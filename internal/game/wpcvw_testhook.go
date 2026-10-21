package game

import (
	"fmt"
	"os"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// PCView is a testing entry: NEWGAME.DBS's maze state with the first six PCFILE.DBS
// characters (as MainMaze), straight into the character view of member m (game state 17).
func PCView(dir string, m int) (code int) {
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
	for i := 0; i < partyCount; i++ {
		memberPortrait[i] = uint8(i)
		loadPortrait(int(party[i].Portrait), i)
	}
	menuKeys = getMessage(300)
	curMember = int16(min(m, max(partyCount-1, 0)))
	returnCmd, gameState = 5, 17
	wpcvwMain()
	fmt.Fprintf(os.Stderr, "wiz6 -pcview: left the character view for game_state %d\n", gameState)
	shutdownGame(0)
	return 0
}
