package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// TestWpopsScreens runs WPOPS's start of a combat round against a made-up group in
// NEWGAME.DBS's maze, with scripted keys, and writes PNGs to $WPOPS_SHOTS (skipped
// when unset).
func TestWpopsScreens(t *testing.T) {
	out := os.Getenv("WPOPS_SHOTS")
	if out == "" {
		t.Skip("WPOPS_SHOTS not set")
	}
	if dataDir = os.Getenv("WIZ6_DATA"); dataDir == "" {
		dataDir = "../../../wiz6-re/gog/x/app"
	}
	host.Scripted = true
	ready, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		drv.Init()
		calibrateDelay()
		seedRandom()
		initGameData()
		initMainScreen()
		startTestMaze()
		loadMazeLevel(gameStateArg)
		// A minimal encounter: group 1 is two of monster 1, no allies.
		for g := range monsterGroups {
			monsterGroups[g] = &MonsterGroup{}
		}
		b := make([]byte, dbs.Size[MonsterRec]())
		accessScenarioRecord(dbs.RecMonster, 1, b, false)
		rec, _ := dbs.Decode[MonsterRec](b)
		for g := range groupMonsterRec {
			groupMonsterRec[g] = &rec
		}
		loadMonsterPicSlot(int(rec.Pic), 3)
		g := monsterGroups[1]
		g.Count, g.PicSlot, g.Unidentified = 2, 3, 1
		for i := range 2 {
			g.Members[i] = Combatant{Level: 1, HP: 5, HPMax: 9, Pos: int8(i)}
		}
		for i := range 9 {
			monsterGroups[0].Members[i].Pos = -1
		}
		monsterGroupCount = 1
		viewPosGroup[0][1], viewPosGroup[0][2] = 1, 1
		combatKeys = getMessage(3900)
		combatActionWin = createWindow(0, 19, 19, 5, 8, -1, 0, false)
		monsterListWin = createWindow(20, 19, 20, 5, 8, -1, 0, false)
		close(ready)
		gameState = 12
		wpopsOvmain()
		if gameState != 13 {
			t.Errorf("game_state %d after WPOPS", gameState)
		}
		for m := 0; m < partyCount; m++ {
			c := monsterGroups[0].Members[m]
			t.Logf("member %d: action %d choice %d target %d/%d", m, c.Plan[6], c.Choice, c.TargetGroup, c.TargetMember)
		}
		t.Logf("monster 0: action %d choice %d target %d/%d", g.Members[0].Plan[6], g.Members[0].Choice, g.Members[0].TargetGroup, g.Members[0].TargetMember)
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
	shot("wpops1_menu.png")
	tap(0x1c) // fight
	shot("wpops2_fight.png")
	tap(0x1c)
	shot("wpops3_next.png")
	tap(0x4d) // right: spell
	tap(0x1c)
	shot("wpops4_spellbook.png")
	tap(0x01) // escape out of the book
	tap(0x01)
	for range partyCount * 3 {
		tap(0x1c)
	}
	shot("wpops5_confirm.png")
	for range 4 {
		tap(0x1c)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("WPOPS did not finish")
	}
}
