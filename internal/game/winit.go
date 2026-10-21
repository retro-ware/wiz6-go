package game

import (
	"fmt"
	"time"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// WINIT: start-up, the title sequence, the graveyard and the main screen's windows.

const (
	fontCount    = 5   // WFONT0..4: font 0 and the colour fonts
	gmodeInvalid = 255 // a scenario_hdr Gmode the game refuses to start with (exit code 2)
	inputMouse   = 1   // scenario_hdr's Input: use the mouse
)

// winitMain (WINIT ovmain) does what gameState asks: load the data, the title, the
// graveyard or the main screen.
func winitMain() {
	switch gameState {
	case stateLoadData:
		initGameData()
	case stateTitle:
		showTitleSequence()
	case stateGraveyard:
		showGraveyard()
	case stateMainScreen:
		initMainScreen()
	}
}

// loadFontFile (WINIT 45a3) loads font n from file name.
func loadFontFile(name string, n int) {
	f, err := dbs.LoadFont(dbs.Path(dataDir, name))
	if err != nil {
		exitGame(3)
	}
	if n == 0 {
		var g [128][8]uint8
		for c := range g {
			for y := range 8 {
				for x := range 8 {
					g[c][y] |= f.Glyphs[c][y][x] << (7 - x)
				}
			}
		}
		drv.SetMonoFont(g)
		return
	}
	drv.SetFont(n, f.Glyphs)
}

// loadMazeData (WINIT 45e5) loads the maze wall tiles.
func loadMazeData(name string) {
	m, err := dbs.LoadMazeData(dbs.Path(dataDir, name))
	if err != nil {
		exitGame(4)
	}
	drv.LoadMaze(m)
}

// loadMonsterPic (WINIT 4721) loads MONnn.PIC for monster mon into picture slot slot and
// marks the slots it runs over; a missing file is fatal error code.
func loadMonsterPic(mon, slot, code int) {
	stopSound()
	rec := make([]byte, dbs.Size[MonsterPicRec]())
	accessScenarioRecord(dbs.RecMonsterPic, mon, rec, false)
	monsterPics[slot], _ = dbs.Decode[MonsterPicRec](rec)
	drv.LoadPic(slot, openData(fmt.Sprintf("MON%02d.PIC", mon), code))
	for i := 0; i <= int(monsterPics[slot].FrameCount) && slot+i < len(picSlotMonster); i++ {
		picSlotMonster[slot+i] = mon
		picSlotLoaded[slot+i] = true
	}
}

// loadSounds (WINIT 47f5) loads the records of sounds 0..count-1 and the samples of those
// that have them.
func loadSounds(count int) {
	stopSound()
	for i := range count {
		sounds[i] = readSoundRec(i)
		if sounds[i].Size > 0 {
			loadSoundSlot(i, i, 10)
		}
	}
}

// loadSound (WINIT 48ee) loads sound n into slot loadedSound; a sound without data of its
// own takes its fallback's samples.
func loadSound(n int) {
	curSound = n
	stopSound()
	sounds[loadedSound] = readSoundRec(n)
	if sounds[loadedSound].Size == 0 {
		n = sounds[loadedSound].Fallback
		sounds[loadedSound].Size = readSoundRec(n).Size
	}
	loadSoundSlot(n, loadedSound, 15)
}

// showCredits (WINIT 49ca) shows the Sir-Tech logo of CREDITS.PIC.
func showCredits() {
	drv.LoadPic(0, openData("CREDITS.PIC", 3))
	win := createWindow(0, 0, drv.Cols, drv.Rows, 1, borderNone, WinPicture, false)
	drv.FillWorkBuffer(winRect(win), 0)
	drawBigChar(10, 84, 52)
	drawBigChar(11, 84, 84)
	drv.ShowWindow(winRect(win))
	destroyWindow(win, false)
}

func winRect(w *Window) drv.Rect { return drv.Rect{X: int(w.X), Y: int(w.Y), W: int(w.W), H: int(w.H)} }

// initGameData (WINIT 4a89) reads the configuration, sets up the screen, fonts and mouse,
// shows the credits, opens the data files and loads the sounds.
func initGameData() {
	fullscreenWin, mainWindow = nil, nil
	scenarioHdr, _ = dbs.Decode[dbs.ScenarioHeader](openData("SCENARIO.HDR", 1)[:dbs.Size[dbs.ScenarioHeader]()])
	messageDelay = 16
	if scenarioHdr.Gmode == gmodeInvalid {
		exitGame(2)
	}
	stopSound()
	audioOK = host.AudioInit()
	videoFlags = videoEGA | videoMouse // init_video: EGA, and a mouse, which the host always has
	if videoFlags&videoMouse != 0 && scenarioHdr.Input == inputMouse {
		drv.MouseInit()
		mousePresent = true
	} else {
		mousePresent = false
	}
	clickFlag = false
	for n := range fontCount {
		loadFontFile(fmt.Sprintf("WFONT%d.EGA", n), n)
	}
	loadMazeData("MAZEDATA.EGA")
	showCredits()

	var err error
	if scenarioDBS, err = dbs.OpenScenario(dataDir); err != nil {
		exitGame(8)
	}
	pcfileHdr, _ = dbs.Decode[dbs.PCFileHeader](openData("PCFILE.DBS", 6)[:dbs.Size[dbs.PCFileHeader]()])
	rosterSlotUsed = pcfileHdr.Used
	if textTree, err = dbs.ReadMiscTable(dataDir); err != nil {
		exitGame(6)
	}
	if messageDBS, err = dbs.OpenMessages(dataDir); err != nil {
		exitGame(7)
	}
	loadSounds(loadedSound)
	gameState = stateTitle
}

// loadFullscreenPic (WINIT 4e5b) makes the full-screen picture window and loads picture
// name into it; show puts it on the screen.
func loadFullscreenPic(name string, show bool) {
	fullscreenWin = createWindow(0, 0, drv.Cols, drv.Rows, 1, borderNone, WinPicture, false)
	stopSound()
	b := openData(name, 12)
	drv.FillWorkBuffer(winRect(fullscreenWin), 0)
	drv.LoadScreen(b)
	if show {
		drv.ShowWindow(winRect(fullscreenWin))
	}
}

// drawBigChar (WINIT 4ee0) draws frame ch of the picture in slot 0 at pixel (x, y).
func drawBigChar(ch byte, x, y int) {
	drv.DrawFrames(0, x, y, 0, 319, 0, false, []byte{ch})
}

// waitEnterKey (WINIT 4f12) waits secs*10 delay ticks, less if Enter is pressed; whether
// it was.
func waitEnterKey(secs int) bool {
	for n := secs * 10; n > 0; n-- {
		if keyAvailable(keyClassMenu) && getKey(keyClassMenu) == keyEnter {
			return true
		}
		delayTick()
	}
	return false
}

// titleFrame is how long a step of the title scroll takes. The original runs it as fast as
// the machine draws (about 0.2 s a step in DOSBox-X at cycles=auto).
const titleFrame = 200 * time.Millisecond

// showTitleSequence (WINIT 4f57): the Sir-Tech and D. W. Bradley credits on the title page,
// then the title and credits scroll up; a key or a click skips to the end.
func showTitleSequence() {
	flushKeys()
	loadSound(39)
	drv.LoadPic(0, openData("CREDITS.PIC", 9)) // the big characters
	loadFullscreenPic("TITLEPAG.EGA", false)
	win := winRect(fullscreenWin)
	skipped := false
	wait := func(secs int) {
		if !skipped && waitEnterKey(secs) {
			skipped = true
		}
	}
	playSound(4)
	wait(2)
	playSound(13)
	drv.FillWorkBuffer(win, 0)
	drawBigChar(13, 91, 52)
	drawBigChar(9, 107, 82)
	drv.ShowWindow(win)
	wait(72)
	drv.LoadScreen(nil)
	drv.ShowWindow(win)
	playSound(loadedSound)
	wait(10)
	playSound(6)
	drawBigChar(7, 76, 67)
	drawBigChar(8, 76, 99)
	drv.ShowWindow(win)
	playSound(soundHit)
	wait(10)

	// The pieces: their frame, column, the step they start moving, their start row and the
	// row they stop at (pieces 0..2 and 8) or vanish above (the rest).
	pieces := [9]struct {
		glyph             byte
		x, start, y, minY int
	}{
		{7, 76, 0, 67, 3},
		{8, 76, 0, 99, 35},
		{12, 8, 0, 13, 13},
		{1, 20, 4, 144, 13},
		{2, 20, 36, 144, 13},
		{3, 20, 60, 144, 13},
		{4, 20, 88, 144, 13},
		{5, 20, 120, 144, 13},
		{6, 14, 152, 144, 80},
	}
	animating := !skipped
	last := pieces[7].start + pieces[7].y - pieces[7].minY
	next := host.Now()
	for frame := 0; animating && frame < last; frame += 2 {
		drv.LoadScreen(nil)
		for i := 8; i >= 0; i-- {
			p := pieces[i]
			if p.start > frame {
				continue
			}
			y := p.y - (frame - p.start)
			if (i < 3 || i == 8) && y < p.minY {
				y = p.minY
			}
			if y < p.minY {
				continue
			}
			drawBigChar(p.glyph, p.x, y)
		}
		drv.ShowWindow(win)
		if checkScreenClick() {
			animating = false
		}
		if keyAvailable(keyClassMenu) {
			getKey(keyClassMenu)
			animating = false
		}
		for next += titleFrame; host.Now() < next; {
			host.Sleep()
		}
	}
	drv.LoadScreen(nil)
	drawBigChar(7, 76, 3)
	drawBigChar(8, 76, 35)
	drawBigChar(6, 14, 80)
	drv.ShowWindow(win)
	destroyWindow(fullscreenWin, false)
	gameState = stateMainScreen
	if animating {
		for !keyAvailable(keyClassMenu) && !checkScreenClick() {
		}
	} else {
		waitSecondsOrEnter(28)
	}
	flushKeys()
}

// showGraveyard (WINIT 535a) takes down the game windows, shows the graveyard and waits
// for a key.
func showGraveyard() {
	mazeA, mazeB = nil, nil
	partyCount = 0
	for _, w := range []*Window{movePadWin, titleStripWin, pictureWin, fullscreenWin, partyPictureWin, leftPanelWin, rightPanelWin, mainWindow, bottomLineWin} {
		destroyWindow(w, false)
	}
	loadSound(34)
	loadFullscreenPic("GRAVEYRD.EGA", true)
	playSound(loadedSound)
	stopSound()
	destroyWindow(fullscreenWin, false)
	gameState = stateMainScreen
	flushKeys()
	for !keyAvailable(keyClassMenu) && !checkScreenClick() {
	}
	flushKeys()
}

// initMainScreen (WINIT 54a7) loads the monster pictures and DRAGONSC, and makes the game's
// windows.
func initMainScreen() {
	stopSound()
	for j := range picSlotMonster {
		picSlotMonster[j], picSlotLoaded[j] = -1, false
	}
	for _, p := range [][2]int{{3, 0}, {1, 1}, {2, 2}, {7, 3}, {4, 7}, {5, 8}, {6, 9}} {
		loadMonsterPic(p[0], p[1], 9)
	}
	loadFullscreenPic("DRAGONSC.EGA", false)
	partyPictureWin = createWindow(0, 19, 40, 5, 4, borderNone, 0, false)
	fillWindow(partyPictureWin, 26, 3)
	leftPanelWin = createWindow(1, 5, 7, 12, 6, borderLeftPanel, 0, false)
	fillWindow(leftPanelWin, 32, 3)
	rightPanelWin = createWindow(32, 5, 7, 12, 7, borderRightPanel, 0, false)
	fillWindow(rightPanelWin, 32, 3)
	mainWindow = createWindow(0, 18, 40, 1, 14, borderNone, 0, false)
	eraseWindow(mainWindow, false)
	bottomLineWin = createWindow(0, 24, 40, 1, 3, borderNone, 0, false)
	eraseWindow(bottomLineWin, false)
	pictureWin = createWindow(9, 4, 22, 14, 2, borderNone, WinPicture, false)
	titleStripWin = createWindow(2, 0, 36, 3, 5, borderNone, WinPicture, false)
	msgBlockNum = -1
	partyCount = 0
	for i := range memberPcfileIndex {
		memberPcfileIndex[i] = -1
	}
	for j := range 3 {
		picSlotMonster[4+j], picSlotLoaded[4+j] = -1, false
	}
	encounterID = 0
	gameState = stateMasterOptions
	drv.ShowWindow(winRect(fullscreenWin))
}
