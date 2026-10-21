package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// WMEXE: spells, items, breath and summoning in combat.

// spellAnimXY (ROOT g038e): x, y of each step of spell animation kind k, at
// [k*24 + side*12 + mirror*6 + step*2].
var spellAnimXY = [408]uint8{
	80, 61, 102, 62, 121, 64, 208, 61, 186, 62, 175, 64, 120, 64, 97, 62, 79, 61, 176, 64, 191, 62, 210, 61,
	75, 59, 93, 58, 118, 62, 213, 59, 195, 58, 177, 62, 122, 62, 107, 58, 84, 59, 174, 62, 182, 58, 206, 59,
	73, 72, 97, 72, 118, 72, 215, 72, 190, 72, 179, 72, 119, 72, 101, 72, 78, 72, 175, 72, 188, 72, 210, 72,
	84, 61, 107, 63, 125, 64, 212, 61, 190, 63, 179, 64, 124, 64, 102, 63, 83, 61, 179, 64, 193, 63, 214, 61,
	89, 62, 105, 64, 125, 65, 211, 62, 192, 64, 177, 65, 124, 65, 103, 64, 88, 62, 180, 65, 191, 64, 211, 62,
	86, 63, 108, 64, 123, 65, 210, 63, 196, 64, 181, 65, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 124, 64, 104, 63, 77, 62, 180, 64, 200, 63, 220, 62,
	81, 68, 105, 68, 125, 69, 215, 68, 191, 68, 179, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 125, 69, 106, 68, 79, 68, 179, 69, 189, 68, 217, 68,
	84, 68, 116, 67, 131, 67, 212, 68, 188, 67, 181, 67, 130, 67, 107, 67, 77, 68, 182, 67, 197, 67, 219, 68,
	90, 63, 108, 66, 128, 59, 214, 63, 195, 66, 176, 59, 128, 59, 108, 66, 89, 63, 176, 59, 196, 66, 216, 63,
	86, 62, 110, 65, 127, 59, 218, 62, 196, 65, 177, 59, 127, 59, 110, 65, 83, 62, 177, 59, 195, 65, 220, 62,
	85, 59, 114, 61, 127, 64, 219, 59, 190, 61, 177, 64, 127, 64, 114, 61, 84, 59, 177, 64, 190, 61, 219, 59,
	91, 60, 106, 66, 128, 60, 214, 60, 190, 66, 176, 60, 128, 60, 106, 66, 91, 60, 176, 60, 190, 66, 214, 60,
	85, 64, 109, 64, 124, 68, 210, 64, 195, 64, 180, 68, 127, 68, 107, 64, 81, 64, 178, 68, 197, 64, 215, 64,
	81, 60, 102, 63, 124, 65, 208, 60, 186, 63, 172, 65, 124, 65, 102, 63, 79, 60, 172, 65, 186, 63, 210, 60,
	81, 61, 106, 63, 128, 67, 208, 61, 190, 63, 176, 67, 124, 67, 106, 63, 78, 61, 181, 67, 191, 63, 211, 61,
}

// spellAnimation (f8efc, WMEXE) animates spell kind toward fighter i of group g;
// flash (1) leaves a hit flash on that side.
func spellAnimation(g, i, kind int, flash bool) {
	if mexeBackfire == 1 {
		return
	}
	row := 1
	if g == 0 || g == 6 {
		if c := actor(); c.Pos == -1 {
			row = b2i(actGroup != 1)
		} else {
			row = int(c.Pos) / 10
		}
	} else {
		for col := range 5 {
			if int(viewPosGroup[0][col]) == g {
				row = 0
			}
		}
	}
	if viewRowsOpen == 0 {
		row = 0
	}
	playSpellAnimation(actGroup, actMember, kind, row, g, i, flash)
}

// playSpellAnimation (play_spell_animation, WMEXE 695e) draws the three steps of spell
// kind cast by fighter i of group g at group tg, member ti.
func playSpellAnimation(g, i, kind, row, tg, ti int, flash bool) {
	if g == 0 && fighter(g, i).B31 == 0 {
		return
	}
	resetAnimation()
	pic, frame := 7, kind
	if kind >= 5 {
		if kind < 13 {
			pic, frame = 8, kind-5
		} else {
			pic, frame = 9, kind-13
		}
	}
	var step [3]int
	var side, mirror, start int
	symmetric := kind == 6 || kind == 8 || kind >= 10 && kind <= 13 || kind == 15
	if g == 0 || g == 6 {
		side, start = 0, 0
		step = [3]int{2, 1, 0}
		mirror = i % 2
	} else {
		side, start = 1, -1
		step = [3]int{0, 1, 2}
		mirror = 0
		if ti != -1 && ti%2 != 0 {
			mirror = 1
		}
	}
	flip := mirror
	if side == 1 {
		flip = b2i(symmetric == (mirror == 1))
	}
	playSound(mexeSpellSound)
	p := &monsterPics[pic]
	q := int(p.Unk104[frame])
	for s := 0; s < 3; s++ {
		f := byte(int(p.Quads[q][1]) + step[s])
		viewAnimState = -2
		mexeAnimateStep()
		at := kind*24 + side*12 + mirror*6 + s*2
		insertSprite(42, pic, int(spellAnimXY[at]), int(spellAnimXY[at+1]), 72, 248, flip, []byte{f})
		for {
			mexeAnimateStep()
			if viewAnimState == -2 {
				break
			}
		}
	}
	if flash {
		mexeEffect[row][0] = int16(start)
		animCycle = 0
		combatDelay(2)
	}
}

