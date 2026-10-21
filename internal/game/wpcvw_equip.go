package game

// WPCVW: equipping and the derived combat values (resistances, AC, hands, load).

// canEquipItem (WPCVW c8c2) lists in out the inventory items member m may put in equip
// position pos (equipPrimary..equipBoots) and returns how many.
func canEquipItem(m, pos int, out *[10]int) int {
	p := &party[m]
	for i := range out {
		out[i] = -1
	}
	n := 0
	for i := 0; i < int(p.ItemCount); i++ {
		it := &p.Items[i]
		slot := -1
		switch it.Type {
		case 0, 1, 2, 3, 12, 16:
			if pos == equipPrimary {
				slot = equipPrimary
			} else if pos == equipSecondary && it.Flags&itemOffHand != 0 {
				slot = equipSecondary
			}
		case itemTypeMissile, itemTypeShield:
			slot = equipSecondary
		case itemTypeMisc, 6, 7, 8, 9, 10:
			slot = int(it.Type) - itemTypeMisc + equipMisc
		case 13, 14, 15:
			if it.Flags&itemOffHand != 0 {
				slot = equipSecondary
			}
		}
		if pos == equipSecondary {
			// Ammunition only goes with its launcher, and only ammunition does.
			if e := p.Equipped[equipPrimary]; e != noEquippedItem {
				w := &p.Items[e]
				if w.Type == itemTypeLauncher {
					if w.Icon == 11 && it.Icon != 22 || w.Icon == 13 && it.Icon != 23 || w.Icon == 12 && it.Icon != 83 {
						slot = -1
					} else if it.Type != itemTypeMissile {
						slot = -1
					}
				} else if it.Type == itemTypeMissile {
					slot = -1
				}
			} else if it.Type == itemTypeMissile {
				slot = -1
			}
		}
		for s := range equipSlots {
			if int(p.Equipped[s]) == i {
				slot = -1
			}
		}
		if slot != pos {
			continue
		}
		rec := itemRec(int(it.ID))
		if testBit(mask16(rec.ClassMask), int(p.Class)) && testBit(mask16(rec.RaceMask), int(p.Race)) &&
			testBit([]byte{rec.SexMask, rec.Flags}, int(p.Sex)) {
			out[n] = i
			n++
		}
	}
	return n
}

// sortEquippedFirst (WPCVW c726) moves the equipped items of member m to the top of the
// inventory, in equip position order.
func sortEquippedFirst(m int) {
	p := &party[m]
	top := 0
	for s := range equipSlots {
		i := int(p.Equipped[s])
		if i == noEquippedItem {
			continue
		}
		it, name, color := p.Items[i], pvItemNames[i], pvItemRowColor[i]
		for j := i; j > top; j-- {
			p.Items[j] = p.Items[j-1]
			pvItemNames[j] = pvItemNames[j-1]
			pvItemRowColor[j] = pvItemRowColor[j-1]
			for t := s + 1; t < equipSlots; t++ {
				if int(p.Equipped[t]) == j-1 {
					p.Equipped[t] = uint8(j)
				}
			}
		}
		p.Items[top], pvItemNames[top], pvItemRowColor[top] = it, name, color
		p.Equipped[s] = uint8(top)
		top++
	}
}

// addResistance (add_skill_bonus, WPCVW cbf4) adds v to resistance r of member m, kept
// to 0..125.
func addResistance(m, r, v int) {
	p := &party[m]
	p.Resist[r] = uint8(min(max(int(int16(int(p.Resist[r])+v)), 0), maxResist)) // C int sum wraps
}

// levelCap is member m's level capped at limit, or its level before a class
// change (LevelBeforeClassChange) when that was higher.
func levelCap(p *PC, limit int, minus1 bool) int {
	l := int(p.Level)
	if l < int(p.LevelBeforeClassChange) {
		l = int(p.LevelBeforeClassChange)
	}
	if minus1 {
		if l <= limit {
			return l - 1
		}
		return limit
	}
	return min(l, limit)
}

