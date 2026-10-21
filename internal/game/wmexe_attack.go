package game

// WMEXE: melee and missile attacks, damage and status effects.

// mexeRaiseSkill is try_raise_skill (WMEXE 77d4, the same as trySkillGain) for member m;
// the C also calls it for monsters, writing past the party (skipped here).
func mexeRaiseSkill(m, skill, difficulty int) {
	if m < len(party) {
		trySkillGain(m, skill, difficulty)
	}
}

// fatigueMod (f4572, WMEXE) is the to-hit (kind 0) or armour (kind 1) penalty for
// having stamina left of max.
func fatigueMod(kind, stamina, max int) int {
	tired := 100 - int32(stamina)*100/int32(max)
	pick := func(hit, ac int) int {
		if kind == 0 {
			return hit
		}
		return ac
	}
	switch {
	case tired <= 50:
		return 0
	case tired <= 66:
		return pick(5, 0)
	case tired <= 75:
		return pick(10, 1)
	case tired <= 90:
		return pick(20, 2)
	}
	return pick(40, 4)
}

// pickWeighted (pick_weighted, WMEXE 6eea) picks 0..7 by the percent weights w, else 4.
func pickWeighted(w []uint8) int {
	roll := random(100)
	for i := 0; i < 8; i++ {
		if int(w[i]) > roll {
			return i
		}
		roll -= int(w[i])
	}
	return 4
}

// attackFx is an attack's hit effect chances (8) and its drain strength: Hand.Data and
// B0C, or MonsterAttack.Unk4.
type attackFx [9]uint8

func handFx(h *Hand) attackFx {
	var fx attackFx
	copy(fx[:8], h.Data[:])
	fx[8] = h.B0C
	return fx
}

// useActorItemCharge (use_item_charge_x, WMEXE 7096) is useItemCharge for the acting
// member that also recounts the load. It uses up one charge of the acting
// member's item i; the last one removes the item.
func useActorItemCharge(i int) {
	p := &party[actMember]
	it := &p.Items[i]
	if it.Qty == 0 {
		return
	}
	it.Qty--
	if it.Qty != 0 {
		return
	}
	for s := 0; s < 8; s++ {
		if int(p.Equipped[s]) != i {
			continue
		}
		switch s {
		case 0, 1:
			clearHandAttackX(s)
		case 2:
			p.AC[2] = 0
		default:
			p.AC[s] = p.AC[0]
		}
		p.Equipped[s] = 255
	}
	p.Items[i].ID = 0
	p.ItemCount--
	for k := i; int(p.ItemCount) > k; k++ {
		p.Items[k] = p.Items[k+1]
		for s := 0; s < 8; s++ {
			if int(p.Equipped[s]) == k+1 {
				p.Equipped[s] = uint8(k)
			}
		}
	}
	calcCarriedWeight(actMember)
	p.LoadCat = uint8(getLoadCategory(actMember).StaminaDrain)
}

// clearHandAttackX (clear_hand_attack_x, WMEXE 6f24) is WPCVW's clearHandAttack for
// the acting member, which also cancels the member's swings.
func clearHandAttackX(h int) {
	clearHandAttack(actMember, h)
	c := fighter(0, actMember)
	c.B24[6] = 10
	for k := range 4 {
		c.B24[k] = 0
	}
}

// killCombatant (kill_combatant, WMEXE 7848) kills fighter i of group g, crediting the
// acting member and taking the monster off the view.
func killCombatant(g, i int) {
	c := fighter(g, i)
	if g != 0 {
		if actGroup == 0 && actMember >= 0 && actMember < partyCount {
			party[actMember].Kills++
		}
		if rec := groupMonsterRec[g]; rec.Flags&1 != 0 {
			setBit(hdr.Unk140[20:], int(rec.Unk211[7]), true) // DS:0x3498
		}
		if c.Pos != -1 {
			flashOffView(c, 0)
		}
	}
	c.Pos = -1
	c.Condition = 3
	c.HP, c.Stamina = 0, 0
	c.B35[1], c.B35[2] = 0, 0
	for k := range 4 {
		c.B24[k] = 0
	}
	if g >= 1 && g <= 5 {
		rec := groupMonsterRec[g]
		combatGold += int32(uint16(rec.Unk64)) | int32(rec.Unk66)<<16
	}
}

