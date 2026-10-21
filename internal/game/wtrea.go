package game

import (
	"strconv"
	"strings"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

// WTREA: treasure after a fight (game state 15) and a chest found in the maze (21): the
// chest menu, its traps, the gold and the items. Both end in the character view (22).
//
// The overlay carries its own copy of WMAZE's maze view renderer (render_maze_view and the
// *_t helpers 9cd3..b131, fba83); the copies are identical, so the port calls drawMazeView.
// f9d8a (an event lookup) is a dead copy nothing calls.

// ovmain (WTREA 4572).
func wtreaOvmain() {
	trBlink = -1
	trChestOpen = 0
	trLidFrame = -1
	trTrapFrame = -1
	switch gameState {
	case 15:
		gameState = 22
		treasureAfterCombat()
	case 21:
		gameState = 22
		treasureFound()
	default:
		gameState = 22
	}
}

// WTREA data segment.
var (
	trBlink       = -1 // g501c: the blinking label shown next (0, 1), -1 none
	trChestOpen   int  // g501e: 1 draws the open chest
	trLidFrame    = -1 // g5020: the lid animation step, -1 none
	trTrapFrame   = -1 // g5022: the trap animation step, -1 none
	trChestKind   int  // g5024: the chest picture (-1 none, 0, 1)
	trChestPhase  int  // g5026: the chest's idle frame
	trTrapAnim    int  // g5028: which trap animation
	trTrapMember  int  // g502a: the member the trap aims at
	trAnimCounter int  // g502c: counts to 10 between animation frames

	trTextWin *Window // g502e: the text window
	trTextRow int     // g5030
	trTextCol int     // g5032

	trBlinkWin   *Window   // g5038
	trBlinkX     int       // g5036
	trBlinkY     int       // g5034
	trBlinkColor [2]int    // g503a, g503c
	trBlinkText  [2]string // 0x503e, 0x5056 (24 bytes each)

	trLine     string // 0x506e: the line being built (20 bytes)
	trKeys     string // 0x50be: message 5100, the menu keys (Esc, left, up, right, down, Enter)
	trRow      int    // g50c8: the row of party_picture_win the results go to
	trTrapName string // 0x50ca: the name of the chest's trap
	trTrapPool []byte // 0x50de (g50de): its letters, '$' once guessed
	trNum      string // 0x50f2 (g50f2)
	trNum2     string // 0x5142 (g5142)
)

// trTextControls are print_text_line's control characters (0x4fee).
const trTextControls = "!%#&]@"

// animate_chest_lid (WTREA 45de) draws lid step n of the chest picture (slot 0) and returns
// the next step, -1 after the last.
func animateChestLid(n int) int {
	frame, mirror := 0, 0
	switch n {
	case 0:
		frame = 5
	case 1:
		frame = 3
	case 2:
		frame, mirror = 3, 1
	case 3:
		frame = 2
	case 4:
		frame, mirror = 2, 1
	case 5:
		frame = 1
	case 6:
		frame, mirror = 1, 1
	case 7:
		return -1
	}
	p := &monsterPics[0]
	drv.DrawFrames(0, int(p.Pairs[frame][0]), int(p.Unk57[frame])-8, 72, 248, mirror, false, []byte{byte(frame)})
	return n + 1
}

// trTrapPos (ROOT g038e): where each trap animation's frames go, [anim][facing][frame] x, y.
var trTrapPos = [16][2][2][3][2]uint8{}

func init() {
	b := []uint8{80, 61, 102, 62, 121, 64, 208, 61, 186, 62, 175, 64, 120, 64, 97, 62, 79, 61, 176, 64, 191, 62, 210, 61, 75, 59, 93, 58, 118, 62, 213, 59, 195, 58, 177, 62, 122, 62, 107, 58, 84, 59, 174, 62, 182, 58, 206, 59, 73, 72, 97, 72, 118, 72, 215, 72, 190, 72, 179, 72, 119, 72, 101, 72, 78, 72, 175, 72, 188, 72, 210, 72, 84, 61, 107, 63, 125, 64, 212, 61, 190, 63, 179, 64, 124, 64, 102, 63, 83, 61, 179, 64, 193, 63, 214, 61, 89, 62, 105, 64, 125, 65, 211, 62, 192, 64, 177, 65, 124, 65, 103, 64, 88, 62, 180, 65, 191, 64, 211, 62, 86, 63, 108, 64, 123, 65, 210, 63, 196, 64, 181, 65, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 124, 64, 104, 63, 77, 62, 180, 64, 200, 63, 220, 62, 81, 68, 105, 68, 125, 69, 215, 68, 191, 68, 179, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 125, 69, 106, 68, 79, 68, 179, 69, 189, 68, 217, 68, 84, 68, 116, 67, 131, 67, 212, 68, 188, 67, 181, 67, 130, 67, 107, 67, 77, 68, 182, 67, 197, 67, 219, 68, 90, 63, 108, 66, 128, 59, 214, 63, 195, 66, 176, 59, 128, 59, 108, 66, 89, 63, 176, 59, 196, 66, 216, 63, 86, 62, 110, 65, 127, 59, 218, 62, 196, 65, 177, 59, 127, 59, 110, 65, 83, 62, 177, 59, 195, 65, 220, 62, 85, 59, 114, 61, 127, 64, 219, 59, 190, 61, 177, 64, 127, 64, 114, 61, 84, 59, 177, 64, 190, 61, 219, 59, 91, 60, 106, 66, 128, 60, 214, 60, 190, 66, 176, 60, 128, 60, 106, 66, 91, 60, 176, 60, 190, 66, 214, 60, 85, 64, 109, 64, 124, 68, 210, 64, 195, 64, 180, 68, 127, 68, 107, 64, 81, 64, 178, 68, 197, 64, 215, 64, 81, 60, 102, 63, 124, 65, 208, 60, 186, 63, 172, 65, 124, 65, 102, 63, 79, 60, 172, 65, 186, 63, 210, 60}
	for i, v := range b {
		trTrapPos[i/24][i/12%2][i/6%2][i/2%3][i%2] = v
	}
}

// animate_chest_trap (WTREA 4695) draws frame step of trap animation anim (pictures in slots
// 7..9) aimed at member (-1 the party) and returns the next step, -1 after the last.
func animateChestTrap(member, anim, step int) int {
	if step == 3 {
		return -1
	}
	slot, sub := 7, anim
	switch {
	case anim >= 13:
		slot, sub = 9, anim-13
	case anim >= 5:
		slot, sub = 8, anim-5
	}
	symmetric := anim == 6 || anim == 8 || anim >= 10 && anim <= 13 || anim == 15
	right, mirror := 0, 1
	if symmetric {
		mirror = 0
	}
	if member != -1 && member%2 != 0 {
		right, mirror = 1, 1-mirror
	}
	p := &monsterPics[slot]
	frame := p.Quads[p.Unk104[sub]][1] + uint8(step)
	pos := trTrapPos[anim][1][right][step]
	drv.DrawFrames(slot, int(pos[0]), int(pos[1]), 72, 248, mirror, false, []byte{frame})
	return step + 1
}

// animate_treasure_view (WTREA 4811) draws the next frame of the view: the maze, the chest
// (open or idle), the lid and the trap.
func animateTreasureView() {
	switch trChestKind {
	case 0:
		trChestPhase = (trChestPhase + 1) % 3
	case 1:
		trChestPhase = (trChestPhase + 1) % 2
	}
	drawMazeView()
	if trChestOpen == 1 {
		f := trChestKind*3 + trChestPhase + 1
		p := &monsterPics[3]
		drv.DrawFrames(3, int(p.Pairs[f][0]), int(p.Unk57[f]), 72, 248, 0, false, []byte{byte(f)})
	}
	if trLidFrame >= 0 {
		trLidFrame = animateChestLid(trLidFrame)
	}
	if trTrapFrame >= 0 {
		trTrapFrame = animateChestTrap(trTrapMember, trTrapAnim, trTrapFrame)
	}
	drvShowWindow(pictureWin)
	if trLidFrame == 1 {
		playSound(5)
	}
}

// trAnimate is one idle step of the treasure view: every 10th call flips the animation phase
// and draws a frame, else an ambient sound or a pause.
func trAnimate() {
	if trAnimCounter == 0 {
		animPhase = (animPhase + 1) % 2
		animateTreasureView()
	} else {
		idleStep()
	}
	trAnimCounter = (trAnimCounter + 1) % 10
}

// wait_animating (WTREA 48df) animates for seconds, less on Enter or a click.
func trWaitAnimating(seconds int) {
	flushKeys()
	for n := seconds * 5; n > 0; n-- {
		if checkScreenClick() || keyAvailable(1) && int8(getKey(1)) == 13 {
			return
		}
		trAnimate()
	}
}

// wait_more_prompt (WTREA 4950) blinks the more mark until Enter or a click.
func trWaitMorePrompt() {
	blink := 3
	flushKeys()
	for {
		if keyAvailable(1) {
			if int8(getKey(1)) == 13 {
				return
			}
			continue
		}
		if checkScreenClick() {
			return
		}
		if trAnimCounter == 0 {
			animPhase = (animPhase + 1) % 2
			animateTreasureView()
			setCursor(trTextWin, 39, 4)
			if blink < 2 {
				putChar(trTextWin, ' ', 1)
			} else {
				putCharHi(trTextWin, 126, 1)
			}
			blink = (blink + 1) % 4
		} else {
			idleStep()
		}
		trAnimCounter = (trAnimCounter + 1) % 10
	}
}

// end_text_page (WTREA 4a03): a timed pause (timed 1), or the more prompt and a clean page.
func trEndTextPage(timed int) {
	if timed == 1 {
		trWaitAnimating(messageDelay)
		return
	}
	trWaitMorePrompt()
	clearWindow(trTextWin, true)
	trTextRow = 0
}

// print_text_line (WTREA 4a31) is WMAZE's print_text_line without the sound control '~'.
func trPrintTextLine(s string) {
	for len(s) > 0 {
		k := strings.IndexByte(trTextControls, s[0])
		if k < 0 {
			break
		}
		switch k {
		case 0:
			s = s[1:]
			trTextRow++
		case 1:
			s = s[1:]
			trWaitAnimating(messageDelay)
		case 2:
			s = s[min(2, len(s)):]
		case 3:
			s = s[1:]
		case 4:
			s = s[1:]
			clearWindow(trTextWin, true)
			trTextRow = 0
		case 5:
			s = s[1:]
			trEndTextPage(0)
		}
	}
	if trTextRow > 4 {
		trEndTextPage(0)
	}
	col := 20 - (len(s)+1)/2
	if len(s) > 0 {
		switch s[0] {
		case '$':
			s, col = s[1:], 0
		case '^':
			s, col = s[1:], trTextCol
		}
	}
	setCursor(trTextWin, col, trTextRow)
	trTextCol = col
	trTextRow++
	printString(trTextWin, s, 5) // g0002
}

// show_message_text (WTREA 4b36) shows messages msg, msg+1, ... up to the first missing one.
func trShowMessageText(msg, timed int) {
	trTextCol, trTextRow = 0, 0
	trTextWin = createWindow(0, 19, 40, 5, 20, -1, 0, true)
	for {
		s := getMessage(msg)
		if msgNotFound {
			break
		}
		trPrintTextLine(s)
		msg++
	}
	trEndTextPage(timed)
	destroyWindow(trTextWin, false)
}

// blink_selection (WTREA 4bb5) shows the next of the two blink labels at the blink spot.
func blinkSelection() {
	setCursor(trBlinkWin, trBlinkX, trBlinkY)
	printColored(trBlinkWin, trBlinkText[trBlink], trBlinkColor[trBlink])
	trBlink = (trBlink + 1) % 2
}

// get_chest_input (WTREA 4c2f) animates until a key of keys (its index) or a click in
// regions (the region index; menuClicked set).
func getChestInput(keys string, regions []byte) int {
	phase := animPhase
	if trBlink != -1 {
		blinkSelection()
	}
	menuClicked = 0
	for {
		if keyAvailable(1) {
			if k := strings.IndexByte(keys, byte(getKey(1))); k >= 0 {
				return k
			}
			continue
		}
		k := checkClickRegions(regions)
		if clickFlag {
			menuClicked = 1
			return k - 1
		}
		if trAnimCounter == 0 {
			animPhase = (animPhase + 1) % 2
			animateTreasureView()
			if trBlink != -1 && animPhase == phase {
				blinkSelection()
			}
		} else {
			idleStep()
		}
		trAnimCounter = (trAnimCounter + 1) % 10
	}
}

// trInput is get_chest_input with the menu keys and the click regions.
func trInput() int { return getChestInput(trKeys, clickRegions[:]) }

// print_chest_prompt (WTREA 5097) shows message msg at column x of the message line.
func printChestPrompt(msg, x int) {
	eraseWindow(mainWindow, false)
	eraseCursorMark(mainWindow, false)
	deferWindowDraw(mainWindow)
	trLine = strings.Repeat(" ", 20)
	setCursor(mainWindow, 20, 0)
	printStringHi(mainWindow, trLine, 3)
	setCursor(mainWindow, x, 0)
	printMessage(msg, mainWindow, 18)
	mainWindow.CurX++
	resumeWindowDraw(mainWindow)
}

// print_option_cell (WTREA 512e) redraws option i of a column menu plain.
func printOptionCell(win *Window, i, width, rows, x, y int, s string) {
	setCursor(win, i/rows*width+x, i%rows+y)
	printStringHi(win, s, 3)
}

// chest_menu_select (WTREA 5169) is WMAZE's menu: the messages firstMsg+k whose enabled[k]
// is 1 (-1 ends the list) in columns of rows; withCancel adds the message line's exit. It
// returns the chosen k, -1 for exit.
func chestMenuSelect(win *Window, firstMsg int, enabled []int, x0, y0, width, rows, color, withCancel int) int {
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
		x, y := n/rows*width+x0, n%rows+y0
		setCursor(win, x, y)
		s := getMessage(firstMsg + k)
		if !mousePresent && withCancel == 0 && n == 0 {
			printString(win, s, -color)
		} else {
			printStringHi(win, s, 3)
		}
		labels, keys = append(labels, s), append(keys, k)
		setClickRegion(n, win.X+byte(x), win.Y+byte(y), byte(len(s)+int(win.X)+x-1), win.Y+byte(y))
	}
	n := len(labels)
	setClickRegion(n, 255, 0, 0, 0)
	if withCancel == 1 {
		setClickRegion(n, mainWindow.X+byte(exitX), mainWindow.Y+byte(exitY), byte(len(exitLo)+int(mainWindow.X)+exitX-1), mainWindow.Y+byte(exitY))
		setClickRegion(n+1, 255, 0, 0, 0)
	}
	sel := 0
	if withCancel == 1 {
		sel = -1
	}
	plain := func() { printOptionCell(win, sel, width, rows, x0, y0, labels[sel]) }
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printString(mainWindow, exitLo, -color)
			} else {
				setCursor(win, sel/rows*width+x0, sel%rows+y0)
				printString(win, labels[sel], -color)
			}
		}
		key := trInput()
		if menuClicked == 1 {
			sel = -1
			if key < n {
				sel = key
			}
			key = 5
		} else {
			switch key {
			case 1:
				if sel != -1 && sel-rows >= 0 {
					plain()
					sel -= rows
				}
			case 2:
				if sel != -1 {
					if sel%rows > 0 {
						plain()
						sel--
					} else if withCancel == 1 {
						plain()
						sel = -1
					}
				}
			case 3:
				if sel != -1 && sel+rows < n {
					plain()
					sel += rows
				}
			case 4:
				if sel == -1 {
					setCursor(mainWindow, exitX, exitY)
					printStringHi(mainWindow, exitHi, 3)
					sel = 0
				} else if sel%rows < rows-1 && sel+1 < n {
					plain()
					sel++
				}
			}
		}
		if key == 5 {
			break
		}
	}
	if sel == -1 {
		return -1
	}
	return keys[sel]
}

