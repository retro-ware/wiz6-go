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

// Combat groups: 0 is the party, 1..5 the monster groups, allyGroup the party's
// summoned allies.
const (
	allyGroup    = 6
	combatGroups = 7
)

// Action kinds (Combatant.Plan[planAction]), as WPOPS sets them.
const (
	actionAttack  = 0
	actionDefend  = 1 // Choice: 0 (shield skill), 1, defendParry or defendRest
	actionEquip   = 2
	actionCast    = 3
	actionUseItem = 4
	actionMove    = 5 // Choice: the new place
	actionHide    = 6
	actionRun     = 7
	actionBreath  = 8
	actionNone    = 10

	defendParry = 2 // turns into a riposte when struck
	defendRest  = 3 // wins back stamina
)

// Combatant.Stance values.
const (
	stanceHiddenStruck = -1 // hidden and striking from hiding (doAttack)
	stanceNone         = 0
	stanceHidden       = 1
	stanceFled         = 2
	stanceCharging     = 3 // set at Combatant.B35[b35BlinkAt]'s count, lands at its swing
)

// ROOT data: where the hit flash goes at view row, col (g006a x, g007e y).
var (
	effectX = [2][5]int{{0, -32, 32, -16, 16}, {0, -20, 20, -10, 10}}
	effectY = [2][5]int{{-3, 0, 0, -6, -6}, {-2, 0, 0, -4, -4}}
)

// ovmain (WMEXE 7230).
func wmexeOvmain() {
	if gameState == stateRunCombatRound {
		runCombatRound()
	}
}

// fighter is fighter i of group g.
func fighter(g, i int) *Combatant { return &monsterGroups[g].Members[i] }

// targetOf is fighter c's target, nil for none (TargetGroup 255: the C reads past
// monster_groups then).
func targetOf(c *Combatant) *Combatant {
	g, i := int(uint8(c.TargetGroup)), int(uint8(c.TargetMember))
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
	return cstr(groupMonsterRec[g].Names[int(monsterGroups[g].Unidentified)<<1][:])
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
		if keyAvailable(keyClassMenu) && int8(getKey(keyClassMenu)) == keyEnter {
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
		viewAnimState = viewAnimShow
	}
}

