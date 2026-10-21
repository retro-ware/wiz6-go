package game

import (
	"strconv"
	"strings"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// WPCVW: the character sheet (game state 17: stats, inventory, equip, spells, trade,
// swag, skills, modify) and the level up after combat (game state 22).

// ovmain (WPCVW 4572).
func wpcvwOvmain() {
	pvBlinkState = -1
	if gameState == 17 {
		wpcvwViewCharacter(int(curMember))
	}
	if gameState == 22 {
		levelUpEntry()
	}
}

// WPCVW bss (0x553c..0x583a).
var (
	pvBlinkState   = -1       // g553c: -1 no blinking cursor, else which of pvBlinkText shows next
	pvItemNames    [10]string // 0x55de: the inventory's item names
	pvItemRowColor [10]int    // item_row_color: the colour each inventory row is printed in
	pvSwagNames    [12]string // 0x5692: the swag bag's item names
	pvCardWin      *Window    // g5752: the member card window on the right
	pvBlinkWin     *Window    // g5754: where the blinking cursor is
	pvBlinkX       int        // g5756
	pvBlinkY       int        // g5758
	pvBlinkText    [2]string  // 0x575a, 0x5772: the two blink phases
	pvBlinkColor   [2]int     // g578a, g578c
	pvSpellTarget  int        // g578e: the member a spell is cast on
	pvNamesLoaded  bool       // g57fc: pvItemNames holds the shown member's names
	pvSkillShown   [30]bool   // g57fe..g583a: the skills the skills page lists
	pvClassAllowed [14]bool   // class_allowed (0x57e0)
	pvChosenItem   int        // g43ca (root): the item the last choose_item picked
)

// WPCVW data segment strings (0x4fee..0x549e).
const (
	wpcvwBlankLabel = "               " // 0x4fee
	wpcvwNoPoints   = "***"             // 0x4ffe, 0x5002: shown for a cost the member cannot pay
	wpcvwACOpen     = " ("              // 0x5192
)

// The stat tables are a letter per stat, 'A' being 0: STR INT PIE VIT DEX SPD PER.
var (
	// classReqs (0x5006, and WPCMK's copy): the stats each class needs. The equal table at
	// 0x5084 is the floor a new class raises the stats to; it is this one too.
	classReqs = [14]string{"MAAAAAAA", "AMAAAAAA", "AAMAAAIA", "AAAAMIAA", "KIILKIIA", "ANAANAAA", "AKAAMIMA",
		"KOAOAAKA", "KALLKLIA", "APPAAAIA", "MJMMJJOA", "MLAJMOIA", "NINAKNIA", "MKKMMMAA"}
	// raceMinStats (0x5102): the race's starting stats, the floor after a class change.
	raceMinStats = [11]string{"JIIJJIIA", "HKKHJJIA", "LGKMHHHA", "KHNKIGGA", "IHGJKHNA", "FLGGKOMA",
		"MFFOIKDA", "KHGMKIGA", "HKHHKMKA", "IGMKIIKA", "KKGKHHJA"}
	// resistTables (0x5195..0x52c0): per class group, the base resistances (letter, 'A' 0,
	// times 4) and their growth per level (digit).
	resistTables = [10][2]string{
		{"ACBCBCCCCHCHBA", "02222232112230"}, {"ACCCCDCCCHHKDA", "02222232123230"},
		{"ACCCCDCCCIFKEA", "02222232121230"}, {"ACHCHDCCCHFIDA", "02323232111230"},
		{"ACKCHHGCCIECBA", "02322322121220"}, {"ACBCCDECCEEEGA", "02121212111330"},
		{"ACICIBHCCFKFFA", "02323112113330"}, {"ACECCEGCCIFKFA", "02222222122220"},
		{"ACKCGDHCCFKIDA", "02322212112320"}, {"ACHCHHGCCCECHA", "02322322112230"},
	}
	// classSkills (0x52c1): per class, which weapon, physical, academic and personal skills
	// the skills pages always list ('1').
	classSkills = [14][4]string{
		{"1111111110", "0100000", "00000", "11100000"}, {"1000111100", "0101000", "00000", "11100010"},
		{"0001101010", "0101000", "00000", "11101000"}, {"1111111110", "0101111", "00000", "11100000"},
		{"1111111110", "0100111", "00000", "11110000"}, {"1001110000", "0100000", "00000", "11110000"},
		{"1111111110", "0111111", "00000", "11100010"}, {"1001111000", "0101000", "00000", "11100100"},
		{"1111111110", "0101000", "00000", "11101000"}, {"0001101010", "0101000", "00000", "11101010"},
		{"1111111110", "0101000", "00000", "11101000"}, {"1111111110", "0101000", "00000", "11100011"},
		{"1001111101", "0101001", "00000", "11100101"}, {"1111111111", "0100111", "00000", "11110001"},
	}
)

// ---------------------------------------------------------------- root helpers

// setRegion is set_click_region with int arguments.
func setRegion(i, x1, y1, x2, y2 int) {
	setClickRegion(i, byte(x1), byte(y1), byte(x2), byte(y2))
}

// endRegions ends the click region table at i.
func endRegions(i int) { setClickRegion(i, 255, 0, 0, 0) }

// copyThroughChar (ROOT 0c82) is s up to and including the first ch.
func copyThroughChar(s string, ch byte) string {
	if i := strings.IndexByte(s, ch); i >= 0 {
		return s[:i+1]
	}
	return s
}

// itemRec reads item record id (access_scenario_record type 1).
func itemRec(id int) ItemRec {
	b := make([]byte, dbs.Size[ItemRec]())
	accessScenarioRecord(dbs.RecItem, id, b, false)
	rec, _ := dbs.Decode[ItemRec](b)
	return rec
}

// mask16 is a 16-bit mask as the byte pair test_bit reads.
func mask16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }

