package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
)

// MainMaze is Main for testing the maze until WBASE's start_game is in (the -maze flag):
// it boots like WINIT without the title, starts NEWGAME.DBS's map and position with the
// first six characters of PCFILE.DBS and runs WMAZE until the game leaves the maze.
func MainMaze(dir string) (code int) {
	dataDir = dir
	defer func() {
		if r := recover(); r != nil {
			c, ok := r.(exitCode)
			if !ok {
				panic(r)
			}
			code = int(c)
		}
	}()
	drv.Init()
	calibrateDelay()
	seedRandom()
	initGameData()
	initMainScreen()
	startTestMaze()
	for gameState == 5 || gameState == 6 || gameState == 23 || gameState == 25 {
		if gameState == 25 {
			gameState = 5
		}
		wmazeMain()
		stopSound()
	}
	fmt.Fprintf(os.Stderr, "wiz6 -maze: left the maze for game_state %d\n", gameState)
	shutdownGame(0)
	return 0
}

// startTestMaze is load_game(0), the roster and setup_game_windows (WBASE), cut down.
func startTestMaze() {
	g, err := dbs.DecodeSaveGame(openData("NEWGAME.DBS", 9))
	if err != nil {
		exitGame(9)
	}
	hdr.LevelFlags = g.Hdr.LevelFlags
	for i := range 14 {
		mapCacheA[i], mapCacheB[i] = g.MapA[i], g.MapB[i]
	}
	s := g.State
	partyX, partyY, cellX, cellY, partyBlock, partyFacing = s.PartyX, s.PartyY, s.CellX, s.CellY, s.Block, s.Facing
	gameStateArg = int(s.Map)
	saveState = s
	partyCounters = s.Counters
	roster := openData("PCFILE.DBS", 6)
	partyCount = 0
	for i := range 6 {
		pc, err := dbs.Decode[PC](roster[24+432*i : 24+432*(i+1)])
		if err != nil || pc.Name[0] == 0 {
			break
		}
		party[partyCount] = pc
		memberPcfileIndex[partyCount] = i
		partyCount++
	}
	g4f88 = 0
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	g4fb0 = createWindow(20, 19, 20, 5, 8, -1, 0, false)
	fillWindow(g4fb0, 32, 3)
	curEvent = [4]int16{}
	g4f86 = 0
	mazeA, mazeB = nil, nil
	gameState = 6
}
