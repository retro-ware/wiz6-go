// Package game is the port of WROOT.EXE and its overlays: the root runtime (windows,
// text, messages, keys, clicks, random numbers, sound, disks) that every overlay calls,
// main's game_state loop, and the overlays themselves.
package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
)

// exitCode unwinds the game to Main with the code exit_game gave.
type exitCode int

// exitGame (ROOT 00d5) shuts down and leaves with code (1..21 name the failure).
func exitGame(code int) {
	shutdownGame(code)
	panic(exitCode(code))
}

// shutdownGame (ROOT 0000) stops the sound and the mouse.
func shutdownGame(code int) {
	if code >= 3 && mousePresent {
		drv.MouseShutdown()
	}
	stopSound()
}

// Main runs the game from dir until it quits; the exit code.
func Main(dir string) (code int) {
	dataDir = dir
	defer func() {
		if r := recover(); r != nil {
			c, ok := r.(exitCode)
			if !ok {
				panic(r)
			}
			if c != 0 {
				fmt.Fprintf(os.Stderr, "wiz6: exit %d\n", int(c))
			}
			code = int(c)
		}
	}()
	drv.Init()
	calibrateDelay()
	seedRandom()
	gameState = 0
	for {
		switch gameState {
		case 8:
			ensureDiskFor(3, 0)
			winitMain()
		case 0, 1, 2:
			winitMain()
		case 7, 24:
			ensureDiskFor(1, 0)
			wbaseMain()
		case 4:
			wbaseMain()
		case 25:
			ensureDiskFor(2, gameStateArg)
			gameState = 5
			wmazeMain()
		case 5, 6, 23:
			wmazeMain()
		case 16:
			wpcmkMain()
		case 17, 22:
			wpcvwMain()
		case 15, 21:
			wtreaOvmain()
		case 10, 11, 14:
			wmeleOvmain()
		case 13:
			wmexeOvmain()
		case 18:
			wmnpcOvmain()
		case 12:
			wpopsOvmain()
		case 19, 20:
			wdoptOvmain()
		case 3:
			shutdownGame(0)
			return 0
		default:
			overlayMissing()
		}
		stopSound()
	}
}

// The overlays not ported yet.
var (
	wmazeMain = overlayMissing
	wpcvwMain = overlayMissing
)

func overlayMissing() {
	fmt.Fprintf(os.Stderr, "wiz6: game_state %d: overlay not ported yet\n", gameState)
	for !keyAvailable(0) && !checkScreenClick() {
	}
	exitGame(0)
}

// openData reads a data file (make_data_path and open); fatal error code if it is missing.
func openData(name string, code int) []byte {
	b, err := os.ReadFile(dbs.Path(dataDir, name))
	if err != nil {
		exitGame(code)
	}
	return b
}

// ensureDiskFor (ROOT 1124) asks for the floppy an action needs. The port runs from one
// directory, the hard disk layout (g3592 1), where no disk is ever asked for.
func ensureDiskFor(kind, arg int) {}

// promptInsertDisk (ROOT 0d26) asks for a floppy: never needed from one directory.
func promptInsertDisk(label string, disk int, closeFiles, reopen bool) {}

// accessScenarioRecord (ROOT 0882) reads (write false) or writes record index of type typ.
func accessScenarioRecord(typ, index int, buf []byte, write bool) {
	stopSound()
	if write {
		if err := scen.WriteRecord(typ, index, buf); err != nil {
			exitGame(9)
		}
		return
	}
	b, err := scen.Record(typ, index)
	if err != nil {
		exitGame(9)
	}
	copy(buf, b)
}

// getMessage (ROOT 075b) is message msg of MSG.DBS, decoded; msgNotFound says it is missing.
func getMessage(msg int) string {
	b := msgs.Get(msg)
	msgNotFound = b == nil
	return string(b)
}

// printMessage (ROOT 083f) prints message msg into win; colours from 16 on are colour font
// color-15.
func printMessage(msg int, win *Window, color int) {
	printColored(win, getMessage(msg), color)
}

func printColored(win *Window, s string, color int) {
	if color >= 16 {
		printStringHi(win, s, color-15)
	} else {
		printString(win, s, color)
	}
}