// effectMsg (f7a16, WMEXE) prints fighter i of group g with message n; shake (1) on a
// member plays sound and redraws the card.
func effectMsg(g, i, n, shake, sound int) {
	mexeName = combatantName(g, i)
	mexeVerb = getMessage(n)
	mexeText = ""
	if mexeMsgMode == 2 {
		clearWindowArea(partyPictureWin, 0, 2, 39, 4)
		clearMsgLine(2)
		printCombatCentered(mexeName, nameColor(g), 2)
		printCombatCentered(mexeVerb, 6, 3)
	} else {
		clearMsgLine(4)
		printCombatLine(mexeName, mexeVerb, mexeText, nameColor(g), 6, 0)
	}
	if shake == 1 && g == 0 {
		if sound > 0 {
			playSound(sound)
		}
		drawCardX(i, true)
	}
	combatDelay(int(g4fc8) / 2)
}

// loadPartyResist puts member i's resistances into the party's monster record.
func loadPartyResist(g, i int) {
	if g == 0 {
		groupMonsterRec[0].Resist = party[i].Resist
	}
}

// resistReduce (f7af7, WMEXE) lowers n by a random part of up to resistance r less mod
// percent of group g.
func resistReduce(g, n, r, mod int) int {
	cut := int(int16(int32(n) * int32(int(groupMonsterRec[g].Resist[r])-mod) / 100)) // C: (int) of a long
	cut = min(max(cut, 0), n)
	return n - random(int(int16(cut+1)))
}

// damageEffect (do_damage_effect, WMEXE 7b59) puts fighter i of group g into condition
// cond (3: dead) unless it resists r; says so with message n, or "resists" if
// report is 1.
func damageEffect(g, i, n, r, cond, mod, report int) {
	c := fighter(g, i)
	if int(uint8(c.Condition)) >= cond || c.Toggle != 0 {
		return
	}
	loadPartyResist(g, i)
	if r != 0 && random(100) <= int(groupMonsterRec[g].Resist[r])-mod {
		if report == 1 {
			effectMsg(g, i, 3471, 0, 0)
		}
		return
	}
	c.Condition = int8(cond)
	sound := 0
	if cond >= 3 {
		killCombatant(g, i)
		if g == 0 {
			sound = int(party[i].Sex) + 8
		}
	}
	effectMsg(g, i, n, 1, sound)
}

// statusEffect (do_status_effect, WMEXE 7c5a) adds count d sides + base rounds of status
// s to fighter i of group g unless it resists r (base 255: for good); poison (5) also
// gets strength base.
func statusEffect(g, i, n, r, s, mod, count, sides, base, report int) {
	c := fighter(g, i)
	if uint8(c.Condition) >= 2 || c.Toggle != 0 {
		return
	}
	loadPartyResist(g, i)
	d := rollDice(formatDice(count, sides, base))
	d = resistReduce(g, d, r, mod)
	if d <= 0 || random(100) <= int(groupMonsterRec[g].Resist[r])-mod {
		if report == 1 {
			effectMsg(g, i, 3471, 0, 0)
		}
		return
	}
	switch {
	case base == 255:
		c.Status[s] = 255
	case int(c.Status[s])+d > 250:
		c.Status[s] = 250
	default:
		c.Status[s] += uint8(d)
	}
	if s == 5 {
		c.B35[0] = int8(base)
	}
	effectMsg(g, i, n, 1, 0)
}

// damageRoll (f7ddc, WMEXE) deals times x count d sides damage to fighter i of group g,
// lessened by resistance r (wholly resisted on a roll), and reports it; the damage.
func damageRoll(g, i, count, sides, times, r, mod int) int {
	mexeMsgMode = 4
	c := fighter(g, i)
	if uint8(c.Condition) >= 3 || c.Toggle != 0 {
		return 0
	}
	dmg := 0
	for range times {
		for range count {
			dmg = int(int16(dmg + random(sides) + 1)) // C int wraps
		}
	}
	loadPartyResist(g, i)
	if random(100) <= int(groupMonsterRec[g].Resist[r])-mod {
		dmg = 0
	} else {
		dmg = resistReduce(g, dmg, r, mod)
	}
	clearWindowArea(partyPictureWin, 0, 2, 39, 4)
	setCursor(partyPictureWin, 1, 2)
	combatDelay(1)
	mexeName = combatantName(g, i)
	printCombatCentered(mexeName, nameColor(g), 2)
	if dmg <= 0 {
		printMsgCentered(3310, 14, 3)
	} else {
		printMsgCentered(3302, 4, 3)
		combatDelay(2)
		mexeName = itoa(dmg)
		mexeVerb = getMessage(3309)
		setCursor(partyPictureWin, 0, 4)
		mexeText = ""
		printCombatLine(mexeName, mexeVerb, mexeText, 5, 5, 0)
		c.HP -= int16(dmg)
		if c.HP <= 0 {
			combatDelay(int(g4fc8) / 4)
			damageEffect(g, i, 3486, 0, 3, 0, 0)
		} else {
			if g == 0 {
				playSound(2)
				drawCardX(i, true)
			}
			combatDelay(int(g4fc8) / 4)
		}
	}
	combatDelay(int(g4fc8) / 4)
	return dmg
}

