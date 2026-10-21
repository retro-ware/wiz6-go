package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// MainNPC is Main for testing WMNPC (the -npc flag): it boots like MainMaze, shows the
// maze and meets NPC record npc (game state 18).
func MainNPC(dir string, npc int) (code int) {
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
	g433a = npc
	gameState = 18
	wmnpcOvmain()
	fmt.Fprintf(os.Stderr, "wiz6 -npc: left WMNPC for game_state %d\n", gameState)
	shutdownGame(0)
	return 0
}
