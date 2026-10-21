package game

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// Random numbers: three linear congruential generators summed, Wichmann-Hill style (ROOT
// 25b9, 2556; seeds cs:1d3b, 1d3d, 1d3f).
var seeds = [3]int{3000, 1, 29999}

// randomGenerators are the generators: s = multiplier*s mod modulus, computed without overflow as
// multiplier*(s%quotient) - remainder*(s/quotient) (Schrage's method).
var randomGenerators = [3]struct{ multiplier, quotient, remainder, modulus int }{
	{171, 177, 2, 30269},
	{172, 176, 35, 30307},
	{170, 178, 63, 30323},
}

// randomMax is the largest number randomWord gives.
const randomMax = math.MaxInt16

// seedRandom (ROOT, asm) seeds the second generator from the BIOS tick count.
func seedRandom() {
	if DetRNG {
		seeds[1] = 2 // a BIOS tick count of 0
		return
	}
	seeds[1] = int(host.Now()/biosTick+2) & randomMax
}

// DetRNG (testing, -detrng) makes the random numbers independent of timing: a fixed seed and
// no step on each key poll. tools/parity patches WROOT.EXE the same way.
var DetRNG bool

// stepRandom (ROOT 25b9, asm) moves each generator on one step.
func stepRandom() {
	for i, g := range randomGenerators {
		s := int(int16(seeds[i]))
		s = g.multiplier*(s%g.quotient) - g.remainder*(s/g.quotient)
		if s < 0 {
			s += g.modulus
		}
		seeds[i] = int(int16(s))
	}
}

// randomWord (ROOT 2556, asm) is a number 0..randomMax.
func randomWord() int {
	stepRandom()
	sum := 0
	for i, g := range randomGenerators {
		s := int(uint16(seeds[i]))
		sum = (sum + (s%g.modulus)*10000/g.modulus&randomMax) & randomMax
	}
	return sum
}

// random (ROOT 09e2) is a number 0..n-1, or 0 for n <= 0.
func random(n int) int {
	if os.Getenv("WIZ6_RNGLOG") != "" {
		pc, _, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "rng %d %s:%d\n", n, runtime.FuncForPC(pc).Name(), line)
	}
	n = int(int16(n)) // the C's int argument
	r := 0
	if n > 0 {
		r = randomWord() % n
	}
	if dtLog != nil {
		dtLog("rng", n, r)
	}
	return r
}

// setBit (ROOT 2881) sets (v true) or clears bit n of a.
func setBit(a []byte, n int, v bool) {
	if v {
		a[n/8] |= 1 << (n % 8)
	} else {
		a[n/8] &^= 1 << (n % 8)
	}
}

// testBit (ROOT 28af) is bit n of a.
func testBit(a []byte, n int) bool { return a[n/8]&(1<<(n%8)) != 0 }

// setBitfield (ROOT 28da) stores the width-bit value val at index idx of a packed array.
func setBitfield(a []byte, idx, width, val int) {
	pos := idx * width
	o, sh := pos>>3, pos&7
	m := (1<<width - 1) << sh
	w := int(a[o])
	if o+1 < len(a) {
		w |= int(a[o+1]) << 8
	}
	w = w&^m | (val<<sh)&m
	a[o] = byte(w)
	if o+1 < len(a) {
		a[o+1] = byte(w >> 8)
	}
}

// getBitfield (ROOT 2925) is the width-bit value at index idx of a packed array.
func getBitfield(a []byte, idx, width int) int {
	pos := idx * width
	o := pos >> 3
	w := int(a[o])
	if o+1 < len(a) {
		w |= int(a[o+1]) << 8
	}
	return w >> (pos & 7) & (1<<width - 1)
}

// extractBits (ROOT 2962) is width bits of the 16-bit word v from bit shift on.
func extractBits(v, shift, width int) int { return int(uint16(v)) >> shift & (1<<width - 1) }

// Delays. calibrate_delay counts the busy loops of one BIOS tick (54.9 ms); delay_tick
// runs 2/132 of a tick's worth and delay_long two ticks less a delay_tick.
const (
	biosTick  = 54925 * time.Microsecond
	tickDelay = 2 * biosTick / 132
	longDelay = 2*biosTick - tickDelay
)

var delayUntil time.Duration // when the current run of delays ends

// calibrateDelay (ROOT, asm) starts the delay clock.
func calibrateDelay() { delayUntil = host.Now() }

func delay(d time.Duration) {
	if dtLog != nil { // no wall-clock waits in the difftests
		return
	}
	now := host.Now()
	if delayUntil < now-10*time.Millisecond {
		delayUntil = now // a delay after other work starts afresh
	}
	delayUntil += d
	for host.Now() < delayUntil {
		host.Sleep()
	}
}

// delayTick (ROOT 2858, asm) waits about 0.83 ms.
func delayTick() { delay(tickDelay) }

// delayLong (ROOT 282f, asm) waits about 109 ms.
func delayLong() { delay(longDelay) }

// The keyboard: keys come in as PC scan codes and are kept, as the BIOS would give them
// (ASCII or 0 for an extended key, scan code), in a keyBufSize-key buffer (cs:262b).
const keyBufSize = 20

var (
	keyBuf []key
	shift  bool
)

