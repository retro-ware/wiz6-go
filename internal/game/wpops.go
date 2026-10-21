package game

import "github.com/retro-ware/wiz6-go/internal/drv"

// WPOPS: the start of a combat round (game state 12). Each able party member picks an
// action and a target, the monsters choose theirs, and the game goes on to WMEXE (13).
// The overlay's maze-view copies are WMAZE's (drawMazeView and its helpers).

// ovmain (WPOPS 4572).
func wpopsOvmain() {
	if gameState == stateStartCombatRound {
		startCombatRound()
	}
}

// ROOT data g0092: the view columns each party member attacks first, by member.
var memberColumns = [6][5]int{{1, 0, 3, 2, 4}, {2, 0, 4, 1, 3}, {0, 1, 2, 3, 4}, {4, 0, 2, 3, 1}, {3, 0, 1, 4, 2}, {0, 2, 1, 4, 3}}

// WPOPS data and bss.
var (
	// g5043..: monster taken as a target, at group*9+index (group 1..5). The back
	// commands mark at 9 more (the C's g504c), so they flag the next group.
	wpopsTargetTaken [64]uint8
)

// clearTargets is setmem(g504c, 45, 0).
func clearTargets() { clear(wpopsTargetTaken[9:54]) }

// combatantAble is the C's "condition < 2 and neither status 2 nor 3".
func combatantAble(c *Combatant) bool {
	return uint8(c.Condition) < conditionDown && c.Status[statusParalyzed] == 0 && c.Status[statusAsleep] == 0
}

// groupName is the group's monster name, singular or plural, identified or not.
func groupName(g int) string {
	grp := monsterGroups[g]
	n := 1
	if grp.Count == 1 {
		n = 0
	}
	return cstr(groupMonsterRec[g].Names[int(grp.Unidentified)<<1+n][:])
}

// drawCombatTitle (WPOPS 4580).
func drawCombatTitle() {
	deferWindowDraw(mainWindow)
	setCursor(mainWindow, 22, 0)
	printMessage(3503, mainWindow, 18)
	resumeWindowDraw(mainWindow)
}

// printCombatPrompt (WPOPS 45b5) centres text, extra columns wider, in the main window.
func printCombatPrompt(text string, extra int) {
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	drawCombatTitle()
	deferWindowDraw(mainWindow)
	setCursor(mainWindow, 10-(len(text)+extra+1)/2, 0)
	printStringHi(mainWindow, text, 3)
	resumeWindowDraw(mainWindow)
}

// drawMonsterGroups (WPOPS 4623) lists the groups: count, name and how many can fight.
func drawMonsterGroups(win *Window, refresh bool) {
	drawCombatTitle()
	deferWindowDraw(win)
	clearWindow(win, true)
	for g := 1; g <= monsterGroupCount; g++ {
		grp := monsterGroups[g]
		grp.B414 = 0
		for i := 0; i < int(grp.Count); i++ {
			if combatantAble(&grp.Members[i]) {
				grp.B414++
			}
		}
		setCursor(win, 0, g-1)
		putChar(win, grp.Count+'0', 6)
		win.CurX++
		printString(win, groupName(g), 14)
		setCursor(win, 19, g-1)
		putChar(win, grp.B414+'0', 9)
	}
	resumeWindowDraw(win)
	if refresh {
		refreshWindow(mainWindow)
		refreshWindow(win)
	}
}

// drawAllyGroup (WPOPS 478f) shows group 6, the party's allies, under the view.
func drawAllyGroup() {
	w := allyStripWin
	deferWindowDraw(w)
	fillWindow(w, 28, 3)
	setCursor(w, 4, 0)
	printMessage(3516, w, 18)
	clearWindowArea(w, 0, 1, 21, 1)
	grp := monsterGroups[allyGroup]
	grp.B414 = 0
	for i := 0; i < int(grp.Count); i++ {
		if combatantAble(&grp.Members[i]) {
			grp.B414++
		}
	}
	setCursor(w, 1, 1)
	putChar(w, grp.Count+'0', 4)
	w.CurX++
	printString(w, groupName(6), 12)
	setCursor(w, 20, 1)
	putChar(w, grp.B414+'0', 9)
	resumeWindowDraw(w)
}

// animRunLength is Unk114[seq] of the picture; seq 9 runs into the next field.
func animRunLength(p *MonsterPicRec, seq int) int {
	if seq < len(p.Unk114) {
		return int(p.Unk114[seq])
	}
	return int(p.Quads[0][0])
}

