package game

import (
	"strings"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// WMNPC: meeting an NPC (game state 18). The party talks to it, trades, casts or uses an
// item, pilfers, fights it or leaves; the NPC's event scripts decide what follows.
//
// The overlay's maze view copy (fd8d1..ff6be) is WMAZE's renderer, so the port calls
// drawMazeView. The spell window pieces (8822..8c28) are WPOPS's, the item helpers WDOPT's
// (remove_item, use_item_charge) and WPCVW's, draw_sprite (4c09) is WPOPS's and the title
// (53e5) is drawCombatTitle. The blinking cursor (g522a..g524b) is WPCVW's pvBlink* state:
// the two overlays never run together.

// ovmain (WMNPC 6a6d).
func wmnpcOvmain() {
	pvBlinkState = -1
	npcID = g433a
	runNPCEncounter()
}

// WMNPC data segment.
var (
	npcRec       NPCRec       // 0x5146: the NPC
	npcID        int          // g5274: its record
	npcFlags     [2]byte      // 0x5278: its flag bits, from scenario_hdr's per-NPC pair
	npcSound     int          // g5224: the NPC's sound, 0 none
	npcShowGold  int          // g527a: the member panels show gold, not names
	npcLeaves    int          // g5276: the NPC turns away at the end
	npcGoldGiven int32        // g52cc: gold given so far, for the gift scripts
	npcCharm     int16        // g52d0: added to the party's attitude (spell 50)
	npcTarget    int          // g5322: the member a spell or item is cast on
	npcPilfered  int          // g5892: each try makes the next harder
	npcSeqForce  [10][5]int16 // g50de: the animation sequence forced by pic slot, column
	npcPicAlt    int          // g5142: pictures 40 and 54 show their other look
	npcKeys      string       // 0x526a: message 1500, the menu keys

	npcTextWin   *Window // g5264: the text window, nil when closed
	npcTextRows  int     // g5262: its last row
	npcTextRow   int     // loot_count: the row print_text_line writes next
	npcTextCol   int     // g5268: the column of the last line
	npcScrollCol int     // g5144: where a new line starts

	// The member's items listed by npc_choose_member_item (g5326, g5324 of them) and
	// their names by slot (0x5352).
	npcItemSlots []int
	npcItemNames [22]string

	// The NPC's wares (g5856 ids; the rest by ware) and those on offer (g54b4, g54b2).
	npcWares    [30]npcWare
	npcWareList []int
)

// npcWare is one item the NPC sells.
type npcWare struct {
	ID    int    // g5856, 0 sold
	Name  string // 0x54f0
	Type  uint8  // g56d0
	Qty   int    // g56ee: the charges it comes with
	Icon  uint8  // g572a
	Class uint16 // g5748
	Race  uint16 // g5784
	Sex   uint8  // g57c0
	Price int32  // g57de
}

// npcTextControls are print_text_line's control characters (0x5004), in the order of
// the npcText... constants.
const npcTextControls = "!%#&]@"

const (
	npcTextNewRow  = iota // '!': skip a row
	npcTextPause          // '%': wait the message delay
	npcTextShowPic        // '#': show the monster picture of the next byte
	npcTextHidePic        // '&': take it away
	npcTextClear          // ']': clear the window
	npcTextMore           // '@': the more prompt
)

// npcMenuInput's menu keys: their indexes in npcKeys (message 1500).
const (
	npcKeyLeft   = 1
	npcKeyUp     = 2
	npcKeyRight  = 3
	npcKeyDown   = 4
	npcKeySelect = 5
)

// npcMainMenu's choices (messages 1530..).
const (
	npcMenuTalk = iota
	npcMenuTrade
	npcMenuCast
	npcMenuUse
	npcMenuPoolGold
	npcMenuPilfer
	npcMenuFight
	npcMenuLeave
)

// npcReact's event kinds (NPCRec.Entries B[2]).
const (
	npcReactItemGiven    = 1
	npcReactGoldGiven    = 2
	npcReactItemSold     = 3
	npcReactItemBought   = 4
	npcReactFight        = 6
	npcReactParting      = 7
	npcReactPilferCaught = 10
)

// Event script opcodes (EventRec.Ops Op), as npcRunScript runs them.
const (
	npcOpEnd = iota
	npcOpSay
	npcOpFight
	npcOpLeave
	npcOpShowPicture
	npcOpWait
	npcOpAskWord
	npcOpMore
	npcOpStepBack
	npcOpGiveItem
	npcOpTakeItem
	npcOpPayGold
	npcOpChargeGold
	npcOpNothing
	npcOpSetMapFlag
	npcOpSaveMapFlag
	npcOpSetNPCFlag
	npcOpSaveNPCFlag
	npcOpIfMapFlagClear
	npcOpIfMapFlagSet
	npcOpIfNPCFlagClear
	npcOpIfNPCFlagSet
	npcOpIfHasItem
	npcOpIfLacksItem
	npcOpIfChance
	npcOpRunIfMapFlagClear
	npcOpRunIfMapFlagSet
	npcOpRunIfNPCFlagClear
	npcOpRunIfNPCFlagSet
	npcOpRunIfChance
	npcOpRaiseStatus7
	npcOpCurse
	npcOpGainStats
	npcOpStamina
	npcOpRun
	npcOpSayRandom
	npcOpFace
	npcOpWalk
	npcOpRunIfUnable
	npcOpEndMeeting
	npcOpRestart
	npcOpFullMessage
	npcOpExchangeItem
	npcOpForceSequence
	npcOpPicAlt
)

// NPCRec.Unk8D bits.
const (
	npcNoBuySell  = 2 // the trade menu offers no buying or selling
	npcCharmProof = 4 // spell 50 fails
	npcRumorProof = 8 // spell 56 fails
)

const (
	spellCharmNPC  = 50 // adds to npcCharm
	spellDrawRumor = 56 // the NPC tells a rumour
	spellLockInNPC = 39 // with spellMapInNPC: does nothing in a meeting
	spellMapInNPC  = 60
	itemCurseWard  = 245 // spares the party op 31's curse
	itemEquipped   = 1   // an item's Flags bit
	itemImportant  = 2   // an item record's Flags bit: the NPC takes it only as Unk68[7]/[8]
	packSlots      = 10  // Items[0..9] the pack, Items[10..] the swag
	itemSlots      = 22
	maxStackQty    = 250
	maxStat        = 18
	maxAttitude    = 100
	minAttitude    = -100
	mapFlagCount   = 79 // map flags 1..78 a script can set
	npcFlagCount   = 15 // NPC flags 1..14
	npcFlagsOffset = 28 // scenarioHdr.Unk140: the NPCs' flag pairs start here
)

// npcRecord reads scenario record idx of type typ.
func npcRecord[T any](typ, idx int) T {
	b := make([]byte, dbs.Size[T]())
	accessScenarioRecord(typ, idx, b, false)
	r, _ := dbs.Decode[T](b)
	return r
}

// npcName is the NPC's name.
func npcName() string { return cstr(npcRec.Name[:]) }

// npcLevel is the NPC's level (g51cf), at least 1 where the C divides by it.
func npcLevel() int { return int(npcRec.Unk86[3]) }

// npcAttitude is member m's attitude to the NPC (g4550).
func npcAttitude(m int) *int8 { return &party[m].NPCAttitude[npcID] }

// npcMood is member m's attitude plus personality and charm: below 0 the NPC is hostile.
func npcMood(m int) int {
	return int(int16(int(*npcAttitude(m)) + int(party[m].Stats[statPersonality]) + int(npcCharm))) // C int wraps
}

// --- The view

// npcChooseFrames (WMNPC 4572) starts the animation of the picture in slot at view column
// col: the picture's fixed start, or a random sequence and step.
func npcChooseFrames(slot, col int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	p := &monsterPics[slot]
	for i := 0; i < int(p.Unk309[3]); i++ {
		var seq, step int
		if p.Unk309[0] > 0 && p.Unk309[0] != 255 {
			seq, step = int(p.Unk309[0])/25, int(p.Unk309[0])%25
			if picSlotMonster[slot] == 54 && i == 0 {
				seq, step = 0, 0
			}
		} else {
			seq = random(int(p.SeqCount[i])) + int(p.SeqFirst[i])
			step = random(animRunLength(p, seq)) + int(p.SeqStartFrame[seq])
		}
		f := &monsterFrames[col][i]
		f[0], f[1] = uint8(seq), uint8(step)
		f[2] = uint8(random(int(p.Quads[step][3])) + int(p.Quads[step][2]))
	}
}

// npcBuildFrames (WMNPC 46b8) steps the animation of the picture in slot at column col and
// returns its frames, one per layer. A forced sequence (npcSeqForce) replaces the random
// next one; pictures 40 and 54 change a layer with npcPicAlt.
func npcBuildFrames(slot, col int) []byte {
	p := &monsterPics[slot]
	var frames []byte
	for layer := 0; layer < int(p.Unk309[3]); layer++ {
		st := &monsterFrames[col][layer]
		step := int(st[1])
		ticks := int(st[2]) - 1
		if ticks <= 0 {
			seq := int(st[0])
			step++
			if int(p.SeqStartFrame[seq])+animRunLength(p, seq)-1 < step {
				if f := npcSeqForce[slot][col]; f > 0 {
					seq = int(f)
				} else {
					seq = random(int(p.SeqCount[layer])) + int(p.SeqFirst[layer])
				}
				st[0] = uint8(seq)
				step = int(p.SeqStartFrame[seq])
			}
			st[1] = uint8(step)
			ticks = random(int(p.Quads[step][3])) + int(p.Quads[step][2])
		}
		st[2] = uint8(ticks)
		f := p.Quads[step][1]
		switch picSlotMonster[slot] {
		case 40:
			if layer == 4 {
				f = 0
				if npcPicAlt == 1 && len(frames) > 0 {
					f = frames[len(frames)-1] + 5
				}
			}
		case 54:
			if layer == 0 && npcPicAlt == 1 {
				f = 0
			}
		}
		if f != 0 {
			frames = append(frames, f)
		}
	}
	return frames
}

// npcQueueSprite (WMNPC 49db) queues the picture at view row, col (viewPosGroup holds its
// slot), shifted dx pixels unless the picture has a fixed start.
func npcQueueSprite(row, col, dx int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	slot := int(viewPosGroup[row][col])
	if slot == 0 {
		return
	}
	p := &monsterPics[slot]
	var frames []byte
	if row == 0 {
		frames = npcBuildFrames(slot, col)
	} else if p.Unk306[0] != 0 {
		frames = []byte{p.Unk306[0]}
	}
	if len(frames) == 0 {
		return
	}
	mirror := col % 2
	if random(10) == 9 {
		monsterJitterX[row][col] = uint8(random(int(p.Unk309[1])))
		monsterJitterY[row][col] = uint8(random(int(p.Unk309[2])))
	}
	f := int(frames[0])
	if p.Unk309[0] != 0 {
		dx = 0
	}
	depth := int(p.Quads[f][0])
	if depth == 0 {
		depth = int(viewDepthBase[row*5+col])
	}
	x := int(p.Offsets[row][col][0]) + int(monsterJitterX[row][col]) + dx + int(p.FrameX[f][mirror])
	if x < 0 {
		x = 0
	} else if x >= 248 {
		return
	}
	y := int(p.Offsets[row][col][1]) + int(monsterJitterY[row][col]) + int(p.FrameY[f])
	insertSprite(depth, slot, x, y, 72, 248, mirror, frames)
}

// npcAnimStep (WMNPC 55dc) is one tick of the view: -3 shows it, -2 draws the maze, -1
// queues the pictures, 0..11 draw sprite n.
func npcAnimStep() {
	if dtLog != nil { // the difftest stubs it
		return
	}
	switch viewAnimState {
	case -3:
		drvShowWindow(pictureWin)
		animPhase = (animPhase + 1) % 2
	case -2:
		drawMazeView()
		for i := range spriteList {
			spriteList[i].Depth = 255
		}
		spriteCount = 0
	case -1:
		for r := range 2 {
			for c := range 5 {
				npcQueueSprite(r, c, 0)
			}
		}
	default:
		drawSprite(viewAnimState)
	}
	viewAnimState++
	if viewAnimState == 12 {
		viewAnimState = -3
	}
}

// npcAnimateCycle (WMNPC 5690) runs a whole animation cycle.
func npcAnimateCycle() {
	if dtLog != nil { // the difftest stubs it
		return
	}
	viewAnimState = -2
	for {
		npcAnimStep()
		if viewAnimState == -2 {
			break
		}
	}
}

// npcGesture (WMNPC 56a1) sways the NPC four times with its sound: kind 0 as it comes up
// (after message 2025), 1 as it goes.
func npcGesture(kind int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	if kind == 0 {
		s := getMessage(2025)
		eraseWindow(mainWindow, true)
		printCentered(mainWindow, s, 0, 18)
	}
	left := random(2) == 0
	beat := random(4)
	for i := range 4 {
		if (random(8) == 0 || i == beat) && npcSound > 0 {
			playSound(npcSound)
		}
		viewAnimState = -2
		npcAnimStep()
		dx := int(flashShift[kind*4+i])
		if left {
			dx = -dx
		}
		npcQueueSprite(0, 0, dx)
		viewAnimState = 0
		for {
			npcAnimStep()
			if viewAnimState == -2 {
				break
			}
		}
	}
}

// npcLoadPic (WMNPC 5508) loads monster picture pic into slot.
func npcLoadPic(pic, slot int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	loadMonsterPic(pic, slot, 21)
}

// --- Input

// npcWaitKey (WMNPC 578c) animates until a key of class comes.
func npcWaitKey(class int) int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	for !keyAvailable(class) {
		paceViewCycle()
		npcAnimStep()
	}
	return int(uint8(getKey(class)))
}

