package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// WMELE: combat bookkeeping (game states 10: start an encounter, 11: redraw the combat
// screen, 14: end a round). State 12 (WPOPS) picks the actions, 13 (WMEXE) runs them.
// The view behind the monsters is WMAZE's drawMazeView: WMELE's copy (7652..92dc) and
// WMEXE's (fd4c9..ff2b6) are the same code.

// Combat state in the root data segment that the combat overlays share.
// TODO(merge): move to state.go.
var (
	combatOver     int16           // 0x4356 combat_over
	monsterJitterX [2][5]uint8     // g4348: random x offset of the monster at row, col
	monsterJitterY [2][5]uint8     // g433e: random y offset
	combatWinA     *Window         // g438a: the action window (left)
	combatWinB     *Window         // g438c: the monster list window (right)
	g4352          int16           // set by an item of kind 7 (do_use_item_action)
	initiative     int             // g435a: the initiative count of the round, 100 down to 1
	actMember      int             // g435c: the acting combatant's index
	actGroup       int             // g435e: the acting combatant's group
	animCycle      int16           // 0x3644 anim_cycle: 0..2, effect sprites step at 0
	monsterFrames  [5][10][3]uint8 // 0x3648: per view column and frame: frame, step, ticks left
	g4fcc          int16           // 0x4fcc: nonzero, every swing hits (a debug switch)
)

// ROOT data: where a monster at row*5+col moves when the row ahead opens (g00ba, row*10
// + col), its sprite depth (g00b0) and the x jitter of the encounter flash (g00c4).
var (
	viewAdvance   = [10]int8{0, 1, 2, 1, 2, 12, 1, 2, 3, 4}
	viewDepthBase = [10]uint8{25, 30, 30, 20, 20, 10, 15, 15, 5, 5}
	flashShift    = [8]uint8{88, 44, 22, 11, 11, 22, 44, 88}
)

// WMELE data segment (bss 0x500e..).
var (
	meleAnnounce int16       // g500e: show message 2025 before the next encounter flash
	meleNPCFirst int16       // g5010: the encounter came from an NPC; column 0 first, no flash
	meleNewFight int16       // g5012: the combat just started, monsters do not advance yet
	meleFlash    [2][5]int16 // g50b4: flash the monster at view row, col
)

// ovmain (WMELE 4572): start an encounter (10), end a combat round (14) and redraw the
// combat screen (11), in that order, so state 10 runs on into 11.
func wmeleOvmain() {
	meleAnnounce, meleNPCFirst, meleNewFight = 0, 0, 0
	if gameState == 10 {
		beginCombat()
	}
	if gameState == 14 {
		endCombatRound()
	}
	if gameState == 11 {
		redrawCombatScreen()
	}
}

// monsterRecord reads monster record id.
func monsterRecord(id int) MonsterRec {
	b := make([]byte, dbs.Size[MonsterRec]())
	accessScenarioRecord(dbs.RecMonster, id, b, false)
	r, _ := dbs.Decode[MonsterRec](b)
	return r
}

// sidePanel is the side panel showing party member i.
func sidePanel(i int) *Window {
	if i%2 == 0 {
		return g4fba
	}
	return g4fb8
}

// waitAnimatingView (wait_animating_view, WMELE 45a6) animates the view for tenths of
// a second; a click or Enter ends it early.
func waitAnimatingView(tenths int) {
	flushKeys()
	for n := tenths * 10; n > 0; n-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			return
		}
		animateViewStep()
	}
}

// clearCombatGroups (clear_combat_groups, WMELE 47b1) empties the seven groups and the
// view positions.
func clearCombatGroups() {
	combatKeys = getMessage(3900)
	viewPosGroup = [2][5]int16{}
	for g := range 7 {
		*monsterGroups[g] = MonsterGroup{}
	}
	monsterGroupCount = 0
	combatOver = 0
	viewAnimState = -2
}

// findMonsterPicSlot (f4858, WMELE) is the picture slot (4..6) holding monster picture
// mon, loading it into a free slot if none does; -1 if there is no room. Some pictures
// take two slots.
func findMonsterPicSlot(mon int) int {
	wide := 0
	switch mon {
	case 9, 13, 27, 28, 32, 34, 45, 47, 50, 54:
		wide = 1
	}
	free := -1
	for s := 4; 7-wide > s; s++ {
		there, empty := true, true
		for k := 0; k <= wide; k++ {
			if picSlotMonster[s+k] != mon || !picSlotLoaded[s+k] {
				there = false
			}
		}
		if there {
			return s
		}
		for k := 0; k <= wide; k++ {
			if picSlotLoaded[s+k] {
				empty = false
			}
		}
		if empty {
			free = s
		}
	}
	if free != -1 {
		picSlotMonster[free] = -1
		loadMonsterPicSlot(mon, free) // load_monster_pic_mele (46cd) is the same
	}
	return free
}