// staminaPct is the stamina percentage the sheet shows (16-bit, as the C computes it),
// at least 1 while there is any stamina.
func staminaPct(p *PC) int {
	n := int(int16(int(p.Stamina)*100)) / int(p.StaminaMax)
	if n < 1 && p.Stamina > 0 {
		n = 1
	}
	return n
}

// --------------------------------------------------------------- menus and input

// blinkCursor (WPCVW 8817) prints the next phase of the blinking cursor.
func blinkCursor() {
	setCursor(pvBlinkWin, pvBlinkX, pvBlinkY)
	printColored(pvBlinkWin, pvBlinkText[pvBlinkState], pvBlinkColor[pvBlinkState])
	pvBlinkState = (pvBlinkState + 1) % 2
}

// getMenuInputBlink (get_menu_input, WPCVW 8891) waits for a key of menuKeys (its index)
// or a click on a click region (its index, with menuClicked set), blinking the cursor
// meanwhile.
func getMenuInputBlink() int {
	idle := 0
	phase := animPhase
	if pvBlinkState != -1 {
		blinkCursor()
	}
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if k := strings.IndexByte(menuKeys, byte(getKey(1))); k >= 0 {
				return k
			}
			continue
		}
		r := checkClickRegions(clickRegions[:])
		if clickFlag {
			menuClicked = 1
			return r - 1
		}
		if idle == 0 {
			animPhase = (animPhase + 1) % 2
			if pvBlinkState != -1 && animPhase == phase {
				blinkCursor()
			}
		} else {
			delayTick()
			delayTick()
		}
		idle = (idle + 1) % 10
	}
}

// wpcvwWaitKeyOrClick (WPCVW 8d3a) shows message 409 in w and waits for Enter or a click.
func wpcvwWaitKeyOrClick(w *Window) {
	fillWindow(w, ' ', 3)
	refreshWindow(w)
	printCentered(w, getMessage(409), 2, 18)
	wpcvwWaitEnterOrClick()
	fillWindow(w, ' ', 3)
	refreshWindow(w)
}

// wpcvwWaitEnterOrClick waits for Enter or a mouse click (the loops of WPCVW 8d3a and b6c4).
func wpcvwWaitEnterOrClick() {
	for wait := true; wait; {
		if checkScreenClick() {
			wait = false
		}
		if keyAvailable(1) && int8(getKey(1)) == 13 {
			wait = false
		}
	}
}

// wpcvwShowCenteredMsg (WPCVW e8b6) shows s centred on row 1 of w, beeps if beep, and waits.
func wpcvwShowCenteredMsg(w *Window, s string, beep bool) {
	fillWindow(w, ' ', 3)
	refreshWindow(w)
	setCursor(w, 20-(len(s)+1)/2, 1)
	printStringHi(w, s, 3)
	if beep {
		playSound(0)
	}
	waitSecondsOrEnter(messageDelay)
}

