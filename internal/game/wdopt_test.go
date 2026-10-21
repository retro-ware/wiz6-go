package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// TestWdoptScreens runs WDOPT's use item (20) and cast spell (19) in NEWGAME.DBS's maze
// with scripted keys and writes PNGs to $WDOPT_SHOTS (skipped when unset).
func TestWdoptScreens(t *testing.T) {
	out := os.Getenv("WDOPT_SHOTS")
	if out == "" {
		t.Skip("WDOPT_SHOTS not set")
	}
	if dataDir = os.Getenv("WIZ6_DATA"); dataDir == "" {
		dataDir = "../../../wiz6-re/gog/x/app"
	}
	host.Scripted = true
	ready, done, mapNext, mapGo := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		drv.Init()
		calibrateDelay()
		seedRandom()
		initGameData()
		initMainScreen()
		startTestMaze()
		loadMazeLevel(gameStateArg)
		redrawMazeView()
		close(ready)
		for _, s := range []gameStateID{stateUseItemInMaze, stateCastSpellInMaze} {
			gameState, curMember = s, 0
			for m := partyCount - 1; m >= 0 && s == 19; m-- {
				if party[m].SP[0].Cur > 0 {
					curMember = int16(m)
				}
			}
			wdoptOvmain()
			if gameState != 5 {
				t.Errorf("game_state %d after WDOPT", gameState)
			}
		}
		close(mapNext)
		<-mapGo
		wdoptShowOverheadMap(6) // the 60 spell at power 6
	}()
	<-ready
	tap := func(code byte) {
		host.PushKey(code)
		host.PushKey(code | 0x80)
		time.Sleep(400 * time.Millisecond)
	}
	shot := func(name string) {
		time.Sleep(300 * time.Millisecond)
		px, w, h := host.FrameRGBA()
		if err := host.WritePNG(filepath.Join(out, name), px, w, h); err != nil {
			t.Error(err)
		}
	}
	const enter, up, down, right = 0x1c, 0x48, 0x50, 0x4d
	shot("wdopt1_items.png")
	tap(down)
	shot("wdopt2_items_down.png")
	tap(enter) // use the second item
	time.Sleep(4 * time.Second)
	shot("wdopt3_used.png")
	tap(enter)
	time.Sleep(time.Second)
	shot("wdopt4_book.png")
	tap(enter)
	shot("wdopt6_spells.png")
	tap(enter)
	shot("wdopt7_power.png")
	tap(enter)
	time.Sleep(time.Second)
	shot("wdopt8_cast.png")
	for range 10 {
		select {
		case <-mapNext:
		default:
			tap(enter)
			continue
		}
		break
	}
	<-mapNext
	time.Sleep(time.Second)
	close(mapGo)
	time.Sleep(2 * time.Second)
	shot("wdopt9_map.png")
	tap(enter)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		shot("wdopt10_stuck.png")
		t.Fatal("WDOPT did not return")
	}
}