// generateMonsterGroup (generate_monster_group, WMELE 4965) makes group n of monster mon:
// rolls its size, levels, hit points and stamina, and may add a companion group.
func generateMonsterGroup(mon, n int) {
	monsterGroupCount++
	*groupMonsterRec[n] = monsterRecord(mon)
	rec, grp := groupMonsterRec[n], monsterGroups[n]
	grp.MonsterID = uint8(mon)
	grp.PicSlot = uint8(findMonsterPicSlot(int(rec.Pic)))
	if n == 1 {
		if rec.Sound > 0 {
			loadSound(int(rec.Sound)) // load_sound_mele (45f1) is the same
		} else {
			curSound = -1
		}
	}
	switch hdr.Level {
	case 1:
		if rec.GroupSize.Base > 0 {
			rec.GroupSize.Base--
		}
	case 2:
		rec.GroupSize.Base++
	}
	grp.Count = uint8(rollDice(rec.GroupSize))
	grp.Count = min(max(grp.Count, 1), 9)
	grp.B412 = 1
	grp.B396 = [6]int8{}
	grp.B402 = [4]uint8{}
	if hdr.Level == 2 {
		rec.HP.Count++
	}
	for i := 0; int(grp.Count) > i; i++ {
		c := &grp.Members[i]
		c.Level = int16(rec.HP.Count)
		c.HPMax = int16(rollDice(rec.HP))
		c.HP = c.HPMax
		c.StaminaMax = int16(rollDice(rec.Stamina))
		c.Stamina = c.StaminaMax
		c.Pos = -1
		switch hdr.Level {
		case 1:
			c.B38, c.B39, c.B41 = 2, 2, -5
		case 2:
			c.B38, c.B39, c.B41 = -2, -2, 5
		}
	}
	limit := min((gameStateArg+1)/2+1+b2i(g4338 > 0), 5)
	chance := 100
	switch hdr.Level {
	case 1:
		chance = 150
	case 2:
		chance = 75
	}
	if n >= limit {
		return
	}
	mon = -1
	for i := 0; i < 2 && mon == -1; i++ {
		if rec.CompChance[i] > 0 && random(chance) < int(rec.CompChance[i]) {
			mon = int(rec.Companion[i])
		}
	}
	if mon != -1 {
		generateMonsterGroup(mon, n+1)
	}
}

// fillPartyGroup (fill_party_group, WMELE 4d5b) copies the party into group 0.
func fillPartyGroup() {
	grp := monsterGroups[0]
	grp.Count = uint8(partyCount)
	grp.B396 = [6]int8{}
	if partyCounters[4].A > 0 {
		grp.B396[4] = int8(partyCounters[4].B)
	}
	grp.B402 = [4]uint8{}
	for i := 0; i < partyCount; i++ {
		p, c := &party[i], &grp.Members[i]
		c.Level = p.Level
		c.HPMax, c.HP = p.HPMax, p.HP
		c.StaminaMax, c.Stamina = p.StaminaMax, p.Stamina
		c.Condition = int8(p.Condition)
		c.Status = p.Status
		c.B35[0] = int8(p.PoisonStr)
		c.Pos = -1
	}
}

// picFrameSteps is MonsterPicRec byte 114+f, the step count of frame f (the table runs
// on into Quads for f = 9).
func picFrameSteps(p *MonsterPicRec, f int) int {
	if f < len(p.Unk114) {
		return int(p.Unk114[f])
	}
	return int(p.Quads[0][f-len(p.Unk114)])
}

// chooseMonsterFrames (choose_monster_frames, WMELE 4f02) starts the animation of every
// frame of group g's picture at view column pos.
func chooseMonsterFrames(g, pos int) {
	pic := int(monsterGroups[g].PicSlot)
	p := &monsterPics[pic]
	for i := 0; int(p.Unk309[3]) > i; i++ {
		var frame, step int
		if p.Unk309[0] > 0 && p.Unk309[0] != 255 && picSlotMonster[pic] != 46 && picSlotMonster[pic] != 36 {
			frame, step = int(p.Unk309[0])/25, int(p.Unk309[0])%25
			if picSlotMonster[pic] == 54 && i == 0 {
				frame, step = 0, 0
			}
		} else {
			frame = random(int(p.Unk94[i])) + int(p.Unk84[i])
			step = random(picFrameSteps(p, frame)) + int(p.Unk104[frame])
		}
		f := &monsterFrames[pos][i]
		f[0], f[1] = uint8(frame), uint8(step)
		f[2] = uint8(random(int(p.Quads[step][3])) + int(p.Quads[step][2]))
	}
}

