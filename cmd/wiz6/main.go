// Command wiz6 runs Wizardry VI: Bane of the Cosmic Forge.
package main

import (
	"time"

	"github.com/retro-ware/wiz6-go/internal/host"
)

func main() {
	host.Scripted = true
	time.AfterFunc(time.Second, func() { host.Screenshot("/tmp/claude-501/s.png") })
	host.Run("Wizardry VI", func() {
		px := make([]byte, 320*200)
		for i := range px {
			px[i] = byte(i / 320 / 13)
		}
		host.SetPalette(0, []host.Color{{0, 0, 0}, {0, 0, 42}, {0, 42, 0}, {0, 42, 42}, {42, 0, 0}, {42, 0, 42}, {42, 21, 0}, {42, 42, 42}, {21, 21, 21}, {21, 21, 63}, {21, 63, 21}, {21, 63, 63}, {63, 21, 21}, {63, 21, 63}, {63, 63, 21}, {63, 63, 63}})
		host.Present(px, 320, 200)
		time.Sleep(3 * time.Second)
	})
}
