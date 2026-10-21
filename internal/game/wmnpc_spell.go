package game

import "strings"

// npc_choose_spell (WMNPC 8ce8) is WDOPT's choose_spell with the NPC's menu input: member
// m's spell book. Pick a realm, a known spell, then a power level the member can pay;
// power<<8 + spell, or -1. A spell cast on a member also asks for it (npcTarget).
func npcChooseSpell(_ *Window, m int) int {
	p := &party[m]
	win := createWindow(0, 18, 23, 7, 25, -1, 0, false)
	list := createWindow(22, 19, 18, 6, 26, -5, 0, false)
	done := func() {
		pvBlinkState = -1
		destroyWindow(win, false)
		destroyWindow(list, false)
	}
	wpopsDrawSpellWindow(win, m)
	drawListFrame(list, true)
	realm := 0
	for i := range 6 {
		x, y := i/3*6+1, i%3<<1+1
		setClickRegion(i, win.X+byte(x), win.Y+byte(y), win.X+byte(x), win.Y+byte(y))
	}
	setClickRegion(6, win.X+13, win.Y+3, win.X+13, win.Y+3)
	var spells [20]int
	var count, sel int
	for {
		// The realm.
		setClickRegion(7, 255, 0, 0, 0)
		redraw := true
		pvBlinkWin = win
		for {
			if redraw {
				pvBlinkState = 1
				clearWindowArea(win, 13, 1, 20, 1)
				setCursor(win, 13, 1)
				if realm == -1 {
					pvBlinkX, pvBlinkY = 13, 3
					printMessage(3956, win, 9)
					clearWindowArea(win, 18, 5, 20, 5)
					count = -1
					drawListFrame(list, true)
				} else {
					pvBlinkX, pvBlinkY = realm/3*6+1, realm%3<<1+1
					printMessage(realm+3950, win, realmColors[realm])
					clearWindowArea(win, 18, 5, 20, 5)
					count = wpcvwCountKnownSpells(m, realm, spells[:])
					wpopsDrawSpellList(list, realm, 0, false, count, spells[:])
				}
				pvBlinkText = [2]string{string(rune(realm + 35)), "c"}
				pvBlinkColor = [2]int{17, 5}
				redraw = false
			}
			c := npcMenuInput()
			if menuClicked == 1 {
				if c == realm {
					c = 5
				} else {
					wpopsUnhighlightRealm(win, realm)
					if c < 6 {
						realm, c = c, 0
					} else {
						realm, c, count = -1, 5, -1
					}
				}
				redraw = true
			} else {
				switch c {
				case 1:
					if realm == -1 {
						wpopsUnhighlightRealm(win, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						wpopsUnhighlightRealm(win, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						wpopsUnhighlightRealm(win, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						wpopsUnhighlightRealm(win, realm)
						redraw = true
						if realm += 3; realm >= 6 {
							realm = -1
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						wpopsUnhighlightRealm(win, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if c == 5 && count != 0 {
				break
			}
		}
		if realm == -1 {
			done()
			return -1
		}
		// The spell.
		wpopsUnhighlightRealm(win, realm)
		sel = 0
		back := false
		pvBlinkWin, pvBlinkX, pvBlinkY = list, 2, 2
		pvBlinkText[0] = strings.Repeat(" ", 15)
		pvBlinkColor[0] = 0
		setClickRegion(7, list.X+1, list.Y, list.X+1, list.Y)
		setClickRegion(8, list.X+1, list.Y+4, list.X+1, list.Y+4)
		setClickRegion(9, list.X+2, list.Y+2, list.X+19, list.Y+2)
		setClickRegion(10, 255, 0, 0, 0)
		for {
			pvBlinkState = 1
			pvBlinkText[1] = getMessage(spells[sel] + 4000)
			pvBlinkColor[1] = realmColors[realm]
			wpopsDrawSpellList(list, realm, sel, true, count, spells[:])
			cost := int(spellTable[spells[sel]].Cost)
			setCursor(win, 18, 5)
			if int(p.SP[realm].Cur) < cost {
				printString(win, "***", realmColors[realm])
			} else {
				printInt(win, cost, 3, realmColors[realm])
			}
			c := npcMenuInput()
			if menuClicked == 1 {
				switch c {
				case 0, 1, 2, 3, 4, 5:
					realm, c = c, 1
				case 6:
					done()
					return -1
				case 7:
					c = 2
				case 8:
					c = 4
				case 9:
					c = 5
				}
			}
			switch c {
			case 1:
				back = true
			case 2:
				if sel > 0 {
					sel--
				}
			case 4:
				if sel+1 < count {
					sel++
				}
			}
			if c == 5 || back {
				break
			}
		}
		if !back {
			break
		}
	}
	// The power level.
	spell := spells[sel]
	sp := &spellTable[spell]
	drawListFrame(list, false)
	deferWindowDraw(list)
	setCursor(list, 2, 2)
	printMessage(spell+4000, list, 18)
	resumeWindowDraw(list)
	refreshWindow(list)
	pvBlinkWin = win
	level := 1
	maxLevel := min(max(int(p.Level)-int(sp.Level)+1, 1), 6)
	for i := range maxLevel {
		setClickRegion(i, win.X+byte(i)+15, win.Y+3, win.X+byte(i)+15, win.Y+3)
	}
	setClickRegion(maxLevel, win.X+13, win.Y+3, win.X+13, win.Y+3)
	setClickRegion(maxLevel+1, 255, 0, 0, 0)
	unmark := func() { // the level's digit back to plain
		setCursor(win, level+14, 3)
		putChar(win, byte(level+23), 5)
	}
	redraw := true
	var cost int
	for {
		cost = int(sp.Cost) * level
		if redraw {
			redraw = false
			pvBlinkState = 1
			if level == 0 {
				pvBlinkX, pvBlinkY = 13, 3
				pvBlinkText = [2]string{string(rune(34)), "c"}
				pvBlinkColor = [2]int{17, 5}
				clearWindowArea(win, 13, 1, 20, 1)
				setCursor(win, 13, 1)
				printMessage(3956, win, 9)
				clearWindowArea(win, 18, 5, 20, 5)
			} else {
				pvBlinkX, pvBlinkY = level+14, 3
				pvBlinkText = [2]string{" ", string(rune(level + 23))}
				pvBlinkColor = [2]int{0, 7}
				setCursor(win, 18, 5)
				if int(p.SP[realm].Cur) < cost {
					printString(win, "***", realmColors[realm])
				} else {
					printInt(win, cost, 3, realmColors[realm])
				}
			}
		}
		c := npcMenuInput()
		if menuClicked == 1 {
			if c+1 == level {
				c = 5
			} else {
				unmark()
				level = 0
				if c < maxLevel {
					level = c + 1
				}
				c = 0
				if level == 0 {
					c = 5
				}
			}
			redraw = true
		} else {
			switch c {
			case 1:
				if level > 0 {
					unmark()
					level--
					redraw = true
				}
			case 3:
				if level == 0 {
					wpopsUnhighlightRealm(win, -1)
					clearWindowArea(win, 13, 1, 20, 1)
					setCursor(win, 13, 1)
					printMessage(realm+3950, win, realmColors[realm])
					level, redraw = 1, true
				} else if level < maxLevel {
					unmark()
					level++
					redraw = true
				}
			}
		}
		if c == 5 && int(p.SP[realm].Cur) >= cost {
			break
		}
	}
	done()
	if level == 0 {
		return -1
	}
	if sp.Target == 1 {
		refreshAllWindows()
		if npcTarget = npcChooseMember(2030, true); npcTarget == -1 {
			return -1
		}
	}
	return level<<8 + spell
}