// spellPower (calc_spell_level_value, WMEXE 7816).
func spellPower(spell, power int) int { return int(spellTable[spell].Level)*3 + power*3 - 6 }

// sideOf is 1 when the acting fighter is on the party's side (spells at "the other
// side" hit the monsters).
func partySide() bool { return actGroup == 0 || actGroup == 6 }

// doCastAction (do_cast_action, WMEXE b03c) casts the acting fighter's spell (W10: power
// <<8 | spell): it may fizzle, backfire or be silenced; the stamina it costs.
func doCastAction(c *Combatant) int {
	monsterGroups[0].Count = uint8(partyCount)
	power, spell := int(c.W10)/256, int(c.W10)%256
	sp := &spellTable[spell]
	mexeName = combatantName(actGroup, actMember)
	mexeVerb = getMessage(3308)
	mexeText = getMessage(spell + 4000)
	printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, realmColors[sp.Realm])
	mexeBackfire, mexeFizzle = 0, 0
	grp := monsterGroups[actGroup]
	if random(int(uint8(grp.B396[5]))+int(c.Level)+1) < int(uint8(grp.B396[5])) {
		mexeFizzle = 1
	}
	if actGroup == 0 {
		p := &party[actMember]
		var chance int
		if p.Class == 5 || p.Class == 4 || p.Class == 13 {
			chance = int(p.Skills[25])/3 + 90 - int(sp.Level)*power
		} else {
			chance = int(p.Skills[13])/3 + 90 - int(sp.Level)*power
			mexeRaiseSkill(actMember, 13, 2)
		}
		if random(100) > chance {
			if hdr.Level == 1 || random(36) <= int(p.Stats[7])+18 {
				mexeFizzle = 1
			} else {
				mexeFizzle, mexeBackfire = 0, 1
			}
		}
	} else {
		chance := int(c.Level)*3 + 88 - int(sp.Level)*power
		if random(100) > chance {
			if hdr.Level == 2 || random(7) != 0 {
				mexeFizzle = 1
			} else {
				mexeFizzle, mexeBackfire = 0, 1
			}
		}
	}
	silenceable := true
	if actGroup == 0 {
		cl := party[actMember].Class
		silenceable = !(cl == 5 || cl == 4 || cl == 13)
	} else {
		k := groupMonsterRec[actGroup].Unk208
		silenceable = !(k == 5 || k == 4 || k == 13)
	}
	checkMiscast(sp)
	switch {
	case silenceable && c.Status[0] > 0:
		combatDelay(2)
		printMsgCentered(3200, 3, 2)
		combatDelay(int(g4fc8) / 2)
	case mexeFizzle == 1:
		combatDelay(2)
		printMsgCentered(3201, 5, 2)
		combatDelay(int(g4fc8) / 2)
	default:
		backfire(c)
		doCastSpell(c, spell, power)
	}
	if actGroup == 0 {
		party[actMember].SP[sp.Realm].Cur -= int16(int(sp.Cost) * power)
	}
	return random(power) + int(sp.Level) + power
}

// checkMiscast keeps a backfire only for spells that can (flag 2) and a fizzle only for
// those that can (flag 1).
func checkMiscast(sp *Spell) {
	if mexeBackfire == 1 && sp.B3&2 == 0 {
		mexeBackfire, mexeFizzle = 0, 1
	}
	if mexeFizzle == 1 && sp.B3&1 == 0 {
		mexeFizzle = 0
	}
}

// backfire turns a backfired spell on its caster.
func backfire(c *Combatant) {
	if mexeBackfire == 1 {
		combatDelay(2)
		printMsgCentered(3202, 4, 2)
		combatDelay(int(g4fc8) / 2)
		c.B31, c.B32 = int8(actGroup), int8(actMember)
	}
}

