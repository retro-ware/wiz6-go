package game

import "strconv"

// WMEXE: one combat round (game state 13). Every combatant acts at its initiative
// counts, from 100 down to 1; then the round ends in state 14 (WMELE), or in 17 when a
// member stops to change equipment. The maze view behind the monsters is WMAZE's
// drawMazeView (WMEXE's copy, fd4c9..ff2b6, is the same code).

// WMEXE data segment.
var (
	mexeMsgMode    int         // g500a: 2 = the spell layout of the message lines, 4 = the melee one
	mexeName       string      // 0x500c: the first part of a combat line (usually a name)
	mexeText       string      // 0x505c: its last part
	mexeVerb       string      // 0x50ac: its middle part
	mexeBackfire   int16       // g50fc: the spell turned on its caster; no spell animation
	mexeEffect     [2][5]int16 // g50fe: the hit flash at view row, col: its step, -1 none
	mexeFlashSound int16       // g5112: play the flash sound at the next frame
	mexeSpellSound int         // g5114: the sound of the spell animation
	mexeDelayed    int16       // g5116: combat_delay ran since it was cleared
	mexeFizzle     int16       // g5118: the spell fizzled
)

// ROOT data: where the hit flash goes at view row, col (g006a x, g007e y).
var (
	effectX = [2][5]int{{0, -32, 32, -16, 16}, {0, -20, 20, -10, 10}}
	effectY = [2][5]int{{-3, 0, 0, -6, -6}, {-2, 0, 0, -4, -4}}
)

// ovmain (WMEXE 7230).
func wmexeOvmain() {
	if gameState == 13 {
		runCombatRound()
	}
}

// fighter is fighter i of group g.
func fighter(g, i int) *Combatant { return &monsterGroups[g].Members[i] }

// targetOf is fighter c's target, nil for none (B31 255: the C reads past
// monster_groups then).
func targetOf(c *Combatant) *Combatant {
	g, i := int(uint8(c.B31)), int(uint8(c.B32))
	if g >= len(monsterGroups) || i >= len(monsterGroups[g].Members) {
		return nil
	}
	return fighter(g, i)
}

// actor is the acting fighter.
func actor() *Combatant { return fighter(actGroup, actMember) }

// combatantName (get_combatant_name, WMEXE 6ba3): a member's name, or the group's
// monster name as identified so far (singular).
func combatantName(g, i int) string {
	if g == 0 {
		return memberName(i)
	}
	return cstr(groupMonsterRec[g].Names[int(monsterGroups[g].B412)<<1][:])
}

// nameColor is the colour of a name of group g: 1 party, 6 monsters.
func nameColor(g int) int {
	if g == 0 {
		return 1
	}
	return 6
}

// printWrapped (print_wrapped, WMEXE 6bf1) prints s word by word into the message
// window, wrapping at its edge.
func printWrapped(s string, color int) {
	w := partyPictureWin
	for len(s) > 0 {
		sp := charIndex(s, ' ')
		n := sp
		if sp < 0 {
			n = len(s)
		}
		if int(w.CurX)+n+1 > int(w.W) {
			windowNewline(w)
		}
		if w.CurX == 0 {
			w.CurX++
		}
		if sp >= 0 {
			printString(w, s[:sp], color)
			w.CurX++
			s = s[sp+1:]
		} else {
			printString(w, s, color)
			w.CurX++
			s = ""
		}
	}
}

// printCombatCentered (print_combat_centered, WMEXE 6cc3) prints s centered on row.
func printCombatCentered(s string, color, row int) {
	setCursor(partyPictureWin, max(20-(len(s)+1)/2, 0), row)
	printWrapped(s, color)
}

// printMsgCentered (print_msg_centered, WMEXE 6d0b) prints message n centered on row.
func printMsgCentered(n, color, row int) { printCombatCentered(getMessage(n), color, row) }

// printCombatLine (print_combat_line, WMEXE 6d32) prints a, b and c centered on the
// cursor's row, each in its colour.
func printCombatLine(a, b, c string, ca, cb, cc int) {
	gaps := b2i(a != "") + b2i(b != "") + b2i(c != "")
	setCursor(partyPictureWin, max(20-(len(a)+len(b)+len(c)+gaps)/2, 0), int(partyPictureWin.CurY))
	for _, part := range []struct {
		s string
		c int
	}{{a, ca}, {b, cb}, {c, cc}} {
		if part.s != "" {
			printWrapped(part.s, part.c)
		}
	}
}