// choose_party_member (WTREA 5649) lets the player pick a member under prompt msg; -1 exit.
// refreshAll 0 refreshes only the prompt and the picker.
func trChooseMember(msg, refreshAll int) int {
	if partyCount < 2 {
		return 0
	}
	win := createWindow(0, 19, 19, 5, 25, -4, 0, false)
	fillWindow(win, ' ', 3)
	exitHi, exitLo := getMessage(2027), getMessage(2028)
	printChestPrompt(msg, 10-(len(getMessage(msg))+6)/2)
	exitX, exitY := int(mainWindow.CurX), int(mainWindow.CurY)
	if refreshAll == 0 {
		refreshWindow(mainWindow)
		refreshWindow(win)
	} else {
		refreshAllWindows()
	}
	setCursor(mainWindow, exitX, exitY)
	if mousePresent {
		printStringHi(mainWindow, exitHi, 3)
	} else {
		printString(mainWindow, exitLo, -5)
	}
	for m := 0; m < partyCount; m++ {
		x, y := m%2*9+2, m/2+1
		setCursor(win, x, y)
		name := memberName(m)
		printStringHi(win, name, 3)
		setClickRegion(m, win.X+byte(x), win.Y+byte(y), byte(len(name)+int(win.X)+x-1), win.Y+byte(y))
	}
	setClickRegion(partyCount, mainWindow.X+byte(exitX), mainWindow.Y+byte(exitY), byte(len(exitLo)+int(mainWindow.X)+exitX-1), mainWindow.Y+byte(exitY))
	setClickRegion(partyCount+1, 255, 0, 0, 0)
	sel := -1
	for {
		if !mousePresent {
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printString(mainWindow, exitLo, -5)
			} else {
				setCursor(win, sel%2*9+2, sel/2+1)
				printString(win, memberName(sel), -5)
			}
		}
		key := trInput()
		if menuClicked == 1 {
			sel = -1
			if key < partyCount {
				sel = key
			}
			break
		}
		switch key {
		case 1:
			if sel != -1 && sel%2 > 0 {
				drawMemberName(win, sel)
				sel--
			}
		case 2:
			if sel != -1 {
				drawMemberName(win, sel)
				if sel -= 2; sel < 0 {
					sel = -1
				}
			}
		case 3:
			if sel != -1 && sel%2 < 1 && sel+1 < partyCount {
				drawMemberName(win, sel)
				sel++
			}
		case 4:
			if sel == -1 {
				setCursor(mainWindow, exitX, exitY)
				printStringHi(mainWindow, exitHi, 3)
				sel = 0
			} else if sel+2 < partyCount {
				drawMemberName(win, sel)
				sel += 2
			}
		}
		if key == 5 {
			break
		}
	}
	destroyWindow(win, false)
	return sel
}

