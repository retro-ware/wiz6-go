package game

import (
	"fmt"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// WINIT: start-up, the title sequence, the graveyard and the main screen's windows.

// winitMain (WINIT ovmain) does what gameState asks: 0 load the data, 1 the title, 8 the
// graveyard, 2 the main screen.
func winitMain() {
	switch gameState {
	case 0:
		initGameData()
	case 1:
		showTitleSequence()
	case 8:
		showGraveyard()
	case 2:
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

// loadSound (WINIT 48ee) loads sound n into slot 14; a sound without data of its own takes
// its fallback's samples.
func loadSound(n int) {
	curSound = n
	stopSound()
	sounds[14] = readSoundRec(n)
	if sounds[14].Size == 0 {
		n = sounds[14].Fallback
		sounds[14].Size = readSoundRec(n).Size
	}
	loadSoundSlot(n, 14, 15)
}

// showCredits (WINIT 49ca) shows the Sir-Tech logo of CREDITS.PIC.
func showCredits() {
	drv.LoadPic(0, openData("CREDITS.PIC", 3))
	win := createWindow(0, 0, 40, 25, 1, -1, WinPicture, false)
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
	hdr, _ = dbs.Decode[dbs.ScenarioHeader](openData("SCENARIO.HDR", 1)[:dbs.Size[dbs.ScenarioHeader]()])
	messageDelay = 16
	if hdr.Gmode == 255 {
		exitGame(2)
	}
	stopSound()
	audioOK = host.AudioInit()
	videoFlags = 0x1001 // init_video: EGA, and a mouse, which the host always has
	if videoFlags&0x1000 != 0 && hdr.Input == 1 {
		drv.MouseInit()
		mousePresent = true
	} else {
		mousePresent = false
	}
	clickFlag = false
	for n := range 5 {
		loadFontFile(fmt.Sprintf("WFONT%d.EGA", n), n)
	}
	loadMazeData("MAZEDATA.EGA")
	showCredits()

	var err error
	if scen, err = dbs.OpenScenario(dataDir); err != nil {
		exitGame(8)
	}
	pcfileHdr, _ = dbs.Decode[dbs.PCFileHeader](openData("PCFILE.DBS", 6)[:24])
	rosterSlotUsed = pcfileHdr.Used
	if misc, err = dbs.ReadMiscTable(dataDir); err != nil {
		exitGame(6)
	}
	if msgs, err = dbs.OpenMessages(dataDir); err != nil {
		exitGame(7)
	}
	loadSounds(14)
	gameState = 1
}

// loadFullscreenPic (WINIT 4e5b) makes the full-screen picture window and loads picture
// name into it; show puts it on the screen.
func loadFullscreenPic(name string, show bool) {
	fullscreenWin = createWindow(0, 0, 40, 25, 1, -1, WinPicture, false)
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
		if keyAvailable(1) && getKey(1) == 13 {
			return true
		}
		delayTick()
	}
	return false
}

// titleFrame is how long a step of the title scroll takes. The original runs it as fast as
// the machine draws (about 0.2 s a step in DOSBox-X at cycles=auto).
const titleFrame = 200 * 1000 * 1000 // ns

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
	playSound(14)
	wait(10)
	playSound(6)
	drawBigChar(7, 76, 67)
	drawBigChar(8, 76, 99)
	drv.ShowWindow(win)
	playSound(7)
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
		if keyAvailable(1) {
			getKey(1)
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
	gameState = 2
	if animating {
		for !keyAvailable(1) && !checkScreenClick() {
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
	for _, w := range []*Window{g4fb0, g4fac, pictureWin, fullscreenWin, partyPictureWin, g4fba, g4fb8, mainWindow, bottomLineWin} {
		destroyWindow(w, false)
	}
	loadSound(34)
	loadFullscreenPic("GRAVEYRD.EGA", true)
	playSound(14)
	stopSound()
	destroyWindow(fullscreenWin, false)
	gameState = 2
	flushKeys()
	for !keyAvailable(1) && !checkScreenClick() {
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
	partyPictureWin = createWindow(0, 19, 40, 5, 4, -1, 0, false)
	fillWindow(partyPictureWin, 26, 3)
	g4fba = createWindow(1, 5, 7, 12, 6, -2, 0, false)
	fillWindow(g4fba, 32, 3)
	g4fb8 = createWindow(32, 5, 7, 12, 7, -3, 0, false)
	fillWindow(g4fb8, 32, 3)
	mainWindow = createWindow(0, 18, 40, 1, 14, -1, 0, false)
	eraseWindow(mainWindow, false)
	bottomLineWin = createWindow(0, 24, 40, 1, 3, -1, 0, false)
	eraseWindow(bottomLineWin, false)
	pictureWin = createWindow(9, 4, 22, 14, 2, -1, WinPicture, false)
	g4fac = createWindow(2, 0, 36, 3, 5, -1, WinPicture, false)
	msgBlockNum = -1
	partyCount = 0
	for i := range memberPcfileIndex {
		memberPcfileIndex[i] = -1
	}
	for j := range 3 {
		picSlotMonster[4+j], picSlotLoaded[4+j] = -1, false
	}
	g433a = 0
	gameState = 4
	drv.ShowWindow(winRect(fullscreenWin))
}