// clearMsgLine (clear_msg_line, WMEXE 779c) clears message row and waits a tenth.
func clearMsgLine(row int) {
	clearWindowArea(partyPictureWin, 0, row, 39, row)
	setCursor(partyPictureWin, 1, row)
	combatDelay(1)
}

// showCombatMsg (show_combat_msg, WMEXE 8036).
func showCombatMsg(n int) {
	clearMsgLine(2)
	printMsgCentered(n, 5, 2)
	combatDelay(int(delaySetting) / 2)
}

// combatDelay (combat_delay, WMEXE 774b) animates the view for tenths of a second; a
// click or Enter ends it early.
func combatDelay(tenths int) {
	mexeDelayed = 1
	if dtLog != nil {
		return
	}
	flushKeys()
	for n := tenths * 10; n > 0; n-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			return
		}
		paceViewCycle()
		mexeAnimateStep()
	}
}

// mexeAnimateStep (animate_step, WMEXE 765a) is WPOPS's view tick plus the hit flashes and
// their sound.
func mexeAnimateStep() {
	if dtLog != nil {
		return
	}
	switch viewAnimState {
	case -3:
		drvShowWindow(pictureWin)
		if mexeFlashSound == 1 {
			playSound(5)
		}
		mexeFlashSound = 0
		animPhase = (animPhase + 1) % 2
		animCycle = (animCycle + 1) % 3
	case -2:
		drawMazeView()
		for i := range spriteList {
			spriteList[i].Depth = 255
		}
		spriteCount = 0
	case -1:
		for r := range 2 {
			for c := range 5 {
				queueCombatantSprite(r, c)
				queueEffectSprite(r, c)
			}
		}
	default:
		drawSprite(viewAnimState) // draw_sprite (655c) is the same
	}
	viewAnimState++
	if viewAnimState == 12 {
		viewAnimState = -3
	}
}

// resetAnimation (reset_animation, WMEXE 773a) runs a whole view cycle.
func resetAnimation() {
	if dtLog != nil {
		return
	}
	viewAnimState = -2
	for {
		mexeAnimateStep()
		if viewAnimState == -2 {
			break
		}
	}
}