// show_centered_line (WTREA 59e9) replaces row y of win with s centred, beeps if asked and
// waits the message delay.
func showCenteredLine(win *Window, y int, s string, beep bool, color int) {
	clearWindowArea(win, 0, y, 39, y)
	trWaitAnimating(1)
	setCursor(win, 20-(len(s)+1)/2, y)
	printString(win, s, color)
	if beep {
		playSound(0)
	}
	trWaitAnimating(messageDelay)
}

// print_member_message (WTREA 5af6) is WMAZE's, on party_picture_win.
func trPrintMemberMessage(m int, s1, s2, s3 string, c0, c1, c2, c3 int) {
	w := partyPictureWin
	clearWindow(w, true)
	name := memberName(m)
	gaps := 1
	for _, s := range []string{s1, s2, s3} {
		if s != "" {
			gaps++
		}
	}
	setCursor(w, 20-(len(name)+len(s1)+len(s2)+len(s3)+gaps)/2, 1)
	printString(w, name, c0)
	w.CurX++
	for i, s := range []string{s1, s2, s3} {
		if s != "" {
			printString(w, s, []int{c1, c2, c3}[i])
			w.CurX++
		}
	}
}

// inflict_condition (WTREA 5c0f) is WMAZE's with the treasure view's messages.
func trInflictCondition(m, msg, resist, cond int) {
	p := &party[m]
	if (resist == 0 || random(100) > int(p.Resist[resist])) && int(p.Condition) < cond {
		p.Condition = uint8(cond)
		if cond >= 3 {
			p.HP = 0
			p.Stamina = 0
			playSound(int(p.Sex) + 8)
		}
		trPrintMemberMessage(m, getMessage(msg), "", "", 1, 6, 0, 0)
		drawMemberStatus(m)
		trWaitAnimating(messageDelay)
	}
}