// npc_input_line (WMNPC 57ac) reads up to max characters at the cursor of win, echoed in
// color; digits 1 takes only digits.
func npcInputLine(win *Window, max, color, digits int) string {
	if dtLog != nil { // the difftest stubs it
		return ""
	}
	x, y := int(win.CurX), int(win.CurY)
	var s []byte
	putChar(win, 'a', 1) // the cursor glyph
	flushKeys()
	for {
		c := npcWaitKey(keyClassMenu)
		if c == keyEnter {
			break
		}
		setCursor(win, x, y)
		putChar(win, ' ', 1)
		setCursor(win, x, y)
		switch {
		case c == keyLeft: // Backspace
			if len(s) > 0 {
				s = s[:len(s)-1]
				x--
				setCursor(win, x, y)
				putChar(win, ' ', 1)
				setCursor(win, x, y)
			} else {
				playSound(soundBeep)
			}
		case len(s) >= max:
			playSound(soundBeep)
		case digits == inputPrintable && c >= ' ' && c <= 127 || digits == inputDigits && c >= '0' && c <= '9':
			s = append(s, byte(c))
			putChar(win, byte(c), color)
			x++
		}
		putChar(win, 'a', 1)
	}
	setCursor(win, x, y)
	putChar(win, ' ', 1)
	return string(s)
}