// groupWard (f8064, WMEXE) lessens n by group g's ward k (B396[k]) when it holds
// against a roll of chance.
func groupWard(g, k, n, chance int) int {
	ward := int(uint8(monsterGroups[g].B396[k]))
	if random(chance) < ward {
		for cut := int(int32(ward) * int32(n) / 100); cut > 0; cut-- {
			n -= random(2)
		}
	}
	return max(n, 1)
}

// wardedDice (f80cc, WMEXE) passes n through the ward against element r and then the
// general one.
func wardedDice(g, r, n int) int {
	switch r {
	case 1:
		n = groupWard(g, 0, n, 10)
	case 3:
		n = groupWard(g, 1, n, 10)
	case 5:
		n = groupWard(g, 3, n, 10)
	}
	return groupWard(g, 4, n, 10)
}

// wardPair wards the dice count and sides, saying so if a ward held.
func wardPair(g, r int, ns ...*int) {
	changed := false
	for _, n := range ns {
		if *n > 0 {
			v := wardedDice(g, r, *n)
			changed = changed || v != *n
			*n = v
		}
	}
	if changed {
		showCombatMsg(3213)
	}
}

// targets calls f for the fighters of group g that who names: one (who >= 0) or up to
// -who of them from a random first.
func targets(g, who int, f func(i int)) {
	if who >= 0 {
		f(who)
		return
	}
	count := int(monsterGroups[g].Count)
	n := min(-who, count)
	first := random(count - n)
	for i := first; first+n > i; i++ {
		f(i)
	}
}

// targetRun calls f for |who| fighters of group g (at most all) from a random first;
// unlike targets, a who >= 0 is a count too (f88b1, f8bdb).
func targetRun(g, who int, f func(i int)) {
	count := int(monsterGroups[g].Count)
	n := min(mexeAbs(who), count)
	first := random(count - n)
	for i := first; first+n > i; i++ {
		f(i)
	}
}

// hitDamage (f8149, WMEXE) deals times x count d sides damage of element r to the
// fighters who of group g.
func hitDamage(g, who, count, sides, times, r, mod int) {
	wardPair(g, r, &count, &sides)
	targets(g, who, func(i int) {
		damageRoll(g, i, count, sides, times, r, mod)
		if g == 0 {
			drawCardX(i, false)
		}
	})
}

// hitEffect (f825f, WMEXE) puts the fighters who of group g into condition s
// (isDamage) or status s for count d sides + base rounds.
func hitEffect(g, who, n, r, s, mod int, isDamage bool, count, sides, base int) {
	mexeMsgMode = 2
	wardPair(g, r, &count, &sides)
	targets(g, who, func(i int) {
		if isDamage {
			damageEffect(g, i, n, r, s, mod, 1)
		} else {
			statusEffect(g, i, n, r, s, mod, count, sides, base, 1)
		}
		if g == 0 {
			drawCardX(i, false)
		}
	})
}

// hitDamageThen (f83f3, WMEXE) deals damage as hitDamage and puts each one hurt into
// condition or status s as hitEffect, unreported.
func hitDamageThen(g, who, n, r, times, mod, count, sides, s int, isDamage bool, count2, sides2, base2, _, _ int) {
	mexeMsgMode = 4
	wardPair(g, r, &count, &sides, &count2, &sides2)
	targets(g, who, func(i int) {
		if damageRoll(g, i, count, sides, times, r, mod) > 0 {
			if isDamage {
				damageEffect(g, i, n, r, s, mod, 0)
			} else {
				statusEffect(g, i, n, r, s, mod, count2, sides2, base2, 0)
			}
		}
		if g == 0 {
			drawCardX(i, false)
		}
	})
}

