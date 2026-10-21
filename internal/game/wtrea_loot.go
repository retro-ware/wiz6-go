package game

import (
	"strings"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// WTREA: the gold, experience and items of a treasure, and handing the items out.

// The items found (at most 6).
var (
	lootCount     int       // loot_count
	lootItem      [6]int    // loot_item: item record
	lootQty       [6]int    // g51e2: quantity
	lootLabel     [6]string // 0x51ee (18 bytes each): "qty name"
	lootIcon      [6]int    // g525a
	lootWeight    [6]int    // g5268: tenths of a pound, quantity included
	lootImportant [6]int    // g5274: the item must not be left behind
)

// TreasureEntry.Kind values.
const (
	treasureGold = 1 // gold between B and A
	treasureItem = 2 // item A plus up to B-1
	treasureExp  = 4 // A times B experience
)

// treasure_after_combat (WTREA 9b80): the fight is won; share its gold, then its treasure.
func treasureAfterCombat() {
	trChestKind = -1
	trRow = 2
	trLine = getMessage(msgYouHaveDefeatedMonsters)
	trKeys = getMessage(msgTreasureKeys)
	trAnimCounter = 0
	flushKeys()
	deferWindowDraw(movePadWin)
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	clearWindow(partyPictureWin, false)
	refreshAllWindows()
	showCenteredLine(partyPictureWin, 0, trLine, true, 3)
	if combatGold > 0 {
		shareCombatGold()
	}
	if treasureID != -1 {
		returnCmd = int16(stateTreasureAfterFight)
		openTreasure()
	}
}

// treasure_found (WTREA 9c29): the treasure of the current cell event.
func treasureFound() {
	trChestKind = -1
	trRow = 1
	trKeys = getMessage(msgTreasureKeys)
	trAnimCounter = 0
	flushKeys()
	deferWindowDraw(movePadWin)
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	clearWindow(partyPictureWin, false)
	if treasureID != -1 {
		returnCmd = int16(stateTreasureFound)
		openTreasure()
	}
	cellEventIndex++
}

// trLiving is the number of members with no condition.
func trLiving() int {
	n := 0
	for m := 0; m < partyCount; m++ {
		if party[m].Condition == conditionOK {
			n++
		}
	}
	return n
}

// share_combat_gold (WTREA 9ab6) gives each living member an equal share of the fight's
// gold as experience.
func shareCombatGold() {
	share := combatGold / int32(trLiving())
	trLine = getMessage(msgEverySurvivorEarnsExp)
	trNum = itoa(int(share))
	trLine = replaceCharWithString(trLine, trNum, '$')
	showCenteredLine(partyPictureWin, 2, trLine, false, 6)
	for m := 0; m < partyCount; m++ {
		if party[m].Condition == conditionOK {
			party[m].Exp += share
		}
	}
}

// open_treasure (WTREA 9601) opens treasure treasureID: the chest, if any, then its gold,
// experience and items.
func openTreasure() {
	b := make([]byte, dbs.Size[TreasureRec]())
	accessScenarioRecord(dbs.RecTreasure, int(treasureID), b, false)
	chest, _ := dbs.Decode[TreasureRec](b)
	treasureID = -1
	if chest.ChestType > 0 {
		trRow = 1
		if handleChest(&chest) == -1 {
			return
		}
		clearWindow(partyPictureWin, false)
	}
	if returnCmd == int16(stateTreasureFound) {
		markChestOpened()
	}
	refreshAllWindows()
	var exp, gold int32
	lootCount = 0
	for _, e := range chest.Entries {
		if random(100) >= int(e.Chance) {
			continue
		}
		switch e.Kind {
		case treasureGold:
			gold += int32(int16(random(int(e.A)-int(e.B)) + int(e.B))) // C int sum wraps
		case treasureExp:
			exp += int32(e.A) * int32(e.B)
		case treasureItem:
			if lootItem[lootCount] = int(int16(random(int(e.B)) + int(e.A))); lootItem[lootCount] > 0 { // C int sum wraps
				lootCount++
			}
		}
	}
	living := trLiving()
	if living == 0 {
		gameState = stateGraveyard
		return
	}
	gold /= int32(living)
	for i := 0; i < lootCount; i++ {
		trLine = getMessage(msgYouFound)
		lootLabel[i] = getMessage(5005)
		var kind string
		if lootItem[i] > 0 {
			rec := itemRec(lootItem[i])
			lootLabel[i] = replaceCharWithString(lootLabel[i], cstr(rec.Name[:]), '^')
			lootQty[i] = 1
			if stackable[rec.Type] == 1 {
				lootQty[i] = int(uint8(rollDice(rec.Charges)))
			}
			lootWeight[i] = lootQty[i] * int(rec.Weight)
			lootIcon[i] = int(rec.Icon)
			lootImportant[i] = 0
			if rec.Flags&itemImportant != 0 {
				lootImportant[i] = 1
			}
			switch rec.Type {
			case itemTypePotion:
				kind = getMessage(msgPotion)
			case itemTypeScroll:
				kind = getMessage(msgScroll)
			case itemTypePowder:
				kind = getMessage(msgPowder)
			}
		}
		trNum2 = itoa(lootQty[i])
		lootLabel[i] = replaceCharWithString(lootLabel[i], trNum2, '$')
		trLine = replaceCharWithString(trLine, lootLabel[i], '%')
		trLine = strings.TrimRight(replaceCharWithString(trLine, kind, '@'), " ")
		showCenteredLine(partyPictureWin, trRow, trLine, true, 1)
	}
	if gold > 0 {
		msg := getMessage(msgEachShareLootWorth)
		trNum = itoa(int(gold))
		trLine = replaceCharWithString(msg, trNum, '$')
		showCenteredLine(partyPictureWin, trRow, trLine, false, 5)
	}
	if exp > 0 {
		msg := getMessage(msgEveryoneReceivesExpBonus)
		trNum = itoa(int(exp))
		trLine = replaceCharWithString(msg, trNum, '$')
		showCenteredLine(partyPictureWin, trRow, trLine, true, 4)
	}
	for m := 0; m < partyCount; m++ {
		if party[m].Condition == conditionOK {
			party[m].Exp += exp
			party[m].Gold += gold
		}
	}
	if lootCount > 0 {
		distributeLoot()
	}
	trChestOpen = 0
	animateTreasureView()
	trWaitAnimating(2)
}

// mark_chest_opened (WTREA 8f81) shows the cell event's text and updates its level flags.
func markChestOpened() {
	ev := cellEventIndex
	bits := int(uint16(mazeB.DirBits[ev]))
	if mazeB.Unk288[ev] > 0 {
		trShowMessageText(int(mazeB.Unk288[ev]), 0)
	}
	bit := extractBits(bits, 0, 8)
	setBit(levelFlags[gameStateArg][:], bit, false)
	if extractBits(bits, 11, 1) == 1 {
		setBit(scenarioHdr.LevelFlags[gameStateArg][:], bit, false)
	}
	if n := extractBits(bits, 9, 2); n > 0 {
		setBit(levelFlags[gameStateArg][:], int(mazeB.FlagBit[ev]), true)
		if n > 1 {
			setBit(scenarioHdr.LevelFlags[gameStateArg][:], int(mazeB.FlagBit[ev]), true)
		}
	}
}

// give_item_to_member (WTREA 8583) puts found item loot into member's pack (or swag),
// stacking it where it can; -1 if there is no room.
func giveItemToMember(member, loot int) int {
	p := &party[member]
	id := lootItem[loot]
	rec := itemRec(id)
	var qty uint8
	if stackable[rec.Type] == 1 {
		qty = uint8(lootQty[loot])
	} else {
		qty = uint8(rollDice(rec.Charges))
	}
	stacks := func(i int) bool {
		it := p.Items[i]
		return int(it.ID) == id && stackable[it.Type] == 1 && int(it.Qty)+int(qty) <= maxStackQty
	}
	place := 1 // 1 a new pack slot, 2 a new swag slot, 0 a stack
	slot := int(p.ItemCount)
	for i := slot - 1; i >= 0; i-- {
		if stacks(i) {
			slot, place = i, 0
		}
	}
	if slot >= packSlots {
		place = 2
		slot = int(p.SwagCount) + packSlots
		for i := int(p.SwagCount) - 1; i >= 0; i-- {
			if stacks(i + packSlots) {
				slot, place = i+packSlots, 0
			}
		}
		if slot >= itemSlots {
			return -1
		}
	}
	it := p.Items[slot]
	if place > 0 {
		it.Qty = 0
		if place == 1 {
			p.ItemCount++
		} else {
			p.SwagCount++
		}
	}
	it.ID = int16(id)
	it.Type = rec.Type
	it.Icon = rec.Icon
	it.Weight = rec.Weight
	it.Qty += qty
	it.Flags = 0
	for _, f := range [][2]uint8{{recItemOffHand, itemOffHand}, {recItemTwoHanded, itemTwoHanded}, {16, 16}, {32, 32}, {itemImportant, itemKept}} {
		if rec.Flags&f[0] != 0 {
			it.Flags |= f[1]
		}
	}
	p.Items[slot] = it
	calcCarriedWeight(member)
	return 0
}

// draw_loot_line (WTREA 8848) draws found item i plain in the list.
func drawLootLine(w *Window, i int) {
	setCursor(w, 1, i)
	printStringHi(w, lootLabel[i], 3)
	setCursor(w, 18, i)
	putCharHi(w, byte(lootIcon[i]+1), 4)
}

// remove_loot_entry (WTREA 88d2) takes found item i off the list.
func removeLootEntry(i int) {
	lootCount--
	for ; i < lootCount; i++ {
		lootItem[i] = lootItem[i+1]
		lootLabel[i] = lootLabel[i+1]
		lootQty[i] = lootQty[i+1]
		lootWeight[i] = lootWeight[i+1]
		lootIcon[i] = lootIcon[i+1]
		lootImportant[i] = lootImportant[i+1]
	}
}

// distribute_loot (WTREA 8952) lets the player hand the found items out; important items
// cannot be left.
func distributeLoot() {
	if dtLog != nil { // interactive: the difftest stubs it
		return
	}
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
	eraseCursorMark(bottomLineWin, false)
	win := createWindow(0, 19, 19, 5, 20, borderRightEdge, 0, false)
	fillWindow(win, ' ', 3)
	list := createWindow(20, 19, 20, 5, 21, borderTopEdge, 0, false)
	fillWindow(list, ' ', 3)
	deferWindowDraw(win)
	setCursor(win, 2, 1)
	printMessage(msgTakeItem, win, 18)
	win.CurX++
	noneX, noneY := int(win.CurX), int(win.CurY)
	none := getMessage(msgLootNone)
	printStringHi(win, none, 3)
	g := func(x, y int, ch byte) { putGlyphAtC1(win, x, y, ch) }
	g(1, 2, 17)
	g(18, 2, 19)
	g(1, 4, 6)
	g(18, 4, 8)
	for x := 2; x < 18; x++ {
		g(x, 2, 18)
		g(x, 4, 7)
	}
	g(1, 3, 4)
	g(18, 3, 5)
	clearWindowArea(win, 2, 3, 17, 3)
	resumeWindowDraw(win)
	sel := 0
	for {
		for sel >= lootCount {
			sel--
		}
		printChestPrompt(msgDivvyLoot, 2)
		deferWindowDraw(list)
		fillWindow(list, ' ', 3)
		for i := 0; i < lootCount; i++ {
			drawLootLine(list, i)
			setClickRegion(i, list.X+1, list.Y+byte(i), list.X+19, list.Y+byte(i))
		}
		resumeWindowDraw(list)
		setClickRegion(lootCount, win.X+byte(noneX), win.Y+byte(noneY), byte(len(none)+int(win.X)+noneX-1), win.Y+byte(noneY))
		setClickRegion(lootCount+1, clickRegionsEnd, 0, 0, 0)
		refreshAllWindows()
		hiNone := func() {
			setCursor(win, noneX, noneY)
			printStringHi(win, none, 3)
			sel = 0
		}
		for {
			if sel == -1 {
				if !mousePresent {
					setCursor(win, noneX, noneY)
					printString(win, none, -3)
				}
				clearWindowArea(win, 2, 3, 17, 3)
			} else {
				// Column 21 of the 20-column list: set_cursor wraps it to column 1.
				setCursor(list, 21, sel)
				printString(list, lootLabel[sel], -3)
				setCursor(win, 2, 3)
				printMessage(msgWeight, win, 11)
				setCursor(win, 12, 3)
				printInt(win, lootWeight[sel]/10, 4, 3)
				putChar(win, '.', 3)
				printInt(win, lootWeight[sel]%10, 1, 3)
			}
			k := trInput()
			if menuClicked == 1 {
				if k == sel {
					k = menuKeyEnter
				} else {
					if sel != -1 {
						drawLootLine(list, sel)
					}
					if k < lootCount {
						sel, k = k, menuKeyEscape
					} else {
						sel, k = -1, menuKeyEnter
					}
				}
			} else {
				switch k {
				case menuKeyLeft:
					if sel != -1 {
						drawLootLine(list, sel)
						sel = -1
					}
				case menuKeyUp:
					if sel == -1 {
						hiNone()
					} else {
						drawLootLine(list, sel)
						if sel--; sel < 0 {
							sel = -1
						}
					}
				case menuKeyRight:
					if sel == -1 {
						hiNone()
					}
				case menuKeyDown:
					if sel == -1 {
						hiNone()
					} else {
						drawLootLine(list, sel)
						if sel++; sel >= lootCount {
							sel = -1
						}
					}
				}
			}
			if k == menuKeyEnter {
				break
			}
		}
		if sel != -1 {
			if m := trChooseMember(msgWhoTakesIt, 0); m != -1 && lootItem[sel] > 0 {
				if giveItemToMember(m, sel) == -1 {
					printChestPrompt(msgLootFull, 6)
					refreshWindow(mainWindow)
					playSound(soundBeep)
					trWaitAnimating(messageDelay)
				} else {
					removeLootEntry(sel)
				}
			}
		}
		important := false
		if sel == -1 {
			for i := 0; i < lootCount; i++ {
				if lootImportant[i] == 1 {
					important = true
				}
			}
			if important {
				printChestPrompt(msgImportantItem, 1)
				refreshWindow(mainWindow)
				playSound(soundBeep)
				trWaitAnimating(messageDelay)
			}
		}
		if lootCount <= 0 || sel == -1 && !important {
			break
		}
	}
	destroyWindow(list, false)
	destroyWindow(win, false)
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
}