// putChar (ROOT 22b7, asm) puts ch at win's cursor in font 0 colour color and moves the
// cursor on, wrapping at the right edge and the bottom. A negative colour draws inverted and
// leaves the cell dirty, so the next refresh draws it plain again.
func putChar(win *Window, ch byte, color int) {
	inverse := color < 0
	if inverse {
		color = -color
	}
	putCell(win, Cell{ch, uint8(color) << 4}, func(col, row int) { drv.Glyph(col, row, ch, uint8(color), inverse) }, inverse)
}

// putCharHi (ROOT 23e3, asm) puts ch at win's cursor in colour font font (1..4).
func putCharHi(win *Window, ch byte, font int) {
	putCell(win, Cell{ch, uint8(font)}, func(col, row int) { drv.GlyphHi(col, row, ch, font) }, false)
}

func putCell(win *Window, c Cell, draw func(col, row int), mark bool) {
	if id := windowID(win); id != 0 {
		if i := int(win.CurY)*int(win.W) + int(win.CurX); i < len(win.Cells) {
			win.Cells[i] = c
		}
		if win.Flags&WinDeferred == 0 {
			col, row := int(win.X)+int(win.CurX), int(win.Y)+int(win.CurY)
			if m := row*drv.Cols + col; m >= 0 && m < len(cellOwner) && cellOwner[m]&0x3f == id {
				cellOwner[m] = id
				if mark {
					cellOwner[m] |= cellDirty
				}
				draw(col, row)
				drv.Present()
			}
		}
	}
	win.CurX++
	if win.CurX >= win.W {
		win.CurX = 0
		win.CurY++
		if win.CurY >= win.H {
			win.CurY = 0
		}
	}
}

// printString (ROOT 24e9, asm) puts each byte of s with putChar.
func printString(win *Window, s string, color int) {
	for i := 0; i < len(s); i++ {
		putChar(win, s[i], color)
	}
}

// printStringHi (ROOT 251d, asm) puts each byte of s with putCharHi.
func printStringHi(win *Window, s string, font int) {
	for i := 0; i < len(s); i++ {
		putCharHi(win, s[i], font)
	}
}

// printLong (ROOT 0507) prints value right-aligned in width columns; a longer number shows
// its last width digits.
func printLong(win *Window, value int32, width, color int) {
	s := strconv.FormatInt(int64(value), 10)
	for ; len(s) < width; width-- {
		if color >= 16 {
			putCharHi(win, ' ', color-15)
		} else {
			putChar(win, ' ', color)
		}
	}
	printColored(win, s[len(s)-width:], color)
}

// printInt (ROOT 05a5) is printLong for an int.
func printInt(win *Window, value, width, color int) {
	printLong(win, int32(int16(value)), width, color)
}

// printCentered (ROOT 0b7e) prints s centred on row of a 40-column window.
func printCentered(win *Window, s string, row, color int) {
	setCursor(win, 20-(len(s)+1)/2, row)
	printColored(win, s, color)
}

// waitSecondsOrEnter (ROOT 09a3) waits up to seconds, less if Enter is pressed.
func waitSecondsOrEnter(seconds int) {
	for n := seconds * 10; n > 0; n-- {
		if keyAvailable(1) && getKey(1) == 13 {
			return
		}
		delayTick()
	}
}

// The click regions (cs:3598): x1, y1, x2, y2 in cells; x1 255 ends the table.
var clickRegions [4 * 40]byte

// setClickRegion (ROOT 0c16) sets click region i.
func setClickRegion(i int, x1, y1, x2, y2 byte) {
	copy(clickRegions[4*i:], []byte{x1, y1, x2, y2})
}

// clearClickFlag (ROOT 0bd1).
func clearClickFlag() { clickFlag = false }

// checkClickRegions (ROOT 0bd8) is the 1-based click region under the pointer, 0 for none;
// clickFlag says whether it was clicked.
func checkClickRegions(regions []byte) int {
	clearClickFlag()
	if !mousePresent {
		return 0
	}
	r := drv.PollClick(regions)
	clickFlag = r < 0
	if r < 0 {
		return -r
	}
	return r
}

// checkScreenClick (ROOT 0c42) is whether the mouse was clicked anywhere.
func checkScreenClick() bool {
	if !mousePresent {
		return false
	}
	setClickRegion(0, 0, 0, 39, 24)
	setClickRegion(1, 255, 0, 0, 0)
	checkClickRegions(clickRegions[:])
	return clickFlag
}

// le16 is the little-endian word at b[o:].
func le16(b []byte, o int) int { return int(binary.LittleEndian.Uint16(b[o:])) }
