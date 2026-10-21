package game

// WPCVW: equipping and the derived combat values (resistances, AC, hands, load).

// canEquipItem (WPCVW c8c2) lists in out the inventory items member m may put in equip
// position pos (0 primary, 1 secondary, 2 armour .. 7) and returns how many.
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
			if pos == 0 {
				slot = 0
			} else if pos == 1 && it.Flags&4 != 0 {
				slot = 1
			}
		case 4, 11:
			slot = 1
		case 5, 6, 7, 8, 9, 10:
			slot = int(it.Type) - 3
		case 13, 14, 15:
			if it.Flags&4 != 0 {
				slot = 1
			}
		}
		if pos == 1 {
			// Ammunition only goes with its launcher, and only ammunition does.
			if e := p.Equipped[0]; e != 255 {
				w := &p.Items[e]
				if w.Type == 3 {
					if w.Icon == 11 && it.Icon != 22 || w.Icon == 13 && it.Icon != 23 || w.Icon == 12 && it.Icon != 83 {
						slot = -1
					} else if it.Type != 4 {
						slot = -1
					}
				} else if it.Type == 4 {
					slot = -1
				}
			} else if it.Type == 4 {
				slot = -1
			}
		}
		for s := 0; s < 8; s++ {
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
	for s := 0; s < 8; s++ {
		i := int(p.Equipped[s])
		if i == 255 {
			continue
		}
		it, name, color := p.Items[i], pvItemNames[i], pvItemRowColor[i]
		for j := i; j > top; j-- {
			p.Items[j] = p.Items[j-1]
			pvItemNames[j] = pvItemNames[j-1]
			pvItemRowColor[j] = pvItemRowColor[j-1]
			for t := s + 1; t < 8; t++ {
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

// addSkillBonus (add_skill_bonus, WPCVW cbf4) adds v to resistance r of member m, kept
// to 0..125.
func addSkillBonus(m, r, v int) {
	p := &party[m]
	p.Resist[r] = uint8(min(max(int(p.Resist[r])+v, 0), 125))
}

// wpcvwLevelCap is member m's level capped at limit, or its level before a class
// change (Unk1AF) when that was higher.
func wpcvwLevelCap(p *PC, limit int, minus1 bool) int {
	l := int(p.Level)
	if l < int(p.Unk1AF) {
		l = int(p.Unk1AF)
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
	lv := wpcvwLevelCap(p, 16, true)
	for r := 0; r < 14; r++ {
		p.Resist[r] = 0
		addSkillBonus(m, r, int(t[1][r]-'0')*lv+int(t[0][r]-'A')*4)
	}
}

// setAttackCounts (WPCVW ccd2) sets member m's attacks per round (Unk1A9 primary, Unk1AA
// secondary, at most most) and Unk1AB.
func setAttackCounts(m, most, a, b, c, d int) {
	p := &party[m]
	lv := wpcvwLevelCap(p, 20, false)
	f := func(stat, x, y int) uint8 {
		v := uint8(stat/6 + (lv+x-y)/x - 1)
		if v < 1 {
			v = 1
		}
		return min(v, uint8(most))
	}
	p.Unk1A9 = f(int(p.Stats[4]), a, b)
	p.Unk1AA = f(int(p.Stats[5]), c, d)
	p.Unk1AB = p.Stats[5] + uint8(lv)
}

// resetCombatValues (WPCVW cdb3) recomputes member m's bare resistances, attacks, hands
// and AC from class, race, stats and level.
func resetCombatValues(m int) {
	p := &party[m]
	g := map[uint8]int{0: 0, 4: 1, 8: 1, 10: 2, 11: 3, 13: 4, 1: 5, 5: 5, 7: 6, 2: 7, 9: 7, 12: 8, 3: 9, 6: 9}
	setBaseResists(m, resistTables[g[p.Class]])
	vit := int(p.Stats[3]) / 5 * 5
	switch p.Race {
	case 0:
		addSkillBonus(m, 11, 20)
	case 2, 4:
		addSkillBonus(m, 6, vit)
		addSkillBonus(m, 12, vit)
	case 3:
		addSkillBonus(m, 12, vit)
	case 1:
		addSkillBonus(m, 9, 20)
	case 5:
		for _, r := range []int{12, 4, 7, 10, 11} {
			addSkillBonus(m, r, 25)
		}
	case 6, 7:
		addSkillBonus(m, 10, 25)
		addSkillBonus(m, 8, 10)
		addSkillBonus(m, 3, -20)
	case 8:
		addSkillBonus(m, 7, 15)
		addSkillBonus(m, 4, 20)
		addSkillBonus(m, 3, 20)
	case 9:
		addSkillBonus(m, 3, 20)
	case 10:
		addSkillBonus(m, 12, 25)
		addSkillBonus(m, 3, 15)
	}
	if p.Stats[3] > 15 {
		addSkillBonus(m, 6, 5)
		addSkillBonus(m, 11, 5)
	}
	if p.Stats[1] > 15 {
		addSkillBonus(m, 9, 5)
	}
	if p.Stats[5] > 15 {
		addSkillBonus(m, 7, 5)
		addSkillBonus(m, 4, 5)
	}
	switch p.Class {
	case 0, 4, 8, 10, 11, 12, 13:
		setAttackCounts(m, 4, 7, 5, 6, 3)
	case 1, 5, 7, 9:
		setAttackCounts(m, 2, 12, 12, 7, 7)
	case 2, 3, 6:
		setAttackCounts(m, 3, 10, 8, 7, 5)
	}
	clearHandAttack(m, 0)
	h1 := &p.Hands[1]
	*h1 = Hand{Damage: Dice{Count: 2, Sides: 2}, Flags: 32}
	h0 := &p.Hands[0]
	if p.Stats[0] > 15 {
		h0.Damage.Base++
	}
	if p.Stats[0] > 17 {
		h0.Damage.Base++
	}
	if p.Stats[4] > 15 {
		h0.ToHit -= 5
	}
	if p.Stats[4] > 17 {
		h0.ToHit -= 5
		h1.ToHit -= 5
	}
	p.AC[0] = 10
	if p.Stats[5] > 15 {
		p.AC[0]--
	}
	if p.Stats[5] > 17 {
		p.AC[0]--
	}
	if p.Race == 5 {
		p.AC[0] -= 2
	}
	if p.Class == 13 || p.Class == 12 {
		l := 20
		if p.Level <= 20 {
			l = int(p.Level) / 2
		}
		p.AC[0] -= uint8(l + int(p.Skills[16])/10)
	}
	p.AC[1], p.AC[2] = 0, 0
	for s := 3; s < 8; s++ {
		p.AC[s] = p.AC[0]
	}
}

// doEquipItems (WPCVW d399) re-equips member m position by position, rebuilds its
// combat values from what it wears, then offers to invoke the special powers.
func doEquipItems(sheet, menu *Window, m int) {
	p := &party[m]
	var power [8]int
	for s := range p.Equipped {
		p.Equipped[s] = 255
	}
	for i := 0; i < int(p.ItemCount); i++ {
		p.Items[i].Flags &^= 1
	}
	p.Regen, p.Drain = 0, 0
	p.Resist = [14]uint8{}
	side := g4fb8
	if m%2 == 0 {
		side = g4fba
	}
	setCursor(side, 3, m/2<<2+1)
	putCharHi(side, 37, 4)
	putCharHi(side, 38, 4)
	resetCombatValues(m)
	drawInventoryPanel(sheet, m)
	cancel := getMessage(359)
	for s := 0; s < 8; s++ {
		var cand [10]int
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
			k := wpcvwGetMenuInput()
			if menuClicked == 1 {
				sel = -1
				if k < n {
					sel = k
				}
				break
			}
			if k == 2 || k == 4 {
				if sel == -1 {
					setCursor(menu, cx, cy)
					printStringHi(menu, cancel, 3)
					sel = 0
				} else {
					unhighlightItemRow(sheet, cand[sel], 4, false)
					if k == 2 {
						sel--
					} else if sel++; sel >= n {
						sel = -1
					}
				}
			}
			if k == 5 {
				break
			}
		}
		if sel != -1 {
			i := cand[sel]
			p.Equipped[s] = uint8(i)
			p.Items[i].Flags |= 1
			rec := itemRec(int(p.Items[i].ID))
			power[s] = int(rec.Unk68)
			if s <= 1 {
				h := &p.Hands[s]
				p.AC[1] -= rec.AC
				h.B0D = rec.HandD
				h.Data = rec.HandData
				kiri := int(p.Skills[29]) / 5
				if p.Class == 12 || p.Class == 13 {
					h.Data[6] += uint8(kiri + 5)
				} else if kiri > 0 {
					h.Data[6] += uint8(kiri)
				}
				h.B0C = rec.HandC
				if rec.Type == 4 {
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
			for r := 1; r < 14; r++ {
				addSkillBonus(m, r, int(rec.SkillBonus[r]))
			}
			p.Regen = max(p.Regen, rec.Regen)
			p.Drain = max(p.Drain, rec.Drain)
			if s == 0 && p.Items[i].Flags&8 != 0 {
				s++ // the C skips the secondary position after an identified weapon
			}
			sortEquippedFirst(m)
		}
		drawInventoryPanel(sheet, m)
	}

	for s := 0; s < 8; s++ {
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
		if wpcvwRunMenu(menu, 381, []int{1, 1, -1}, int(menu.CurX), 2, 3, 1, 5, 0) != 1 {
			continue
		}
		switch v := power[s]; {
		case v < 9:
			if p.Stats[v-1] < 20 {
				p.Stats[v-1]++
			}
		case v == 9:
			p.Unk1A9 = max(p.Unk1A9, 4)
		case v == 10:
			p.Unk1AA = max(p.Unk1AA, 4)
		case v == 11:
			p.Condition = 0
			p.Status = [10]uint8{}
			p.HP, p.Stamina = p.HPMax, p.StaminaMax
		case v == 12:
			if p.AgeDays/365 > 18 {
				p.AgeDays -= 365
			}
			if p.Stats[3] < 18 {
				p.Stats[3]++
			}
		case v == 13:
			n := int16(random(5) + 2)
			p.HPMax += n
			p.HP += n
		}
		useItemChargeShown(m, int(p.Equipped[s]))
		drawSheet(sheet, menu, m)
	}
}