// doUseItemAction (do_use_item_action, WMEXE b44d) uses the acting member's item W10:
// its spell, if it has one.
func doUseItemAction(c *Combatant) {
	mexeName = combatantName(actGroup, actMember)
	power, spell, skill := 0, 0, -1
	if actGroup == 0 {
		p := &party[actMember]
		item := int(c.W10)
		rec := itemRec(int(p.Items[item].ID))
		mexeText = cstr(rec.Name[:])
		if rec.Unk28 != 0 {
			power, spell = int(rec.Unk28)/256, int(rec.Unk28)%256
			switch rec.Type {
			case 12:
				mexeVerb = getMessage(3313)
			case 13:
				mexeVerb = getMessage(3314)
				skill = 24
			case 16:
				mexeVerb = getMessage(3315)
			default:
				switch {
				case spell >= 80:
					mexeVerb = getMessage(3318)
				case power != 0:
					mexeVerb = getMessage(3312)
					skill = 22
				default:
					if p.Class == 6 {
						power = int(p.Skills[12])/15 + 2 - int(spellTable[spell].Level)
						mexeRaiseSkill(actMember, 12, 1)
						power = min(max(power, 0), 6)
					}
					if power == 0 {
						mexeVerb = getMessage(3305)
					} else {
						mexeVerb = getMessage(3304)
					}
				}
			}
			useItemCharge(item)
		} else if rec.Type == 2 || rec.Type == 4 {
			mexeVerb = getMessage(3317)
		} else {
			mexeVerb = getMessage(3316)
		}
		if rec.Unk69 == 7 {
			g4352 = 1
			skill = -1
			mexeVerb = getMessage(3312)
		}
		printCombatLine(mexeName, mexeVerb, mexeText, 1, 14, 11)
	}
	if power == 0 {
		combatDelay(int(g4fc8) / 2)
		return
	}
	mexeBackfire, mexeFizzle = 0, 0
	roll := random(int(spellTable[spell].Level)*10+power*5) - 10
	if actGroup == 0 {
		if skill != -1 {
			mexeRaiseSkill(actMember, skill, 3)
			if int(party[actMember].Skills[skill]) < roll {
				if hdr.Level == 1 || random(36) <= int(party[actMember].Stats[7])+18 {
					mexeFizzle = 1
				} else {
					mexeBackfire = 1
				}
			}
		}
	} else if int(c.Level)*5-2 < roll {
		if hdr.Level == 2 || random(7) != 0 {
			mexeFizzle = 1
		} else {
			mexeBackfire = 1
		}
	}
	checkMiscast(&spellTable[spell])
	if mexeFizzle == 1 {
		combatDelay(2)
		printMsgCentered(3201, 5, 2)
		combatDelay(int(g4fc8) / 2)
		return
	}
	backfire(c)
	doCastSpell(c, spell, power)
}

// doBreath (fb934, WMEXE) is a monster's breath (W10: its kind) at its target group,
// as strong as it is healthy; the stamina it costs.
func doBreath(c *Combatant) int {
	mexeDelayed = 0
	g, i := int(uint8(c.B31)), int(uint8(c.B32))
	pow, hp := 1, 25
	for range 6 {
		if int(c.HP) > hp {
			pow++
		}
		hp <<= 1
	}
	pow = min(max(int(int32(pow)*int32(c.Stamina)/int32(c.StaminaMax))+1, 1), 6)
	mod := pow << 2
	silenced := 0
	if c.Status[0] > 0 {
		silenced = 1
	}
	cost := 0
	smax := int(c.StaminaMax)
	switch c.W10 {
	case 1:
		actorLine(3340, 3700, 12, 0)
		spellAnimation(g, 0, 0, true)
		hitDamage(g, -pow, 2, 5, pow, 1, mod)
		cost = random(4) + smax/6
	case 2:
		actorLine(3341, 3705, 9, 0)
		spellAnimation(g, 0, 9, false)
		hitDamage(g, -pow, 2, 5, pow, 7, mod)
		cost = random(2) + smax/20
	case 3:
		actorLine(3340, 3701, 2, 0)
		spellAnimation(g, 0, 1, true)
		hitDamage(g, -pow, 3, 4, pow, 3, mod)
		cost = random(4) + smax/6
	case 8:
		actorLine(3340, 3706, 6, 0)
		spellAnimation(g, 0, 3, true)
		hitDamage(g, -pow, 1, 7, pow, 8, mod)
		cost = random(6) + smax/4
	case 10:
		if !actorLine(3345, -1, 6, silenced) {
			spellAnimation(g, 0, 16, false)
			hitDamage(g, -(pow + 1), 2, 4, pow, 10, mod)
		}
		cost = random(4) + smax/10
	case 12:
		if !actorLine(3346, -1, 5, silenced) {
			spellAnimation(g, 0, 16, false)
			hitDamageThen(g, -(pow + 1), 3479, 10, pow, mod, 1, 4, 7, false, pow*10, 2, pow*10, 0, 0)
		}
		cost = random(4) + smax/10
	case 4:
		actorLine(3341, 3702, 15, 0)
		spellAnimation(g, 0, 16, false)
		hitEffect(g, -pow, 3475, 4, 3, mod, false, pow*2, 1, pow)
		cost = random(4) + smax/10
	case 5:
		actorLine(3342, 3703, 14, 0)
		spellAnimation(g, 0, 3, false)
		hitDamageThen(g, -(pow + 1), 3480, 5, pow, mod, 1, 6, 8, false, pow, 4, pow, 0, 0)
		cost = random(4) + smax/6
	case 6:
		actorLine(3340, 3704, 11, 0)
		spellAnimation(g, 0, 3, false)
		hitDamageThen(g, -pow, 3477, 6, pow, mod, 1, 5, 5, false, pow*10, 2, pow, 0, 0)
		cost = random(3) + smax/8
	case 7:
		actorLine(3343, -1, 5, 0)
		spellAnimation(g, 0, 16, false)
		hitEffect(g, i, 3498, 13, 6, 15, false, 5, 5, 5)
		cost = random(2) + smax/20
	case 9:
		actorLine(3344, 3707, 7, 0)
		spellAnimation(g, 0, 16, false)
		hitEffect(g, -(pow + 1), 3474, 9, 2, mod, false, pow, 2, pow)
		cost = random(3) + smax/10
	case 11:
		if !actorLine(3347, -1, 3, silenced) {
			spellAnimation(g, 0, 16, false)
			hitEffect(g, -10, 3473, 10, 1, mod, false, pow, 1, pow)
		}
		cost = random(3) + smax/8
	}
	if mexeDelayed == 0 {
		combatDelay(int(g4fc8) / 2)
	}
	return cost
}