// npcMenuInput (WMNPC 5994) animates until a menu key (its index in npcKeys) or a click
// on a click region (its index, menuClicked set), blinking the cursor.
func npcMenuInput() int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	phase := animPhase
	if pvBlinkState != -1 {
		blinkCursor()
	}
	flushKeys()
	menuClicked = 0
	for {
		if keyAvailable(keyClassMenu) {
			if c := charIndex(npcKeys, int(uint8(getKey(keyClassMenu)))); c >= 0 {
				return c
			}
			continue
		}
		r := checkClickRegions(clickRegions[:])
		if clickFlag {
			menuClicked = 1
			return r - 1
		}
		paceViewCycle()
		npcAnimStep()
		if viewAnimState == -2 && pvBlinkState != -1 && animPhase == phase {
			blinkCursor()
		}
	}
}

// npcWait (WMNPC 5a1e) animates for tenths of 10 ticks, less on Enter or a click.
func npcWait(tenths int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	flushKeys()
	for n := tenths * 10; n > 0; n-- {
		if checkScreenClick() {
			return
		}
		if keyAvailable(keyClassMenu) && int8(getKey(keyClassMenu)) == keyEnter {
			return
		}
		paceViewCycle()
		npcAnimStep()
	}
}

// npcMorePrompt (WMNPC 5a69) blinks the more mark of the text window until Enter or a click.
func npcMorePrompt() {
	if dtLog != nil { // the difftest stubs it
		return
	}
	blink := 3
	flushKeys()
	for {
		if keyAvailable(keyClassMenu) {
			if int8(getKey(keyClassMenu)) == keyEnter {
				return
			}
			continue
		}
		if checkScreenClick() {
			return
		}
		paceViewCycle()
		npcAnimStep()
		if viewAnimState == -2 {
			setCursor(npcTextWin, 39, npcTextRows)
			if blink < 2 {
				putChar(npcTextWin, ' ', 1)
			} else {
				putCharHi(npcTextWin, 126, 1)
			}
			blink = (blink + 1) % 4
		}
	}
}

// --- Text

// npcEndPage (WMNPC 5af3): a timed pause (timed 1), or the more prompt and a clean page.
func npcEndPage(timed int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	if timed == 1 {
		npcWait(messageDelay)
		return
	}
	npcMorePrompt()
	clearWindow(npcTextWin, true)
	npcTextRow = 0
}

// npcPrintTextLine (WMNPC 5b21) is WTREA's print_text_line, where '#' shows the monster
// picture of the next byte in place of the NPC and '&' takes it away.
func npcPrintTextLine(s string) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	for len(s) > 0 {
		k := strings.IndexByte(npcTextControls, s[0])
		if k < 0 {
			break
		}
		s = s[1:]
		switch k {
		case npcTextNewRow:
			npcTextRow++
		case npcTextPause:
			npcWait(messageDelay)
		case npcTextShowPic:
			if len(s) > 0 {
				npcLoadPic(int(s[0]), 4)
				s = s[1:]
			}
			viewPosGroup[0][0] = 4
			npcChooseFrames(4, 0)
			npcAnimateCycle()
		case npcTextHidePic:
			viewPosGroup[0][0] = 0
			npcAnimateCycle()
		case npcTextClear:
			clearWindow(npcTextWin, true)
			npcTextRow = 0
		case npcTextMore:
			npcEndPage(0)
		}
	}
	if npcTextRow > npcTextRows {
		npcEndPage(0)
	}
	col := 20 - (len(s)+1)/2
	if len(s) > 0 {
		switch s[0] {
		case '$':
			s, col = s[1:], 0
		case '^':
			s, col = s[1:], npcTextCol
		}
	}
	setCursor(npcTextWin, col, npcTextRow)
	npcTextCol = col
	npcTextRow++
	printString(npcTextWin, s, 5) // g0002
}

// npcShowMessageText (WMNPC 5c61) shows messages msg, msg+1, ... up to the first missing
// one in a text window; timed 1 ends with a pause, else with the more prompt.
func npcShowMessageText(msg, timed int) {
	npcTextCol, npcTextRow = 0, 0
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	npcTextWin = createWindow(0, 19, 40, 5, 30, borderNone, 0, true)
	npcTextRows = 4
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		npcPrintTextLine(s)
		msg++
	}
	npcEndPage(timed)
	destroyWindow(npcTextWin, false) // inline, as the C (not npc_close_text)
	npcTextWin = nil
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
}

// npcNotice (WMNPC 5d0c) shows s centred in a text window for the message delay.
func npcNotice(s string, beep bool) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	npcTextWin = createWindow(0, 19, 40, 5, 30, borderNone, 0, true)
	printCentered(npcTextWin, s, 1, 5)
	if beep {
		playSound(soundBeep)
	}
	npcWait(messageDelay)
	npcCloseText()
}