// queueCombatantSprite (queue_combatant_sprite, WMEXE 6339) is WMELE's
// queueMonsterSprite without the shift, the group check and the clipping.
func queueCombatantSprite(row, col int) {
	g := int(viewPosGroup[row][col])
	if g == 0 || row > int(viewRowsOpen) {
		return
	}
	slot := int(monsterGroups[g].PicSlot)
	p := &monsterPics[slot]
	var frames []byte
	if row == 0 {
		frames = buildMonsterAnimFrames(g, col) // build_monster_frames (5fa0)
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
	y := int(p.Offsets[row][col][1]) + int(monsterJitterY[row][col]) + int(p.FrameY[f])
	insertSprite(depth, slot, x, y, 72, 248, mirror, frames)
}

// queueEffectSprite (queue_effect_sprite, WMEXE 6742) queues the hit flash at view row,
// col (picture slot 0) and steps it every third frame.
func queueEffectSprite(row, col int) {
	e := &mexeEffect[row][col]
	if *e == -1 {
		return
	}
	if animCycle == 1 && *e == 1 {
		mexeFlashSound = 1
	}
	frame, mirror := 1, 0
	switch row*10 + int(*e) {
	case 0:
		frame = 5
	case 1:
		frame = 3
	case 2:
		frame, mirror = 3, 1
	case 3, 11:
		frame = 2
	case 4, 12:
		frame, mirror = 2, 1
	case 5, 13:
		frame = 1
	case 6, 14:
		frame, mirror = 1, 1
	case 7, 15:
		*e = -1
		return
	case 10:
		frame = 4
	}
	depth := 30
	switch row*10 + col {
	case 13, 14:
		depth = 9
	case 11, 12:
		depth = 19
	case 10:
		depth = 14
	case 3, 4:
		depth = 24
	case 1, 2:
		depth = 50
	case 0:
		depth = 29
	}
	p := &monsterPics[0]
	y := int(p.FrameY[frame]) + effectY[row][col]
	if row == 1 {
		y -= 8
	}
	insertSprite(depth, 0, int(p.FrameX[frame][0])+effectX[row][col], y, 72, 248, mirror, []byte{byte(frame)})
	if animCycle == 0 {
		*e++
	}
}

// resetEffects (reset_effects, WMEXE d07c) clears the hit flashes.
func resetEffects() {
	for r := range mexeEffect {
		for c := range mexeEffect[r] {
			mexeEffect[r][c] = -1
		}
	}
}

// flashGroup shows the hit flash at fighter c's place and takes it off the view.
func flashOffView(c *Combatant, flash int16) {
	if c.Pos != -1 {
		r, col := c.Pos/10, c.Pos%10
		animCycle = 0
		mexeEffect[r][col] = flash
		resetAnimation()
		viewPosGroup[r][col] = 0
		monsterJitterX[r][col], monsterJitterY[r][col] = 0, 0
	}
	c.Pos = -1
}

// groupEffectMsg (fd0ad, WMEXE) announces a lasting group effect and flashes the
// acting monster group's row.
func groupEffectMsg(n, color int) {
	printMsgCentered(n, color, 0)
	if actGroup != 0 && actGroup != 6 {
		animCycle = 0
		mexeEffect[b2i(actGroup >= 3)][0] = 1
		resetAnimation()
	}
	combatDelay(int(delaySetting) / 4)
}

// runCombatRound (run_combat_round, WMEXE d109) runs the round: at each initiative count
// every fighter whose swing is due acts, charging fighters land, and the groups'
// lasting clouds strike.
func runCombatRound() {
	resetEffects()
	refreshAllWindows()
	for initiative = 100; initiative > 0; initiative-- {
		for actGroup = 0; actGroup < 7; actGroup++ {
			grp := monsterGroups[actGroup]
			for actMember = 0; int(grp.Count) > actMember; actMember++ {
				c := actor()
				if uint8(c.B35[1]) > 0 && int(uint8(c.B35[2])) == initiative {
					if c.Toggle == 0 {
						if int(uint8(c.B24[c.B33])) != initiative {
							c.Toggle = 3
							clearWindow(partyPictureWin, true)
							combatDelay(2)
							actorLine(3518, -1, 6, 0)
							combatDelay(int(delaySetting) / 2)
						}
						c.B35[2] = int8(max(initiative-random(10)-5, 1))
					} else if c.Toggle == 3 {
						landCharge(c)
					}
				}
				switch {
				case int(uint8(c.B24[c.B33])) == initiative:
					if uint8(c.B35[1]) > 0 && c.Toggle == 3 {
						landCharge(c)
					}
					performAction()
					c.B33++
					if gameState != 13 {
						return
					}
				case c.B24[c.B33] == 0:
					c.B33++
				}
				if uint8(c.B33) > 3 {
					c.B33 = 3
				}
			}
			if grp.Count == 0 {
				continue
			}
			for k := range 4 {
				power := int(grp.B402[k])
				if int(groupB406(grp, k)) != initiative || power <= 0 {
					continue
				}
				clearWindow(partyPictureWin, true)
				combatDelay(2)
				chance := power*5 + 25
				switch k {
				case 0:
					groupEffectMsg(3220, 4)
					hitDamage(actGroup, -10, 2, 4, power, 1, chance)
				case 1:
					groupEffectMsg(3221, 3)
					hitDamageThen(actGroup, -10, 3477, 6, power, chance, 1, 4, 5, false, power*10, 2, power, 0, 0)
				case 2:
					groupEffectMsg(3222, 6)
					hitDamageThen(actGroup, -10, 3480, 5, power, chance, 2, 3, 8, false, power, 4, power, 0, 0)
				case 3:
					groupEffectMsg(3223, 6)
					hitDamage(actGroup, -10, 1, 6, power, 8, chance)
				}
			}
		}
	}
	combatDelay(8)
	gameState = 14
}

// landCharge ends fighter c's charge (message 3519).
func landCharge(c *Combatant) {
	c.Toggle = 0
	c.B35[2] = 0
	clearWindow(partyPictureWin, true)
	combatDelay(2)
	actorLine(3519, -1, 6, 0)
	combatDelay(int(delaySetting) / 2)
}

// groupB406 is MonsterGroup byte 406+k: the initiative at which cloud k strikes.
func groupB406(grp *MonsterGroup, k int) uint8 {
	if k == 0 {
		return grp.B406
	}
	return grp.B407[k-1]
}

func setGroupB406(grp *MonsterGroup, k int, v uint8) {
	if k == 0 {
		grp.B406 = v
	} else {
		grp.B407[k-1] = v
	}
}

// actorLine (fb85f, WMEXE) prints the acting fighter's name with messages verb and text
// (text -1: keep the old one); silenced (1) adds "but is silenced" and returns true.
// A monster of the encounter's own kind uses the encounter's sound for the spell.
func actorLine(verb, text, color, silenced int) bool {
	mexeName = combatantName(actGroup, actMember)
	mexeVerb = getMessage(verb)
	mexeText = ""
	if text >= 0 {
		mexeText = getMessage(text)
	}
	printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), color, color)
	if silenced == 1 {
		combatDelay(2)
		printMsgCentered(3200, 3, 2)
		combatDelay(int(delaySetting) / 2)
		return true
	}
	s := groupMonsterRec[1].Sound
	if actGroup != 0 && s > 0 && s != 6 && groupMonsterRec[actGroup].Sound == s {
		mexeSpellSound = 14
	}
	return false
}