type key struct{ ascii, scan byte }

// PC set-1 scan codes the keyboard code looks at (hardware values).
const (
	scanExtendedPrefix = 0xe0 // comes before the code of a grey (extended) key
	scanBreakBit       = 0x80 // set in the code of a key release
	scanLeftShift      = 0x2a
	scanRightShift     = 0x36
	scanEnter          = 0x1c // with the prefix: keypad Enter
	scanKeypadFirst    = 0x47 // keypad 7 (Home) ..
	scanKeypadLast     = 0x53 // .. keypad . (Del)
	scanLeft           = 0x4b
	scanUp             = 0x48
	scanRight          = 0x4d
	scanDown           = 0x50
)

// The keys get_key returns that are not plain characters.
const (
	keyLeft   = 8 // also Backspace
	keyUp     = 9
	keyRight  = 10
	keyDown   = 11
	keyEnter  = '\r'
	keyEscape = 27 // ASCII ESC
)

// Key classes: which keys keyAvailable and getKey take.
const (
	keyClassAny    = 0 // any key
	keyClassMenu   = 1 // the arrows, Escape, Enter and ' '..'_'
	keyClassDigits = 2 // digits, keyLeft, Escape and Enter
)

// set-1 scan code to unshifted and shifted ASCII.
const (
	scanToASCII        = "\x00\x1b1234567890-=\x08\tqwertyuiop[]\r\x00asdfghjkl;'`\x00\\zxcvbnm,./\x00*\x00 "
	scanToShiftedASCII = "\x00\x1b!@#$%^&*()_+\x08\tQWERTYUIOP{}\r\x00ASDFGHJKL:\"~\x00|ZXCVBNM<>?\x00*\x00 "
)

// pumpKeys turns the host's scan codes into BIOS keys.
func pumpKeys() {
	ext := false
	for {
		sc, ok := host.ReadKey()
		if !ok {
			return
		}
		if sc == scanExtendedPrefix {
			ext = true
			continue
		}
		up, code := sc&scanBreakBit != 0, sc&^scanBreakBit
		e := ext
		ext = false
		if code == scanLeftShift || code == scanRightShift {
			shift = !up
			continue
		}
		if up {
			continue
		}
		k := key{scan: code}
		if !e && int(code) < len(scanToASCII) {
			k.ascii = scanToASCII[code]
			if shift {
				k.ascii = scanToShiftedASCII[code]
			}
		}
		if e && code == scanEnter {
			k.ascii = keyEnter
		}
		if k.ascii == 0 && (code < scanKeypadFirst || code > scanKeypadLast) && !e {
			continue // no BIOS key for it here (shift, ctrl, alt, function keys)
		}
		if len(keyBuf) < keyBufSize {
			keyBuf = append(keyBuf, k)
		}
	}
}

// mapKey is what get_key makes of a BIOS key: lower case to upper, the arrows to
// keyLeft..keyDown, and -1 for other extended keys.
func mapKey(k key) int {
	if k.ascii == 0 {
		switch k.scan {
		case scanLeft:
			return keyLeft
		case scanUp:
			return keyUp
		case scanRight:
			return keyRight
		case scanDown:
			return keyDown
		}
		return -1
	}
	c := int(k.ascii)
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return c
}

// classTakesKey is whether key class class takes c (see keyClassAny..keyClassDigits).
func classTakesKey(class, c int) bool {
	switch {
	case c < 0:
		return false
	case class == keyClassAny:
		return true
	case c == keyEscape || c == keyEnter:
		return true
	case class == keyClassDigits:
		return c == keyLeft || c >= '0' && c <= '9'
	}
	return c >= keyLeft && c <= keyDown || c >= ' ' && c <= '_'
}

// keyAvailable (ROOT 2643, asm) is whether a key of class is waiting; keys the class does
// not take are thrown away. It steps the random numbers.
func keyAvailable(class int) bool {
	if dtLog != nil { // the difftest stubs it: no step_random, the scripted keys
		if len(dtKeys) == 0 {
			if dtIdle++; dtIdle > 1<<14 {
				panic("difftest: waits for input that never comes")
			}
			return false
		}
		return true
	}
	if !DetRNG {
		stepRandom()
	}
	pumpKeys()
	for len(keyBuf) > 0 && !classTakesKey(class, mapKey(keyBuf[0])) {
		keyBuf = keyBuf[1:]
	}
	if len(keyBuf) == 0 {
		drv.Present() // the C draws straight to the screen: show what was drawn before waiting
		host.Sleep()
		return false
	}
	return true
}

// getKey (ROOT 2724, asm) waits for a key of class and returns it.
func getKey(class int) int {
	if dtLog != nil {
		if len(dtKeys) == 0 {
			panic("difftest: input exhausted")
		}
		k := dtKeys[0]
		dtKeys = dtKeys[1:]
		return k
	}
	for !keyAvailable(class) {
	}
	k := mapKey(keyBuf[0])
	keyBuf = keyBuf[1:]
	return k
}

// flushKeys (ROOT 280c, asm) throws away waiting keys and a latched click.
func flushKeys() {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	pumpKeys()
	keyBuf = keyBuf[:0]
	if mousePresent {
		drv.ClearClick()
	}
}