// weakenGroup (f88b1, WMEXE) weakens (weak) or slows the fighters who of group g by up
// to power, unless they resist (element 3).
func weakenGroup(g, who, mod, power int, weak bool) {
	targets(g, who, func(i int) {
		c := fighter(g, i)
		if uint8(c.Condition) >= 3 || c.Toggle != 0 {
			return
		}
		clearWindowArea(partyPictureWin, 0, 2, 39, 4)
		setCursor(partyPictureWin, 1, 2)
		combatDelay(1)
		mexeName = combatantName(g, i)
		printCombatCentered(mexeName, nameColor(g), 2)
		n := resistReduce(g, power, 3, mod)
		if n <= 0 || random(100) <= int(groupMonsterRec[g].Resist[3])-mod {
			printMsgCentered(3471, 14, 3)
			combatDelay(int(g4fc8) / 2)
			return
		}
		msg := 3494
		if weak {
			c.B40 = int8(clampToward((n-1)/2+1, int(c.B40), 1))
			c.B39 = int8(clampToward((n-1)/2+1, int(c.B39), 0))
			c.Level = int16(max(int(c.Level)-n, 1))
			msg = 3491
		} else {
			c.B41 = int8(clampToward(n*3, int(c.B41), 1))
			for k := range 4 {
				c.B24[k] = int8(uint8(c.B24[k]) / 2)
			}
			if c.B41 < 0 {
				for k := min(mexeAbs(int(c.B41))/10, 3); k > 0; k-- {
					c.B24[4-k] = 0
				}
			}
		}
		printMsgCentered(msg, 5, 3)
		if g == 0 {
			drawCardX(i, false)
		}
		combatDelay(int(g4fc8) / 2)
	})
}

func mexeAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// chaosStrike (f8bdb, WMEXE) hits each of the fighters who of group g with one of ten
// random effects, unless it resists (element 2).
func chaosStrike(g, who, power, mod int) {
	targets(g, who, func(i int) {
		if random(100) <= int(groupMonsterRec[g].Resist[2])-mod {
			c := fighter(g, i)
			if uint8(c.Condition) >= 3 || c.Toggle != 0 {
				return
			}
			clearWindowArea(partyPictureWin, 0, 2, 39, 4)
			setCursor(partyPictureWin, 1, 2)
			combatDelay(1)
			mexeName = combatantName(g, i)
			printCombatCentered(mexeName, nameColor(g), 2)
			printMsgCentered(3471, 14, 3)
			combatDelay(int(g4fc8) / 2)
			return
		}
		switch random(10) {
		case 0:
			damageRoll(g, i, 4, 7, power, 12, mod)
		case 1:
			hitEffect(g, i, 3476, 2, 4, mod, false, power*2, 4, power)
		case 2:
			hitEffect(g, i, 3474, 9, 2, mod, false, power, 2, 0)
		case 3:
			hitEffect(g, i, 3475, 4, 3, mod, false, power*2, 1, power)
		case 4:
			hitEffect(g, i, 3480, 5, 8, mod, false, power, 4, power)
		case 5:
			hitEffect(g, i, 3489, 11, 3, mod-25, true, 0, 0, 0)
		case 6:
			damageRoll(g, i, power*2, 4, power*2, 0, mod)
		case 7:
			hitEffect(g, i, 3479, 10, 7, mod, false, power, 2, power)
		case 8:
			hitEffect(g, i, 3485, 11, 2, mod-25, true, 0, 0, 0)
		case 9:
			damageRoll(g, i, 4, 7, power, 1, mod)
		}
	})
}