// wpcvwInputLine (WPCVW 9459) reads up to max characters at w's cursor in colour color (only
// digits with digits) until Enter.
func wpcvwInputLine(w *Window, max, color int, digits bool) string {
	x, y := int(w.CurX), int(w.CurY)
	var buf []byte
	putChar(w, 'a', 1)
	for {
		k := int(uint8(getKey(1)))
		if k == 13 {
			break
		}
		setCursor(w, x, y)
		putChar(w, ' ', 1)
		setCursor(w, x, y)
		switch {
		case k == 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				x--
				setCursor(w, x, y)
				putChar(w, ' ', 1)
				setCursor(w, x, y)
			} else {
				playSound(0)
			}
		case len(buf) >= max:
			playSound(0)
		case !digits && k >= 32 && k <= 127 || digits && k >= '0' && k <= '9':
			buf = append(buf, byte(k))
			putChar(w, byte(k), color)
			x++
		}
		putChar(w, 'a', 1)
	}
	setCursor(w, x, y)
	putChar(w, ' ', 1)
	return string(buf)
}

// wpcvwInputNumber (WPCVW 95c6) asks with message msg for a number of up to max digits; 0 for
// none.
func wpcvwInputNumber(w *Window, msg, max int) int32 {
	fillWindow(w, ' ', 3)
	deferWindowDraw(w)
	setCursor(w, 2, 1)
	printMessage(msg, w, 18)
	x, y := int(w.CurX), int(w.CurY)
	printString(w, strings.Repeat(" ", max+1), 1)
	resumeWindowDraw(w)
	refreshAllWindows()
	setCursor(w, x, y)
	s := wpcvwInputLine(w, max, 5, true)
	n, _ := strconv.ParseInt(s, 10, 32) // atol: digits only, at most 8
	return int32(n)
}

// wpcvwDrawTextFrame (WPCVW 8dd4) draws the text window's frame.
func wpcvwDrawTextFrame(w *Window) {
	fillWindow(w, 0, 1)
	setFrameGlyph(w, 0, 0, 9)
	setFrameGlyph(w, 39, 0, 10)
	setFrameGlyph(w, 0, 15, 6)
	setFrameGlyph(w, 39, 15, 8)
	for i := 1; i < 39; i++ {
		setFrameGlyph(w, i, 0, 12)
		setFrameGlyph(w, i, 15, 7)
	}
	for i := 1; i < 15; i++ {
		setFrameGlyph(w, 0, i, 4)
		setFrameGlyph(w, 39, i, 5)
	}
	refreshWindow(w)
}

// wpcvwShowTextPages (WPCVW 8ea4) shows messages msg, msg+1, ... until one is missing, a line
// each, centred unless it starts with '$' (left) or '^' (under the previous); '!' skips a
// line and '@' starts a new page. menu shows the page prompt.
func wpcvwShowTextPages(menu *Window, msg int) {
	w := createWindow(0, 4, 40, 16, 31, -1, 0, false)
	wpcvwDrawTextFrame(w)
	row, prevX := 1, 1
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		if strings.HasPrefix(s, "!") {
			s = s[1:]
			row++
		}
		if strings.HasPrefix(s, "@") {
			s = s[1:]
			row = 13
		}
		if row == 13 {
			wpcvwWaitKeyOrClick(menu)
			wpcvwDrawTextFrame(w)
			row = 1
		}
		row++
		x := 20 - (len(s)+1)/2
		if strings.HasPrefix(s, "$") {
			s = s[1:]
			x = 1
		} else if strings.HasPrefix(s, "^") {
			s = s[1:]
			x = prevX
		}
		setCursor(w, x, row)
		printString(w, s, textColor)
		prevX = x
		msg++
	}
	wpcvwWaitKeyOrClick(menu)
	destroyWindow(w, false)
}

// --------------------------------------------------------------------- printing

// wpcvwPrintAt (WPCVW 50ba).
func wpcvwPrintAt(w *Window, x, y int, s string, color int) {
	setCursor(w, x, y)
	printString(w, s, color)
}

// wpcvwPrintMsg (WPCVW 5115, and b195) prints message msg in font 0.
func wpcvwPrintMsg(msg int, w *Window, color int) { printString(w, getMessage(msg), color) }

// putGlyphAtC1 (WPCVW 9f7f) puts glyph ch of font 1 at (x, y).
func putGlyphAtC1(w *Window, x, y int, ch byte) {
	setCursor(w, x, y)
	putCharHi(w, ch, 1)
}

// wpcvwPrintNumberCol16 (WPCVW de9f) prints n in column 16 of row y.
func wpcvwPrintNumberCol16(w *Window, y, n int) {
	setCursor(w, 16, y)
	printInt(w, n, 3, 1)
}
