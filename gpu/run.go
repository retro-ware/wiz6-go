package gpu

import (
	"errors"
	"time"

	"github.com/gogpu/gogpu"
	_ "github.com/gogpu/wgpu/hal/allbackends" // the desktop GPU backends
)

// Game is what RunGame runs, like ebiten.Game: Update runs TPS times a
// second, Draw once a frame into a screen of the size Layout returns for the
// window's size.
type Game interface {
	Update() error
	Draw(screen *Image)
	Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)
}

// ErrTermination returned by Update ends RunGame without an error.
var ErrTermination = errors.New("gpu: termination")

var win struct {
	app                             *gogpu.App
	requestedWidth, requestedHeight int // requested size
	title                           string
	tps                             int
	resizable                       bool
	runWhenUnfocused                bool // keep updating while unfocused (always on here)
	fullscreen                      bool
	fps                             float64
	frames                          int
	fpsStart                        time.Time

	// the screen's place in the framebuffer (for the cursor)
	screenX, screenY, screenScale float64
}

// The window's size and ticks a second until the game sets them.
const (
	defaultWidth, defaultHeight = 640, 480
	defaultTPS                  = 60
	// maxTicksPerFrame caps the Updates one frame runs after a stall.
	maxTicksPerFrame = 8
)

func init() {
	win.requestedWidth, win.requestedHeight, win.tps, win.title = defaultWidth, defaultHeight, defaultTPS, ""
}

func SetWindowSize(w, h int) {
	win.requestedWidth, win.requestedHeight = w, h
	if win.app != nil {
		win.app.RequestSize(w, h)
	}
}

func SetWindowTitle(t string) {
	win.title = t
	if win.app != nil {
		win.app.SetTitle(t)
	}
}

// WindowResizingMode says whether the player may resize the window.
type WindowResizingMode int

const (
	WindowResizingModeDisabled WindowResizingMode = iota
	WindowResizingModeEnabled
)

func SetWindowResizingMode(m WindowResizingMode) { win.resizable = m == WindowResizingModeEnabled }
func SetTPS(n int)                               { win.tps = n }
func SetRunnableOnUnfocused(b bool)              { win.runWhenUnfocused = b }
func ActualFPS() float64                         { return win.fps }
func IsFullscreen() bool                         { return win.fullscreen }

func SetFullscreen(on bool) {
	win.fullscreen = on
	if win.app != nil {
		win.app.SetFullscreen(on)
	}
}

// MonitorType is the window's monitor.
type MonitorType struct{}

// Monitor returns the window's monitor.
func Monitor() *MonitorType { return &MonitorType{} }

// DeviceScaleFactor is framebuffer pixels per window unit.
func (*MonitorType) DeviceScaleFactor() float64 {
	if win.app == nil || win.app.ScaleFactor() == 0 {
		return 1
	}
	return win.app.ScaleFactor()
}

// RunGameOptions are RunGameWithOptions' options.
type RunGameOptions struct {
	InitUnfocused bool // ponytail: gogpu has no unfocused start; the window opens focused
}

func RunGame(g Game) error { return RunGameWithOptions(g, nil) }

// RunGameWithOptions opens the window and runs g until the window closes or
// Update returns an error.
func RunGameWithOptions(g Game, o *RunGameOptions) error {
	cfg := gogpu.DefaultConfig().WithTitle(win.title).WithSize(win.requestedWidth, win.requestedHeight).
		WithResizable(win.resizable).WithContinuousRender(true)
	if win.fullscreen {
		cfg = cfg.WithFullscreen()
	}
	app := gogpu.NewApp(cfg)
	win.app = app
	initInput(app)
	var (
		screen *Image
		last   time.Time
		acc    float64
		runErr error
	)
	app.OnDraw(func(dc *gogpu.Context) {
		p := app.DeviceProvider()
		view := dc.SurfaceView()
		if p == nil || view == nil || runErr != nil {
			return
		}
		if screen == nil && last.IsZero() {
			if err := adoptDevice(p.Device(), p.Queue(), p.SurfaceFormat()); err != nil {
				runErr = err
				app.Quit()
				return
			}
			last = time.Now()
			win.fpsStart = last
		}
		now := time.Now()
		acc += now.Sub(last).Seconds() * float64(win.tps)
		last = now
		n := int(acc)
		acc -= float64(n)
		if n > maxTicksPerFrame { // ponytail: a long stall drops ticks instead of catching up
			n = maxTicksPerFrame
		}
		if screen == nil {
			n = max(n, 1) // Update comes before the first Draw
		}
		for range n {
			tickInput()
			if err := g.Update(); err != nil {
				if err != ErrTermination {
					runErr = err
				}
				app.Quit()
				return
			}
		}
		ww, wh := app.Size()
		fw, fh := dc.FramebufferSize()
		if ww == 0 || wh == 0 || fw == 0 || fh == 0 { // minimised
			return
		}
		sw, sh := g.Layout(ww, wh)
		sw, sh = max(sw, 1), max(sh, 1)
		if screen == nil || screen.r.Dx() != sw || screen.r.Dy() != sh {
			if screen != nil {
				screen.Deallocate()
			}
			screen = NewImage(sw, sh)
		}
		screen.Clear()
		g.Draw(screen)
		win.screenScale = min(float64(fw)/float64(sw), float64(fh)/float64(sh))
		win.screenX, win.screenY = (float64(fw)-float64(sw)*win.screenScale)/2, (float64(fh)-float64(sh)*win.screenScale)/2
		present(screen, view, fw, fh)
		win.frames++
		if d := now.Sub(win.fpsStart).Seconds(); d >= 1 {
			win.fps, win.frames, win.fpsStart = float64(win.frames)/d, 0, now
		}
	})
	if err := app.Run(); err != nil {
		return err
	}
	return runErr
}