// resetAnimation (reset_animation, WMEXE 773a) runs a whole view cycle.
func resetAnimation() {
	if dtLog != nil {
		return
	}
	viewAnimState = viewAnimMaze
	for {
		mexeAnimateStep()
		if viewAnimState == viewAnimMaze {
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
		monsterJitterX[row][col] = uint8(random(int(p.Unk309[picJitterX])))
		monsterJitterY[row][col] = uint8(random(int(p.Unk309[picJitterY])))
	}
	f := int(frames[0])
	depth := int(p.Frames[f][frameDepth])
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

// flashOffView shows the hit flash at fighter c's place and takes it off the view.
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
	if actGroup != 0 && actGroup != allyGroup {
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
		for actGroup = 0; actGroup < combatGroups; actGroup++ {
			grp := monsterGroups[actGroup]
			for actMember = 0; int(grp.Count) > actMember; actMember++ {
				c := actor()
				if uint8(c.B35[b35Blink]) > 0 && int(uint8(c.B35[b35BlinkAt])) == initiative {
					if c.Stance == stanceNone {
						if int(uint8(c.Plan[c.Swing])) != initiative {
							c.Stance = stanceCharging
							clearWindow(partyPictureWin, true)
							combatDelay(2)
							actorLine(msgBlinksOut, -1, 6, 0)
							combatDelay(int(delaySetting) / 2)
						}
						c.B35[b35BlinkAt] = int8(max(initiative-random(10)-5, 1))
					} else if c.Stance == stanceCharging {
						landCharge(c)
					}
				}
				switch {
				case int(uint8(c.Plan[c.Swing])) == initiative:
					if uint8(c.B35[b35Blink]) > 0 && c.Stance == stanceCharging {
						landCharge(c)
					}
					performAction()
					c.Swing++
					if gameState != stateRunCombatRound {
						return
					}
				case c.Plan[c.Swing] == 0:
					c.Swing++
				}
				if uint8(c.Swing) > 3 {
					c.Swing = 3
				}
			}
			if grp.Count == 0 {
				continue
			}
			for k := range 4 {
				power := int(grp.CloudPower[k])
				if int(cloudStrike(grp, k)) != initiative || power <= 0 {
					continue
				}
				clearWindow(partyPictureWin, true)
				combatDelay(2)
				chance := power*5 + 25
				switch k {
				case 0:
					groupEffectMsg(msgFirestorm, 4)
					hitDamage(actGroup, -10, 2, 4, power, resistFire, chance)
				case 1:
					groupEffectMsg(msgPoisonGas, 3)
					hitDamageThen(actGroup, -10, msgCombatStatusBase+statusPoisoned, resistPoison, power, chance, 1, 4, statusPoisoned, false, power*10, 2, power, 0, 0)
				case 2:
					groupEffectMsg(msgToxicVapors, 6)
					hitDamageThen(actGroup, -10, msgCombatStatusBase+statusNauseated, 5, power, chance, 2, 3, statusNauseated, false, power, 4, power, 0, 0)
				case 3:
					groupEffectMsg(msgAcidCloud, 6)
					hitDamage(actGroup, -10, 1, 6, power, 8, chance)
				}
			}
		}
	}
	combatDelay(8)
	gameState = stateEndCombatRound
}

// landCharge ends fighter c's charge (message 3519).
func landCharge(c *Combatant) {
	c.Stance = stanceNone
	c.B35[b35BlinkAt] = 0
	clearWindow(partyPictureWin, true)
	combatDelay(2)
	actorLine(msgBlinksIn, -1, 6, 0)
	combatDelay(int(delaySetting) / 2)
}

// cloudStrike is initiative at which cloud k strikes.
func cloudStrike(grp *MonsterGroup, k int) uint8 {
	if k == 0 {
		return grp.Cloud0Strike
	}
	return grp.Cloud1to3Strike[k-1]
}

func setCloudStrike(grp *MonsterGroup, k int, v uint8) {
	if k == 0 {
		grp.Cloud0Strike = v
	} else {
		grp.Cloud1to3Strike[k-1] = v
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
		printMsgCentered(msgSilenced, 3, 2)
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

// MonsterRec.Stats, MonsterRec.Unk168 and MonsterRec.Unk211 indexes.
const (
	monIntelligence = 0
	monDexterity    = 1
	monSpeed        = 2

	monCastChance   = 0 // Unk168: percent chance to cast
	monBreathChance = 2 // Unk168: percent chance to breathe

	monAttackPower      = 0 // Unk211: PC.AttackPower
	monPrimaryAttacks   = 1 // Unk211: swings per attack
	monSecondaryAttacks = 2 // Unk211: planned swings
	monSex              = 3
	monRegen            = 4
	monBreath           = 5 // Unk211: the breath (actionBreath choice)
	monKillFlag         = 7 // Unk211: global flag set when a Flags&1 monster dies
)

// fillPartyRecord copies member m's attributes into the party's monster record
// (group_monster_rec[0]), which the attack code reads like a monster's.
func fillPartyRecord(m int, acSum bool) {
	r, p := groupMonsterRec[0], &party[m]
	r.Strength = p.Stats[statStrength]
	r.Stats = [3]uint8{p.Stats[statIntelligence], p.Stats[statDexterity], p.Stats[statSpeed]}
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
	r.Unk211[monAttackPower], r.Unk211[monSecondaryAttacks], r.Unk211[monPrimaryAttacks] = p.AttackPower, p.SecondaryAttacks, p.PrimaryAttacks
	r.Unk211[monSex] = p.Sex
	r.Class = p.Class
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
		if uint8(c.Condition) < conditionDead && c.Stance == stanceNone {
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
		if uint8(grp.Members[k].Condition) < conditionDead {
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
	if uint8(c.Condition) >= conditionDown || c.Status[statusParalyzed] > 0 || c.Status[statusAsleep] > 0 {
		return
	}
	clearWindow(partyPictureWin, true)
	msg, turned, leave := 0, false, 0
	if c.Status[statusAfraid] > 0 && random(100) > 80 && c.Plan[planAction] != actionRun {
		msg = 3209
	}
	if msg == 0 && c.Status[statusInsane] > 0 && random(100) > 40 {
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
	if msg == 0 && c.Status[statusBlinded] > 0 && random(100) > 80 {
		msg, turned = 3204, false
	}
	if msg == 0 && c.Status[statusNauseated] > 0 && random(100) > 75 {
		msg, turned = 3203, false
	}
	if groupMonsterRec[actGroup].Flags&4 != 0 && targetOf(c) != nil && targetOf(c).Status[6] > 0 {
		msg = 3218
		leave = 1
		if monsterGroups[actGroup].MonsterID == 151 {
			c.Stance = stanceCharging
			c.B35[b35Blink] = 1
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
				c.Condition = conditionDead
				c.HP = 0
			}
			for k := range planSwings {
				c.Plan[k] = 0
			}
			combatDelay(int(delaySetting))
			return
		}
		combatDelay(int(delaySetting) / 2)
		if !turned {
			c.Status[statusAsleep]++
			return
		}
		c.Plan[planAction], c.Swing = actionAttack, 0
		c.SecondAttack = -1
		c.Choice = firstAttackMode(actGroup, actMember)
		c.Plan[planSlot] = 0
		c.TargetGroup = int8(actGroup)
		c.TargetMember = int8(random(int(monsterGroups[actGroup].Count)))
		clearWindow(partyPictureWin, true)
		combatDelay(2)
		mexeName = combatantName(actGroup, actMember)
		mexeVerb = getMessage(msgTurnsOnComrades)
		mexeText = ""
		printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, 0)
		playSound(soundBeep)
		combatDelay(int(delaySetting) / 2)
		clearWindow(partyPictureWin, true)
	}
	if t := targetOf(c); t != nil && (uint8(t.Condition) >= conditionDead || t.Stance != stanceNone) {
		if c.TargetGroup == 0 {
			c.TargetMember = int8(nextLivingMember(int(uint8(c.TargetMember))))
		} else {
			c.TargetMember = int8(nextTargetMonster(int(uint8(c.TargetGroup)), int(uint8(c.TargetMember))))
		}
	}
	if turned && int(uint8(c.TargetMember)) == actMember {
		c.Status[statusAsleep]++
		return
	}
	if actGroup == 0 {
		fillPartyRecord(actMember, true)
	} else if c.TargetGroup == 0 {
		fillPartyRecord(int(uint8(c.TargetMember)), false)
	}
	switch uint8(c.Plan[planAction]) {
	case actionAttack:
		t := targetOf(c)
		if t == nil || uint8(t.Condition) >= conditionDead || t.Stance != stanceNone {
			break
		}
		slot := int(uint8(c.Plan[planSlot+int(uint8(c.Swing))%2]))
		if actGroup == 0 && party[actMember].Equipped[slot] == noEquippedItem {
			slot = 0
		}
		kind := int(c.Choice)
		if uint8(c.Swing)%2 == 1 {
			kind = int(c.SecondAttack)
		}
		combatDelay(2)
		doAttack(c, slot, kind)
		if kind == 3 || kind == 6 {
			reduceStamina(c, random(4)+3)
		} else {
			reduceStamina(c, random(3)+2)
		}
	case actionCast:
		combatDelay(2)
		reduceStamina(c, doCastAction(c))
	case actionBreath:
		combatDelay(2)
		reduceStamina(c, doBreath(c))
	case actionDefend:
		if c.Choice == defendRest {
			n := int(int32(c.StaminaMax) * 5 / 100)
			c.Stamina += int16(random(n) + 1)
			if c.Stamina > c.StaminaMax {
				c.Stamina = c.StaminaMax
			}
		}
	case actionEquip:
		combatDelay(2)
		doChangeEquipment()
		reduceStamina(c, random(4)+3)
		resetAnimation()
		clearWindow(partyPictureWin, false)
		gameState = stateViewCharacter
		returnCmd = int16(stateRunCombatRound)
		curMember = int16(actMember)
		return
	case actionUseItem:
		combatDelay(2)
		doUseItemAction(c)
		reduceStamina(c, random(3)+1)
	case actionMove:
		combatDelay(2)
		doAdvanceRetreat(actMember, int(c.Choice))
		reduceStamina(c, random(4)+3)
	case actionHide:
		combatDelay(2)
		doHide(c)
		reduceStamina(c, random(3)+1)
	case actionRun:
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
	mexeVerb = getMessage(msgChangesEquipment)
	mexeText = ""
	printCombatLine(mexeName, mexeVerb, mexeText, 1, 15, 0)
	combatDelay(int(delaySetting) / 2)
}

// doHide (do_hide, WMEXE c064): a member hides from the monsters, a monster from the
// first three members.
func doHide(c *Combatant) {
	var chance int
	if actGroup == 0 {
		chance = int(party[actMember].Level) + int(party[actMember].Skills[skillHide]) + 50
		for g := 1; g <= monsterGroupCount; g++ {
			chance -= int(monsterGroups[g].Count) + int(groupMonsterRec[g].HP.Count) + int(groupMonsterRec[g].Stats[monIntelligence])
		}
	} else {
		chance = int(c.Level)*5 + 50
		for i := 0; i < min(partyCount, 3); i++ {
			chance -= int(party[i].Level) + int(party[i].Stats[statIntelligence])
		}
	}
	actorFlavour(msgHides)
	combatDelay(int(delaySetting) / 4)
	if random(100) < chance {
		c.Stance = stanceHidden
		mexeRaiseSkill(actMember, skillHide, 1)
		printMsgCentered(msgShadows, 6, 2)
	} else {
		printMsgCentered(msgFailed, 6, 2)
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
		chance = chance - int(r.HP.Count)*int(monsterGroups[g].Count) - (int(r.Stats[monSpeed])-int(groupMonsterRec[actGroup].Stats[monSpeed]))*2
	}
	if noEscape == 1 {
		chance = 0
	}
	actorFlavour(msgRuns)
	combatDelay(int(delaySetting) / 4)
	if random(100) < chance {
		c.Stance = stanceFled
		printMsgCentered(msgAway, 6, 2)
	} else {
		printMsgCentered(msgBlocked, 6, 2)
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
		mexeVerb = getMessage(msgAdvances)
		step = -2
	} else {
		mexeVerb = getMessage(msgRetreats)
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

// MSG.DBS message ids.
const (
	msgBodyAreaBase          = 3320 // plus the hit area: BACK, WING, ..
	msgTreeAreaBase          = 3330 // plus the hit area on a plant: TRUNK, BRANCH, ..
	msgPartyAttackVerbBase   = 3350 // plus the attack kind: SWINGS..SHOOTS
	msgMonsterAttackVerbBase = 3371 // plus AttackKind: CLAWS, BITES, ..
	msgCombatStatusBase      = 3472 // plus the status: SILENCED..NAUSEOUS
	msgCombatConditionBase   = 3483 // plus the condition: CRITICAL!..DIES!
	msgItemNameBase          = 3799 // plus the item id (1-based)

	msgSilenced            = 3200
	msgFizzle              = 3201
	msgBackfire            = 3202
	msgTurnsOnComrades     = 3211
	msgSpellDeflected      = 3212
	msgSpellReduced        = 3213
	msgDeflected           = 3214
	msgAppears             = 3215
	msgAppear              = 3216
	msgNothingAppears      = 3217
	msgFirestorm           = 3220
	msgPoisonGas           = 3221
	msgToxicVapors         = 3222
	msgAcidCloud           = 3223
	msgLethalCloudsReduced = 3224
	msgAirBubbleDestroyed  = 3225
	msgAirBubbleReduced    = 3226
	msgMissed              = 3301
	msgHit                 = 3302
	msgNoPenetration       = 3303
	msgPlays               = 3304
	msgAbuses              = 3305
	msgCasts               = 3308
	msgUnaffected          = 3310
	msgChangesEquipment    = 3311
	msgInvokes             = 3312
	msgUses                = 3313
	msgIncants             = 3314
	msgTosses              = 3315
	msgFumblesWith         = 3316
	msgFumblesWithThrown   = 3317 // FUMBLES WITH
	msgConsumes            = 3318
	msgBreathes            = 3340
	msgSpews               = 3341
	msgVomits              = 3342
	msgGazes               = 3343
	msgEmits               = 3344
	msgShrieks             = 3345
	msgSings               = 3346
	msgCries               = 3347
	msgMonsterThrows       = 3384 // THROWS
	msgEffectUnaffected    = 3471 // UNAFFECTED!
	msgSlainSpell          = 3482 // SLAIN!
	msgIrritated           = 3487
	msgCaptured            = 3488
	msgWhithersAndDies     = 3489
	msgSuffocated          = 3490
	msgHeld                = 3492
	msgDispelled           = 3493
	msgCritical            = 3495
	msgBeholden            = 3496
	msgHypnotized          = 3498
	msgFire                = 3700
	msgIce                 = 3701
	msgDarkVapor           = 3702
	msgStinkingOdor        = 3703
	msgPoison              = 3704
	msgBoulders            = 3705
	msgAcid                = 3706
	msgWhiteMist           = 3707
	msgDrained             = 3750
)