// sideGroups calls f for the groups of a side: monsters (1) or the party and its allies.
func sideGroups(monsters bool, f func(g int)) {
	if monsters {
		for g := 1; g <= monsterGroupCount; g++ {
			f(g)
		}
		return
	}
	f(0)
	if monsterGroups[6].Count > 0 {
		f(6)
	}
}

// groupStatus (f86aa, WMEXE) puts all of group g into condition or status s; a ward
// against element 5 or 11 spares the stronger ones.
func groupStatus(g, n, r, s, mod int, isDamage bool, count, sides, base int) {
	mexeMsgMode = 2
	strong := 0
	if s == 3 && isDamage {
		if r == 5 {
			if int(uint8(monsterGroups[g].B396[3]))/5 > count {
				clearMsgLine(2)
				printMsgCentered(3212, 5, 2)
				combatDelay(int(g4fc8) / 2)
				return
			}
			strong = 2
		}
		if r == 11 {
			strong = 3
		}
	}
	for i := 0; int(monsterGroups[g].Count) > i; i++ {
		m := mod
		if strong > 0 && int(fighter(g, i).Level) > count*strong {
			m = -100
		}
		hitEffect(g, i, n, r, s, m, isDamage, count, sides, base)
	}
}

// doAttack (do_attack, WMEXE 4c48) runs the acting fighter c's swings with hand or
// attack slot of kind (0..8: 4 throw, 8 the shield or the second attack).
func doAttack(c *Combatant, slot, kind int) {
	tg, ti := int(uint8(c.B31)), int(uint8(c.B32))
	anim, spell, quiet := -1, -1, false
	mexeMsgMode = 4
	mexeName = combatantName(actGroup, actMember)
	mexeVerb, mexeText = "", ""
	thrownOrBash := kind == 4 || kind == 8
	atkRec := groupMonsterRec[actGroup]
	if actGroup == 0 {
		p := &party[actMember]
		mexeVerb = getMessage(kind + 3350)
		item := int(p.Equipped[slot])
		if kind == 8 {
			item = int(p.Equipped[1])
		}
		if c.Toggle != 0 && !thrownOrBash {
			mexeVerb = getMessage(random(2) + 3385)
		}
		if item != 255 {
			rec := itemRec(int(p.Items[item].ID))
			mexeText = cstr(rec.Name[:])
			if thrownOrBash {
				if kind == 4 && rec.Unk28 != 0 {
					spell = int(rec.Unk28)
					mexeVerb = getMessage(3384)
				}
				switch p.Items[item].Icon {
				case 0:
					anim = 5
				case 20:
					anim = 11
				case 6:
					anim = 10
				case 18:
					anim = 13
				case 21:
					anim = 12
				case 22, 83:
					anim = 7
				case 23:
					anim = 9
				case 32, 100, 101:
					anim = 14
				case 34:
					anim = 3
				}
			}
		}
	} else if atkRec.AttackKind[slot] > 3 {
		mexeVerb = getMessage(int(atkRec.AttackKind[slot]) + 3371)
	} else {
		mexeVerb = getMessage(kind + 3350)
		if c.Toggle != 0 && !thrownOrBash {
			mexeVerb = getMessage(random(2) + 3385)
		}
		if thrownOrBash {
			d := atkRec.Attacks[slot].Damage
			switch atkRec.AttackVerb[slot] {
			case 2:
				anim = 6
			case 7:
				anim = 9
			case 9, 10:
				anim = 10
			case 13, 18, 19, 24:
				anim = 8
			case 28:
				anim = 12
			case 32:
				anim = 14
				spell = int(d.Count)<<8 + int(d.Sides)
			}
			if spell != -1 {
				mexeVerb = getMessage(3384)
			}
		}
		if v := atkRec.AttackVerb[slot]; v > 0 {
			mexeText = getMessage(int(v) + 3799)
		}
	}
	printCombatLine(mexeName, mexeVerb, mexeText, nameColor(actGroup), 14, 11)
	if spell == -1 {
		mexeName = combatantName(tg, ti)
		printCombatCentered(mexeName, nameColor(tg), 2)
	}
	if anim != -1 {
		spellAnimation(tg, ti, anim, spell != -1)
		quiet = true
	}

	t := fighter(tg, ti)
	tRec := groupMonsterRec[tg]
	sum := func(r *MonsterRec) int { return int(r.Unk135[2]) + int(r.Unk135[1]) + int(r.Unk135[0]) }
	statDiff := (sum(tRec) - sum(atkRec)) / 2
	bonus := int(c.B40)
	var toHit, ac, acArea, mult int
	var dice Dice
	var fx attackFx
	if actGroup == 0 {
		p := &party[actMember]
		skill := 9
		if kind == 4 {
			skill = 5
		} else if e := p.Equipped[slot]; e != 255 {
			switch p.Items[e].Icon {
			case 1, 14, 15, 16, 51, 53, 55, 58, 59:
				skill = 1
			case 2, 3, 20:
				skill = 2
			case 4, 5, 6, 52, 54, 56, 98:
				skill = 3
			case 7, 8, 9, 10, 17, 19, 57, 90:
				skill = 4
			case 11, 12:
				skill = 7
			case 13:
				skill = 6
			case 24, 38:
				skill = 8
			default:
				skill = 0
			}
		}
		if skill != 9 || p.Skills[9] > 0 || p.Class == 12 || p.Class == 13 {
			mexeRaiseSkill(actMember, skill, 2)
		}
		lc := getLoadCategory(actMember)
		load := lc.B
		if slot == 0 {
			load = lc.A
		}
		toHit = fatigueMod(0, int(c.Stamina), int(c.StaminaMax)) + int(p.ToHit) + int(p.Hands[slot].ToHit) +
			int(c.B39)*5 + statDiff + load - int(p.Skills[skill])/3
		dice = p.Hands[slot].Damage
		fx = handFx(&p.Hands[slot])
		item := int(p.Equipped[slot])
		if kind == 8 {
			if sh := int(p.Equipped[1]); sh != 255 && p.Items[sh].Type == 4 {
				dice = p.Hands[1].Damage
				fx = handFx(&p.Hands[1])
				toHit += int(p.Hands[1].ToHit)
				bonus = 0
				useActorItemCharge(sh)
			}
		} else if kind == 4 && item != 255 {
			if ty := p.Items[item].Type; ty == 14 || ty < 12 && ty != 2 {
				p.Items[item].Qty = 1
			}
			bonus = 0
			useActorItemCharge(item)
		}
		def := fatigueMod(1, int(t.Stamina), int(t.StaminaMax)) + int(t.B38)
		if partyCounters[3].A > 0 {
			def += int(partyCounters[3].B)
		}
		ac = int(int8(tRec.AC[0])) + def
		acArea = kindAC(tRec, kind) + def
	} else {
		a := &atkRec.Attacks[slot]
		toHit = fatigueMod(0, int(c.Stamina), int(c.StaminaMax)) + int(c.B39)*5 + int(a.ToHit) + statDiff
		dice = a.Damage
		copy(fx[:], a.Unk4[:])
		if thrownOrBash {
			bonus = 0
		}
		loadAC := 0
		if tg == 0 {
			loadAC = getLoadCategory(ti).AC
		}
		def := fatigueMod(1, int(t.Stamina), int(t.StaminaMax)) + int(t.B38)
		if tg == 0 && partyCounters[1].A > 0 {
			def -= int(partyCounters[1].B)
		}
		ac = int(int8(tRec.AC[0])) + def + loadAC
		acArea = def + loadAC
		if tg != 0 {
			acArea += kindAC(tRec, kind)
		}
	}

	mult = 1
	if c.Toggle != 0 {
		ac = max(ac, 10)
		mult = 2
	}
	if c.Status[4] > 0 {
		toHit = max(toHit, 90)
		ac = min(ac, 0)
	}
	if c.Status[8] > 0 || c.Status[1] > 0 || c.Status[9] > 0 {
		toHit += 25
	}
	if t.Status[4] > 0 || t.Status[2] > 0 || t.Status[3] > 0 || t.Condition == 2 {
		ac = 15
		mult = 2
	}
	if t.Status[8] > 0 || t.Status[1] > 0 {
		ac += 4
	}
	strength := 0
	if atkRec.Unk134 > 15 {
		strength++
	}
	if atkRec.Unk134 > 17 {
		strength++
	}
	bonus += strength
	swings := min(max(random(int(atkRec.Unk211[1])-int(t.B41)/10)+1, 1), 4)
	switch kind {
	case 1:
		ac -= 2
		acArea++
		bonus++
	case 2:
		acArea -= 2
		bonus += strength + 2
	case 3:
		ac -= 4
		acArea -= 4
		mult = 2
	case 4:
		swings, mult = 1, 1
		ac--
		acArea++
	case 5:
		ac++
		acArea--
	case 6:
		ac -= 2
		bonus += strength + 2
	case 7:
		ac += 2
		acArea -= 2
		mult = 1
	case 8:
		swings, mult = 1, 1
		acArea += 2
	}
	if t.B24[6] == 7 {
		ac = max(ac, 10)
		mult = 2
	}
	if t.Status[4] == 0 && t.Status[2] == 0 && t.Status[3] == 0 && t.Condition == 0 {
		if t.B24[6] == 1 {
			switch t.W10 {
			case 0:
				if tg == 0 {
					ac -= int(party[ti].Skills[8])/10 + 1
					mexeRaiseSkill(ti, 8, 2)
				} else {
					ac -= int(t.Level)/4 + 1
				}
				mult = 1
			case 1:
				ac -= 2
				mult = 1
			case 2:
				if !thrownOrBash {
					// The parry turns into a riposte at the next count.
					t.B24[6], t.B33 = 0, 0
					t.B24[0] = int8(initiative - 1)
					t.B31, t.B32 = int8(actGroup), int8(actMember)
					t.W12 = -1
					t.W10 = firstAttackMode(tg, ti)
					t.B24[4] = 0
				}
			case 3:
				ac = 15
			}
		} else if tg == 0 && party[ti].Equipped[1] != 255 && party[ti].Items[party[ti].Equipped[1]].Type == 11 {
			mexeRaiseSkill(ti, 8, 0)
		}
	}
	if s := groupMonsterRec[1].Sound; actGroup != 0 && !quiet && s > 0 && s != 6 && atkRec.Sound == s {
		playSound(14)
	}
	combatDelay(int(g4fc8) / 4)

	for n := 0; n < swings && uint8(t.Condition) < 3; n++ {
		roll := 0
		if g4fcc == 0 {
			roll = random(100)
		}
		var area, areaAC int
		if tg == 0 {
			area = pickWeighted(atkRec.Unk172[10:18])
		} else {
			area = pickWeighted(tRec.Unk172[2:10])
		}
		areaAC = int(int8(tRec.AC[area])) + acArea
		if t.Condition == 2 {
			areaAC = min(areaAC, 0)
		}
		clearWindowArea(partyPictureWin, 0, 3, 39, 4)
		if !quiet {
			playSound(6)
		}
		if tRec.Unk208 == 22 && spell == -1 {
			// Only the right weapon (item 458) once g4352 is set touches this kind.
			if actGroup == 0 {
				e := party[actMember].Equipped[slot]
				if e != 255 && party[actMember].Items[e].ID == 458 && g4352 != 0 {
					mult = 1
				} else {
					roll = 0
				}
			} else {
				roll = 0
			}
		}
		switch {
		case thrownOrBash && random(100) < int(uint8(monsterGroups[tg].B396[2])):
			printMsgCentered(3214, 14, 3)
			playSound(7)
		case roll < 5 || roll < 95 && toHit-ac*5 >= roll && spell == -1:
			printMsgCentered(3301, 14, 3)
		case spell != -1:
			mexeBackfire = 1
			doCastSpell(c, spell%256, spell/256)
		default:
			printMsgCentered(3302, 4, 3)
			playSound(7)
			if tRec.Unk208 == 17 {
				mexeName = getMessage(area + 3330)
			} else {
				mexeName = getMessage(area + 3320)
			}
			if c.Toggle == 1 {
				c.Toggle = -1
			}
			dmg := (rollDice(dice) + bonus) * mult
			if dmg <= 0 || roll < 95 && toHit-areaAC*5 >= roll {
				mexeVerb = getMessage(3303)
				setCursor(partyPictureWin, 0, 4)
				mexeText = ""
				printCombatLine(mexeName, mexeVerb, mexeText, 4, 14, 0)
				break
			}
			mexeVerb = itoa(dmg)
			mexeText = getMessage(3306)
			setCursor(partyPictureWin, 0, 4)
			printCombatLine(mexeName, mexeVerb, mexeText, 4, 5, 5)
			t.HP -= int16(dmg)
			t.Status[2] = 0
			t.Status[3] = uint8(min(max(int(t.Status[3])-(random(3)+2), 0), 255))
			if t.HP <= 0 {
				combatDelay(int(g4fc8) / 4)
				damageEffect(tg, ti, 3486, 0, 3, 0, 0)
				if c.Toggle == -1 {
					c.Toggle = 1
				}
			} else {
				if tg == 0 {
					playSound(2)
					drawCardX(ti, true)
				}
				combatDelay(int(g4fc8) / 4)
				if tRec.Unk208 != 22 {
					applyHitEffects(c, fx, tg, ti)
				}
			}
		}
		combatDelay(int(g4fc8) / 4)
	}
	if c.Toggle == -1 {
		c.Toggle = 0
	}
	if tg == 0 {
		drawCardX(ti, false)
	}
}

