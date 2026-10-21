package game

import (
	"time"

	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// Animation phase periods of the original on a 386DX-33 (DOSBox-X cycles=fixed 7800): its
// view redraw took most of a phase, so the phase flipped every 0.86 s in the maze (measured
// at the first gate view). The combat and NPC views flip once per 15 steps, not 10.
const (
	mazeAnimPeriod = 860 * time.Millisecond
	viewAnimPeriod = mazeAnimPeriod * 15 / 10
)

var lastAnimFlip time.Duration

// animPace waits, at an animation phase flip, until period has passed since the last flip.
// A waiting key or click ends the wait at once; the screen keeps being presented meanwhile.
// It does not step the random numbers.
func animPace(period time.Duration) {
	if dtLog != nil { // no wall-clock waits in the difftests
		return
	}
	for host.Now()-lastAnimFlip < period {
		pumpKeys()
		if len(keyBuf) > 0 || drv.ClickPending() {
			break
		}
		drv.Present()
		host.Sleep()
	}
	lastAnimFlip = host.Now()
}

// paceViewCycle paces the combat/NPC view loops (15 steps, flip at state -3) where they
// idle for input; their whole-cycle redraws are not paced.
func paceViewCycle() {
	if viewAnimState == -3 {
		animPace(viewAnimPeriod)
	}
}
