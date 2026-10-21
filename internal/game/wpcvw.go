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
	if gameState == stateViewCharacter {
		viewCharacter(int(curMember))
	}
	if gameState == stateLevelUp {
		levelUpEntry()
	}
}

// WPCVW bss (0x553c..0x583a).
var (
	pvBlinkState   = -1              // g553c: -1 no blinking cursor, else which of pvBlinkText shows next
	pvItemNames    [packSlots]string // 0x55de: the inventory's item names
	pvItemRowColor [packSlots]int    // item_row_color: the colour each inventory row is printed in
	pvSwagNames    [swagSlots]string // 0x5692: the swag bag's item names
	pvCardWin      *Window           // g5752: the member card window on the right
	pvBlinkWin     *Window           // g5754: where the blinking cursor is
	pvBlinkX       int               // g5756
	pvBlinkY       int               // g5758
	pvBlinkText    [2]string         // 0x575a, 0x5772: the two blink phases
	pvBlinkColor   [2]int            // g578a, g578c
	pvSpellTarget  int               // g578e: the member a spell is cast on
	pvNamesLoaded  bool              // g57fc: pvItemNames holds the shown member's names
	pvSkillShown   [skillCount]bool  // g57fe..g583a: the skills the skills page lists
	pvClassAllowed [classCount]bool  // class_allowed (0x57e0)
	pvChosenItem   int               // g43ca (root): the item the last choose_item picked
)

// WPCVW data segment strings (0x4fee..0x549e).
const (
	wpcvwBlankLabel = "               " // 0x4fee
	wpcvwNoPoints   = "***"             // 0x4ffe, 0x5002: shown for a cost the member cannot pay
	wpcvwACOpen     = " ("              // 0x5192
)

// Character numbers WPCMK and WPCVW share.
const (
	// Equip positions (messages 351+): PC.Equipped and PC.AC indexes; the two hands are
	// also PC.Hands indexes.
	equipPrimary   = 0
	equipSecondary = 1
	equipMisc      = 2 // its AC applies to the whole body
	equipHelmet    = 3
	equipBody      = 4
	equipLegs      = 5
	equipGauntlets = 6
	equipBoots     = 7
	equipSlots     = 8
	acBase         = 0 // PC.AC[0]: the bare armour class, AC[1] what the hands' items add
	acHands        = 1

	// Item.Flags bits besides itemEquipped (wmnpc.go).
	itemCursed    = 2  // cannot be dropped; spell 73 (REMOVE CURSE) destroys it
	itemOffHand   = 4  // may go in the secondary hand (record bit recItemOffHand)
	itemTwoHanded = 8  // a weapon that takes both hands (record bit recItemTwoHanded)
	itemKept      = 64 // an important item (record bit itemImportant): cannot be dropped
	// ItemRec.Flags bits besides itemImportant (wmnpc.go).
	recItemTwoHanded = 4
	recItemOffHand   = 8

	// Item types (ItemRec.Type, messages 1550+) that are tested one by one.
	itemTypeWeaponS   = 0 // WEAPON (S)
	itemTypeWeaponE   = 1 // WEAPON (E)
	itemTypeWeaponT   = 2 // WEAPON (T): thrown, stackable
	itemTypeLauncher  = 3 // WEAPON (L): bows and slings
	itemTypeMissile   = 4 // goes with its launcher
	itemTypeMisc      = 5 // the first of the armour types, misc. item .. boots
	itemTypeHelmet    = 6
	itemTypeBodyArmor = 7
	itemTypeLegArmor  = 8
	itemTypeGauntlets = 9
	itemTypeBoots     = 10
	itemTypeShield    = 11
	itemTypePotion    = 12
	itemTypeScroll    = 13
	itemTypePowder    = 16

	// Hand.Flags attack kinds (bit b is message 3050+b) and the Hand.HitEffects
	// powers (message 470+i; MonsterAttack.HitEffects too, [8] its poison strength).
	handPunch    = 32
	handKick     = 64
	handSleep    = 0
	handParalyze = 1
	handPoison   = 2
	handStone    = 3
	handShred    = 4
	handDrain    = 5
	handCritical = 6
	handKnockout = 7
	hitPoisonStr = 8

	// Spell books: PC.SpellsLearn indexes.
	bookMage      = 0
	bookPriest    = 1
	bookAlchemist = 2
	bookPsionic   = 3
	bookCount     = 4

	maxPower          = 6   // the highest power level of a spell
	realmCount        = 6   // fire, water, air, earth, mental, magic (messages 3950+)
	spellTargetMember = 1   // Spell.Target: cast on a chosen member
	spellTargetGroup  = 2   // Spell.Target: cast on a chosen monster group
	unlimitedSP       = 999 // a realm's spell points while a spell to learn is picked

	// Skills: 0..9 weaponry, 10..16 physical, 17..21 personal, 22..29 academia (messages
	// 5500+), shown a page each (messages 600+).
	firstPhysicalSkill = 10
	firstPersonalSkill = 17
	firstAcademiaSkill = 22
	skillPageWeaponry  = 0
	skillPagePersonal  = 2
	skillPageCount     = 4 // also the skills menu's exit
	maxSkill           = 100

	swagSlots     = itemSlots - packSlots // Items[packSlots..] the swag bag
	portraitCount = 42
	nameLength    = 7 // the letters of a PC.Name
	maxResist     = 125
	daysPerYear   = 365
	highStat      = 15 // a stat above this gives a bonus
	veryHighStat  = 17 // and above this another
	lowStat       = 8  // a stat below this gives a penalty
)