// buildMonsterAnimFrames (build_monster_anim_frames, WPOPS 48e4; WMEXE 5fa0, WMELE
// 6326) steps the animation of the group's monster shown in view column col and returns
// its frames, one per picture layer. A dead, asleep or paralysed monster stays still.
func buildMonsterAnimFrames(g, col int) []byte {
	grp := monsterGroups[g]
	slot := int(grp.PicSlot)
	p := &monsterPics[slot]
	m := 0
	for i := 0; i < int(grp.Count); i++ {
		if int(grp.Members[i].Pos) == col {
			m = i
		}
	}
	c := &grp.Members[m]
	var frames []byte
	for layer := 0; layer < int(p.Unk309[3]); layer++ {
		st := &monsterFrames[col][layer]
		var frame int
		if c.Condition == conditionDown || c.Status[statusParalyzed] > 0 || c.Status[statusAsleep] > 0 {
			frame = int(p.SeqStartFrame[st[0]])
		} else {
			frame = int(st[1])
			ticks := int(st[2]) - 1
			if ticks <= 0 {
				seq := int(st[0])
				frame++
				if int(p.SeqStartFrame[seq])+animRunLength(p, seq)-1 < frame {
					seq = random(int(p.SeqCount[layer])) + int(p.SeqFirst[layer])
					st[0] = uint8(seq)
					frame = int(p.SeqStartFrame[seq])
				}
				st[1] = uint8(frame)
				ticks = random(int(p.Quads[frame][3])) + int(p.Quads[frame][2])
			}
			st[2] = uint8(ticks)
		}
		f := p.Quads[frame][1]
		if picSlotMonster[slot] == 40 && layer == 4 {
			f = 0
		}
		if f != 0 {
			frames = append(frames, f)
		}
	}
	return frames
}

// insertSprite (insert_sprite, WPOPS 4b7d; WMEXE 6239, WMELE 65bf) adds a sprite,
// keeping the list sorted by depth.
func insertSprite(depth, pic, x, y, clipL, clipR, mirror int, frames []byte) {
	pos := 0
	for pos < int(spriteCount) && int(spriteList[pos].Depth) <= depth {
		pos++
	}
	for i := int(spriteCount); i > pos; i-- {
		spriteList[i] = spriteList[i-1]
	}
	if pos >= 12 {
		for {
			playSound(soundBeep) // the C hangs here
		}
	}
	s := &spriteList[pos]
	s.Depth, s.Pic = uint8(depth), uint8(pic)
	s.X, s.Y, s.W, s.H = int16(x), int16(y), int16(clipL), int16(clipR)
	s.Mirror = uint8(mirror)
	s.Layers = [12]int8{}
	for i, f := range frames {
		s.Layers[i] = int8(f)
	}
	spriteCount++
}

// queueMonsterSpriteNoShift (queue_monster_sprite, WPOPS 4c7d) is WMELE's queueMonsterSprite
// without the dx shift, and it takes any nonzero group, not only 1..3.
func queueMonsterSpriteNoShift(row, col int) {
	g := int(viewPosGroup[row][col])
	if g == 0 || row > int(viewRowsOpen) {
		return
	}
	slot := int(monsterGroups[g].PicSlot)
	p := &monsterPics[slot]
	var frames []byte
	if row == 0 {
		frames = buildMonsterAnimFrames(g, col)
	} else if p.Unk306[0] != 0 {
		frames = []byte{p.Unk306[0]}
	}
	if len(frames) == 0 {
		return
	}
	mirror := col % 2
	if random(10) == 9 {
		monsterJitterX[row][col] = uint8(random(int(p.Unk309[1])))
		monsterJitterY[row][col] = uint8(random(int(p.Unk309[2])))
	}
	f := int(frames[0])
	depth := int(p.Quads[f][0])
	if depth == 0 {
		depth = int(viewDepthBase[row*5+col]) - g
	}
	x := int(p.Offsets[row][col][0]) + int(monsterJitterX[row][col]) + int(p.FrameX[f][mirror])
	if x < 0 {
		x = 0
	} else if x >= 248 {
		return
	}
	y := int(p.Offsets[row][col][1]) + int(monsterJitterY[row][col]) + int(p.FrameY[f])
	insertSprite(depth, slot, x, y, 72, 248, mirror, frames)
}

