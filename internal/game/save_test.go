package game

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// TestSaveRoundTrip loads $WIZ6_SAVE (a SAVEGAME.DBS, e.g. one the original wrote under
// DOSBox-X; see tools/parity/save.keys) with load_game, saves it with save_game and expects
// the same bytes. Skipped when unset.
func TestSaveRoundTrip(t *testing.T) {
	src := os.Getenv("WIZ6_SAVE")
	if src == "" {
		t.Skip("WIZ6_SAVE not set")
	}
	orig := os.Getenv("WIZ6_DATA")
	if orig == "" {
		orig = "../../../wiz6-re/gog/x/app"
	}
	want, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dataDir = t.TempDir()
	if out, err := exec.Command("cp", "-R", orig+"/", dataDir).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	if err := os.WriteFile(filepath.Join(dataDir, "SAVEGAME.DBS"), want, 0o644); err != nil {
		t.Fatal(err)
	}
	drv.Init()
	initGameData()
	initMainScreen()
	if loadGame(1) != 0 {
		t.Fatal("load_game failed")
	}
	setupGameWindows()
	loadMazeLevel(gameStateArg)
	initMazeScreen()
	if saveGame() != 0 {
		t.Fatal("save_game failed")
	}
	got, _ := os.ReadFile(filepath.Join(dataDir, "SAVEGAME.DBS"))
	if !bytes.Equal(got, want) {
		n := 0
		for i := range min(len(got), len(want)) {
			if got[i] != want[i] {
				if n < 10 {
					t.Errorf("byte %d: %02x, want %02x", i, got[i], want[i])
				}
				n++
			}
		}
		t.Fatalf("%d bytes differ, length %d want %d", n, len(got), len(want))
	}
}