// npcNoticeName (WMNPC 5d9b) is npcNotice of message msg with '^' the NPC's name.
func npcNoticeName(msg int, beep bool) {
	npcNotice(replaceCharWithString(getMessage(msg), npcName(), '^'), beep)
}

// npcOpenText (WMNPC 6d9d) opens the text window.
func npcOpenText() {
	if dtLog != nil { // the difftest stubs it
		return
	}
	npcScrollCol, npcTextCol, npcTextRow = 0, 0, 0
	deferWindowDraw(mainWindow)
	deferWindowDraw(bottomLineWin)
	npcTextWin = createWindow(0, 19, 40, 5, 30, borderNone, 0, true)
}

// npcCloseText (WMNPC 6de9) closes it.
func npcCloseText() {
	if dtLog != nil { // the difftest stubs it (its windows are null there)
		npcTextWin = nil
		return
	}
	destroyWindow(npcTextWin, false)
	npcTextWin = nil
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
}

// npcScroll (WMNPC 4ecb) scrolls w up a line.
func npcScroll(w *Window) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	deferWindowDraw(w)
	n := int(w.W)
	copy(w.Cells[:n*(int(w.H)-1)], w.Cells[n:])
	clearWindowArea(w, 0, int(w.H)-1, n-1, int(w.H)-1)
	resumeWindowDraw(w)
	refreshWindow(w)
	setCursor(w, npcScrollCol, int(w.H)-1)
}

// npcNewline (WMNPC 4f73) moves the cursor of w to the next line, scrolling at the bottom.
func npcNewline(w *Window) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	if int(w.CurY) >= int(w.H)-1 {
		npcScroll(w)
		return
	}
	setCursor(w, npcScrollCol, int(w.CurY)+1)
}

// npcPrintWrapped (WMNPC c7d9) prints s word-wrapped in the text window, after its leading
// controls: '@' the more prompt and a blank line, '%' a pause, '!' a new line.
func npcPrintWrapped(s string, color int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	w := npcTextWin
lead:
	for len(s) > 0 {
		switch s[0] {
		case '@':
			x, y := int(w.CurX), int(w.CurY)
			npcMorePrompt()
			setCursor(w, 39, 4)
			putChar(w, ' ', 0)
			setCursor(w, x, y)
			npcNewline(w)
			npcNewline(w)
		case '%':
			npcWait(messageDelay)
		case '!':
			npcNewline(w)
		default:
			break lead
		}
		s = s[1:]
	}
	for len(s) > 0 {
		sp := strings.IndexByte(s, ' ')
		n := len(s)
		if sp >= 0 {
			n = sp
		}
		if int(w.CurX)+n+1 > int(w.W) {
			npcNewline(w)
		}
		if w.CurX == 0 {
			w.CurX++
		}
		printString(w, s[:n], color)
		w.CurX++
		if sp < 0 {
			break
		}
		s = s[sp+1:]
	}
}

// npcSay (WMNPC c95b) prints message msg, '^' replaced by repl, and ends the line.
func npcSay(msg int, repl string) {
	npcPrintWrapped(replaceCharWithString(getMessage(msg), repl, '^'), 5)
	npcNewline(npcTextWin)
}

// npcSayAll (WMNPC c999) prints messages msg, msg+1, ... up to the first missing one.
func npcSayAll(msg int) {
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		npcPrintWrapped(s, 5)
		msg++
	}
	npcNewline(npcTextWin)
}

// npcPrintBigLine (WMNPC 6e0e) is print_text_line for the full-screen text window: '!'
// skips a row, '^' keeps the last column.
func npcPrintBigLine(s string) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	for len(s) > 0 && s[0] == '!' {
		s = s[1:]
		npcTextRow++
	}
	col := 19 - (len(s)+1)/2
	if len(s) > 0 && s[0] == '^' {
		s, col = s[1:], npcTextCol
	}
	setCursor(npcTextWin, col, npcTextRow)
	npcTextCol = col
	npcTextRow++
	printString(npcTextWin, s, 5) // g0002
}

// npcFullMessage (WMNPC 6e7e) shows messages msg.. full screen until Enter or a click.
// Then dest 0..2 sends the party to the same place with Flags bit dest set (1); dest 3
// only returns 1; else 0.
func npcFullMessage(msg, dest int) int {
	npcTextCol, npcTextRow = 0, 0
	reopen := npcTextWin != nil
	if reopen {
		npcCloseText()
	}
	npcTextWin = createWindow(1, 1, 38, 23, 30, 1, 0, true)
	npcTextRows = 21
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		npcPrintBigLine(s)
		msg++
	}
	blink, n := 3, 0
	flushKeys()
	for {
		if n == 10 {
			n = 0
			setCursor(npcTextWin, 37, npcTextRows)
			if blink < 2 {
				putChar(npcTextWin, ' ', 1)
			} else {
				putCharHi(npcTextWin, 126, 1)
			}
			blink = (blink + 1) % 4
		} else {
			n++
			delayTick()
			delayTick()
		}
		done := checkScreenClick()
		if keyAvailable(keyClassMenu) && int8(getKey(keyClassMenu)) == keyEnter {
			done = true
		}
		if done {
			break
		}
	}
	npcCloseText()
	if reopen {
		npcOpenText()
	}
	npcTextRows = 4
	switch dest {
	case 0, 1, 2:
		partyX, partyY = 137, 151
		cellX, cellY = 1, 4
		partyBlock = 2
		gameStateArg = 12
		partyFacing = 2
		for m := 0; m < partyCount; m++ {
			setBit(party[m].Flags[:], dest, true)
		}
		gameState = stateLoadMap
		returnCmd = int16(stateNPC)
		return 1
	case 3:
		return 1
	}
	return 0
}

// --- Menus and the member panels

// npcPrompt (WMNPC 541a) shows s on the message line, centred in its left half with room
// for extra more characters.
func npcPrompt(s string, extra int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	drawCombatTitle()
	deferWindowDraw(mainWindow)
	setCursor(mainWindow, 10-(len(s)+extra+1)/2, 0)
	printStringHi(mainWindow, s, 3)
	resumeWindowDraw(mainWindow)
}

// npcDrawNameLine (WMNPC 5488) shows the NPC's name between two ornaments in w.
func npcDrawNameLine(w *Window) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	drawCombatTitle()
	deferWindowDraw(w)
	clearWindow(w, true)
	setCursor(w, 0, 0)
	putChar(w, '1', 6)
	w.CurX++
	printString(w, npcName(), 14)
	setCursor(w, 19, 0)
	putChar(w, '1', 9)
	resumeWindowDraw(w)
}