// placeGroupSprites (place_group_sprites, WMELE 5066) moves group g's monsters one view
// row forward and redraws.
func placeGroupSprites(g int) {
	grp := monsterGroups[g]
	for i := 0; int(grp.Count) > i; i++ {
		c := &grp.Members[i]
		pos := int(c.Pos)
		if pos == -1 {
			continue
		}
		row, col := pos/10, pos%10
		viewPosGroup[row][col] = 0
		monsterJitterX[row][col], monsterJitterY[row][col] = 0, 0
		c.Pos = -1
		next := int(viewAdvance[row*5+col])
		nr, nc := next/10, next%10
		if row == 1 && col == 0 {
			nr, nc = 1, random(2)+1
			if viewPosGroup[nr][nc] != 0 {
				nc = nc%2 + 1
			}
		}
		if next == -1 || viewPosGroup[nr][nc] != 0 {
			continue
		}
		viewPosGroup[nr][nc] = int16(g)
		c.Pos = int8(nr*10 + nc)
		if nr == 0 {
			chooseMonsterFrames(g, nc)
		}
	}
	redrawViewBackground()
	waitAnimatingView(4)
}

// updateCombatant (update_combatant, WMELE 51f0) applies the end of a round to fighter i
// of group g: stamina back while asleep, hp by regen less drain and poison, status
// durations down; true if it died.
func updateCombatant(g, i, regen, drain int) bool {
	c := &monsterGroups[g].Members[i]
	died := false
	if c.Status[2] > 0 {
		c.Stamina += int16(random(int(int32(c.StaminaMax)*5/100)) + 1)
		if c.Stamina > c.StaminaMax {
			c.Stamina = c.StaminaMax
		}
	}
	if c.Stamina <= 0 {
		c.Stamina = 0
		c.Status[2] = uint8(random(6) + 6)
	}
	c.HP += int16(regen - drain - int(uint8(c.B35[0])))
	if c.HP > c.HPMax {
		c.HP = c.HPMax
	}
	if c.HP <= 0 {
		c.HP, c.Stamina, c.Condition = 0, 0, 3
		died = true
		if g != 0 {
			if c.Pos != -1 {
				r, cl := c.Pos/10, c.Pos%10
				monsterJitterX[r][cl], monsterJitterY[r][cl] = 0, 0
				viewPosGroup[r][cl] = 0
				c.Pos = -1
			}
			if rec := groupMonsterRec[g]; rec.Flags&1 != 0 {
				setBit(hdr.Unk140[20:], int(rec.Unk211[7]), true) // DS:0x3498
			}
		}
	}
	for s := range c.Status {
		if c.Status[s] > 0 && c.Status[s] != 255 {
			c.Status[s]--
		}
	}
	if c.Status[5] == 0 {
		c.B35[0] = 0
	}
	switch {
	case c.B41 < 0:
		c.B41++
	case c.B41 > 0:
		c.B41--
	}
	if c.B35[1] != 0 {
		c.B35[1]--
	}
	if c.Toggle == 3 {
		c.Toggle = 0
	}
	return died
}

// updateGroup (update_group, WMELE 53ca) ends the round for group g and drops its dead.
func updateGroup(g int) {
	grp := monsterGroups[g]
	n := 0
	for i := 0; int(grp.Count) > i; i++ {
		if uint8(grp.Members[i].Condition) < 3 {
			updateCombatant(g, i, int(groupMonsterRec[g].Unk211[4]), 0)
		}
		if uint8(grp.Members[i].Condition) < 3 {
			grp.Members[n] = grp.Members[i]
			n++
		}
	}
	grp.Count = uint8(n)
}

// checkGroupSurprise (check_group_surprise, WMELE 54d5): each member may spot each
// unidentified group.
func checkGroupSurprise() {
	for g := 1; g <= monsterGroupCount; g++ {
		for m := 0; m < partyCount; m++ {
			roll := random(100)
			if monsterGroups[g].B412 == 1 && (int(party[m].Skills[23])-gameStateArg*3 > roll || int(party[m].Stats[1]) > roll) {
				monsterGroups[g].B412 = 0
				trySkillGain(m, 23, 0) // maybe_raise_skill (5493) is the same
			}
		}
	}
}