// summonGroup (f8fa8, WMEXE) calls up a group of one of the six monsters (more power,
// more of them): an ally group for the party, a new group for the monsters.
func summonGroup(mons [6]int, power int) {
	combatDelay(int(g4fc8) / 4)
	mexeDelayed = 0
	var g int
	switch {
	case partySide():
		g = 6
		if monsterGroups[6].Count > 0 {
			printMsgCentered(3217, 6, 2)
			return
		}
	case power > 0 && monsterGroupCount < 5:
		monsterGroupCount++
		g = monsterGroupCount
	default:
		printMsgCentered(3217, 6, 2)
		return
	}
	*monsterGroups[g] = MonsterGroup{}
	size := 0
	for {
		size++
		power--
		if !(power > 0 && random(3) == 0 && size < 3) {
			break
		}
	}
	mon := mons[0]
	if power >= 0 && power < 6 {
		mon = mons[power]
	}
	b := make([]byte, dbs.Size[MonsterRec]())
	accessScenarioRecord(dbs.RecMonster, mon, b, false)
	*groupMonsterRec[g], _ = dbs.Decode[MonsterRec](b)
	rec, grp := groupMonsterRec[g], monsterGroups[g]
	grp.MonsterID = uint8(mon)
	rec.GroupSize.Count = uint8(size)
	rec.GroupSize.Base = int16(size - 1)
	grp.Count = uint8(min(max(rollDice(rec.GroupSize), 1), 9))
	grp.B396 = [6]int8{}
	grp.B402 = [4]uint8{}
	grp.B412 = 0
	for i := 0; int(grp.Count) > i; i++ {
		c := &grp.Members[i]
		c.Level = int16(rec.HP.Count)
		c.HPMax = int16(rollDice(rec.HP))
		c.HP = c.HPMax
		c.StaminaMax = int16(rollDice(rec.Stamina))
		c.Stamina = c.StaminaMax
		c.Pos = -1
		c.B24[6] = 10
		c.B33 = 0
		c.B24[0] = 0
	}
	mexeName = itoa(int(grp.Count))
	mexeVerb = cstr(rec.Names[b2i(grp.Count != 1)][:])
	if grp.Count == 1 {
		mexeText = getMessage(3215)
	} else {
		mexeText = getMessage(3216)
	}
	setCursor(partyPictureWin, 20-(len(mexeName)+len(mexeVerb)+len(mexeText)+3)/2, 2)
	printCombatLine(mexeName, mexeVerb, mexeText, 1, 6, 6)
}

// raiseGroupField sets group byte v (a ward or a cloud's strength) toward power.
func raiseWard(w *int8, step int) { *w = int8(clampToward(step, int(uint8(*w)), 0)) }

// startCloud (cases 7, 25, 31, 43 of do_cast_spell) gives group g a cloud k of at least
// power that strikes at the next initiative count.
func startCloud(g, k, power int) {
	grp := monsterGroups[g]
	grp.B402[k] = uint8(max(int(grp.B402[k]), power) + 1)
	setGroupB406(grp, k, uint8(initiative+1))
}