// reduceStamina (reduce_field6, WMEXE 7254) takes n stamina from c.
func reduceStamina(c *Combatant, n int) {
	c.Stamina = max(int16(int(c.Stamina)-n), 0) // C int wraps
}

// clamp_toward_x (WMEXE 8861) is WPCVW's clampToward.

// drawCardX (draw_member_card_x, WMEXE 7353) is WMELE's drawCombatCard: the C reads
// the fighter back from the party group (f5f19) before drawing.
func drawCardX(i int, hilite bool) {
	if dtLog == nil {
		drawCombatCard(i, hilite)
	}
}

// mexeSwapMembers (swap_members, WMEXE c31a) swaps party members i and j and redraws
// their cards.
func mexeSwapMembers(i, j int) {
	swapMembers(j, i)
	drawCardX(i, false)
	drawCardX(j, false)
	combatDelay(3)
}

// fillPartyRecord copies member m's attributes into the party's monster record
// (group_monster_rec[0]), which the attack code reads like a monster's.
func fillPartyRecord(m int, acSum bool) {
	r, p := groupMonsterRec[0], &party[m]
	r.Unk134 = p.Stats[0]
	r.Unk135 = [3]uint8{p.Stats[1], p.Stats[4], p.Stats[5]}
	r.Resist = p.Resist
	r.AC = p.AC
	if acSum {
		r.AC[0] = r.AC[0] + r.AC[1] + r.AC[2]
	} else {
		for k := range r.AC {
			if k != 1 && k != 2 {
				r.AC[k] = r.AC[k] + r.AC[1] + r.AC[2]
			}
		}
	}
	r.Unk211[0], r.Unk211[2], r.Unk211[1] = p.AttackPower, p.SecondaryAttacks, p.PrimaryAttacks
	r.Unk211[3] = p.Sex
	r.Unk208 = p.Class
}

// nextLivingMember (fc4e7, WMEXE) is a standing member next to member i (the one
// before, else the one after), else i.
func nextLivingMember(i int) int {
	j := i
	if i > 0 {
		j = i - 1
	}
	for range 2 {
		c := fighter(0, j)
		if uint8(c.Condition) < 3 && c.Toggle == 0 {
			return j
		}
		if partyCount-1 > i {
			j = i + 1
		} else {
			j = i
		}
	}
	return i
}

// nextTargetMonster (fc546, WMEXE) picks a living fighter of group g: the first one in
// the acting member's view column order, else the first standing one, else i.
func nextTargetMonster(g, i int) int {
	grp := monsterGroups[g]
	for row := range 2 {
		for _, col := range memberColumns[actMember%6] {
			if int(viewPosGroup[row][col]) != g {
				continue
			}
			for k := 0; int(grp.Count) > k; k++ {
				if row*10+col == int(grp.Members[k].Pos) {
					return k
				}
			}
		}
	}
	for k := 0; int(grp.Count) > k; k++ {
		if uint8(grp.Members[k].Condition) < 3 {
			return k
		}
	}
	return i
}

