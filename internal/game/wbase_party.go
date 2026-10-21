package game

import (
	"math"
	"os"
	"strconv"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
)

// WBASE: the roster file, loading a game, the party and the configuration menu.

// pcfileRecordIO (WBASE 459a pcfile_record_io, WPCMK 457f access_roster_record) reads
// (write false) or writes roster record index of PCFILE.DBS.
func pcfileRecordIO(pc *PC, index int, write bool) {
	stopSound()
	f, err := os.OpenFile(dbs.Path(dataDir, "PCFILE.DBS"), os.O_RDWR, 0)
	if err != nil {
		return // the C ignores a failed open too
	}
	defer f.Close()
	off := int64(index)*int64(pcfileHdr.RecSize) + int64(pcfileHdr.Offset)
	if write {
		f.WriteAt(dbs.Encode(pc), off)
		return
	}
	b := dbs.Encode(pc) // a short read leaves the rest of the record as it was
	f.ReadAt(b, off)
	*pc, _ = dbs.Decode[PC](b)
}

// writePcfileHeader writes the PCFILE.DBS header with the roster's slot flags (WPCMK
// create_character, delete_character).
func writePcfileHeader() {
	stopSound()
	f, err := os.OpenFile(dbs.Path(dataDir, "PCFILE.DBS"), os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer f.Close()
	pcfileHdr.Used = rosterSlotUsed
	f.WriteAt(dbs.Encode(&pcfileHdr), 0)
}

// loadGame (WBASE 570a) loads NEWGAME.DBS (resume 0) or SAVEGAME.DBS: the scenario header
// (keeping the configuration in use), the maps, the position and, resuming, the party;
// -1 on a disk error. The port reads both from the data folder: the floppy and drive
// prompts of the C have nothing to ask for.
func loadGame(resume int) int {
	stopSound()
	entering := getMessage(1212)
	title := getMessage(1213)
	diskError := getMessage(2258)
	loading := getMessage(2259)
	name := "NEWGAME.DBS"
	if resume != 0 {
		name = "SAVEGAME.DBS"
	}
	b, err := os.ReadFile(dbs.Path(dataDir, name))
	if err != nil {
		showErrorText(diskError, true)
		return -1
	}
	eraseWindow(mainWindow, false)
	refreshAllWindows()
	if resume == 0 {
		animateAndShowTwoLines(entering, title)
	} else {
		showStatusText(loading)
	}
	next := func(n int) []byte {
		if len(b) < n {
			return nil
		}
		r := b[:n]
		b = b[n:]
		return r
	}
	fail := func() int {
		showErrorText(diskError, false)
		return -1
	}
	saved := scenarioHdr
	h := next(dbs.Size[dbs.ScenarioHeader]())
	if h == nil {
		return fail()
	}
	scenarioHdr, _ = dbs.Decode[dbs.ScenarioHeader](h)
	scenarioHdr.Paths = saved.Paths
	scenarioHdr.SoundMode, scenarioHdr.Delay, scenarioHdr.Disks = saved.SoundMode, saved.Delay, saved.Disks
	scenarioHdr.Level, scenarioHdr.Input, scenarioHdr.Gmode = saved.Level, saved.Input, saved.Gmode
	delaySetting = int16(scenarioHdr.Delay)
	m := dbs.Size[MapA]()
	n := dbs.Size[MapB]()
	for i := 0; i < int(scenarioDBS.Master.RecordCounts[dbs.RecMapA]); i++ { // record_counts[2]
		a, c := next(m), next(n)
		if a == nil || c == nil {
			return fail()
		}
		if i < 14 {
			mapCacheA[i], _ = dbs.Decode[MapA](a)
			mapCacheB[i], _ = dbs.Decode[MapB](c)
		}
	}
	s := next(dbs.Size[dbs.SaveState]())
	if s == nil {
		return fail()
	}
	saveState, _ = dbs.Decode[dbs.SaveState](s)
	st := &saveState
	partyX, partyY, cellX, cellY = st.PartyX, st.PartyY, st.CellX, st.CellY
	partyBlock, partyFacing, gameStateArg = st.Block, st.Facing, int(st.Map)
	partyCounters = st.Counters
	if resume == 1 {
		partyCount = 0
		for range int(st.PartyCount) {
			r := next(dbs.Size[PC]())
			if r == nil {
				return fail()
			}
			memberPortrait[partyCount] = uint8(findFreePortraitSlot(partyCount))
			party[partyCount], _ = dbs.Decode[PC](r)
			partyCount++
		}
	}
	for i := 0; i < partyCount; i++ {
		loadPortrait(int(party[i].Portrait), int(memberPortrait[i]))
		drawMemberPanel(i)
	}
	return 0
}

// The roster list choosePcfileChar shows (pclist_*).
type pclistEntry struct {
	name             string
	sex, race, class int
	slot             int // pclist_pcindex: the PCFILE slot
}

var pclist []pclistEntry

// drawPcfileList (WBASE 6475) draws five rows of the roster list around top, the middle
// one highlighted when highlight is 1.
func drawPcfileList(win *Window, top, highlight int) {
	drawListFrame(win, false)
	deferWindowDraw(win)
	for row := -2; row <= 2; row++ {
		if top+row < 0 || top+row >= len(pclist) {
			continue
		}
		e := &pclist[top+row]
		setCursor(win, 2, row+2)
		if highlight == 1 && row == 0 {
			printString(win, e.name, 5)
			for win.CurX < pclistSexColumn {
				putChar(win, ' ', 1)
			}
			putChar(win, getMessage(e.sex + 140)[0], 7)
			putChar(win, '-', 9)
			printString(win, abbrev3(getMessage(e.race+100)), 6)
			putChar(win, 32, 1)
			printString(win, abbrev3(getMessage(e.class+120)), 3)
			continue
		}
		printStringHi(win, e.name, 3)
		for win.CurX < pclistSexColumn {
			putCharHi(win, ' ', 3)
		}
		putCharHi(win, getMessage(e.sex + 140)[0], 3)
		putCharHi(win, '-', 3)
		printStringHi(win, abbrev3(getMessage(e.race+100)), 3)
		win.CurX++
		printStringHi(win, abbrev3(getMessage(e.class+120)), 3)
	}
	resumeWindowDraw(win)
	refreshWindow(win)
}

// abbrev3 is the first three letters of s: draw_pcfile_list reads messages into a 3-byte
// buffer and puts a NUL after them.
func abbrev3(s string) string { return s[:min(len(s), 3)] }

// choosePcfileChar (WBASE 66a7) lets the player pick an available roster character; its
// place in pclist, or -1.
func choosePcfileChar() int {
	pclist = pclist[:0]
	for i := 0; i < int(pcfileHdr.Count); i++ {
		if rosterSlotUsed[i] == rosterSlotAvailable {
			var pc PC
			pcfileRecordIO(&pc, i, false)
			pclist = append(pclist, pclistEntry{cstr(pc.Name[:]), int(pc.Sex), int(pc.Race), int(pc.Class), i})
		}
	}
	n := len(pclist)
	if n == 0 {
		return -1
	}
	titleWin := createWindow(0, 19, 19, 5, 10, borderRightEdge, 0, false)
	fillWindow(titleWin, ' ', 3)
	listWin := createWindow(20, 19, 20, 5, 20, borderTopEdge, 0, false)
	fillWindow(listWin, ' ', 3)
	title := getMessage(1201)
	showTitleText(title, 10-(len(title)+1)/2)
	eraseCursorMark(bottomLineWin, false)
	deferWindowDraw(titleWin)
	setCursor(titleWin, 6, 1)
	printMessage(1206, titleWin, 18)
	resumeWindowDraw(titleWin)
	drawListFrame(listWin, false)
	refreshAllWindows()
	cancel := getMessage(1207)
	top, onCancel := 0, false
	lx, ly := listWin.X, listWin.Y
	setClickRegion(0, lx+2, ly+2, lx+18, ly+2)
	setClickRegion(1, lx+1, ly, lx+1, ly)
	setClickRegion(2, lx+1, ly+4, lx+1, ly+4)
	setClickRegion(3, titleWin.X+7, titleWin.Y+3, byte(len(cancel)+int(titleWin.X)+6), titleWin.Y+3)
	setClickRegion(4, clickRegionsEnd, 0, 0, 0)
	for {
		setCursor(titleWin, 7, 3)
		if !mousePresent && onCancel {
			printString(titleWin, cancel, -5)
			drawPcfileList(listWin, top, 0)
		} else {
			printStringHi(titleWin, cancel, 3)
			drawPcfileList(listWin, top, 1)
		}
		i := getMenuChoiceAnimated(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			switch i {
			case 0: // the highlighted row
				i = menuKeyEnter
			case 1: // the scroll bar's top
				i = menuKeyUp
			case 2: // its bottom
				i = menuKeyDown
			case 3: // Cancel
				onCancel = true
				i = menuKeyEnter
			}
		}
		switch i {
		case menuKeyLeft:
			onCancel = true
		case menuKeyUp:
			if onCancel {
				onCancel = false
			} else if top > 0 {
				top--
			}
		case menuKeyRight:
			onCancel = false
		case menuKeyDown:
			if onCancel {
				onCancel = false
			} else if top+1 < n {
				top++
			}
		}
		if i == menuKeyEnter {
			break
		}
	}
	destroyWindow(titleWin, false)
	destroyWindow(listWin, false)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	if onCancel {
		return -1
	}
	return top
}

// addPartyMember (WBASE 6a9e) adds a roster character the player picks to the party.
func addPartyMember() {
	pick := choosePcfileChar()
	if pick == -1 {
		return
	}
	slot := pclist[pick].slot
	var pc PC
	pcfileRecordIO(&pc, slot, false)
	rosterSlotUsed[slot] = rosterSlotInParty
	memberPcfileIndex[partyCount] = slot
	memberPortrait[partyCount] = uint8(findFreePortraitSlot(partyCount))
	loadPortrait(int(pc.Portrait), int(memberPortrait[partyCount]))
	party[partyCount] = pc
	drawMemberPanel(partyCount)
	partyCount++
}

// removePartyMember (WBASE 6b30) takes member i out of the party and redraws the panels.
func removePartyMember(i int) {
	rosterSlotUsed[memberPcfileIndex[i]] = rosterSlotAvailable
	partyCount--
	for j := i; j < partyCount; j++ {
		party[j] = party[j+1]
		memberPcfileIndex[j] = memberPcfileIndex[j+1]
		memberPortrait[j] = memberPortrait[j+1]
	}
	fillWindow(leftPanelWin, ' ', 3)
	fillWindow(rightPanelWin, ' ', 3)
	for j := 0; j < partyCount; j++ {
		drawMemberPanel(j)
	}
}

// choosePartyMemberWbase (WBASE 6c2b choose_party_member) lets the player pick a member
// under title msg; the member, or -1. With one member it is that one.
func choosePartyMemberWbase(msg int) int {
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, borderRightEdge, 0, false)
	fillWindow(win, ' ', 3)
	cancelText := getMessage(2027)
	cancelHilite := getMessage(2028)
	title := getMessage(msg)
	showTitleText(title, 10-(len(title)+6)/2)
	cx, cy := int(mainWindow.CurX), int(mainWindow.CurY)
	eraseCursorMark(bottomLineWin, false)
	refreshAllWindows()
	setCursor(mainWindow, cx, cy)
	if mousePresent {
		printStringHi(mainWindow, cancelText, 3)
	} else {
		printString(mainWindow, cancelHilite, -5)
	}
	for i := 0; i < partyCount; i++ {
		x, y := i%2*9+2, i/2+1
		setCursor(win, x, y)
		name := cstr(party[i].Name[:])
		printStringHi(win, name, 3)
		setClickRegion(i, win.X+byte(x), win.Y+byte(y), byte(len(name)+int(win.X)+x-1), win.Y+byte(y))
	}
	mx, my := mainWindow.X+byte(cx), mainWindow.Y+byte(cy)
	setClickRegion(partyCount, mx, my, mx+byte(len(cancelHilite))-1, my)
	setClickRegion(partyCount+1, clickRegionsEnd, 0, 0, 0)
	sel := -1
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, cx, cy)
				printString(mainWindow, cancelHilite, -5)
			} else {
				setCursor(win, sel%2*9+2, sel/2+1)
				printString(win, cstr(party[sel].Name[:]), -5)
			}
		}
		choice := getMenuChoiceAnimated(menuKeys, clickRegions[:])
		if menuClicked == 1 {
			sel = -1
			if choice < partyCount {
				sel = choice
			}
			choice = menuKeyEnter
		}
		switch choice {
		case menuKeyLeft:
			if sel != -1 && sel%2 > 0 {
				drawMemberName(win, sel)
				sel--
			}
		case menuKeyUp:
			if sel != -1 {
				drawMemberName(win, sel)
				if sel-2 >= 0 {
					sel -= 2
				} else {
					sel = -1
				}
			}
		case menuKeyRight:
			if sel != -1 && sel%2 < 1 && sel+1 < partyCount {
				drawMemberName(win, sel)
				sel++
			}
		case menuKeyDown:
			if sel == -1 {
				setCursor(mainWindow, cx, cy)
				printStringHi(mainWindow, cancelText, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if choice == menuKeyEnter {
			break
		}
	}
	destroyWindow(win, false)
	eraseCursorMark(bottomLineWin, false)
	return sel
}

// removeAllMembers (WBASE 6fd4) empties the party.
func removeAllMembers() {
	for partyCount > 0 {
		removePartyMember(partyCount - 1)
	}
}

// pathString is scenarioHdr.Paths[i] as a string; setPath stores s there.
func pathString(i int) string { return cstr(scenarioHdr.Paths[i][:]) }

func setPath(i int, s string) {
	scenarioHdr.Paths[i] = [20]uint8{}
	copy(scenarioHdr.Paths[i][:19], s)
}

// drawSettingValue (WBASE 7585) draws the value of setting item at row in color.
func drawSettingValue(win *Window, item, row, color int) {
	x := 19
	if item < 5 {
		x = 7
	}
	width := 20
	switch {
	case item == 1:
		width = 3
	case item < 5:
		width = 10
	}
	clearWindowArea(win, x, row, width+x-1, row)
	var text string
	switch item {
	case 0:
		text = getMessage(int(scenarioHdr.SoundMode) + 2350)
	case 1:
		text = strconv.Itoa(int(scenarioHdr.Delay))
	case 2:
		text = getMessage(int(scenarioHdr.Level) + 2357)
	case 3:
		text = getMessage(int(scenarioHdr.Input) + 2360)
	case 4:
		text = getMessage(int(scenarioHdr.Gmode) + 2363)
	case 5:
		text = pathString(3)
	case 6:
		text = pathString(0)
	case 7:
		text = pathString(1)
	}
	if text == "" {
		text = " "
	}
	setCursor(win, x, row)
	printString(win, text, color)
}

// settingRow is the row setting sel is drawn on, and settingColor its normal colour.
func settingRow(sel int) int {
	switch {
	case sel < 5:
		return sel
	case sel == 5:
		return 1
	}
	return sel - 3
}

func settingColor(sel int) int {
	switch {
	case sel < 2:
		return 5
	case sel < 5:
		return 6
	case sel == 5:
		return 7
	}
	return 4
}

// doGameConfiguration (WBASE 76b5) is the configuration menu: sound, delay, level, input,
// graphics mode and the drives. Changes are saved to SCENARIO.HDR.
func doGameConfiguration() {
	menuKeys = getMessage(1000)
	changed := false
	win := createWindow(0, 19, 40, 5, 22, borderNone, 0, false)
	fillWindow(win, ' ', 3)
	eraseWindow(mainWindow, false)
	eraseWindow(bottomLineWin, false)
	doneText := getMessage(2308)
	doneHilite := getMessage(2309)
	showMessageLine(2307)
	dx, dy := int(mainWindow.CurX), int(mainWindow.CurY)
	deferWindowDraw(win)
	wx, wy := int(win.X), int(win.Y)
	region := func(i, x1, y1, x2, y2 int) { setClickRegion(i, byte(x1), byte(y1), byte(x2), byte(y2)) }
	for i := range 5 {
		setCursor(win, 1, i)
		printMessage(i+2300, win, 18)
		drawSettingValue(win, i, i, settingColor(i))
		x, width := 7, 10
		if i == 1 {
			x, width = 9, 1
		}
		region(i, wx+x, wy+i, wx+x+width-1, wy+i)
	}
	setCursor(win, 22, 0)
	printMessage(2305, win, 18)
	clearWindowArea(win, 19, 1, 38, 1)
	drawSettingValue(win, 5, 1, 7)
	region(5, wx+19, wy+1, wx+38, wy+1)
	setCursor(win, 22, 2)
	printMessage(2306, win, 18)
	clearWindowArea(win, 19, 3, 38, 3)
	drawSettingValue(win, 6, 3, 4)
	region(6, wx+19, wy+3, wx+38, wy+3)
	clearWindowArea(win, 19, 4, 38, 4)
	drawSettingValue(win, 7, 4, 4)
	region(7, wx+19, wy+4, wx+38, wy+4)
	region(8, wx+7, wy+1, wx+8, wy+1)
	mx, my := int(mainWindow.X)+dx, int(mainWindow.Y)+dy
	region(9, mx, my, mx+len(doneHilite)-1, my)
	region(10, clickRegionsEnd, 0, 0, 0)
	resumeWindowDraw(win)
	refreshAllWindows()
	setCursor(mainWindow, dx, dy)
	if !mousePresent {
		printString(mainWindow, doneHilite, -5)
	} else {
		printStringHi(mainWindow, doneText, 3)
	}
	// drawSel draws setting (or Done, -1) sel highlighted or in its normal colour.
	drawSel := func(sel int, hilite bool) {
		switch {
		case sel == -1 && hilite:
			setCursor(mainWindow, dx, dy)
			printString(mainWindow, doneHilite, -5)
		case sel == -1:
			setCursor(mainWindow, dx, dy)
			printStringHi(mainWindow, doneText, 3)
		case hilite:
			drawSettingValue(win, sel, settingRow(sel), -5)
		default:
			drawSettingValue(win, sel, settingRow(sel), settingColor(sel))
		}
	}
	sel := -1
	for {
		drawSel(sel, !mousePresent)
		prev := sel
		var choice int
		if gameStateID(returnCmd) == stateEnterMap {
			choice = menuInput()
		} else {
			choice = getMenuChoiceAnimated(menuKeys, clickRegions[:])
		}
		if menuClicked == 1 {
			if choice == 8 {
				sel, choice = 1, menuKeyLeft
			} else {
				sel = -1
				if choice < 8 {
					sel = choice
				}
				choice = menuKeyRight
				if sel == -1 {
					choice = menuKeyEnter
				}
			}
		}
		switch choice {
		case menuKeyUp:
			if sel == -1 {
				sel = 7
			} else {
				sel--
			}
		case menuKeyDown:
			switch {
			case sel == -1:
				sel = 0
			case sel < 7:
				sel++
			default:
				sel = -1
			}
		case menuKeyLeft, menuKeyRight:
			changed = true
			d := 1
			if choice == menuKeyLeft {
				d = -1
			}
			switch sel {
			case 0:
				scenarioHdr.SoundMode = uint8((int(scenarioHdr.SoundMode) + d + 6) % 6)
			case 1:
				if v := int(scenarioHdr.Delay) + d; v >= 0 && v <= math.MaxUint8 {
					scenarioHdr.Delay = uint8(v)
				} else {
					playSound(0)
				}
				delaySetting = int16(scenarioHdr.Delay)
			case 2:
				scenarioHdr.Level = uint8((int(scenarioHdr.Level) + d + 3) % 3)
			case 3:
				scenarioHdr.Input = uint8((int(scenarioHdr.Input) + d + 2) % 2)
			case 4, 6, 7:
				playSound(0)
			case 5:
				clearWindowArea(win, 19, 1, 38, 1)
				setCursor(win, 19, 1)
				setPath(3, inputText(win, 19, 7, inputPrintable))
			}
		}
		if !mousePresent && prev != sel {
			drawSel(prev, false)
		}
		if choice == menuKeyEnter && sel == -1 {
			break
		}
	}
	stopSound()
	if changed {
		diskError := getMessage(2258)
		f, err := os.OpenFile(dbs.Path(dataDir, "SCENARIO.HDR"), os.O_RDWR, 0)
		if err != nil {
			exitGame(19)
		}
		if _, err := f.WriteAt(dbs.Encode(&scenarioHdr), 0); err != nil {
			showErrorText(diskError, false)
		}
		f.Close()
	}
	delaySetting = int16(scenarioHdr.Delay)
	if videoFlags&videoMouse != 0 && scenarioHdr.Input == 1 {
		if !mousePresent {
			drv.MouseInit()
		}
		mousePresent = true
	} else {
		if mousePresent {
			drv.MouseShutdown()
		}
		mousePresent = false
	}
	clearClickFlag()
	destroyWindow(win, false)
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
}