// applyHitEffects (apply_hit_effects, WMEXE 47a3) rolls each of the attack's hit
// effects on fighter i of group g: poison, paralysis, sleep, ..., drain.
func applyHitEffects(c *Combatant, fx attackFx, g, i int) {
	mexeMsgMode = 4
	mod := (int(c.Level) - int(fighter(g, i).Level)) * 3
	for k := 0; k < 8; k++ {
		if random(100) >= int(fx[k]) {
			continue
		}
		pow := int(c.Level)/3 + 1
		switch k {
		case 0:
			statusEffect(g, i, 3474, 9, 2, mod, pow, 2, pow, 0)
		case 1:
			statusEffect(g, i, 3475, 4, 3, mod, pow*pow*2, 5, pow, 0)
		case 2:
			statusEffect(g, i, 3477, 6, 5, mod, pow*10, 2, int(fx[8]), 0)
		case 3:
			damageEffect(g, i, 3485, 11, 2, mod, 0)
		case 5:
			if actGroup == 0 {
				break
			}
			drained, msg := 0, 0
			switch groupMonsterRec[actGroup].Unk211[6] {
			case 3:
				drained = rollDice(formatDice(pow*2, 5, pow*3))
				reduceStamina(fighter(g, i), drained)
				msg = 3751
			case 11:
				if random(100) > int(groupMonsterRec[g].Resist[13]) {
					drained = rollDice(formatDice(1, 4, 0))
					t := fighter(g, i)
					t.HPMax -= int16(drained)
					t.HP -= int16(drained)
					if t.HPMax < 1 {
						t.HPMax = 1
					}
					if g == 0 {
						party[i].HP, party[i].HPMax = t.HP, t.HPMax
					}
					msg = 3752
				}
			}
			if drained <= 0 {
				break
			}
			mexeName = combatantName(g, i)
			mexeText = replaceCharWithString(getMessage(msg), itoa(drained), '$')
			mexeVerb = getMessage(3750)
			clearMsgLine(4)
			drainColor := 5
			if g == 0 {
				drainColor = 1
			}
			printCombatLine(mexeName, mexeVerb, mexeText, drainColor, 5, 5)
			if g == 0 {
				playSound(int(party[i].Sex) + 8)
				drawCardX(i, true)
			}
			combatDelay(int(g4fc8) / 2)
			if fighter(g, i).HP <= 0 {
				damageEffect(g, i, 3486, 0, 3, mod, 0)
			}
		case 6:
			r := 13
			if g == 0 {
				r = 11
			}
			if random(100) > int(groupMonsterRec[g].Resist[r])-mod {
				clearMsgLine(4)
				printMsgCentered(3495, 5, 4)
				combatDelay(int(g4fc8) / 4)
				damageEffect(g, i, 3486, 0, 3, mod, 0)
			}
		case 7:
			statusEffect(g, i, 3484, 9, 2, mod, pow*2, 3, pow, 0)
		}
	}
}

// kindAC is the target's armour against attack kind k (rec byte 198+k). A confused
// fighter's second attack has kind -1 (W12), which the C reads as AC[7], the byte before.
func kindAC(rec *MonsterRec, k int) int {
	if k < 0 {
		return int(int8(rec.AC[8+k]))
	}
	return int(int8(rec.Unk198[k]))
}