// setBaseResists (WPCVW cc36) sets member m's resistances from a class group's table:
// base letter*4 plus growth digit per level.
func setBaseResists(m int, t [2]string) {
	p := &party[m]
	lv := levelCap(p, 16, true) // the resistances grow until level 16
	for r := range p.Resist {
		p.Resist[r] = 0
		addResistance(m, r, int(t[1][r]-'0')*lv+int(t[0][r]-'A')*4)
	}
}

// setAttackCounts (WPCVW ccd2) sets member m's attacks per round (PrimaryAttacks and
// SecondaryAttacks, at most most) and AttackPower: one more every step levels from
// level step-shift+1 on.
func setAttackCounts(m, most, primaryStep, primaryShift, secondaryStep, secondaryShift int) {
	p := &party[m]
	lv := levelCap(p, maxCombatLevel, false)
	f := func(stat, step, shift int) uint8 {
		v := uint8(stat/6 + (lv+step-shift)/step - 1)
		if v < 1 {
			v = 1
		}
		return min(v, uint8(most))
	}
	p.PrimaryAttacks = f(int(p.Stats[statDexterity]), primaryStep, primaryShift)
	p.SecondaryAttacks = f(int(p.Stats[statSpeed]), secondaryStep, secondaryShift)
	p.AttackPower = p.Stats[statSpeed] + uint8(lv)
}

// setClassAttackCounts (in WPCVW cdb3 and eff8) is setAttackCounts with member m's
// class's steps.
func setClassAttackCounts(m int) {
	switch party[m].Class {
	case classFighter, classRanger, classValkyrie, classLord, classSamurai, classMonk, classNinja:
		setAttackCounts(m, 4, 7, 5, 6, 3)
	case classMage, classAlchemist, classPsionic, classBishop:
		setAttackCounts(m, 2, 12, 12, 7, 7)
	case classPriest, classThief, classBard:
		setAttackCounts(m, 3, 10, 8, 7, 5)
	}
}

// naturalAC is member m's armour class before items: baseAC, less for speed, faeries and
// a monk's or ninja's level and ninjutsu (as in WPCVW cdb3 and eff8).
func naturalAC(p *PC) int {
	ac := baseAC
	if p.Stats[statSpeed] > highStat {
		ac--
	}
	if p.Stats[statSpeed] > veryHighStat {
		ac--
	}
	if p.Race == raceFaerie {
		ac -= 2
	}
	if p.Class == classNinja || p.Class == classMonk {
		l := 20 // past maxCombatLevel, as in the C
		if p.Level <= maxCombatLevel {
			l = int(p.Level) / 2
		}
		ac -= l + int(p.Skills[skillHide])/10
	}
	return ac
}

// resetCombatValues (WPCVW cdb3) recomputes member m's bare resistances, attacks, hands
// and AC from class, race, stats and level.
func resetCombatValues(m int) {
	p := &party[m]
	g := map[uint8]int{
		classFighter: 0, classRanger: 1, classValkyrie: 1, classLord: 2, classSamurai: 3, classNinja: 4,
		classMage: 5, classAlchemist: 5, classPsionic: 6, classPriest: 7, classBishop: 7, classMonk: 8,
		classThief: 9, classBard: 9,
	}
	setBaseResists(m, resistTables[g[p.Class]])
	vit := int(p.Stats[statVitality]) / 5 * 5
	switch p.Race {
	case raceHuman:
		addResistance(m, 11, 20)
	case raceDwarf, raceHobbit:
		addResistance(m, 6, vit)
		addResistance(m, 12, vit)
	case raceGnome:
		addResistance(m, 12, vit)
	case raceElf:
		addResistance(m, 9, 20)
	case raceFaerie:
		for _, r := range []int{12, 4, 7, 10, 11} {
			addResistance(m, r, 25)
		}
	case raceLizardman, raceDracon:
		addResistance(m, 10, 25)
		addResistance(m, 8, 10)
		addResistance(m, 3, -20)
	case raceFelpurr:
		addResistance(m, 7, 15)
		addResistance(m, 4, 20)
		addResistance(m, 3, 20)
	case raceRawulf:
		addResistance(m, 3, 20)
	case raceMook:
		addResistance(m, 12, 25)
		addResistance(m, 3, 15)
	}
	if p.Stats[statVitality] > highStat {
		addResistance(m, 6, 5)
		addResistance(m, 11, 5)
	}
	if p.Stats[statIntelligence] > highStat {
		addResistance(m, 9, 5)
	}
	if p.Stats[statSpeed] > highStat {
		addResistance(m, 7, 5)
		addResistance(m, 4, 5)
	}
	setClassAttackCounts(m)
	clearHandAttack(m, equipPrimary)
	h1 := &p.Hands[equipSecondary]
	*h1 = Hand{Damage: Dice{Count: bareHandDamage, Sides: bareHandDamage}, Flags: handPunch}
	h0 := &p.Hands[equipPrimary]
	if p.Stats[statStrength] > highStat {
		h0.Damage.Base++
	}
	if p.Stats[statStrength] > veryHighStat {
		h0.Damage.Base++
	}
	if p.Stats[statDexterity] > highStat {
		h0.ToHit -= 5
	}
	if p.Stats[statDexterity] > veryHighStat {
		h0.ToHit -= 5
		h1.ToHit -= 5
	}
	p.AC[acBase] = uint8(naturalAC(p))
	p.AC[acHands], p.AC[equipMisc] = 0, 0
	for s := equipHelmet; s < equipSlots; s++ {
		p.AC[s] = p.AC[acBase]
	}
}