// npcDrawMemberStatus (WMNPC 5078) is drawMemberStatus, showing member m's gold in place
// of the name if gold.
func npcDrawMemberStatus(m, gold int) {
	if dtLog != nil { // the difftest stubs it
		return
	}
	drawMemberStatus(m)
	if gold != 0 {
		win := rightPanelWin
		if m%2 == 0 {
			win = leftPanelWin
		}
		setCursor(win, 0, m/2<<2)
		printLong(win, party[m].Gold, 7, 5)
	}
}

// menuSelectCancel (WMAZE 5ad8 maze_menu_select, WMNPC 5e0c, WTREA 5169) is menuSelect where
// withCancel 1 adds the message line's exit first (chosen: -1); input reads the menu.
func menuSelectCancel(win *Window, firstMsg int, enabled []int, x0, y0, width, rows, color, withCancel, sel int, input func() int) int {
	var exitX, exitY int
	var exitHi, exitLo string
	if withCancel == 1 {
		exitX, exitY = int(mainWindow.CurX), int(mainWindow.CurY)
		exitHi, exitLo = getMessage(2027), getMessage(2028)
		if mousePresent {
			printStringHi(mainWindow, exitHi, 3)
		} else {
			printString(mainWindow, exitLo, -color)
		}
	}
	var labels []string
	var keys []int
	for k := 0; enabled[k] != -1; k++ {
		if enabled[k] != 1 {
			continue
		}
		n := len(labels)
		x, y := menuCell(n, width, rows, x0, y0)
		setCursor(win, x, y)
		s := getMessage(firstMsg + k)
		if !mousePresent && withCancel == 0 && n == sel {
			printString(win, s, -color)
		} else {
			printStringHi(win, s, 3)
		}
		labels, keys = append(labels, s), append(keys, k)
		setClickRegion(n, win.X+byte(x), win.Y+byte(y), byte(len(s)+int(win.X)+x-1), win.Y+byte(y))
	}
	n := len(labels)
	setClickRegion(n, clickRegionsEnd, 0, 0, 0)
	if withCancel == 1 {
		setClickRegion(n, mainWindow.X+byte(exitX), mainWindow.Y+byte(exitY), byte(len(exitLo)+int(mainWindow.X)+exitX-1), mainWindow.Y+byte(exitY))
		setClickRegion(n+1, clickRegionsEnd, 0, 0, 0)
		sel = -1
	}
	plain := func() { redrawMenuItem(win, sel, width, rows, x0, y0, labels[sel]) }
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printString(mainWindow, exitLo, -color)
			} else {
				x, y := menuCell(sel, width, rows, x0, y0)
				setCursor(win, x, y)
				printString(win, labels[sel], -color)
			}
		}
		key := input()
		if menuClicked == 1 {
			sel = -1
			if key < n {
				sel = key
			}
			break
		}
		switch key {
		case npcKeyLeft:
			if sel != -1 && sel-rows >= 0 {
				plain()
				sel -= rows
			}
		case npcKeyUp:
			if sel != -1 {
				if sel%rows > 0 {
					plain()
					sel--
				} else if withCancel == 1 {
					plain()
					sel = -1
				}
			}
		case npcKeyRight:
			if sel != -1 && sel+rows < n {
				plain()
				sel += rows
			}
		case npcKeyDown:
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printStringHi(mainWindow, exitHi, 3)
				sel = 0
			} else if sel%rows < rows-1 && sel+1 < n {
				plain()
				sel++
			}
		}
		if key == npcKeySelect {
			break
		}
	}
	if sel == -1 {
		return -1
	}
	return keys[sel]
}

// npcChooseMember (WMNPC 62ee) lets the player pick a member under prompt msg; withExit
// offers exit (-1) first. It is 0 with one member.
func npcChooseMember(msg int, withExit bool) int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, borderRightEdge, 0, false)
	fillWindow(win, ' ', 3)
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	extra := 0
	if withExit {
		extra = 5
	}
	npcPrompt(getMessage(msg), extra)
	refreshAllWindows()
	var exitX, exitY int
	exitHi, exitLo := getMessage(2027), getMessage(2028)
	if withExit {
		mainWindow.CurX++
		exitX, exitY = int(mainWindow.CurX), int(mainWindow.CurY)
		if mousePresent {
			printStringHi(mainWindow, exitHi, 3)
		} else {
			printString(mainWindow, exitLo, -5)
		}
	}
	for m := 0; m < partyCount; m++ {
		x, y := memberCell(m)
		setCursor(win, x, y)
		name := memberName(m)
		printStringHi(win, name, 3)
		setClickRegion(m, win.X+byte(x), win.Y+byte(y), byte(len(name)+int(win.X)+x-1), win.Y+byte(y))
	}
	setClickRegion(partyCount, clickRegionsEnd, 0, 0, 0)
	sel := 0
	if withExit {
		setClickRegion(partyCount, mainWindow.X+byte(exitX), mainWindow.Y+byte(exitY), byte(len(exitLo)+int(mainWindow.X)+exitX-1), mainWindow.Y+byte(exitY))
		setClickRegion(partyCount+1, clickRegionsEnd, 0, 0, 0)
		sel = -1
	}
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printString(mainWindow, exitLo, -5)
			} else {
				x, y := memberCell(sel)
				setCursor(win, x, y)
				printString(win, memberName(sel), -5)
			}
		}
		key := npcMenuInput()
		if menuClicked == 1 {
			sel = -1
			if key < partyCount {
				sel = key
			}
			break
		}
		switch key {
		case npcKeyLeft:
			if sel != -1 && sel%2 > 0 {
				drawMemberName(win, sel)
				sel--
			}
		case npcKeyUp:
			if sel != -1 {
				if sel-2 >= 0 {
					drawMemberName(win, sel)
					sel -= 2
				} else if withExit {
					drawMemberName(win, sel)
					sel = -1
				}
			}
		case npcKeyRight:
			if sel != -1 && sel%2 < 1 && sel+1 < partyCount {
				drawMemberName(win, sel)
				sel++
			}
		case npcKeyDown:
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printStringHi(mainWindow, exitHi, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if key == npcKeySelect {
			break
		}
	}
	eraseWindow(mainWindow, false)
	destroyWindow(win, false)
	return sel
}

// --- The encounter