// loadGroupPics (load_group_pics, WMELE 5574) finds the pictures of all groups and loads
// group 1's sound.
func loadGroupPics() {
	g4326 = [3]byte{}
	for g := 1; g <= monsterGroupCount; g++ {
		monsterGroups[g].PicSlot = uint8(findMonsterPicSlot(int(groupMonsterRec[g].Pic)))
	}
	if s := groupMonsterRec[1].Sound; s > 0 && int(s) != curSound {
		loadSound(int(s))
	}
}

// compactMonsterGroups (compact_monster_groups, WMELE 55ea) drops the empty groups and
// closes the ranks; with none left the combat is over.
func compactMonsterGroups() {
	count := 0
	for g := 1; g <= monsterGroupCount; g++ {
		updateGroup(g)
		if monsterGroups[g].Count == 0 {
			continue
		}
		count++
		if count != g {
			*monsterGroups[count] = *monsterGroups[g]
			*groupMonsterRec[count] = *groupMonsterRec[g]
			placeGroupSprites(count)
			monsterGroups[g].Count = 0
			continue
		}
		grp := monsterGroups[count]
		for i := 0; int(grp.Count) > i; i++ {
			c := &grp.Members[i]
			if c.Pos == 11 && viewPosGroup[0][3] == 0 {
				viewPosGroup[1][1] = 0
				viewPosGroup[0][3] = int16(count)
				c.Pos = 3
				chooseMonsterFrames(count, 3)
			}
			if c.Pos == 12 && viewPosGroup[0][4] == 0 {
				viewPosGroup[1][2] = 0
				viewPosGroup[0][4] = int16(count)
				c.Pos = 4
				chooseMonsterFrames(count, 4)
			}
		}
	}
	monsterGroupCount = count
	if count == 0 {
		if returnCmd != 8 {
			returnCmd = 15
		}
		combatOver = 1
	}
	loadGroupPics()
	checkGroupSurprise()
}

// advanceMonsterGroups (advance_monster_groups, WMELE 574b): a stronger group behind
// may swap places with the one ahead, announcing it.
func advanceMonsterGroups() {
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	var power [8]int // power[g+1] of group g
	for g := 1; g <= monsterGroupCount; g++ {
		grp, rec := monsterGroups[g], groupMonsterRec[g]
		p := 0
		for i := 0; int(grp.Count) > i; i++ {
			c := &grp.Members[i]
			if c.Condition != 0 || c.Status[2] != 0 || c.Status[3] != 0 {
				continue
			}
			hurt := 0
			for s := range c.Status {
				if c.Status[s] > 0 {
					hurt += int(c.HP / 4)
				}
			}
			if int(c.HP) > hurt {
				p += int(c.HP) - hurt
			}
		}
		if rec.Unk168[0] > 0 {
			p /= int(rec.Unk168[0])/10 + 1
		}
		if rec.Unk168[2] > 0 {
			p = int(int16(int32(p)*int32(rec.Unk168[2])/100 + int32(p)))
		}
		if rec.Unk168[3] > 0 {
			p = int(int16(int32(p)*int32(rec.Unk168[3])/100 + int32(p)))
		}
		if monsterPics[grp.PicSlot].Unk309[0] != 0 {
			p = 0
		}
		power[g+1] = p
	}
	for g := monsterGroupCount; g >= 2; g-- {
		if random(power[g+1]) <= random(power[g]) {
			continue
		}
		clearWindow(partyPictureWin, false)
		refreshAllWindows()
		waitAnimatingView(1)
		grp := monsterGroups[g]
		name := monsterName(g, grp.Count != 1)
		msg := getMessage(3511)
		if grp.Count == 1 {
			msg = getMessage(3510)
		}
		setCursor(partyPictureWin, 20-(len(name)+len(msg)+1)/2, 0)
		printString(partyPictureWin, name, 6)
		printString(partyPictureWin, msg, 14)
		playSound(0)
		ahead := monsterGroups[g-1]
		for i := 0; int(ahead.Count) > i; i++ {
			if pos := ahead.Members[i].Pos; pos != -1 {
				meleFlash[pos/10][pos%10] = 1
			}
		}
		showEncounterView(1)
		for i := 0; int(ahead.Count) > i; i++ {
			c := &ahead.Members[i]
			if c.Pos != -1 {
				viewPosGroup[c.Pos/10][c.Pos%10] = 0
				c.Pos = -1
			}
		}
		redrawViewBackground()
		power[g+1], power[g] = power[g], power[g+1]
		*monsterGroups[g], *monsterGroups[g-1] = *monsterGroups[g-1], *monsterGroups[g]
		*groupMonsterRec[g], *groupMonsterRec[g-1] = *groupMonsterRec[g-1], *groupMonsterRec[g]
		loadGroupPics()
		placeGroupSprites(g - 1)
		arrangeMonsterPositions()
		showEncounterView(0)
	}
}

