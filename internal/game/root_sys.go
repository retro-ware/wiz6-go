package game

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/retro-ware/wiz6-go/internal/drv"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// Random numbers: three linear congruential generators summed (ROOT 25b9, 2556; seeds
// cs:1d3b, 1d3d, 1d3f).
var seeds = [3]int{3000, 1, 0x752f}

var lcg = [3]struct{ a, q, r, m int }{
	{171, 177, 2, 0x763d},
	{172, 176, 35, 0x7663},
	{170, 178, 63, 0x7673},
}

// seedRandom (ROOT, asm) seeds the second generator from the BIOS tick count.
func seedRandom() {
	if DetRNG {
		seeds[1] = 2 // a BIOS tick count of 0
		return
	}
	seeds[1] = int(host.Now()/(54925*time.Microsecond)+2) & 0x7fff
}

// DetRNG (testing, -detrng) makes the random numbers independent of timing: a fixed seed and
// no step on each key poll. tools/parity patches WROOT.EXE the same way.
var DetRNG bool

// stepRandom (ROOT 25b9, asm) moves each generator on one step.
func stepRandom() {
	for i, g := range lcg {
		s := int(int16(seeds[i]))
		s = g.a*(s%g.q) - g.r*(s/g.q)
		if s < 0 {
			s += g.m
		}
		seeds[i] = int(int16(s))
	}
}

// randomWord (ROOT 2556, asm) is a number 0..0x7fff.
func randomWord() int {
	stepRandom()
	sum := 0
	for i, g := range lcg {
		s := seeds[i] & 0xffff
		sum = (sum + (s%g.m)*10000/g.m&0x7fff) & 0x7fff
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
		a[n>>3] |= 1 << (n & 7)
	} else {
		a[n>>3] &^= 1 << (n & 7)
	}
}

// testBit (ROOT 28af) is bit n of a.
func testBit(a []byte, n int) bool { return a[n>>3]&(1<<(n&7)) != 0 }

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
// (ASCII or 0 for an extended key, scan code), in a 20-key buffer (cs:262b).
var (
	keyBuf []key
	shift  bool
)

type key struct{ ascii, scan byte }

// set-1 scan code to unshifted and shifted ASCII.
const (
	plain   = "\x00\x1b1234567890-=\x08\tqwertyuiop[]\r\x00asdfghjkl;'`\x00\\zxcvbnm,./\x00*\x00 "
	shifted = "\x00\x1b!@#$%^&*()_+\x08\tQWERTYUIOP{}\r\x00ASDFGHJKL:\"~\x00|ZXCVBNM<>?\x00*\x00 "
)

// pumpKeys turns the host's scan codes into BIOS keys.
func pumpKeys() {
	ext := false
	for {
		sc, ok := host.ReadKey()
		if !ok {
			return
		}
		if sc == 0xe0 {
			ext = true
			continue
		}
		up, code := sc&0x80 != 0, sc&0x7f
		e := ext
		ext = false
		if code == 0x2a || code == 0x36 {
			shift = !up
			continue
		}
		if up {
			continue
		}
		k := key{scan: code}
		if !e && int(code) < len(plain) {
			k.ascii = plain[code]
			if shift {
				k.ascii = shifted[code]
			}
		}
		if e && code == 0x1c {
			k.ascii = '\r'
		}
		if k.ascii == 0 && (code < 0x47 || code > 0x53) && !e {
			continue // no BIOS key for it here (shift, ctrl, alt, function keys)
		}
		if len(keyBuf) < 20 {
			keyBuf = append(keyBuf, k)
		}
	}
}

// mapKey is what get_key makes of a BIOS key: lower case to upper, the arrows to 8..11,
// and -1 for other extended keys.
func mapKey(k key) int {
	if k.ascii == 0 {
		switch k.scan {
		case 0x4b:
			return 8
		case 0x48:
			return 9
		case 0x4d:
			return 10
		case 0x50:
			return 11
		}
		return -1
	}
	c := int(k.ascii)
	if c >= 'a' && c <= 'z' {
		c &= 0xdf
	}
	return c
}

// accepts is whether key class class takes c: 0 any key, 2 digits, the left arrow (and
// backspace), Escape and Enter, otherwise the arrows, Escape, Enter and 0x20..0x5f.
func accepts(class, c int) bool {
	switch {
	case c < 0:
		return false
	case class == 0:
		return true
	case c == 0x1b || c == 0x0d:
		return true
	case class == 2:
		return c == 8 || c >= '0' && c <= '9'
	}
	return c >= 8 && c <= 11 || c >= 0x20 && c <= 0x5f
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
	for len(keyBuf) > 0 && !accepts(class, mapKey(keyBuf[0])) {
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