// run_npc_encounter (WMNPC ba7d) shows the NPC, stocks its wares, loads the words it knows
// and runs the menu.
func runNPCEncounter() {
	npcTextWin, npcTextRows = nil, 4
	initNPCEncounter()
	npcShowNPC()
	npcStockWares()
	npcLoadVocabulary()
	eraseWindow(mainWindow, false)
	deferWindowDraw(g4fb0)
	monsterListWin = createWindow(20, 19, 20, 5, 10, borderNone, 0, false)
	npcDrawNameLine(monsterListWin)
	npcMainMenu()
	destroyWindow(monsterListWin, false)
	eraseWindow(mainWindow, false)
	resumeWindowDraw(g4fb0)
	clear(picSlotLoaded[4:7])    // g4326
	viewPosGroup = [2][5]int16{} // ba00, which also frees the sprites
}

// init_npc_encounter (WMNPC b8de) clears the sprites and the view and reads NPC npcID.
func initNPCEncounter() {
	npcKeys = getMessage(1500)
	gameState = stateExplore
	for i := range spriteList {
		spriteList[i].Depth = 255
	}
	spriteCount = 0
	viewAnimState = -2
	viewPosGroup = [2][5]int16{}
	picSlotMonster[4], picSlotMonster[5], picSlotMonster[6] = -1, -1, -1 // g4330
	npcSeqForce = [10][5]int16{}
	npcPicAlt = 0
	npcRec = npcRecord[NPCRec](dbs.RecNPC, npcID)
	copy(npcFlags[:], scenarioHdr.Unk140[npcFlagsOffset+2*npcID:])
	npcGoldGiven = 0
}

// npcShowNPC (WMNPC 97c6) loads the NPC's picture (slot 4, the front left of the view) and
// its sound (sounds from 14 on into slot 14) and sways it in.
func npcShowNPC() {
	if dtLog != nil { // the difftest stubs it
		return
	}
	npcLoadPic(int(npcRec.Unk86[1]), 4)
	viewPosGroup[0][0] = 4
	npcChooseFrames(4, 0)
	switch s := int(npcRec.Disarm); {
	case s >= 14:
		if s != curSound {
			loadSound(s)
		}
		npcSound = 14
	case s > 0:
		npcSound = s
	default:
		npcSound = 0
	}
	npcGesture(0)
}

// npcMainMenu (WMNPC b344): the NPC's greeting, then until the party is done: its
// automatic scripts, the menu (talk, trade, cast, use, pool gold, pilfer, fight, leave),
// and a hostile NPC's chance to attack or leave.
func npcMainMenu() {
	npcLeaves, npcPilfered = 0, 0
	enabled := []int{1, 1, 1, 1, 1, 1, 1, 1, -1}
	title := getMessage(1510)
	win := createWindow(0, 19, 19, 5, 18, borderRightEdge, 0, false)
	if g := npcRec.Unk68[1]; g > 0 {
		npcShowMessageText(int(g), 0)
	}
	m := 0
	for {
		npcShowGold = 0
		if npcSound > 0 {
			playSound(npcSound)
		}
		for _, s := range npcRec.Scripts {
			if s > 0 && npcRunScript(m, int(s)) == 1 {
				if npcLeaves == 1 {
					npcGesture(1)
				}
				destroyWindow(win, false)
				return
			}
		}
		fillWindow(win, ' ', 3)
		npcPrompt(title, 0)
		m = -1
		refreshAllWindows()
		choice := menuSelectCancel(win, 1530, enabled, 1, 1, 6, 3, 5, 0, 0, npcMenuInput)
		// pick asks for a member under msg; able is whether one was picked and can act.
		pick := func(msg int) bool {
			m = npcChooseMember(msg, true)
			return m >= 0 && isMemberAble(m)
		}
		switch choice {
		case npcMenuTalk:
			if pick(1512) {
				if npcMood(m) > 0 {
					npcTalk(m)
				} else {
					npcRefuse(1540)
				}
			}
		case npcMenuTrade:
			if pick(1501) {
				if npcMood(m) > 0 {
					npcTrade(m)
				} else {
					npcRefuse(1540)
				}
			}
		case npcMenuCast:
			if pick(1514) {
				npcCastSpell(win, m)
			}
		case npcMenuUse:
			if pick(1515) {
				npcUseItem(win, m)
			}
		case npcMenuPoolGold:
			if pick(1519) {
				npcPoolGold(m)
			}
		case npcMenuPilfer:
			if pick(1511) {
				npcPilfer(m)
			}
		case npcMenuFight:
			for i := 0; i < partyCount; i++ {
				a := npcAttitude(i)
				if *a > 10 {
					*a /= 5
				}
				*a = int8(max(int(*a)-(random(5)+5), minAttitude))
			}
			npcStartFight()
			npcLeaves = 0
			choice = npcMenuLeave
		case npcMenuLeave:
			npcLeaves = 1
		}
		if m < 0 || m > partyCount-1 || random(4) == 0 {
			m = random(partyCount)
		}
		if npcMood(m) < 0 {
			a := int(*npcAttitude(m))
			if a < 0 {
				a = -a
			}
			if int(int16(random(100)+int(party[m].Stats[statPersonality])+int(npcCharm))) < a { // C int wraps
				if random(2) == 1 {
					npcNoticeName(1541, true)
					npcStartFight()
				} else {
					npcNoticeName(1542, false)
					npcLeaves = 1
				}
				choice = npcMenuLeave
			}
		}
		if choice == npcMenuLeave {
			break
		}
	}
	if gameState == stateBeginCombat {
		npcReact(0, npcReactFight, 0, 0, 0)
	} else {
		npcReact(0, npcReactParting, 0, 0, 0)
	}
	if npcLeaves == 1 {
		npcGesture(1)
	}
	destroyWindow(win, false)
}

// npcStartFight sets up a combat with the NPC's monster.
func npcStartFight() {
	g433a = int(npcRec.Unk86[0])
	g4338, g4354 = 1, 0
	gameState = stateBeginCombat
	returnCmd = int16(stateNPC)
}

// npcRefuse (WMNPC a645) shows message msg on the message line with a beep.
func npcRefuse(msg int) {
	eraseWindow(mainWindow, true)
	printCentered(mainWindow, getMessage(msg), 0, 18)
	playSound(soundBeep)
	npcWait(messageDelay)
	eraseWindow(mainWindow, true)
}

// npcPoolGold (WMNPC 5382) gives all the party's gold to member m.
func npcPoolGold(m int) {
	var sum int32
	for i := 0; i < partyCount; i++ {
		sum += party[i].Gold
		party[i].Gold = 0
	}
	party[m].Gold = sum
}

// --- Event scripts