// trap_damage_member (WTREA 5cc3) is WMAZE's with the treasure view's messages; a death
// waits 4 seconds, not 2.
func trTrapDamageMember(m, count, sides int) {
	n := rollDice(formatDice(count, sides, 0))
	p := &party[m]
	if n == 0 || p.Condition >= 3 {
		return
	}
	num := strconv.Itoa(n)
	pre, post := getMessage(5231), getMessage(5232)
	if int(p.HP) < n {
		n = int(p.HP)
	}
	p.HP -= int16(n)
	trPrintMemberMessage(m, pre, num, post, 1, 5, 5, 5)
	drawMemberStatus(m)
	if p.HP <= 0 {
		trWaitAnimating(4)
		trInflictCondition(m, 5264, 0, 3)
	} else {
		playSound(2)
		trWaitAnimating(messageDelay)
	}
}

// inflict_status (WTREA 5dbb) is WMAZE's, except that the resistance save ignores count.
func trInflictStatus(m, msg, resist, status, count, sides, base int) {
	p := &party[m]
	if p.Condition >= 2 {
		return
	}
	n := rollDice(formatDice(count, sides, base))
	if resist != 0 && random(100) <= int(p.Resist[resist]) {
		return
	}
	if base == 255 {
		p.Status[status] = 0xff
	} else {
		if int(p.Status[status])+n > 250 {
			n = 250 - int(p.Status[status])
		}
		p.Status[status] += uint8(n)
	}
	if status == 5 {
		p.PoisonStr = uint8(count)
	}
	trPrintMemberMessage(m, getMessage(msg), "", "", 1, 6, 0, 0)
	drawMemberStatus(m)
	trWaitAnimating(messageDelay)
}
