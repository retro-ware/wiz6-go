package game

import (
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
	saved := hdr
	h := next(dbs.Size[dbs.ScenarioHeader]())
	if h == nil {
		return fail()
	}
	hdr, _ = dbs.Decode[dbs.ScenarioHeader](h)
	hdr.Paths = saved.Paths
	hdr.SoundMode, hdr.Delay, hdr.Disks = saved.SoundMode, saved.Delay, saved.Disks
	hdr.Level, hdr.Input, hdr.Gmode = saved.Level, saved.Input, saved.Gmode
	g4fc8 = int16(hdr.Delay)
	m := dbs.Size[MapA]()
	n := dbs.Size[MapB]()
	for i := 0; i < int(scen.Master.RecordCounts[dbs.RecMapA]); i++ { // record_counts[2]
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
			for win.CurX < 10 {
				putChar(win, 32, 1)
			}
			putChar(win, getMessage(e.sex + 140)[0], 7)
			putChar(win, 45, 9)
			printString(win, abbrev3(getMessage(e.race+100)), 6)
			putChar(win, 32, 1)
			printString(win, abbrev3(getMessage(e.class+120)), 3)
			continue
		}
		printStringHi(win, e.name, 3)
		for win.CurX < 10 {
			putCharHi(win, 32, 3)
		}
		putCharHi(win, getMessage(e.sex + 140)[0], 3)
		putCharHi(win, 45, 3)
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
		if rosterSlotUsed[i] == 1 {
			var pc PC
			pcfileRecordIO(&pc, i, false)
			pclist = append(pclist, pclistEntry{cstr(pc.Name[:]), int(pc.Sex), int(pc.Race), int(pc.Class), i})
		}
	}
	n := len(pclist)
	if n == 0 {
		return -1
	}
	titleWin := createWindow(0, 19, 19, 5, 10, -4, 0, false)
	fillWindow(titleWin, 32, 3)
	listWin := createWindow(20, 19, 20, 5, 20, -5, 0, false)
	fillWindow(listWin, 32, 3)
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
	setClickRegion(4, 255, 0, 0, 0)
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
			case 0:
				i = 5
			case 1:
				i = 2
			case 2:
				i = 4
			case 3:
				onCancel = true
				i = 5
			}
		}
		switch i {
		case 1:
			onCancel = true
		case 2:
			if onCancel {
				onCancel = false
			} else if top > 0 {
				top--
			}
		case 3:
			onCancel = false
		case 4:
			if onCancel {
				onCancel = false
			} else if top+1 < n {
				top++
			}
		}
		if i == 5 {
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
	rosterSlotUsed[slot] = 2
	memberPcfileIndex[partyCount] = slot
	memberPortrait[partyCount] = uint8(findFreePortraitSlot(partyCount))
	loadPortrait(int(pc.Portrait), int(memberPortrait[partyCount]))
	party[partyCount] = pc
	drawMemberPanel(partyCount)
	partyCount++
}

// removePartyMember (WBASE 6b30) takes member i out of the party and redraws the panels.
func removePartyMember(i int) {
	rosterSlotUsed[memberPcfileIndex[i]] = 1
	partyCount--
	for j := i; j < partyCount; j++ {
		party[j] = party[j+1]
		memberPcfileIndex[j] = memberPcfileIndex[j+1]
		memberPortrait[j] = memberPortrait[j+1]
	}
	fillWindow(g4fba, 32, 3)
	fillWindow(g4fb8, 32, 3)
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
	win := createWindow(0, 19, 19, 5, 25, -4, 0, false)
	fillWindow(win, 32, 3)
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
	setClickRegion(partyCount+1, 255, 0, 0, 0)
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
			choice = 5
		}
		switch choice {
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
				printStringHi(mainWindow, cancelText, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if choice == 5 {
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

// pathString is hdr.Paths[i] as a string; setPath stores s there.
func pathString(i int) string { return cstr(hdr.Paths[i][:]) }

func setPath(i int, s string) {
	hdr.Paths[i] = [20]uint8{}
	copy(hdr.Paths[i][:19], s)
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
		text = getMessage(int(hdr.SoundMode) + 2350)
	case 1:
		text = strconv.Itoa(int(hdr.Delay))
	case 2:
		text = getMessage(int(hdr.Level) + 2357)
	case 3:
		text = getMessage(int(hdr.Input) + 2360)
	case 4:
		text = getMessage(int(hdr.Gmode) + 2363)
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
	win := createWindow(0, 19, 40, 5, 22, -1, 0, false)
	fillWindow(win, 32, 3)
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
	region(10, 255, 0, 0, 0)
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
		if returnCmd == 25 {
			choice = menuInput()
		} else {
			choice = getMenuChoiceAnimated(menuKeys, clickRegions[:])
		}
		if menuClicked == 1 {
			if choice == 8 {
				sel, choice = 1, 1
			} else {
				sel = -1
				if choice < 8 {
					sel = choice
				}
				choice = 3
				if sel == -1 {
					choice = 5
				}
			}
		}
		switch choice {
		case 2:
			if sel == -1 {
				sel = 7
			} else {
				sel--
			}
		case 4:
			switch {
			case sel == -1:
				sel = 0
			case sel < 7:
				sel++
			default:
				sel = -1
			}
		case 1, 3:
			changed = true
			d := 1
			if choice == 1 {
				d = -1
			}
			switch sel {
			case 0:
				hdr.SoundMode = uint8((int(hdr.SoundMode) + d + 6) % 6)
			case 1:
				if v := int(hdr.Delay) + d; v >= 0 && v <= 255 {
					hdr.Delay = uint8(v)
				} else {
					playSound(0)
				}
				g4fc8 = int16(hdr.Delay)
			case 2:
				hdr.Level = uint8((int(hdr.Level) + d + 3) % 3)
			case 3:
				hdr.Input = uint8((int(hdr.Input) + d + 2) % 2)
			case 4, 6, 7:
				playSound(0)
			case 5:
				clearWindowArea(win, 19, 1, 38, 1)
				setCursor(win, 19, 1)
				setPath(3, inputText(win, 19, 7, 0))
			}
		}
		if !mousePresent && prev != sel {
			drawSel(prev, false)
		}
		if choice == 5 && sel == -1 {
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
		if _, err := f.WriteAt(dbs.Encode(&hdr), 0); err != nil {
			showErrorText(diskError, false)
		}
		f.Close()
	}
	g4fc8 = int16(hdr.Delay)
	if videoFlags&0x1000 != 0 && hdr.Input == 1 {
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