// npcRunScript (WMNPC 7038) runs the event script of the NPC's topic n (scenario record
// type 6) for member m. 1 ends the encounter; 0 is a failed condition, the end of the
// script, or a sub-script that did not end it.
func npcRunScript(m, n int) int {
	ev := npcRecord[EventRec](dbs.RecEvent, int(npcRec.Topics[n]))
	levelBit := func(bit int) bool { return testBit(levelFlags[gameStateArg][:], bit) }
	npcBit := func(bit int) bool { return testBit(npcFlags[:], bit) }
	// findAny is the slot of item id with member m, or anyone's if anyone.
	findAny := func(id int, anyone bool) int {
		if !anyone {
			return npcFindItem(m, id)
		}
		r := -1
		for i := 0; i < partyCount && r == -1; i++ {
			r = npcFindItem(i, id)
		}
		return r
	}
	step := func(dir int) {
		switch dir {
		case dirNorth:
			partyY++
		case dirEast:
			partyX++
		case dirSouth:
			partyY--
		case dirWest:
			partyX--
		}
		moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY))
		cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
	}
	for _, op := range ev.Ops {
		arg, param := int(op.Arg), int(op.Param)
		switch op.Op {
		case npcOpEnd:
			return 0
		case npcOpSay:
			if param != 0 {
				npcShowMessageText(arg, param-1)
			} else if npcTextWin == nil {
				npcOpenText()
				npcSayAll(arg)
				npcCloseText()
			} else {
				npcSayAll(arg)
			}
		case npcOpFight:
			npcNoticeName(1541, true)
			npcStartFight()
			npcLeaves = 0
			return 1
		case npcOpLeave:
			if param == 0 {
				npcNoticeName(1542, false)
			}
			npcLeaves = 1
			gameState = stateExplore
			returnCmd = int16(stateNPC)
			return 1
		case npcOpShowPicture: // show picture param in slot arg>>8 at view position arg&255
			slot, pos := arg/256, arg%256
			if param > 0 {
				npcLoadPic(param, slot)
				viewPosGroup[pos/5][pos%5] = int16(slot)
				npcChooseFrames(slot, pos)
			} else {
				viewPosGroup[pos/5][pos%5] = 0
			}
			npcAnimateCycle()
		case npcOpWait:
			npcWait(param)
		case npcOpAskWord: // ask for a word
			open := npcTextWin == nil
			if open {
				npcOpenText()
			}
			r := npcInputWord(m, arg, param)
			if open {
				npcCloseText()
			}
			if r > 1 {
				if npcRunScript(m, r) == 1 {
					return 1
				}
			} else if npcTextWin != nil {
				if s := getMessage(arg + 1); len(s) > 0 {
					npcPrintWrapped(s, 5)
					npcNewline(npcTextWin)
				}
			}
		case npcOpMore:
			npcMorePrompt()
		case npcOpStepBack: // step back and turn round
			switch partyFacing {
			case dirNorth:
				partyY--
			case dirEast:
				partyX--
			case dirSouth:
				partyY++
			case dirWest:
				partyX++
			}
			partyFacing = (partyFacing + 2) % dirCount
			moveToMazeCell(&cellX, &cellY, &partyBlock, int(partyX), int(partyY))
			cellEventIndex = int16(mazeB.FirstEvent[partyBlock])
			gameState = stateExplore
			returnCmd = int16(stateNPC)
			return 1
		case npcOpGiveItem: // give item arg to m, or to the first who can take it
			r := npcGiveItemToMember(m, arg, -1)
			who := m
			for i := 0; i < partyCount && r == -1; i++ {
				r, who = npcGiveItemToMember(i, arg, -1), i
			}
			if r == -1 {
				return 0
			}
			if param == 1 {
				npcNotice(replaceCharWithString(getMessage(1547), memberName(who), '^'), true)
			}
		case npcOpTakeItem: // take item arg (a charge of a stack)
			if k := npcFindItem(m, arg); k != -1 {
				if stackable[party[m].Items[k].Type] == 1 {
					useItemCharge(m, k)
				} else {
					removeItem(m, k)
				}
			}
		case npcOpPayGold: // pay arg*param gold
			n := int32(arg) * int32(param)
			party[m].Gold += n
			if npcRec.Gold -= n; npcRec.Gold < 0 {
				npcRec.Gold = 0
			}
		case npcOpChargeGold: // charge arg*10 gold (0 takes everyone's), else run param
			n := int32(arg) * 10
			if n == 0 {
				for i := 0; i < partyCount; i++ {
					party[i].Gold = 0
				}
			}
			if party[m].Gold < n {
				npcPoolGold(m)
				if party[m].Gold < n {
					if npcRunScript(m, param) == 1 {
						return 1
					}
					return 0
				}
			}
			party[m].Gold -= n
			npcRec.Gold += n
		case npcOpNothing:
		case npcOpSetMapFlag, npcOpSaveMapFlag: // set map flag arg (15 also the saved one)
			if arg > 0 && arg < mapFlagCount {
				setBit(levelFlags[gameStateArg][:], arg, param != 0)
				if op.Op == npcOpSaveMapFlag {
					setBit(scenarioHdr.LevelFlags[gameStateArg][:], arg, param != 0)
				}
			}
		case npcOpSetNPCFlag, npcOpSaveNPCFlag: // set NPC flag arg (17 also the saved one)
			if arg > 0 && arg < npcFlagCount {
				setBit(npcFlags[:], arg, param != 0)
				if op.Op == npcOpSaveNPCFlag {
					setBit(scenarioHdr.Unk140[npcFlagsOffset+2*npcID:], arg, param != 0)
				}
			}
		case npcOpIfMapFlagClear:
			if levelBit(arg) {
				return 0
			}
		case npcOpIfMapFlagSet:
			if !levelBit(arg) {
				return 0
			}
		case npcOpIfNPCFlagClear:
			if npcBit(arg) {
				return 0
			}
		case npcOpIfNPCFlagSet:
			if !npcBit(arg) {
				return 0
			}
		case npcOpIfHasItem:
			if findAny(arg, param != 0) == -1 {
				return 0
			}
		case npcOpIfLacksItem:
			if findAny(arg, param != 0) != -1 {
				return 0
			}
		case npcOpIfChance:
			if random(100) > arg {
				return 0
			}
		case npcOpRunIfMapFlagClear:
			if !levelBit(arg) && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpRunIfMapFlagSet:
			if levelBit(arg) && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpRunIfNPCFlagClear:
			if !npcBit(arg) && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpRunIfNPCFlagSet:
			if npcBit(arg) && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpRunIfChance:
			if random(100) < arg && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpRaiseStatus7: // the well members gain status 7
			for i := 0; i < partyCount; i++ {
				if party[i].Condition == conditionOK {
					npcNotice(replaceCharWithString(getMessage(1548), memberName(i), '^'), true)
					party[i].Status[7]++
					npcDrawMemberStatus(i, npcShowGold)
				}
			}
		case npcOpCurse: // a curse on the living, unless one carries item 245; then run arg or param
			next := arg
			for i := 0; i < partyCount; i++ {
				p := &party[i]
				if p.Condition >= conditionDead {
					continue
				}
				if npcFindItem(i, itemCurseWard) != -1 {
					next = param
					break
				}
				npcNotice(replaceCharWithString(getMessage(1549), memberName(i), '^'), true)
				p.HP = int16(random(int(p.HP)) + 1)
				p.HPMax -= int16(random(4) + 1)
				p.HPMax = max(p.HPMax, 1)
				p.HP = min(p.HP, p.HPMax)
				npcDrawMemberStatus(i, npcShowGold)
			}
			if npcRunScript(0, next) == 1 {
				return 1
			}
		case npcOpGainStats: // everyone gains stats arg-1 and param-1 (7 on: a random one), up to 18
			for i := 0; i < partyCount; i++ {
				for _, s := range [2]int{arg - 1, param - 1} {
					if s < 0 {
						continue
					}
					if s > 7 {
						s = random(8)
					}
					if party[i].Stats[s] >= maxStat {
						continue
					}
					party[i].Stats[s]++
					t := replaceCharWithString(getMessage(5015), memberName(i), '^')
					npcNotice(replaceCharWithString(t, getMessage(s+5050), '$'), true)
				}
			}
		case npcOpStamina: // arg d8 of stamina, lost if param
			n := 0
			for range arg {
				n += random(8)
			}
			p := &party[m]
			if param > 0 {
				if p.Stamina -= int16(n); p.Stamina < 0 {
					p.Stamina = 0
					p.Status[statusAsleep] = uint8(n)
				}
			} else {
				p.Stamina = min(p.Stamina+int16(n), p.StaminaMax)
			}
			npcDrawMemberStatus(m, npcShowGold)
		case npcOpRun:
			if npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpSayRandom: // one of param texts, two messages apart
			npcShowMessageText(random(param)<<1+arg, 1)
		case npcOpFace:
			partyFacing = int16(arg)
			npcAnimateCycle()
			npcWait(4)
		case npcOpWalk: // walk arg steps toward param
			for range arg {
				step(param)
				npcAnimateCycle()
				npcWait(4)
			}
		case npcOpRunIfUnable:
			if !isMemberAble(m) && npcRunScript(m, param) == 1 {
				return 1
			}
		case npcOpEndMeeting:
			return 1
		case npcOpRestart: // back to the start of the game
			partyX, partyY = 129, 132
			cellX, cellY = 1, 4
			partyBlock, partyFacing = 0, 0
			gameStateArg = 13
			gameState = stateLoadMap
			returnCmd = int16(stateNPC)
			return 1
		case npcOpFullMessage:
			return npcFullMessage(arg, param)
		case npcOpExchangeItem: // the first member with item param gets item arg for it
			r := -1
			for i := 0; i < partyCount && r == -1; i++ {
				if r = npcFindItem(i, param); r != -1 {
					removeItem(i, r)
					npcGiveItemToMember(i, arg, -1)
				}
			}
		case npcOpForceSequence:
			npcSeqForce[arg/256][arg%256] = int16(param)
		case npcOpPicAlt:
			npcPicAlt = param
		}
	}
	return 0
}

