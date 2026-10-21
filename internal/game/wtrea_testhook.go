package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// MainChest is Main for testing WTREA (the -chest flag): it boots like MainMaze, shows the
// maze and opens treasure record id as a found chest (game state 21).
func MainChest(dir string, id int) (code int) {
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
	treasureID = int16(id)
	gameState = 21
	wtreaOvmain()
	fmt.Fprintf(os.Stderr, "wiz6 -chest: left WTREA for game_state %d\n", gameState)
	shutdownGame(0)
	return 0
}