// The special powers (ItemRec.Unk68) doEquipItems may invoke.
const (
	powerLastStat         = statCount // 1..statCount raise stat power-1
	powerPrimaryAttacks   = 9
	powerSecondaryAttacks = 10
	powerRestore          = 11 // cures and restores hit points and stamina
	powerYouth            = 12
	powerHitPoints        = 13
)

// maxCombatLevel is the last level that improves the attacks and to-hit.
const maxCombatLevel = 20

// doEquipItems (WPCVW d399) re-equips member m position by position, rebuilds its
// combat values from what it wears, then offers to invoke the special powers.
func doEquipItems(sheet, menu *Window, m int) {
	p := &party[m]
	var power [equipSlots]int
	for s := range p.Equipped {
		p.Equipped[s] = noEquippedItem
	}
	for i := 0; i < int(p.ItemCount); i++ {
		p.Items[i].Flags &^= itemEquipped
	}
	p.Regen, p.Drain = 0, 0
	p.Resist = [14]uint8{}
	side := rightPanelWin
	if m%2 == 0 {
		side = leftPanelWin
	}
	setCursor(side, 3, m/2<<2+1)
	putCharHi(side, 37, 4)
	putCharHi(side, 38, 4)
	resetCombatValues(m)
	drawInventoryPanel(sheet, m)
	cancel := getMessage(359)
	for s := 0; s < equipSlots; s++ {
		var cand [packSlots]int
		n := canEquipItem(m, s, &cand)
		if n <= 0 {
			continue
		}
		fillWindow(menu, ' ', 3)
		refreshWindow(menu)
		setCursor(menu, 2, 1)
		printStringHi(menu, replaceCharWithString(getMessage(350), getMessage(s+351), '$'), 3)
		menu.CurX++
		cx, cy := int(menu.CurX), int(menu.CurY)
		if mousePresent {
			printStringHi(menu, cancel, 3)
		} else {
			printString(menu, cancel, -5)
		}
		sx, sy := int(sheet.X), int(sheet.Y)
		for k := 0; k < n; k++ {
			setCursor(sheet, 21, cand[k]+9)
			putChar(sheet, 100, 5)
			printString(sheet, pvItemNames[cand[k]], 4)
			setRegion(k, sx+21, sy+cand[k]+9, sx+36, sy+cand[k]+9)
		}
		setRegion(n, int(menu.X)+cx, int(menu.Y)+cy, len(cancel)+int(menu.X)+cx-1, int(menu.Y)+cy)
		endRegions(n + 1)
		sel := -1
		for {
			if !mousePresent {
				if sel == -1 {
					setCursor(menu, cx, cy)
					printString(menu, cancel, -5)
				} else {
					setCursor(sheet, 22, cand[sel]+9)
					printString(sheet, pvItemNames[cand[sel]], -5)
				}
			}
			k := getMenuInputBlink()
			if menuClicked == 1 {
				sel = -1
				if k < n {
					sel = k
				}
				break
			}
			if k == menuKeyUp || k == menuKeyDown {
				if sel == -1 {
					setCursor(menu, cx, cy)
					printStringHi(menu, cancel, 3)
					sel = 0
				} else {
					unhighlightItemRow(sheet, cand[sel], 4, false)
					if k == menuKeyUp {
						sel--
					} else if sel++; sel >= n {
						sel = -1
					}
				}
			}
			if k == menuKeyEnter {
				break
			}
		}
		if sel != -1 {
			i := cand[sel]
			p.Equipped[s] = uint8(i)
			p.Items[i].Flags |= itemEquipped
			rec := itemRec(int(p.Items[i].ID))
			power[s] = int(rec.Unk68)
			if s <= equipSecondary {
				h := &p.Hands[s]
				p.AC[acHands] -= rec.AC
				h.B0D = rec.HandD
				h.Data = rec.HandData
				kiri := int(p.Skills[skillKirijutsu]) / 5
				if p.Class == classMonk || p.Class == classNinja {
					h.Data[handCritical] += uint8(kiri + 5)
				} else if kiri > 0 {
					h.Data[handCritical] += uint8(kiri)
				}
				h.B0C = rec.HandC
				if rec.Type == itemTypeMissile {
					h.ToHit = -int8(rec.ToHitPen) * 5
				} else {
					h.ToHit -= int8(rec.ToHitPen) * 5
				}
				h.Damage = rec.Damage
				h.Flags = rec.HandFlags
				setCursor(side, s+3, m/2<<2+1)
				putCharHi(side, p.Items[i].Icon+1, 4)
			} else {
				p.AC[s] -= rec.AC
			}
			for r := 1; r < len(p.Resist); r++ {
				addResistance(m, r, int(rec.SkillBonus[r]))
			}
			p.Regen = max(p.Regen, rec.Regen)
			p.Drain = max(p.Drain, rec.Drain)
			if s == equipPrimary && p.Items[i].Flags&itemTwoHanded != 0 {
				s++ // a two-handed weapon leaves no secondary position
			}
			sortEquippedFirst(m)
		}
		drawInventoryPanel(sheet, m)
	}

	for s := range equipSlots {
		if power[s] <= 0 {
			continue
		}
		fillWindow(menu, ' ', 3)
		refreshWindow(menu)
		t := getMessage(380)
		setCursor(menu, 20-(len(t)+1)/2, 1)
		printStringHi(menu, t, 3)
		name := pvItemNames[p.Equipped[s]]
		setCursor(menu, 20-(len(name)+11)/2, 2)
		printStringHi(menu, name, 3)
		menu.CurX++
		putCharHi(menu, '?', 3)
		menu.CurX += 2
		if menuSelect(menu, 381, []int{1, 1, -1}, int(menu.CurX), 2, 3, 1, 5, 0, getMenuInputBlink) != 1 {
			continue
		}
		switch v := power[s]; {
		case v <= powerLastStat: // stat v-1
			if p.Stats[v-1] < 20 {
				p.Stats[v-1]++
			}
		case v == powerPrimaryAttacks:
			p.PrimaryAttacks = max(p.PrimaryAttacks, 4)
		case v == powerSecondaryAttacks:
			p.SecondaryAttacks = max(p.SecondaryAttacks, 4)
		case v == powerRestore:
			p.Condition = conditionOK
			p.Status = [10]uint8{}
			p.HP, p.Stamina = p.HPMax, p.StaminaMax
		case v == powerYouth:
			if p.AgeDays/daysPerYear > adultAge {
				p.AgeDays -= daysPerYear
			}
			if p.Stats[statVitality] < maxStat {
				p.Stats[statVitality]++
			}
		case v == powerHitPoints:
			n := int16(random(5) + 2)
			p.HPMax += n
			p.HP += n
		}
		useItemChargeShown(m, int(p.Equipped[s]))
		drawSheet(sheet, menu, m)
	}
}