// npcReact (WMNPC 7f76) runs the script of the NPC's first reaction to event kind with
// values a, b, c (0 matches anything): 1 an item given, 2 gold given (a<<8 + b, adding up
// until the reaction's amount), 3 an item sold, 4 bought, 6 a fight, 7 parting, 10 a
// pilfer caught. The script's result, or 0.
func npcReact(m, kind, a int, b, c uint8) int {
	if kind == npcReactGoldGiven {
		npcGoldGiven += int32(a)<<8 + int32(b)
	}
	for _, e := range npcRec.Entries {
		if int(e.B[2]) != kind {
			continue
		}
		if kind == npcReactGoldGiven {
			if int32(e.A)<<8+int32(e.B[0]) > npcGoldGiven {
				continue
			}
			npcGoldGiven = 0
			return npcRunScript(m, int(e.B[3]))
		}
		if kind == npcReactItemGiven && e.A == 0 {
			a = 0
		}
		if int(e.A) != a && a != 0 || e.B[0] != b && b != 0 || e.B[1] != c && c != 0 {
			continue
		}
		return npcRunScript(m, int(e.B[3]))
	}
	return 0
}

// --- Items

// npcFindItem (WMNPC 6d11) is the slot of member m's first item id (swag from 10), or -1.
func npcFindItem(m, id int) int {
	p := &party[m]
	for i := 0; i < int(p.ItemCount); i++ {
		if int(p.Items[i].ID) == id {
			return i
		}
	}
	for i := 0; i < int(p.SwagCount); i++ {
		if int(p.Items[i+packSlots].ID) == id {
			return i + packSlots
		}
	}
	return -1
}

// npc_give_item_to_member (WMNPC 6a80) is WTREA's giveItemToMember for qty of item id
// (-1 rolls the item's charges).
func npcGiveItemToMember(member, id, qty int) int {
	p := &party[member]
	rec := itemRec(id)
	if qty == -1 {
		qty = rollDice(rec.Charges)
	}
	stacks := func(i int) bool {
		it := p.Items[i]
		return int(it.ID) == id && stackable[it.Type] == 1 && int(it.Qty)+qty <= maxStackQty
	}
	place := 1 // 1 a new pack slot, 2 a new swag slot, 0 a stack
	slot := int(p.ItemCount)
	for i := slot - 1; i >= 0; i-- {
		if stacks(i) {
			slot, place = i, 0
		}
	}
	if slot >= packSlots {
		place = 2
		slot = int(p.SwagCount) + packSlots
		for i := int(p.SwagCount) - 1; i >= 0; i-- {
			if stacks(i + packSlots) {
				slot, place = i+packSlots, 0
			}
		}
		if slot >= itemSlots {
			return -1
		}
	}
	it := p.Items[slot]
	if place > 0 {
		it.Qty = 0
		if place == 1 {
			p.ItemCount++
		} else {
			p.SwagCount++
		}
	}
	it.ID = int16(id)
	it.Type = rec.Type
	it.Icon = rec.Icon
	it.Weight = rec.Weight
	it.Qty += uint8(qty)
	it.Flags = 0
	for _, f := range [][2]uint8{{8, 4}, {4, 8}, {16, 16}, {32, 32}, {2, 64}} {
		if rec.Flags&f[0] != 0 {
			it.Flags |= f[1]
		}
	}
	p.Items[slot] = it
	calcCarriedWeight(member)
	return 0
}