// drawSprite (draw_sprite, WPOPS 4ebc) draws sprite i into the working buffer. The
// pictures in slots of monsters 20, 35, 54 and 57 place each layer by its own offsets.
func drawSprite(i int) {
	s := &spriteList[i]
	if s.Depth == 255 {
		delayTick()
		delayTick()
		return
	}
	slot := int(s.Pic)
	switch picSlotMonster[slot] {
	case 20, 35, 54, 57:
		p := &monsterPics[slot]
		mir := int(s.Mirror) % 2
		first := uint8(s.Layers[0])
		for j := 0; s.Layers[j] != 0; j++ {
			f := uint8(s.Layers[j])
			if picSlotMonster[slot] == 54 && f == 8 {
				s.Layers[j] = 9
				j--
			}
			x := int(s.X) - int(p.FrameX[first][mir]) + int(p.FrameX[f][mir])
			y := int(s.Y) - int(p.FrameY[first]) + int(p.FrameY[f])
			drv.DrawFrames(slot, x, y, int(s.W), int(s.H), int(s.Mirror), false, []byte{f})
		}
	default:
		var frames []byte
		for _, f := range s.Layers {
			if f == 0 {
				break
			}
			frames = append(frames, uint8(f))
		}
		drv.DrawFrames(slot, int(s.X), int(s.Y), int(s.W), int(s.H), int(s.Mirror), false, frames)
	}
}

// animateCombatStep (WPOPS 511c) is one tick of the combat view: -3 shows it, -2 draws
// the maze, -1 queues the monster sprites, 0..11 draw sprite n.
func animateCombatStep() {
	switch viewAnimState {
	case -3:
		drvShowWindow(pictureWin)
		animPhase = (animPhase + 1) % 2
	case -2:
		drawMazeView()
		for i := range spriteList {
			spriteList[i].Depth = 255
		}
		spriteCount = 0
	case -1:
		for r := range 2 {
			for c := range 5 {
				queueMonsterSpriteNoShift(r, c)
			}
		}
	default:
		drawSprite(viewAnimState)
	}
	viewAnimState++
	if viewAnimState == 12 {
		viewAnimState = viewAnimShow
	}
}

// animateUntilReset (animate_until_reset, WPOPS 51c8; WMEXE 773a, WMELE 6e01) runs a
// whole animation cycle.
func animateUntilReset() {
	viewAnimState = viewAnimMaze
	for {
		animateCombatStep()
		if viewAnimState == viewAnimMaze {
			break
		}
	}
}

// getCombatInput (WPOPS 51d9) waits for a combat key (its index) or a click (its region,
// with menuClicked set), animating the view and blinking the cursor meanwhile.
func getCombatInput() int {
	flushKeys()
	phase := animPhase
	if pvBlinkState != -1 {
		blinkCursor()
	}
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if k := charIndex(combatKeys, getKey(1)); k >= 0 {
				return k
			}
			continue
		}
		r := checkClickRegions(clickRegions[:])
		if clickFlag {
			menuClicked = 1
			return r - 1
		}
		paceViewCycle()
		animateCombatStep()
		if viewAnimState == viewAnimMaze && pvBlinkState != -1 && animPhase == phase {
			blinkCursor()
		}
	}
}

// selectMonsterGroup (WPOPS 55cd) asks for a target group within range (0, 1 the first
// two, 2 three, 3 all five) in the monster list; the group (1..), or -1 for cancel.
func selectMonsterGroup(win *Window, msg, rng, sel int) int {
	if monsterGroupCount == 1 {
		return 1
	}
	list := monsterListWin
	fillWindow(win, 32, 3)
	refreshAllWindows()
	title := replaceCharWithString(getMessage(3600), getMessage(msg), '$')
	setCursor(win, 10-(len(title)+1)/2, 1)
	printStringHi(win, title, 3)
	cancel := getMessage(3606)
	setCursor(win, 7, 3)
	printStringHi(win, cancel, 3)
	count := 5
	switch rng {
	case 0, 1:
		count = 2
	case 2:
		count = 3
	case 3:
		count = 5
	}
	count = min(count, monsterGroupCount)
	for i := range count {
		setClickRegion(i, list.X, list.Y+byte(i), list.X+19, list.Y+byte(i))
	}
	setClickRegion(count, win.X+7, win.Y+3, byte(len(cancel)+int(win.X)+6), win.Y+3)
	setClickRegion(count+1, 255, 0, 0, 0)
	cancelOn := func() {
		setCursor(win, 7, 3)
		printStringHi(win, cancel, 3)
	}
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(win, 7, 3)
				printString(win, cancel, -7)
			} else {
				grp := monsterGroups[sel+1]
				setCursor(list, 0, sel)
				putChar(list, grp.Count+'0', -7)
				putChar(list, ' ', -7)
				printString(list, groupName(sel+1), -7)
				for list.CurX < 19 {
					putChar(list, ' ', -7)
				}
				putChar(list, grp.B414+'0', -7)
			}
		}
		k := getCombatInput()
		if menuClicked == 1 {
			sel = -1
			if k < count {
				sel = k
			}
			break
		}
		switch k {
		case 1:
			if sel != -1 {
				drawMonsterGroups(list, true)
				sel = -1
			}
		case 2:
			if sel == -1 {
				cancelOn()
				sel = 0
			} else {
				drawMonsterGroups(list, true)
				if sel-1 >= 0 {
					sel--
				} else {
					sel = -1
				}
			}
		case 3:
			if sel == -1 {
				cancelOn()
				sel = 0
			}
		case 4:
			if sel == -1 {
				cancelOn()
				sel = 0
			} else {
				drawMonsterGroups(list, true)
				if sel+1 < count {
					sel++
				} else {
					sel = -1
				}
			}
		}
		if k == 5 {
			break
		}
	}
	if sel == -1 {
		return -1
	}
	return sel + 1
}