// performAction (fc66e, WMEXE) runs the acting fighter's action: status effects may
// keep it from acting or turn it on its own side first.
func performAction() {
	groupMonsterRec[0].Flags = 0
	mexeSpellSound = 6
	mexeBackfire, mexeFizzle = 0, 0
	c := actor()
	if uint8(c.Condition) >= 2 || c.Status[3] > 0 || c.Status[2] > 0 {
		return
	}
	clearWindow(partyPictureWin, true)
	msg, turned, leave := 0, false, 0
	if c.Status[1] > 0 && random(100) > 80 && c.B24[6] != 7 {
		msg = 3209
	}
	if msg == 0 && c.Status[7] > 0 && random(100) > 40 {
		msg = random(3) + 3205
		if random(100) > 80 {
			msg, turned = 3208, true
		}
	}
	if msg == 0 && c.Status[6] > 0 {
		msg = 3210
		if random(100) > 25 {
			turned = true
		}
	}
	if msg == 0 && c.Status[4] > 0 && random(100) > 80 {
		msg, turned = 3204, false
	}
	if msg == 0 && c.Status[8] > 0 && random(100) > 75 {
		msg, turned = 3203, false
	}
	if groupMonsterRec[actGroup].Flags&4 != 0 && targetOf(c) != nil && targetOf(c).Status[6] > 0 {
		msg = 3218
		leave = 1
		if monsterGroups[actGroup].MonsterID == 151 {
			c.Toggle = 3
			c.B35[1] = 1
			leave = 2
		}
	}
	if msg > 0 {
		combatDelay(2)
		mexeName = combatantName(actGroup, actMember)
		mexeVerb = getMessage(msg)
		mexeText = ""
		printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, 0)
		if leave > 0 {
			flashOffView(c, 1)
			if leave == 1 {
				c.Condition = 3
				c.HP = 0
			}
			for k := range 4 {
				c.B24[k] = 0
			}
			combatDelay(int(delaySetting))
			return
		}
		combatDelay(int(delaySetting) / 2)
		if !turned {
			c.Status[2]++
			return
		}
		c.B24[6], c.B33 = 0, 0
		c.W12 = -1
		c.W10 = firstAttackMode(actGroup, actMember)
		c.B24[4] = 0
		c.B31 = int8(actGroup)
		c.B32 = int8(random(int(monsterGroups[actGroup].Count)))
		clearWindow(partyPictureWin, true)
		combatDelay(2)
		mexeName = combatantName(actGroup, actMember)
		mexeVerb = getMessage(3211)
		mexeText = ""
		printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, 0)
		playSound(0)
		combatDelay(int(delaySetting) / 2)
		clearWindow(partyPictureWin, true)
	}
	if t := targetOf(c); t != nil && (uint8(t.Condition) >= 3 || t.Toggle != 0) {
		if c.B31 == 0 {
			c.B32 = int8(nextLivingMember(int(uint8(c.B32))))
		} else {
			c.B32 = int8(nextTargetMonster(int(uint8(c.B31)), int(uint8(c.B32))))
		}
	}
	if turned && int(uint8(c.B32)) == actMember {
		c.Status[2]++
		return
	}
	if actGroup == 0 {
		fillPartyRecord(actMember, true)
	} else if c.B31 == 0 {
		fillPartyRecord(int(uint8(c.B32)), false)
	}
	switch uint8(c.B24[6]) {
	case 0:
		t := targetOf(c)
		if t == nil || uint8(t.Condition) >= 3 || t.Toggle != 0 {
			break
		}
		slot := int(uint8(c.B24[4+int(uint8(c.B33))%2]))
		if actGroup == 0 && party[actMember].Equipped[slot] == 255 {
			slot = 0
		}
		kind := int(c.W10)
		if uint8(c.B33)%2 == 1 {
			kind = int(c.W12)
		}
		combatDelay(2)
		doAttack(c, slot, kind)
		if kind == 3 || kind == 6 {
			reduceStamina(c, random(4)+3)
		} else {
			reduceStamina(c, random(3)+2)
		}
	case 3:
		combatDelay(2)
		reduceStamina(c, doCastAction(c))
	case 8:
		combatDelay(2)
		reduceStamina(c, doBreath(c))
	case 1:
		if c.W10 == 3 {
			n := int(int32(c.StaminaMax) * 5 / 100)
			c.Stamina += int16(random(n) + 1)
			if c.Stamina > c.StaminaMax {
				c.Stamina = c.StaminaMax
			}
		}
	case 2:
		combatDelay(2)
		doChangeEquipment()
		reduceStamina(c, random(4)+3)
		resetAnimation()
		clearWindow(partyPictureWin, false)
		gameState = 17
		returnCmd = 13
		curMember = int16(actMember)
		return
	case 4:
		combatDelay(2)
		doUseItemAction(c)
		reduceStamina(c, random(3)+1)
	case 5:
		combatDelay(2)
		doAdvanceRetreat(actMember, int(c.W10))
		reduceStamina(c, random(4)+3)
	case 6:
		combatDelay(2)
		doHide(c)
		reduceStamina(c, random(3)+1)
	case 7:
		combatDelay(2)
		doRunAway(c)
		reduceStamina(c, random(5)+5)
	}
	if actGroup == 0 {
		drawCardX(actMember, false)
	}
}