// monsterName is group g's monster name, plural or not, as identified so far.
func monsterName(g int, plural bool) string {
	return cstr(groupMonsterRec[g].Names[b2i(plural)+int(monsterGroups[g].B412)<<1][:])
}

// copyMemberToCombatant (copy_member_to_combatant, WMELE 5bf3; f5f19 in WMEXE) copies
// fighter i of group 0 back into party member i.
func copyMemberToCombatant(i int) {
	p, c := &party[i], &monsterGroups[0].Members[i]
	p.HP = c.HP
	p.Stamina = c.Stamina
	p.Condition = uint8(c.Condition)
	p.Status = c.Status
	p.PoisonStr = uint8(c.B35[0])
}

// updatePartyAfterRound (update_party_after_round, WMELE 5c7a) ends the round for the
// party, sorts it by condition and ends the combat if all are down or one ran away.
func updatePartyAfterRound() {
	for i := 0; i < partyCount; i++ {
		if uint8(monsterGroups[0].Members[i].Condition) < 3 && updateCombatant(0, i, int(party[i].Regen), int(party[i].Drain)) {
			drawCombatCard(i, false)
			playSound(int(party[i].Sex) + 8)
			setCursor(sidePanel(i), 0, i/2<<2)
			printString(sidePanel(i), memberName(i), -4)
		}
	}
	for i := 0; i < partyCount; i++ {
		copyMemberToCombatant(i)
	}
	for i := 0; partyCount-1 > i; i++ {
		for j := i + 1; j < partyCount; j++ {
			if party[i].Condition > party[j].Condition {
				swapMembers(i, j)
			}
		}
	}
	down, fled := 0, false
	for i := 0; i < partyCount; i++ {
		if party[i].Condition >= 2 {
			down++
		}
		if monsterGroups[0].Members[i].Toggle == 2 {
			fled = true
		}
	}
	if down == partyCount {
		returnCmd = 8
		combatOver = 1
	} else if fled {
		returnCmd = 23
		combatOver = 1
	}
}

// swapMembers exchanges party members i and j with their fighters, roster slots and
// portraits.
func swapMembers(i, j int) {
	party[i], party[j] = party[j], party[i]
	m := &monsterGroups[0].Members
	m[i], m[j] = m[j], m[i]
	memberPcfileIndex[i], memberPcfileIndex[j] = memberPcfileIndex[j], memberPcfileIndex[i]
	memberPortrait[i], memberPortrait[j] = memberPortrait[j], memberPortrait[i]
}

// drawCombatCard (draw_member_panel_mele, WMELE 5fe6; draw_member_card_x, WMEXE 7353)
// copies fighter i back to the party and draws its side panel card; hilite shows the
// name in the highlight colour. The card is WBASE's drawMemberPanel.
func drawCombatCard(i int, hilite bool) {
	copyMemberToCombatant(i)
	drawMemberPanel(i)
	if hilite {
		// The C prints the name highlighted in place of the plain one; drawMemberPanel
		// shows the same frame once this overwrites it.
		setCursor(sidePanel(i), 0, i/2<<2)
		printString(sidePanel(i), memberName(i), -4)
	}
}

// setUpEncounter (start_encounter, WMELE 62ce) rolls the encounter g433a: groups,
// treasure, surprise and the party's group.
func setUpEncounter() {
	combatGold = 0
	if g433a == 128 || g433a == 151 || g433a == 153 {
		g4354 = 1
	}
	clearCombatGroups()
	generateMonsterGroup(g433a, 1)
	treasureID = int16(groupMonsterRec[1].Unk162[g4338])
	checkGroupSurprise()
	fillPartyGroup()
}