// selectPartyMember (WPOPS 5a33) asks for a party member with prompt msg; the member,
// or -1 for cancel (0 with one member).
func selectPartyMember(msg int) int {
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, -4, 0, false)
	fillWindow(win, 32, 3)
	clickText, keyText := getMessage(3607), getMessage(3608)
	printCombatPrompt(getMessage(msg), 5)
	mainWindow.CurX++
	cx, cy := int(mainWindow.CurX), int(mainWindow.CurY)
	refreshWindow(mainWindow)
	refreshWindow(win)
	setCursor(mainWindow, cx, cy)
	if mousePresent {
		printStringHi(mainWindow, clickText, 3)
	} else {
		printString(mainWindow, keyText, -5)
	}
	for m := 0; m < partyCount; m++ {
		x, y := memberCell(m)
		setCursor(win, x, y)
		printStringHi(win, memberName(m), 3)
		setClickRegion(m, win.X+byte(x), win.Y+byte(y), byte(len(memberName(m))+int(win.X)+x-1), win.Y+byte(y))
	}
	setClickRegion(partyCount, mainWindow.X+byte(cx), mainWindow.Y+byte(cy),
		byte(len(keyText)+int(mainWindow.X)+cx-1), mainWindow.Y+byte(cy))
	setClickRegion(partyCount+1, 255, 0, 0, 0)
	sel := -1
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, cx, cy)
				printString(mainWindow, keyText, -5)
			} else {
				x, y := memberCell(sel)
				setCursor(win, x, y)
				printString(win, memberName(sel), -5)
			}
		}
		k := getCombatInput()
		if menuClicked == 1 {
			sel = -1
			if k < partyCount {
				sel = k
			}
			break
		}
		switch k {
		case 1:
			if sel != -1 && sel%2 > 0 {
				drawMemberName(win, sel)
				sel--
			}
		case 2:
			if sel != -1 {
				drawMemberName(win, sel)
				if sel-2 >= 0 {
					sel -= 2
				} else {
					sel = -1
				}
			}
		case 3:
			if sel != -1 && sel%2 < 1 && sel+1 < partyCount {
				drawMemberName(win, sel)
				sel++
			}
		case 4:
			if sel == -1 {
				setCursor(mainWindow, cx, cy)
				printStringHi(mainWindow, clickText, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if k == 5 {
			break
		}
	}
	destroyWindow(win, false)
	return sel
}

// pickTargetMonster (WPOPS 5dbb) picks the monster of group the member attacker hits:
// one not yet taken, searching the view columns in the attacker's order.
func pickTargetMonster(attacker, group int) int {
	grp := monsterGroups[group]
	unplaced := true
	for i := 0; i < int(grp.Count); i++ {
		if grp.Members[i].Pos != -1 {
			unplaced = false
		}
	}
	tries := 0
	if unplaced {
		for {
			for i := 0; i < int(grp.Count); i++ {
				if wpopsTargetTaken[group*9+i] == 0 || tries == attacker {
					wpopsTargetTaken[group*9+i] = 1
					return i
				}
				tries++
			}
		}
	}
	for pass := 0; pass < 10; pass++ {
		for row := range 2 {
			for _, col := range memberColumns[attacker] {
				if int(viewPosGroup[row][col]) != group {
					continue
				}
				for i := 0; i < int(grp.Count); i++ {
					if row*10+col != int(grp.Members[i].Pos) {
						continue
					}
					if wpopsTargetTaken[group*9+i] == 0 || tries == attacker {
						wpopsTargetTaken[group*9+i] = 1
						return i
					}
					tries++
				}
			}
		}
	}
	for {
		playSound(soundBeep) // the C hangs here
	}
}

// printOptionAt (WPOPS 6faa) prints s at option slot i (five rows per column of 20).
func printOptionAt(win *Window, i int, s string, color int) {
	setCursor(win, i/5*20+2, i%5)
	printString(win, s, color)
}

// pickRandomSetBit (WPOPS 7696) counts 1..8 set bits of the byte round from bit 0 and
// returns where it stops (after 100 clear bits at the latest).
func pickRandomSetBit(bits uint8) int {
	b := []byte{bits}
	bit, clearSeen := -1, 0
	n := random(8) + 1
	for {
		bit = (bit + 1) % 8
		if testBit(b, bit) {
			n--
		} else {
			clearSeen++
		}
		if n <= 0 || clearSeen >= 100 {
			return bit
		}
	}
}

// pcItemType is the type of member m's item i; the C reads past the inventory for
// 255 (none), here that is type 0.
// ponytail: an out-of-range index reads as type 0, the C reads whatever follows.
func pcItemType(m, i int) uint8 {
	if i >= len(party[m].Items) {
		return 0
	}
	return party[m].Items[i].Type
}

// Party action menu entries past the action kinds.
const (
	menuBack      = 9 // back one member
	menuStartOver = 10
)

// choosePartyActions (WPOPS 76e9) sets each member's swings, then asks every able
// member for an action and target (fight, parry, ..., back, start over) and confirms.
func choosePartyActions(win *Window) {
	for m := 0; m < partyCount; m++ {
		p, c := &party[m], &monsterGroups[0].Members[m]
		power := max(random(10)+int(p.AttackPower)+int(c.B41), 1)
		step := power / int(p.SecondaryAttacks)
		if uint8(c.B35[1]) > 0 {
			c.B35[2] = int8(random(power/2) + random(20) + power/2)
		}
		c.Swing, c.Plan[planSlot], c.Plan[planSlot+1] = 0, 0, 1
		swings := min(max(int(p.SecondaryAttacks)+int(c.B41)/10, 1), 4)
		for i := range 4 {
			if i >= swings || p.Condition >= conditionDown || p.Status[statusParalyzed] > 0 || p.Status[statusAsleep] > 0 {
				c.Plan[i] = 0
			} else {
				c.Plan[i] = int8(power)
				power -= step
			}
		}
	}
	clearTargets()
	done := 0
	var doneMembers [6]int
	var enabled [12]int // 0..10 the actions; [11] ends the list
	for member := 0; member < partyCount; {
		p := &party[member]
		if p.Condition < conditionDown && p.Status[statusParalyzed] == 0 && p.Status[statusAsleep] == 0 {
			doneMembers[done] = member
			done++
			c := &monsterGroups[0].Members[member]
			for i := range 11 {
				enabled[i] = 1
			}
			if p.Race != raceDracon {
				enabled[actionBreath] = 0
			}
			if c.Stance != stanceNone || p.Class != classThief && p.Class != classNinja && p.Class != classRanger && p.Class != classBard && p.Class != classMonk && p.Skills[skillHide] == 0 {
				enabled[actionHide] = 0
			}
			if p.ItemCount == 0 {
				enabled[actionUseItem] = 0
			}
			if done < 2 {
				enabled[menuBack], enabled[menuStartOver] = 0, 0
			}
			if e := int(p.Equipped[equipPrimary]); e != noEquippedItem && p.Items[e].Type == itemTypeLauncher {
				ammo := int(p.Equipped[equipSecondary])
				if ammo == noEquippedItem || p.Items[ammo].Type != itemTypeMissile {
					enabled[actionAttack] = 0
				} else {
					bow, arrow := p.Items[e].Icon, p.Items[ammo].Icon
					if bow == 11 && arrow != 22 || bow == 13 && arrow != 23 || bow == 12 && arrow != 83 {
						enabled[actionAttack] = 0
					}
				}
			}
			if member >= 3 && p.Hands[0].B0D == 0 && c.Stance != stanceHidden {
				enabled[actionAttack] = 0
			}
			enabled[11] = -1
			name := copyName(memberName(member))
			prompt := func(msg int) {
				fillWindow(win, 32, 3)
				printCombatPrompt(replaceCharWithString(getMessage(msg), name, '$'), 0)
			}
			var action, choice, target int
			back := false
			for {
				drawMonsterGroups(monsterListWin, false)
				prompt(3500)
				refreshAllWindows()
				n := 0
				for i := 0; enabled[i] != -1; i++ {
					if enabled[i] == 1 {
						n++
					}
				}
				rows := 3
				if n > 9 {
					rows = 4
				}
				action = menuSelect(win, 3000, enabled[:], 1, 1, 6, rows, 5, 0, getCombatInput)
				choice = -1
				c.TargetGroup, c.TargetMember = -1, 0
				retry := false
				switch action {
				case actionAttack:
					prompt(3501)
					weapon := int(p.Equipped[equipPrimary])
					kinds := 0
					var attacks [9]int
					if weapon != noEquippedItem && p.Items[weapon].Type == itemTypeLauncher {
						kinds, choice = 1, 8
					}
					if choice == -1 {
						for i := range 8 {
							attacks[i] = b2i(testBit([]byte{p.Hands[0].Flags}, i))
							if i != 4 && member >= 3 && weapon != noEquippedItem && p.Items[weapon].Type == 2 {
								attacks[i] = 0
							}
							if attacks[i] == 1 {
								kinds++
								choice = i
							}
						}
						attacks[8] = -1
					}
					if kinds > 1 {
						refreshAllWindows()
						choice = menuSelect(win, 3050, attacks[:], 3, 1, 8, 3, 5, 0, getCombatInput)
					}
					if choice != -1 {
						rng := int(p.Hands[0].B0D)
						if rng == 2 && choice != 4 {
							rng = 0
						}
						if rng == 0 && member >= 3 && c.Stance != stanceHidden {
							playSound(soundBeep)
							target = -1
						} else {
							target = selectMonsterGroup(win, 3602, rng, 0)
						}
					}
					if choice == -1 || target == -1 {
						retry = true
					} else {
						c.TargetGroup = int8(target)
					}
				case actionDefend: // parry and the like
					prompt(3502)
					refreshAllWindows()
					for i := range 4 {
						enabled[i] = 1
					}
					if pcItemType(member, int(p.Equipped[equipSecondary])) != itemTypeShield {
						enabled[0] = 0 // the shield choice
					}
					enabled[4] = -1 // the C leaves it so: a retry shows only four actions
					if choice = menuSelect(win, 3100, enabled[:], 3, 1, 9, 2, 5, 0, getCombatInput); choice == -1 {
						retry = true
					}
				case actionCast:
					choice = chooseCombatSpell(win, member)
					pvBlinkState = -1
					retry = choice == -1
				case actionUseItem:
					choice = chooseCombatItem(win, member)
					retry = choice == -1
				case actionMove:
					choice = selectPartyMember(3611)
					retry = choice == -1
				case actionBreath:
					target = selectMonsterGroup(win, 3605, 1, 0)
					choice = 8
					if target == -1 {
						retry = true
					} else {
						c.TargetGroup = int8(target)
					}
				case menuBack:
					clearTargets()
					for i := 0; member-1 > i; i++ {
						markTarget(i)
					}
					member = doneMembers[done-2] - 1
					done -= 2
					back = true
				case menuStartOver:
					clearTargets()
					member = -1
					done = 0
					back = true
				}
				if !retry {
					break
				}
			}
			if !back {
				c.Choice = int16(choice)
				c.SecondAttack = -1
				c.Plan[planAction] = int8(action)
				if target = int(uint8(c.TargetGroup)); target >= 1 && target <= 5 {
					c.TargetMember = int8(pickTargetMonster(member, target))
				}
				ammo := int(p.Equipped[equipSecondary])
				if action == actionAttack {
					if ammo == noEquippedItem && p.Equipped[equipPrimary] == noEquippedItem {
						c.SecondAttack = int16(pickRandomSetBit(p.Hands[0].Flags))
					} else if ammo != noEquippedItem {
						switch t := p.Items[ammo].Type; {
						case t == 2 || t == 12 || t == 16:
							c.SecondAttack = 4
						case t < 2 && (p.Hands[1].B0D == 1 || member < 3 && target < 3):
							var b int
							for tries := 0; ; {
								b = pickRandomSetBit(p.Hands[1].Flags)
								tries++
								if b != 4 || tries >= 10 {
									break
								}
							}
							if b != 4 {
								c.SecondAttack = int16(b)
							}
						}
					}
				}
			}
		}
		member++
		if member == partyCount {
			drawMonsterGroups(monsterListWin, false)
			fillWindow(win, 32, 3)
			printCombatPrompt(getMessage(3506), 0)
			refreshAllWindows()
			ok := []int{1, 1, 1, -1}
			if done == 0 {
				ok[1], ok[2] = 0, 0
			}
			switch menuSelect(win, 3507, ok, 3, 1, 14, 4, 5, 0, getCombatInput) {
			case 2:
				clearTargets()
				member, done = 0, 0
			case 1:
				clearTargets()
				for i := 0; partyCount-1 > i; i++ {
					markTarget(i)
				}
				member = doneMembers[done-1]
				done--
			}
		}
	}
	for m := 0; m < partyCount; m++ {
		c := &monsterGroups[0].Members[m]
		if c.Status[1] > 0 && random(100) > int(party[m].Stats[statPersonality])+80 {
			c.Plan[planAction] = actionRun
		}
		if c.Plan[planAction] == actionDefend || c.Plan[planAction] == actionMove {
			c.Plan[0] = 1
		}
		if c.Plan[planAction] == actionAttack {
			if c.SecondAttack == -1 {
				c.Plan[1], c.Plan[3] = 0, 0
			}
		} else {
			c.Plan[1], c.Plan[2], c.Plan[3] = 0, 0, 0
		}
	}
}

// markTarget re-marks member m's target after a back command (at the C's g504c,
// one group past where pickTargetMonster marks it).
func markTarget(m int) {
	c := &monsterGroups[0].Members[m]
	if uint8(c.TargetGroup) > 0 {
		wpopsTargetTaken[9+int(uint8(c.TargetGroup))*9+int(uint8(c.TargetMember))] = 1
	}
}

// findMostWounded (WPOPS 83ab) aims monster at the most wounded of group (each better
// one taken at even odds); the damage found.
func findMostWounded(monster *Combatant, group int) int {
	best := 0
	grp := monsterGroups[group]
	for i := 0; i < int(grp.Count); i++ {
		if d := int(grp.Members[i].HPMax - grp.Members[i].HP); d > best && random(2) == 0 {
			monster.TargetGroup, monster.TargetMember = int8(group), int8(i)
			best = d
		}
	}
	return best
}

// pickRandomRow (WPOPS 843b) picks one of the group's attacks at random.
func pickRandomRow(g int) int {
	for i := 0; ; i = (i + 1) % 3 {
		if groupMonsterRec[g].AttackKind[i] != 0 && random(2) != 1 {
			return i
		}
	}
}

// attackBits is the attack's effect bits; attack kinds 2 and 3 are fixed at 4 and 8.
func attackBits(rec *MonsterRec, a int) int16 {
	switch rec.AttackKind[a] {
	case 2:
		return 4
	case 3:
		return 8
	}
	return int16(pickRandomSetBit(rec.Attacks[a].Effect))
}

// chooseGroupActions (WPOPS 8480) chooses action and target for each monster of group
// against targetGroup: a spell, a breath, or an attack within reach.
func chooseGroupActions(group, targetGroup int) {
	grp, rec := monsterGroups[group], groupMonsterRec[group]
	for i := 0; i < int(grp.Count); i++ {
		m := &grp.Members[i]
		power := max(random(10)+int(m.B41)+int(rec.Unk211[0]), 1)
		step := power / int(rec.Unk211[2])
		if rec.Flags&2 != 0 {
			m.B35[1] = 1
		}
		if uint8(m.B35[1]) > 0 {
			m.B35[2] = int8(random(power/2) + random(20) + power/2)
		}
		m.Swing = 0
		m.Plan[planSlot] = int8(pickRandomRow(group))
		for n := 0; ; {
			m.Plan[planSlot+1] = int8(pickRandomRow(group))
			n++
			if m.Plan[planSlot+1] != m.Plan[planSlot] || n >= 3 {
				break
			}
		}
		for s := range 4 {
			if int(rec.Unk211[2]) <= s || uint8(m.Condition) >= conditionDown || m.Status[statusParalyzed] > 0 || m.Status[statusAsleep] > 0 {
				m.Plan[s] = 0
			} else {
				m.Plan[s] = int8(power)
				power -= step
			}
		}
		act := -1
		m.TargetGroup = int8(targetGroup)
		for {
			m.TargetMember = int8(random(int(monsterGroups[targetGroup].Count)))
			if uint8(monsterGroups[uint8(m.TargetGroup)].Members[uint8(m.TargetMember)].Condition) < conditionDead {
				break
			}
		}
		if random(100) < int(rec.Unk168[0]) {
			act = actionCast
			m.Plan[planAction] = actionCast
			roll, limit, spell := random(100), 5, -1
			for k := 0; spell == -1; k++ {
				if roll < limit {
					spell = int(rec.Unk138[k])
				}
				switch {
				case limit < 15:
					limit += 5
				case limit < 55:
					limit += 10
				default:
					limit += 15
				}
			}
			if spellTable[spell].Target == spellTargetMember {
				if group == allyGroup {
					if findMostWounded(m, 0) == 0 && findMostWounded(m, allyGroup) == 0 {
						act = -1
					}
				} else if findMostWounded(m, group) == 0 && findMostWounded(m, random(monsterGroupCount)+1) == 0 {
					act = -1
				}
			}
			lv := min(max(int(m.Level)-int(spellTable[spell].Level)+1, 1), 6)
			for random(int(m.Level))+1 < lv {
				lv--
			}
			m.Choice = int16(lv<<8 + spell)
		}
		if act == -1 && random(100) < int(rec.Unk168[2]) {
			act = actionBreath
			m.Plan[planAction] = actionBreath
			m.Choice = int16(rec.Unk211[5])
		}
		if act == -1 {
			reach := 0
			if uint8(m.Condition) < conditionDown && m.Status[statusParalyzed] == 0 && m.Status[statusAsleep] == 0 {
				a := int(uint8(m.Plan[planSlot]))
				switch {
				case group == allyGroup:
					reach = 9
				case group >= 3 && rec.Attacks[a].Range <= 1:
				default:
					reach = 3
					far := 0
					if group >= 3 {
						far = 2
					}
					if far < int(rec.Attacks[a].Range) {
						reach = 6
					}
				}
			}
			tg := monsterGroups[targetGroup]
			reach = min(reach, int(tg.Count))
			alive := 0
			for k := 0; k < reach; k++ {
				if uint8(tg.Members[k].Condition) < conditionDead {
					alive++
				}
			}
			if alive > 0 {
				m.Plan[planAction] = actionAttack
				m.TargetGroup = int8(targetGroup)
				m.TargetMember = int8(random(alive))
				m.Choice = attackBits(rec, int(uint8(m.Plan[planSlot])))
			} else {
				m.Plan[0] = 0
			}
		}
		m.SecondAttack = -1
		if m.Plan[planAction] == actionAttack {
			a := int(uint8(m.Plan[planSlot+1]))
			if group >= 3 && rec.Attacks[a].Range <= 1 {
				m.Plan[1], m.Plan[3] = 0, 0
			} else {
				m.SecondAttack = attackBits(rec, a)
			}
		} else {
			m.Plan[1], m.Plan[2], m.Plan[3] = 0, 0, 0
		}
	}
}

// chooseMonsterActions (WPOPS 8a45): every group fights the party, or now and then the
// party's allies, who in turn fight one of the groups.
func chooseMonsterActions() {
	monsterGroups[0].Count = uint8(partyCount)
	allies := monsterGroups[allyGroup].Count > 0
	for g := 1; g <= monsterGroupCount; g++ {
		if !allies || random(4) != 1 || groupMonsterRec[g].Unk208 == 22 {
			chooseGroupActions(g, 0)
		} else {
			chooseGroupActions(g, allyGroup)
		}
	}
	if !allies {
		return
	}
	g := 1
	for g < monsterGroupCount && random(3) == 0 {
		g++
	}
	chooseGroupActions(allyGroup, g)
}

// startCombatRound (WPOPS 8aea) sets up the windows (a shorter view over the allies'
// line), gets everyone's actions and hands over to WMEXE (game state 13).
func startCombatRound() {
	pvBlinkState = -1
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
	drawMonsterGroups(monsterListWin, false)
	resumeWindowDraw(combatActionWin)
	resumeWindowDraw(monsterListWin)
	eraseCursorMark(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	if monsterGroups[allyGroup].Count > 0 {
		destroyWindow(pictureWin, false)
		pictureWin = createWindow(9, 4, 22, 12, 2, -1, 2, false)
		allyStripWin = createWindow(9, 16, 22, 2, 16, -1, 0, false)
		drawAllyGroup()
	}
	choosePartyActions(combatActionWin)
	if allyStripWin != nil {
		destroyWindow(allyStripWin, false)
		allyStripWin = nil
		destroyWindow(pictureWin, false)
		pictureWin = createWindow(9, 4, 22, 14, 2, -1, 2, false)
		viewAnimState = viewAnimMaze
		for {
			animateCombatStep()
			if viewAnimState == viewAnimShow {
				break
			}
		}
	}
	deferWindowDraw(combatActionWin)
	deferWindowDraw(monsterListWin)
	eraseWindow(mainWindow, false)
	eraseWindow(bottomLineWin, false)
	clearWindow(partyPictureWin, false)
	chooseMonsterActions()
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	gameState = stateRunCombatRound
}