// doCastSpell (do_cast_spell, WMEXE 9405) works spell at power from fighter c on its
// target (B31, B32).
func doCastSpell(c *Combatant, spell, power int) {
	tg, ti := int(uint8(c.B31)), int(uint8(c.B32))
	mexeDelayed = 0
	mod := spellPower(spell, power)
	if tg != 255 {
		mod = mod - int(fighter(tg, ti).Level) + int(c.Level)
	}
	other := b2i(partySide()) == 1 // the spell's side for "all monsters"
	own := monsterGroups[actGroup]
	me := actor()
	switch spell {
	case 0:
		spellAnimation(tg, ti, 2, true)
		hitDamage(tg, ti, 1, 7, power, 1, mod)
	case 1:
		spellAnimation(tg, ti, 3, false)
		hitEffect(tg, -(power + 1), 3476, 2, 4, mod, false, power, 2, power)
	case 2:
		spellAnimation(tg, 0, 0, true)
		hitDamage(tg, -(power + 3), 2, 5, power, 1, mod)
	case 3:
		raiseWard(&own.B396[0], power*15)
	case 4:
		spellAnimation(tg, 0, 0, true)
		hitDamage(tg, -(power + 4), 3, 3, power, 1, mod)
	case 5:
		spellAnimation(tg, 0, 2, true)
		hitDamage(tg, -(power + 3), 1, 10, power, 1, mod)
	case 6:
		spellAnimation(tg, 0, 2, true)
		chaosStrike(tg, -(power + 3), power, mod)
	case 7:
		spellAnimation(tg, 0, 0, true)
		hitDamage(tg, -10, 2, 4, power, 1, mod)
		startCloud(tg, 0, power)
	case 8:
		spellAnimation(b2i(actGroup == 0), 0, 0, true)
		sideGroups(other, func(g int) { hitDamage(g, -10, 5, 5, power, 1, mod) })
	case 9:
		spellAnimation(tg, ti, 1, true)
		hitDamage(tg, ti, 1, 6, power, 3, mod)
	case 11:
		spellAnimation(tg, 0, 16, false)
		hitEffect(tg, -10, 3473, 10, 1, mod, false, power, 3, power)
	case 12:
		spellAnimation(tg, 0, 16, false)
		weakenGroup(tg, -(power + 3), mod, power, true)
	case 13:
		spellAnimation(tg, 0, 16, false)
		weakenGroup(tg, -(power + 3), mod, power, false)
	case 14:
		for i := 0; int(own.Count) > i; i++ {
			if m := &own.Members[i]; uint8(m.Condition) < 3 {
				m.B41 = int8(clampToward(power*2, int(m.B41), 0))
			}
		}
	case 16:
		raiseWard(&own.B396[1], power*15)
	case 17:
		spellAnimation(tg, 0, 1, true)
		hitDamage(tg, -(power + 4), 3, 4, power, 3, mod)
	case 18:
		spellAnimation(tg, ti, 16, false)
		hitEffect(tg, ti, 3475, 4, 3, mod, false, power*2, 1, power)
	case 19:
		spellAnimation(tg, ti, 1, true)
		hitDamage(tg, ti, 3, 10, power, 3, mod)
	case 20:
		spellAnimation(tg, ti, 3, false)
		hitDamageThen(tg, ti, 3477, 6, power, mod, 1, 5, 5, false, power*10, 2, power, 0, 0)
	case 21:
		raiseWard(&own.B396[2], power*15)
	case 22:
		spellAnimation(tg, 0, 3, true)
		hitDamageThen(tg, -3, 3480, 5, power, mod, 1, 6, 8, false, power, 4, power, 0, 0)
	case 23:
		raiseWard(&own.B396[3], power*15)
	case 24:
		spellAnimation(tg, 0, 16, false)
		hitEffect(tg, -(power + 3), 3472, 12, 0, mod, false, power, 2, 0)
	case 25:
		spellAnimation(tg, 0, 3, false)
		hitDamageThen(tg, -10, 3477, 6, power, mod, 1, 4, 5, false, power*10, 2, power, 0, 0)
		startCloud(tg, 1, power)
	case 27:
		spellAnimation(tg, 0, 4, false)
		hitDamage(tg, -(power + 3), 1, 7, power, 5, mod)
	case 28:
		clearMsgLine(3)
		printMsgCentered(3224, 3, 3)
		combatDelay(int(g4fc8) / 2)
		for k := range own.B402 {
			own.B402[k] = uint8(max(int(own.B402[k])-power, 0))
		}
		if v := int(uint8(own.B396[3])); v > 0 {
			v = max(v-power*10, 0)
			own.B396[3] = int8(v)
			clearMsgLine(3)
			if v == 0 {
				printMsgCentered(3225, 3, 3)
			} else {
				printMsgCentered(3226, 3, 3)
			}
		}
	case 29:
		spellAnimation(tg, ti, 3, false)
		if random(100) < power*5 {
			hitEffect(tg, ti, 3489, 6, 3, mod, true, 0, 0, 0)
		} else {
			hitDamageThen(tg, ti, 3477, 6, power, mod, 3, 5, 5, false, power*10, 4, power*10, 0, 0)
		}
	case 31:
		spellAnimation(tg, 0, 3, false)
		hitDamageThen(tg, -10, 3480, 5, power, mod, 2, 3, 8, false, power, 4, power, 0, 0)
		startCloud(tg, 2, power)
	case 32:
		spellAnimation(tg, 0, 3, true)
		hitDamageThen(tg, -(power + 3), 3480, 5, power, mod, 1, 6, 8, false, power, 4, power, 0, 0)
	case 33:
		spellAnimation(tg, 0, 4, false)
		sideGroups(other, func(g int) { groupStatus(g, 3490, 5, 3, mod, true, power, power, power) })
	case 34:
		spellAnimation(b2i(actGroup == 0), 0, 4, false)
		sideGroups(other, func(g int) { hitDamage(g, -10, 4, 4, power, 5, mod) })
	case 35:
		spellAnimation(tg, ti, 3, true)
		hitDamage(tg, ti, 1, 7, power, 8, mod)
	case 36:
		spellAnimation(tg, ti, 3, false)
		hitEffect(tg, -3, 3487, 8, 9, mod, false, power, 2, power)
	case 37:
		t := fighter(tg, ti)
		t.B38 = int8(clampToward(power, int(t.B38), 1))
	case 40:
		spellAnimation(tg, 0, 15, false)
		hitDamage(tg, -(power + 3), 2, 4, power, 7, mod)
	case 42:
		spellAnimation(tg, ti, 2, false)
		hitEffect(tg, ti, 3488, 4, 3, mod, false, power*2, 1, power)
	case 43:
		spellAnimation(tg, 0, 3, true)
		hitDamage(tg, -10, 1, 6, power, 8, mod)
		startCloud(tg, 3, power)
	case 44:
		spellAnimation(tg, ti, 4, true)
		n := resistReduce(tg, power, 0, mod)
		for i := 0; int(monsterGroups[tg].Count) > i; i++ {
			m := fighter(tg, i)
			m.B38 = int8(clampToward(n, int(m.B38), 0))
		}
	case 45:
		switch {
		case partySide():
			summonGroup([6]int{212, 213, 214, 215, 216, 217}, power)
		case gameStateArg == 10:
			summonGroup([6]int{214, 214, 214, 214, 214, 214}, power)
		case gameStateArg == 11:
			summonGroup([6]int{214, 214, 215, 215, 216, 216}, power)
		case gameStateArg == 12:
			summonGroup([6]int{214, 214, 214, 215, 215, 217}, power)
		default:
			summonGroup([6]int{}, 0)
		}
	case 47:
		spellAnimation(tg, ti, 16, true)
		hitDamageThen(tg, ti, 3479, 10, power, mod, 1, 7, 7, false, power, 1, power, 0, 0)
	case 48:
		spellAnimation(tg, 0, 16, false)
		hitEffect(tg, -(power + 3), 3474, 9, 2, mod, false, power, 2, 0)
	case 49:
		n := (power-1)/2 + 1
		for i := 0; int(own.Count) > i; i++ {
			m := &own.Members[i]
			m.B38 = int8(clampToward(n, int(m.B38), 1))
			m.B39 = int8(clampToward(n, int(m.B39), 1))
		}
	case 50:
		spellAnimation(tg, ti, 16, false)
		hitEffect(tg, ti, 3496, 9, 3, mod, false, power*2, 1, power)
	case 54:
		for g := 1; g <= monsterGroupCount; g++ {
			if random(100) < power*10+80-gameStateArg*5 {
				monsterGroups[g].B412 = 0
			}
		}
	case 55:
		spellAnimation(tg, 0, 16, false)
		hitEffect(tg, -(power + 3), 3492, 9, 3, mod, false, power, 2, 0)
	case 58:
		spellAnimation(tg, 0, 16, true)
		hitDamageThen(tg, -(power + 3), 3479, 10, power, mod, 1, 7, 7, false, power*2, 2, power, 0, 0)
	case 59:
		switch {
		case partySide():
			summonGroup([6]int{218, 219, 220, 221, 222, 223}, power)
		case gameStateArg == 9:
			summonGroup([6]int{218, 218, 219, 219, 219, 222}, power)
		case gameStateArg == 10, gameStateArg == 12:
			summonGroup([6]int{219, 219, 219, 222, 222, 223}, power)
		case gameStateArg == 11:
			summonGroup([6]int{214, 214, 210, 216, 210, 216}, power)
		default:
			summonGroup([6]int{}, 0)
		}
	case 61:
		spellAnimation(tg, ti, 16, false)
		hitEffect(tg, ti, 3482, 11, 3, mod-35, true, 0, 0, 0)
	case 63:
		spellAnimation(tg, 0, 16, true)
		sideGroups(other, func(g int) { hitDamage(g, -10, 4, 4, power, 10, mod) })
	case 65:
		spellAnimation(tg, ti, 15, false)
		hitDamage(tg, ti, 1, 8, power, 12, mod)
	case 66:
		spellAnimation(tg, 0, 2, true)
		hitDamage(tg, -(power + 1), 1, 7, power, 12, mod)
	case 67, 75:
		spellAnimation(tg, 0, 2, true)
		kind, msg := uint8(20), 3493
		if spell == 75 {
			kind, msg = 21, 3497
		}
		if groupMonsterRec[tg].Unk208 == kind {
			mod += (int(c.Level) - int(fighter(tg, ti).Level)) * 3
		} else {
			mod = -100
		}
		hitEffect(tg, -power, msg, 11, 3, mod, true, 0, 0, 0)
	case 69:
		if me.B35[1] == 0 {
			me.B35[2] = int8(random(initiative/2) + initiative/2)
		}
		me.B35[1] = int8(uint8(me.B35[1]) + uint8(power*3))
	case 71:
		switch {
		case partySide():
			if random(2) == 0 {
				summonGroup([6]int{200, 201, 202, 203, 204, 205}, power)
			} else {
				summonGroup([6]int{206, 207, 208, 209, 210, 211}, power)
			}
		case own.MonsterID == 150 || own.MonsterID == 153:
			summonGroup([6]int{211, 211, 211, 211, 211, 211}, 1)
		case gameStateArg == 9:
			summonGroup([6]int{200, 207, 202, 208, 203, 209}, power)
		case gameStateArg == 10:
			summonGroup([6]int{201, 201, 208, 208, 208, 208}, power)
		case gameStateArg == 11:
			summonGroup([6]int{202, 202, 208, 203, 203, 210}, power)
		case gameStateArg == 12:
			summonGroup([6]int{206, 206, 208, 208, 209, 209}, power)
		default:
			summonGroup([6]int{}, 0)
		}
	case 72:
		raiseWard(&monsterGroups[tg].B396[5], power*2)
	case 76:
		spellAnimation(tg, 0, 4, false)
		sideGroups(other, func(g int) { hitDamage(g, -10, 4, 4, power, 11, mod) })
	case 78:
		spellAnimation(tg, 0, 16, false)
		sideGroups(other, func(g int) { groupStatus(g, 3486, 11, 3, mod, true, power, power, power) })
	case 79:
		spellAnimation(tg, ti, 4, false)
		k := groupMonsterRec[tg].Unk208
		if !(k == 22 && g4352 != 0 || k == 20 || k == 21) {
			mod = -100
		}
		t := fighter(tg, ti)
		n := max(int(int32(t.HP)<<3/int32(t.HPMax)), 2)
		hitDamage(tg, ti, 1, n, power, 0, mod)
	case 10:
		t := fighter(tg, ti)
		t.Stamina = min(t.Stamina+int16(rollDice(formatDice(power*2, 8, power))), t.StaminaMax)
		if tg == 0 {
			drawCardX(ti, false)
		}
	case 15:
		cure(tg, ti, &fighter(tg, ti).Status[3], power*25)
	case 26:
		t := fighter(tg, ti)
		if v := int(uint8(t.B35[0])) - power; v <= 0 {
			t.B35[0], t.Status[5] = 0, 0
		} else {
			t.B35[0] = int8(v)
		}
		if v := int(t.Status[5]) - power*5; v <= 0 {
			t.Status[5], t.B35[0] = 0, 0
		} else {
			t.Status[5] = uint8(v)
		}
		if tg == 0 {
			drawCardX(ti, false)
		}
	case 57:
		cure(tg, ti, &fighter(tg, ti).Status[7], power*25)
	case 51:
		t := fighter(tg, ti)
		for _, s := range []int{1, 2, 4, 8, 9} {
			t.Status[s] = uint8(max(int(t.Status[s])-power*2, 0))
		}
		if tg == 0 {
			drawCardX(ti, false)
		}
	case 74:
		spellAnimation(tg, 0, 4, false)
		dmg := damageRoll(tg, ti, power<<2, 4, power<<2, 11, mod)
		if tg == 0 {
			drawCardX(ti, false)
		}
		tg, ti = actGroup, actMember
		power = dmg / 7
		fallthrough
	case 64:
		t := fighter(tg, ti)
		t.HP = min(t.HP+int16(rollDice(formatDice(power, 7, power))), t.HPMax)
		if tg == 0 {
			drawCardX(ti, false)
		}
	case 80:
		me.Stamina = min(me.Stamina+int16(rollDice(formatDice(power, 8, power))), me.StaminaMax)
		me.HP = min(me.HP+int16(rollDice(formatDice(power, 2, 0))), me.HPMax)
		if actGroup == 0 {
			drawCardX(actMember, false)
		}
	case 81:
		if actGroup == 0 {
			p := &party[actMember]
			for r := range p.SP {
				p.SP[r].Cur = min(p.SP[r].Cur+int16(rollDice(formatDice(power, 10, 0))), p.SP[r].Max)
			}
		}
	case 53:
		partyCounters[2].A += int16(power * 50)
		partyCounters[2].B = int16(clampToward(power*5+70, int(partyCounters[2].B), 0))
	case 68:
		partyCounters[3].A += int16(power * 35)
		partyCounters[3].B = int16(clampToward((power-1)/2+1, int(partyCounters[3].B), 0))
	case 41:
		partyCounters[1].A += int16(power * 50)
		partyCounters[1].B = int16(clampToward((power-1)/2+1, int(partyCounters[1].B), 0))
	case 30:
		partyCounters[5].A += int16(power * 5)
	case 38:
		partyCounters[0].A += int16(100 * power)
	case 70:
		raiseWard(&own.B396[4], power*10)
		if actGroup == 0 {
			partyCounters[4].A += int16(power * 50)
			partyCounters[4].B = int16(clampToward(power*10, int(partyCounters[4].B), 0))
		}
	}
	if mexeDelayed == 0 {
		combatDelay(int(g4fc8) / 2)
	}
}

// cure lowers status s of fighter i of group g by n.
func cure(g, i int, s *uint8, n int) {
	*s = uint8(max(int(*s)-n, 0))
	if g == 0 {
		drawCardX(i, false)
	}
}
