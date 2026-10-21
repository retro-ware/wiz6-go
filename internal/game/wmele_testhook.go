package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// MainFight is Main for testing the combat overlays (the -fight flag): it boots like
// MainMaze, shows the maze and starts the map's encounter (monster mon instead when mon > 0), then runs WMELE,
// WPOPS and WMEXE until the combat is over.
func MainFight(dir string, mon int) (code int) {
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
	loadMazeLevel(gameStateArg)
	initMazeScreen()
	refreshAllWindows()
	returnCmd = 5
	g4fc8 = int16(hdr.Delay) // as load_game does
	if g4fc8 == 0 {
		g4fc8 = 5
	}
	startEncounter(0)
	if mon > 0 {
		g433a = mon
	}
	for {
		switch gameState {
		case 10, 11, 14:
			wmeleOvmain()
		case 12:
			wpopsOvmain()
		case 13:
			wmexeOvmain()
		case 17:
			wpcvwMain()
		default:
			fmt.Fprintf(os.Stderr, "wiz6 -fight: left the combat for game_state %d\n", gameState)
			shutdownGame(0)
			return 0
		}
		stopSound()
	}
}