// endEncounter (end_encounter, WMELE 6fb6) closes the combat windows and frees the
// combat buffers.
func endEncounter() {
	destroyWindow(combatWinA, false)
	destroyWindow(combatWinB, false)
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
	resumeWindowDraw(g4fb0)
	eraseWindow(mainWindow, false)
	deferWindowDraw(partyPictureWin)
	fillWindow(partyPictureWin, 26, 3, false)
	for r := range 5 {
		setCursor(partyPictureWin, 19, r)
		putCharHi(partyPictureWin, 27, 3)
	}
	resumeWindowDraw(partyPictureWin)
	eraseCursorMark(bottomLineWin, false)
	for g := range 7 {
		groupMonsterRec[g], monsterGroups[g] = nil, nil
	}
	g4326 = [3]byte{}
}

// redrawCombatScreen (redraw_combat_screen, WMELE 70ce) redraws the party and the
// monsters, lets the monsters advance (not in the first round) and goes on to the
// action menu.
func redrawCombatScreen() {
	for i := 0; i < partyCount; i++ {
		drawCombatCard(i, false)
	}
	arrangeMonsterPositions()
	showEncounterView(0)
	if meleNewFight == 0 {
		advanceMonsterGroups()
	}
	meleNewFight = 0
	gameState = 12
}

// tickCombatEffects (tick_combat_effects, WMELE 7113) runs down the groups' timed
// effects after a round.
func tickCombatEffects() {
	for g := range 7 {
		grp := monsterGroups[g]
		for k := range 6 {
			v := int(uint8(grp.B396[k]))
			if v <= 0 {
				continue
			}
			var d int
			switch k {
			case 0, 1:
				d = v / 10
			case 2, 3:
				d = v / 12
			case 4:
				d = v / 20
			case 5:
				d = 1
			}
			v -= max(d, 1)
			grp.B396[k] = int8(max(v, 0))
		}
		for k := range grp.B402 {
			if grp.B402[k] > 0 {
				grp.B402[k]--
			}
		}
	}
	if partyCounters[4].A > 0 {
		partyCounters[4].B = int16(uint8(monsterGroups[0].B396[4]))
	}
	if partyCounters[4].B <= 0 {
		partyCounters[4].A, partyCounters[4].B = 0, 0
	}
}

// endCombatRound (end_combat_round, WMELE 724f) finishes a round; when the combat is
// over it leaves for return_cmd, else for the next round (11).
func endCombatRound() {
	updatePartyAfterRound()
	compactMonsterGroups()
	updateGroup(6)
	tickCombatEffects()
	fillWindow(combatWinA, 32, 3, false)
	clearWindow(combatWinB, false)
	clearWindow(partyPictureWin, false)
	if combatOver == 1 {
		for i := 0; i < partyCount; i++ {
			drawCombatCard(i, false)
		}
		endEncounter()
		gameState = int(returnCmd)
		return
	}
	gameState = 11
	refreshAllWindows()
}

// beginCombat (f72d1, WMELE) allocates the combat buffers and windows, rolls the
// encounter and announces an NPC's companions.
func beginCombat() {
	meleNewFight = 1
	g4352 = 0
	for g := range 7 {
		groupMonsterRec[g] = new(MonsterRec)
		monsterGroups[g] = new(MonsterGroup)
	}
	for s := range wpopsSprites {
		wpopsSprites[s] = Sprite{Depth: 255}
	}
	g4326 = [3]byte{}
	monsterJitterX[0], monsterJitterY[0] = [5]uint8{}, [5]uint8{}
	deferWindowDraw(g4fb0)
	combatWinA = createWindow(0, 19, 19, 5, 11, -4, 0, false)
	combatWinB = createWindow(20, 19, 20, 5, 10, -1, 0, false)
	g4fae = nil
	deferWindowDraw(combatWinB)
	deferWindowDraw(combatWinA)
	clearWindow(partyPictureWin, false)
	setUpEncounter()
	meleAnnounce, meleNPCFirst = 1, 0
	if returnCmd == 18 {
		meleAnnounce, meleNPCFirst = 0, 1
	}
	if returnCmd == 18 && monsterGroupCount > 1 {
		deferWindowDraw(mainWindow)
		deferWindowDraw(bottomLineWin)
		clearWindow(partyPictureWin, false)
		refreshAllWindows()
		waitAnimatingView(1)
		name, msg := cstr(groupMonsterRec[1].Names[0][:]), getMessage(3512)
		setCursor(partyPictureWin, 20-(len(name)+len(msg)+1)/2, 0)
		printString(partyPictureWin, name, 6)
		printString(partyPictureWin, msg, 14)
		playSound(0)
	}
	updatePartyAfterRound()
	gameState = 11
}