// firstAttackMode is the lowest attack mode fighter i of group g has (party: the bits
// of its primary hand; monsters: of its first attack), else 8.
func firstAttackMode(g, i int) int16 {
	bits := groupMonsterRec[g].Attacks[0].Effect
	if g == 0 {
		bits = party[i].Hands[0].Flags
	}
	for m := 0; m < 8; m++ {
		if bits&(1<<m) != 0 {
			return int16(m)
		}
	}
	return 8
}

// doChangeEquipment (do_change_equipment, WMEXE c012).
func doChangeEquipment() {
	mexeName = combatantName(actGroup, actMember)
	mexeVerb = getMessage(3311)
	mexeText = ""
	printCombatLine(mexeName, mexeVerb, mexeText, 1, 15, 0)
	combatDelay(int(delaySetting) / 2)
}

// doHide (do_hide, WMEXE c064): a member hides from the monsters, a monster from the
// first three members.
func doHide(c *Combatant) {
	var chance int
	if actGroup == 0 {
		chance = int(party[actMember].Level) + int(party[actMember].Skills[16]) + 50
		for g := 1; g <= monsterGroupCount; g++ {
			chance -= int(monsterGroups[g].Count) + int(groupMonsterRec[g].HP.Count) + int(groupMonsterRec[g].Unk135[0])
		}
	} else {
		chance = int(c.Level)*5 + 50
		for i := 0; i < min(partyCount, 3); i++ {
			chance -= int(party[i].Level) + int(party[i].Stats[1])
		}
	}
	actorFlavour(3515)
	combatDelay(int(delaySetting) / 4)
	if random(100) < chance {
		c.Toggle = 1
		mexeRaiseSkill(actMember, 16, 1)
		printMsgCentered(3520, 6, 2)
	} else {
		printMsgCentered(3521, 6, 2)
	}
	combatDelay(int(delaySetting) / 4)
}

// actorFlavour prints the acting fighter's name and message n.
func actorFlavour(n int) {
	mexeName = combatantName(actGroup, actMember)
	mexeVerb = getMessage(n)
	mexeText = ""
	printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, 0)
}

// doRunAway (do_run_away, WMEXE c1e4).
func doRunAway(c *Combatant) {
	chance := int(c.Level) + 90
	for g := 1; g <= monsterGroupCount; g++ {
		r := groupMonsterRec[g]
		chance = chance - int(r.HP.Count)*int(monsterGroups[g].Count) - (int(r.Unk135[2])-int(groupMonsterRec[actGroup].Unk135[2]))*2
	}
	if g4354 == 1 {
		chance = 0
	}
	actorFlavour(3517)
	combatDelay(int(delaySetting) / 4)
	if random(100) < chance {
		c.Toggle = 2
		printMsgCentered(3522, 6, 2)
	} else {
		printMsgCentered(3523, 6, 2)
	}
	combatDelay(int(delaySetting) / 4)
}

// doAdvanceRetreat (do_advance_retreat, WMEXE c3ee) moves member from to place to,
// swapping it past the others.
func doAdvanceRetreat(from, to int) {
	if from == to {
		return
	}
	side := 0
	if from%2 != to%2 {
		side = -1
		if to%2 == 0 {
			side = 1
		}
	}
	mexeName = combatantName(actGroup, actMember)
	step := 2
	if from > to {
		mexeVerb = getMessage(3513)
		step = -2
	} else {
		mexeVerb = getMessage(3514)
	}
	mexeText = ""
	printCombatLine(mexeName, mexeVerb, mexeText, 1, 14, 0)
	combatDelay(3)
	for to+side != from {
		mexeSwapMembers(from, from+step)
		from += step
	}
	if side != 0 {
		mexeSwapMembers(to+side, to)
	}
	combatDelay(int(delaySetting) / 2)
}

// itoa is int_to_string.
func itoa(n int) string {
	if dtLog != nil {
		dtLog("num", n)
	}
	return strconv.Itoa(n)
}