var (
	spellBookBits   = [bookCount]uint8{8, 4, 1, 2} // the Spell.Books bit of each book
	skillGroupStart = [4]int{0, firstPhysicalSkill, firstPersonalSkill, firstAcademiaSkill}
)

// The stat tables are a letter per stat, 'A' being 0: STR INT PIE VIT DEX SPD PER.
var (
	// classReqs (0x5006, and WPCMK's copy): the stats each class needs. The equal table at
	// 0x5084 is the floor a new class raises the stats to; it is this one too.
	classReqs = [classCount]string{"MAAAAAAA", "AMAAAAAA", "AAMAAAIA", "AAAAMIAA", "KIILKIIA", "ANAANAAA", "AKAAMIMA",
		"KOAOAAKA", "KALLKLIA", "APPAAAIA", "MJMMJJOA", "MLAJMOIA", "NINAKNIA", "MKKMMMAA"}
	// raceMinStats (0x5102): the race's starting stats, the floor after a class change.
	raceMinStats = [raceCount]string{"JIIJJIIA", "HKKHJJIA", "LGKMHHHA", "KHNKIGGA", "IHGJKHNA", "FLGGKOMA",
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
	// classSkills (0x52c1): per class, which weaponry, physical, personal and academia
	// skills the skills pages always list ('1').
	classSkills = [classCount][4]string{
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
func endRegions(i int) { setClickRegion(i, clickRegionsEnd, 0, 0, 0) }

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
		if keyAvailable(keyClassMenu) {
			if k := strings.IndexByte(menuKeys, byte(getKey(keyClassMenu))); k >= 0 {
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

// waitKeyOrClick (WPCVW 8d3a) shows message 409 in w and waits for Enter or a click.
func waitKeyOrClick(w *Window) {
	fillWindow(w, ' ', 3)
	refreshWindow(w)
	printCentered(w, getMessage(409), 2, 18)
	waitEnterOrClick()
	fillWindow(w, ' ', 3)
	refreshWindow(w)
}

// waitEnterOrClick waits for Enter or a mouse click (the loops of WPCVW 8d3a and b6c4).
func waitEnterOrClick() {
	for wait := true; wait; {
		if checkScreenClick() {
			wait = false
		}
		if keyAvailable(keyClassMenu) && int8(getKey(keyClassMenu)) == keyEnter {
			wait = false
		}
	}
}

// showCenteredMsg (WPCVW e8b6) shows s centred on row 1 of w, beeps if beep, and waits.
func showCenteredMsg(w *Window, s string, beep bool) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	fillWindow(w, ' ', 3)
	refreshWindow(w)
	setCursor(w, 20-(len(s)+1)/2, 1)
	printStringHi(w, s, 3)
	if beep {
		playSound(soundBeep)
	}
	waitSecondsOrEnter(messageDelay)
}

// inputNumber (WPCVW 95c6) asks with message msg for a number of up to max digits; 0 for
// none.
func inputNumber(w *Window, msg, max int) int32 {
	fillWindow(w, ' ', 3)
	deferWindowDraw(w)
	setCursor(w, 2, 1)
	printMessage(msg, w, 18)
	x, y := int(w.CurX), int(w.CurY)
	printString(w, strings.Repeat(" ", max+1), 1)
	resumeWindowDraw(w)
	refreshAllWindows()
	setCursor(w, x, y)
	s := inputText(w, max, 5, inputDigits)
	n, _ := strconv.ParseInt(s, 10, 32) // atol: digits only, at most 8
	return int32(n)
}

// drawTextFrame (WPCVW 8dd4) draws the text window's frame.
func drawTextFrame(w *Window) {
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

// showTextPages (WPCVW 8ea4) shows messages msg, msg+1, ... until one is missing, a line
// each, centred unless it starts with '$' (left) or '^' (under the previous); '!' skips a
// line and '@' starts a new page. menu shows the page prompt.
func showTextPages(menu *Window, msg int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	w := createWindow(0, 4, 40, 16, 31, borderNone, 0, false)
	drawTextFrame(w)
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
			waitKeyOrClick(menu)
			drawTextFrame(w)
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
	waitKeyOrClick(menu)
	destroyWindow(w, false)
}

// --------------------------------------------------------------------- printing

// printAt (WPCVW 50ba).
func printAt(w *Window, x, y int, s string, color int) {
	setCursor(w, x, y)
	printString(w, s, color)
}

// putGlyphAtC1 (WPCVW 9f7f) puts glyph ch of font 1 at (x, y).
func putGlyphAtC1(w *Window, x, y int, ch byte) {
	setCursor(w, x, y)
	putCharHi(w, ch, 1)
}

// printNumberCol16 (WPCVW de9f) prints n in column 16 of row y.
func printNumberCol16(w *Window, y, n int) {
	setCursor(w, 16, y)
	printInt(w, n, 3, 1)
}
