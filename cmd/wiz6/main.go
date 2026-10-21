// Command wiz6 runs Wizardry VI: Bane of the Cosmic Forge from its data files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/retro-ware/wiz6-go/internal/game"
	"github.com/retro-ware/wiz6-go/internal/host"
)

func main() {
	data := flag.String("data", "", "the folder with SCENARIO.DBS and the other data files (default: ./data when it has them, else .)")
	imp := flag.String("import", "", "find Wizardry VI in this folder, zip, CD image or GOG installer, copy it into -data and exit")
	shots := flag.String("shot", "", "write the game's frame to PNGs: SECONDS:FILE[,SECONDS:FILE...]")
	input := flag.String("input", "", "replace the keyboard and mouse with a script: SECONDS:ACTION[,...], "+
		"ACTION kHEX (tap a scan code), pHEX/rHEX (press/release a scan code), aX/Y (put the mouse at X, Y), mDX/DY (move the mouse), bN (click button N), dN/uN (button N down/up)")
	steps := flag.String("steps", "", "testing: -input ACTIONs done one by one once the screen has settled (see -stepdir); ACTION/SECS waits at least SECS after the previous one")
	stepDir := flag.String("stepdir", ".", "testing: with -steps, write DIR/NNN.png, the settled frame after step NNN")
	stepWait := flag.Float64("stepwait", 8, "testing: with -steps, seconds before the first step at least")
	detRNG := flag.Bool("detrng", false, "testing: fixed random seed, no random step per key poll (matches tools/parity's patched WROOT.EXE)")
	maze := flag.Bool("maze", false, "testing: boot straight into NEWGAME.DBS's maze with the first six PCFILE.DBS characters")
	pcview := flag.Int("pcview", -1, "testing: as -maze, but open the character view of party member N")
	fight := flag.Int("fight", -1, "testing: as -maze, but start an encounter (with monster N when N > 0)")
	chest := flag.Int("chest", -1, "testing: as -maze, but open treasure record N as a found chest")
	npc := flag.Int("npc", -1, "testing: as -maze, but meet NPC record N")
	flag.Parse()
	if *imp != "" {
		if *data == "" {
			*data = "data"
		}
		if err := importGame(*imp, *data); err != nil {
			fmt.Fprintln(os.Stderr, "wiz6: import:", err)
			os.Exit(1)
		}
		return
	}
	if *data == "" {
		*data = "."
		if _, err := os.Stat(filepath.Join("data", "SCENARIO.DBS")); err == nil {
			*data = "data"
		}
	}
	if _, err := os.Stat(filepath.Join(*data, "SCENARIO.DBS")); err != nil {
		fmt.Fprintln(os.Stderr, "wiz6: no SCENARIO.DBS in", *data)
		os.Exit(1)
	}
	for _, s := range strings.Split(*shots, ",") {
		secs, path, ok := strings.Cut(s, ":")
		d, err := time.ParseDuration(secs + "s")
		if !ok || err != nil {
			continue
		}
		time.AfterFunc(d, func() { host.Screenshot(path) })
	}
	game.DetRNG = *detRNG
	if *input != "" || *steps != "" {
		host.Scripted = true
		host.MouseTo(160, 100) // where the original's mouse driver reset puts the pointer
	}
	if *steps != "" {
		go runSteps(strings.Split(*steps, ","), *stepDir, time.Duration(*stepWait*float64(time.Second)))
	}
	for _, s := range strings.Split(*input, ",") {
		secs, act, ok := strings.Cut(s, ":")
		d, err := time.ParseDuration(secs + "s")
		if !ok || err != nil || act == "" {
			continue
		}
		time.AfterFunc(d, func() { scriptAction(act) })
	}
	code := 0
	if err := host.Run("Wizardry VI", func() {
		if *fight >= 0 {
			code = game.MainFight(*data, *fight)
		} else if *chest >= 0 {
			code = game.MainChest(*data, *chest)
		} else if *npc >= 0 {
			code = game.MainNPC(*data, *npc)
		} else if *pcview >= 0 {
			code = game.PCView(*data, *pcview)
		} else if *maze {
			code = game.MainMaze(*data)
		} else {
			code = game.Main(*data)
		}
		host.Quit()
	}); err != nil {
		fmt.Fprintln(os.Stderr, "wiz6:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// scriptAction performs one -input action.
func scriptAction(act string) {
	switch act[0] {
	case 'k':
		var code uint
		if _, err := fmt.Sscanf(act[1:], "%x", &code); err == nil {
			host.PushKey(byte(code))
			time.AfterFunc(100*time.Millisecond, func() { host.PushKey(byte(code) | host.KeyBreakBit) })
		}
	case 'p', 'r':
		var code uint
		if _, err := fmt.Sscanf(act[1:], "%x", &code); err == nil {
			if act[0] == 'r' {
				code |= host.KeyBreakBit
			}
			host.PushKey(byte(code))
		}
	case 'a':
		var x, y float64
		if _, err := fmt.Sscanf(act[1:], "%g/%g", &x, &y); err == nil {
			host.MouseTo(x, y)
		}
	case 'm':
		var dx, dy float64
		if _, err := fmt.Sscanf(act[1:], "%g/%g", &dx, &dy); err == nil {
			host.MouseMove(dx, dy)
		}
	case 'd', 'u':
		var b int
		if _, err := fmt.Sscanf(act[1:], "%d", &b); err == nil {
			host.MouseButton(b, act[0] == 'd')
		}
	case 'b':
		var b int
		if _, err := fmt.Sscanf(act[1:], "%d", &b); err == nil {
			host.MouseButton(b, true)
			time.AfterFunc(100*time.Millisecond, func() { host.MouseButton(b, false) })
		}
	}
}

// Settling for -steps: a step's frame is taken once the screen has not changed for settle,
// or after settleMax for screens that keep animating.
const (
	settle    = 1500 * time.Millisecond
	settleMax = 8 * time.Second
)

// runSteps does each action once the screen has settled, saving the settled frame before
// the next action (and after the last), then quits.
func runSteps(acts []string, dir string, first time.Duration) {
	waitSettled := func(min time.Duration) {
		t0 := time.Now()
		for time.Since(t0) < min || host.Unchanged() < settle && time.Since(t0) < settleMax {
			time.Sleep(20 * time.Millisecond)
		}
	}
	shot := func(i int) {
		px, w, h := host.FrameRGBA()
		if err := host.WritePNG(filepath.Join(dir, fmt.Sprintf("%03d.png", i)), px, w, h); err != nil {
			fmt.Fprintln(os.Stderr, "wiz6:", err)
		}
	}
	waitSettled(first)
	for i, a := range acts {
		// ACTION/SECS: at least SECS since the previous step (as the original gets in a parity run).
		min := 500 * time.Millisecond
		if a2, secs, ok := strings.Cut(a, "/"); ok {
			a = a2
			if f, err := strconv.ParseFloat(secs, 64); err == nil {
				min = time.Duration(f * float64(time.Second))
			}
		}
		if i > 0 {
			waitSettled(min)
			shot(i - 1)
		}
		fmt.Fprintf(os.Stderr, "wiz6: step %d %s\n", i, a)
		scriptAction(a)
	}
	waitSettled(500 * time.Millisecond)
	shot(len(acts) - 1)
	host.Quit()
}
