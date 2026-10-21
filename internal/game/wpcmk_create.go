package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// WPCMK: making a character: race, sex, class, bonus points, karma, portrait, skills,
// starting spells and items.

// WPCMK data: the class and race tables (ds:4ffd..5459). Stat strings are 8 letters
// STR..KAR, 'A' = 0.
var (
	// classSkillMasks are per class the skills a class can learn ('1'): weaponry (skills
	// 0..9), physical (10..16), personal (17..21) and academia (22..29).
	classSkillMasks = [14][4]string{
		{"1111111110", "0100000", "00000", "11100000"},
		{"1000111100", "0101000", "00000", "11100010"},
		{"0001101010", "0101000", "00000", "11101000"},
		{"1111111110", "0101111", "00000", "11100000"},
		{"1111111110", "0100111", "00000", "11110000"},
		{"1001110000", "0100000", "00000", "11110000"},
		{"1111111110", "0111111", "00000", "11100010"},
		{"1001111000", "0101000", "00000", "11100100"},
		{"1111111110", "0101000", "00000", "11101000"},
		{"0001101010", "0101000", "00000", "11101010"},
		{"1111111110", "0101000", "00000", "11101000"},
		{"1111111110", "0101000", "00000", "11100011"},
		{"1001111101", "0101001", "00000", "11100101"},
		{"1111111111", "0100111", "00000", "11110001"},
	}
	// classMinimums are the stats each class needs (the C has two equal copies, for
	// checking and for raising).
	classMinimums = [14]string{
		"MAAAAAAA", "AMAAAAAA", "AAMAAAIA", "AAAAMIAA", "KIILKIIA", "ANAANAAA", "AKAAMIMA",
		"KOAOAAKA", "KALLKLIA", "APPAAAIA", "MJMMJJOA", "MLAJMOIA", "NINAKNIA", "MKKMMMAA",
	}
	// raceBase are the races' base stats.
	raceBase = [11]string{
		"JIIJJIIA", "HKKHJJIA", "LGKMHHHA", "KHNKIGGA", "IHGJKHNA", "FLGGKOMA", "MFFOIKDA",
		"KHGMKIGA", "HKHHKMKA", "IGMKIIKA", "KKGKHHJA",
	}
	// classResists are the resistances, 'A' = 0 in steps of 4, by class.
	classResists = [14]string{
		0: "ACBCBCCCCHCHBA", 4: "ACCCCDCCCHHKDA", 8: "ACCCCDCCCHHKDA", 10: "ACCCCDCCCIFKEA",
		11: "ACHCHDCCCHFIDA", 13: "ACKCHHGCCIECBA", 1: "ACBCCDECCEEEGA", 5: "ACBCCDECCEEEGA",
		7: "ACICIBHCCFKFFA", 2: "ACECCEGCCIFKFA", 9: "ACECCEGCCIFKFA", 12: "ACKCGDHCCFKIDA",
		3: "ACHCHHGCCCECHA", 6: "ACHCHHGCCCECHA",
	}
	// classItems are the items each class starts with (give_class_items).
	classItems = [14][5]int{
		{8, 135, 132, 130, 141}, {18, 122, 123, 130, 335}, {24, 122, 123, 130, 316},
		{6, 120, 121, 131, 27}, {31, 33, 126, 127, 131}, {18, 122, 123, 130, 326},
		{29, 30, 120, 121, 55}, {1, 122, 123, 130, 241}, {22, 124, 125, 130, 138},
		{24, 122, 123, 130, 163}, {9, 133, 134, 131, 143}, {10, 4, 122, 123, 130},
		{25, 47, 122, 123, 130}, {47, 160, 158, 159, 161},
	}
)

// Race, sex and class numbers the rules test.
const (
	raceHuman  = 0
	raceFaerie = 5
	raceLizard = 6
	sexFemale  = 1
	classMonk  = 12
	classNinja = 13
	classValk  = 8
)

// setClassSkillMasks (WPCMK 55ad) marks the skills the class can learn, or already has.
func setClassSkillMasks(masks [4]string) {
	base := [4]int{0, 10, 17, 22}
	for g, m := range masks {
		for i := range len(m) {
			s := base[g] + i
			pcmkSkillShown[s] = m[i] == '1' || newPC.Skills[s] > 0
		}
	}
}

// printSkillValue (WPCMK 5691) prints v at column 16 of row.
func printSkillValue(win *Window, row, v int) {
	setCursor(win, 16, row)
	printInt(win, v, 3, 1)
}

// drawSkillPage (WPCMK 5b3b) draws skill page page (weaponry, physical, personal,
// academia) into win; the skills listed, in order.
func drawSkillPage(win *Window, page int) []int {
	setClassSkillMasks(classSkillMasks[newPC.Class])
	fillWindow(win, 0, 1, false)
	g := func(x, y int, ch byte) { setFrameGlyph(win, x, y, ch) }
	g(0, 0, 14)
	g(19, 0, 10)
	g(0, 15, 11)
	g(19, 15, 8)
	for i := 1; i < 19; i++ {
		g(i, 0, 12)
		g(i, 2, 12)
		g(i, 13, 12)
		g(i, 15, 7)
	}
	for i := 1; i < 15; i++ {
		g(0, i, 13)
		g(19, i, 5)
	}
	g(0, 2, 33)
	g(0, 9, 15)
	g(0, 11, 15)
	g(0, 13, 16)
	g(19, 2, 10)
	g(19, 13, 10)
	deferWindowDraw(win)
	setCursor(win, 1, 1)
	printStringHi(win, "                  ", 3)
	setCursor(win, 6, 1)
	printMessage(page+600, win, 18)
	pages := [4]struct {
		left, right byte
		first, last int
		color       int
	}{{1, 1, 0, 9, 2}, {36, 37, 10, 16, 14}, {36, 37, 17, 21, 12}, {33, 33, 22, 29, 11}}
	p := pages[page]
	putGlyphAtC4(win, 2, 1, p.left+1)
	putGlyphAtC4(win, 17, 1, p.right+1)
	var list []int
	for i := p.first; i <= p.last; i++ {
		if pcmkSkillShown[i] {
			setCursor(win, 1, len(list)+3)
			printMessage(i+5500, win, p.color)
			setCursor(win, 16, len(list)+3)
			printInt(win, int(newPC.Skills[i]), 3, 1)
			list = append(list, i)
		}
	}
	setCursor(win, 1, 14)
	printMessage(5530, win, 9)
	setCursor(win, 16, 14)
	printInt(win, int(newPC.SkillPoints), 3, 1)
	resumeWindowDraw(win)
	refreshWindow(win)
	return list
}

// distributeSkillPoints (WPCMK 56bc) lets the player spend skill points on the listed
// skills of the page in skillWin; win holds the instructions.
func distributeSkillPoints(win, skillWin *Window, skills []int) {
	points := int(newPC.SkillPoints)
	count := len(skills)
	fillWindow(win, 32, 3, false)
	refreshWindow(win)
	setCursor(win, 7, 1)
	printMessage(610, win, 18)
	setCursor(win, 2, 2)
	printMessage(606, win, 18)
	win.CurX += 3
	printMessage(607, win, 18)
	var added [10]int
	row := 0
	x, y := win.X, win.Y
	setClickRegion(0, x+2, y+2, x+2, y+2)
	setClickRegion(1, x+3, y+2, x+3, y+2)
	setClickRegion(2, x+22, y+2, x+22, y+2)
	setClickRegion(3, x+23, y+2, x+23, y+2)
	setClickRegion(4, x+13, y+3, x+13, y+3)
	setClickRegion(5, 255, 0, 0, 0)
	mark := func(ch byte, color int) {
		setCursor(skillWin, 15, row+3)
		putChar(skillWin, ch, color)
	}
	mark(100, 4)
	for {
		setCursor(win, 7, 3)
		msg, ex := 609, byte(18)
		if points > 0 {
			msg, ex = 608, 13
		}
		printMessage(msg, win, 18)
		setClickRegion(4, x+ex, y+3, x+ex, y+3)
		key := getMenuInput(menuKeys, clickRegions[:])
		if menuClicked == 1 && key >= 0 && key <= 4 {
			key = [5]int{1, 3, 2, 4, 5}[key]
		}
		s := &newPC.Skills[skills[row]]
		switch key {
		case 1:
			if added[row] > 0 {
				*s--
				added[row]--
				points++
				playSound(1)
				printSkillValue(skillWin, row+3, int(*s))
				printSkillValue(skillWin, 14, points)
			} else {
				playSound(0)
			}
		case 2:
			mark(32, 7)
			if row > 0 {
				row--
			} else {
				row = count - 1
			}
			mark(100, 4)
		case 3:
			if *s < 100 && points > 0 {
				*s++
				added[row]++
				points--
				playSound(1)
				printSkillValue(skillWin, row+3, int(*s))
				printSkillValue(skillWin, 14, points)
			} else {
				playSound(0)
			}
		case 4:
			mark(32, 7)
			if row+1 < count {
				row++
			} else {
				row = 0
			}
			mark(100, 4)
		}
		if key == 5 {
			break
		}
	}
	mark(32, 7)
	newPC.SkillPoints = uint8(points)
}

// skillPointsScreen (WPCMK 604d) goes through the skill pages until the skill points are
// spent; the personal page only when the character has a personal skill.
func skillPointsScreen(win *Window) {
	fillWindow(win, 32, 3, false)
	refreshWindow(win)
	skillWin := createWindow(20, 4, 20, 16, 25, -1, 0, false)
	personal := false
	for i := 17; i < 22; i++ {
		if newPC.Skills[i] > 0 {
			personal = true
		}
	}
	for page := 0; ; {
		list := drawSkillPage(skillWin, page)
		distributeSkillPoints(win, skillWin, list)
		page = (page + 1) % 4
		if page == 2 && !personal {
			page++
		}
		if newPC.SkillPoints == 0 {
			break
		}
	}
	destroyWindow(skillWin, false)
}

// raiseStatsToMinimum (WPCMK 71df) raises each stat to its minimum in min.
func raiseStatsToMinimum(min string) {
	for i := range 8 {
		if v := min[i] - 'A'; newPC.Stats[i] < v {
			newPC.Stats[i] = v
		}
	}
}

// checkClassEligible (WPCMK 7212) sets whether the bonus points pay for the class's
// minimums; with apply it raises the stats, paying the points.
func checkClassEligible(class int, min string, apply bool) {
	points := bonusPoints
	for stat := range 8 {
		for v := int(newPC.Stats[stat]); int(min[stat])-'A' > v; v++ {
			points--
			if apply {
				newPC.Stats[stat]++
				bonusPoints--
				drawStats()
			}
		}
	}
	classEligible[class] = points >= 0
}

// computeEligibleClasses (WPCMK 7274) sets classEligible; only women may be valkyries.
func computeEligibleClasses() {
	for c := range 14 {
		if c == classValk && newPC.Sex != sexFemale {
			classEligible[c] = false
			continue
		}
		checkClassEligible(c, classMinimums[c], false)
	}
}

// applySexModifiers (WPCMK 75d2): women have 2 less strength, 1 more personality and karma.
func applySexModifiers(sex int) {
	if sex == sexFemale {
		newPC.Stats[0] -= 2
		newPC.Stats[6]++
		newPC.Stats[7]++
	}
}

// chooseFromMenu shows the prompt msg at column x, the status title status and the menu
// of messages first.. in menuWin; the chosen one.
func chooseFromMenu(x, msg, status, first int, enabled []int) int {
	fillWindow(promptWin, 32, 3, false)
	refreshWindow(promptWin)
	setCursor(promptWin, x, 1)
	printMessage(msg, promptWin, 18)
	fillWindow(menuWin, 32, 3, false)
	refreshWindow(menuWin)
	printStatusCentered(status)
	c := menuSelect(menuWin, first, enabled, 1, 1, 10, 11, 5, 0)
	clearStatusLine()
	fillWindow(menuWin, 32, 3, false)
	refreshWindow(menuWin)
	return c
}

// chooseRace (WPCMK 75f1) asks for the race and sets its base stats.
func chooseRace() {
	r := chooseFromMenu(9, 1104, 1116, 100, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1})
	newPC.Race = uint8(r)
	setCursor(sheetWin, 15, 1)
	printMsgColor(int(newPC.Race)+100, sheetWin, 1)
	raiseStatsToMinimum(raceBase[r])
	drawStats()
}

// chooseSex (WPCMK 770a) asks for the sex.
func chooseSex() {
	s := chooseFromMenu(10, 1105, 1117, 140, []int{1, 1, -1})
	newPC.Sex = uint8(s)
	setCursor(sheetWin, 13, 1)
	putChar(sheetWin, getMessage(int(newPC.Sex) + 140)[0], 1)
	putChar(sheetWin, 45, 1)
	applySexModifiers(s)
	drawStats()
}

// chooseClass (WPCMK 7845) asks for one of the classes the bonus points allow and raises
// the stats to its minimums.
func chooseClass() {
	computeEligibleClasses()
	enabled := make([]int, 15)
	for c := range 14 {
		if classEligible[c] {
			enabled[c] = 1
		}
	}
	enabled[14] = -1
	c := chooseFromMenu(7, 1106, 1118, 120, enabled)
	newPC.Class = uint8(c)
	printMsgAt(int(newPC.Class)+120, sheetWin, 13, 2, 1)
	checkClassEligible(c, classMinimums[c], true)
	drawStats()
}

// distributeBonusPoints (WPCMK 7969) lets the player spend the remaining bonus points on
// the stats. The cursor only reaches the first seven; karma is cast after.
func distributeBonusPoints() {
	w := promptWin
	fillWindow(w, 32, 3, false)
	refreshWindow(w)
	setCursor(w, 7, 1)
	printMessage(1120, w, 18)
	setCursor(w, 1, 2)
	printMessage(1108, w, 18)
	w.CurX += 2
	printMessage(1109, w, 18)
	var added [8]int
	stat := 0
	mark := func(ch byte) {
		setCursor(sheetWin, 7, stat+5)
		putChar(sheetWin, ch, 7)
	}
	mark(98)
	x, y := w.X, w.Y
	setClickRegion(0, x+1, y+2, x+1, y+2)
	setClickRegion(1, x+2, y+2, x+2, y+2)
	setClickRegion(2, x+21, y+2, x+21, y+2)
	setClickRegion(3, x+22, y+2, x+22, y+2)
	setClickRegion(4, 255, 0, 0, 0)
	for {
		setCursor(w, 12, 3)
		if bonusPoints == 0 {
			printMessage(1110, w, 18)
			setClickRegion(4, x+18, y+3, x+18, y+3)
			setClickRegion(5, 255, 0, 0, 0)
		} else {
			for w.CurX < 38 {
				putCharHi(w, 32, 3)
			}
			setClickRegion(4, 255, 0, 0, 0)
		}
		key := getMenuInput(menuKeys, clickRegions[:])
		if menuClicked == 1 && key >= 0 && key <= 4 {
			key = [5]int{1, 3, 2, 4, 5}[key]
		}
		switch key {
		case 1:
			if added[stat] > 0 {
				newPC.Stats[stat]--
				added[stat]--
				bonusPoints++
				playSound(1)
				drawStats()
			} else {
				playSound(0)
			}
		case 2:
			mark(32)
			if stat > 0 {
				stat--
			} else {
				stat = 6
			}
			mark(98)
		case 3:
			if newPC.Stats[stat] < 18 && bonusPoints > 0 {
				newPC.Stats[stat]++
				added[stat]++
				bonusPoints--
				playSound(1)
				drawStats()
			} else {
				playSound(0)
			}
		case 4:
			mark(32)
			if stat < 6 {
				stat++
			} else {
				stat = 0
			}
			mark(98)
		}
		if key == 5 && bonusPoints > 0 {
			playSound(0)
		}
		if key == 5 && bonusPoints <= 0 {
			break
		}
	}
	mark(32)
}

// rollKarma (WPCMK 7d9b) rolls karma until Enter or a click (women get one more).
func rollKarma() {
	fillWindow(promptWin, 32, 3, false)
	refreshWindow(promptWin)
	setCursor(promptWin, 9, 1)
	printMessage(1111, promptWin, 18)
	for rolling := true; rolling; {
		newPC.Stats[7] = uint8(random(19))
		if newPC.Sex == sexFemale {
			newPC.Stats[7]++
		}
		setCursor(sheetWin, 5, 12)
		printInt(sheetWin, int(newPC.Stats[7]), 2, 1)
		if checkScreenClick() {
			rolling = false
		}
		if keyAvailable(1) && byte(getKey(1)) == 13 {
			rolling = false
		}
	}
	drawStats()
}

// initNPCAttitudes (WPCMK 7e92, with 7e5f) sets every NPC's attitude from karma: 40 less
// 4 per point away from 9.
func initNPCAttitudes() {
	for i := range newPC.NPCAttitude {
		d := int(newPC.Stats[7]) - 9
		if d < 0 {
			d = -d
		}
		newPC.NPCAttitude[i] = int8(40 - d<<2)
	}
}

// addStartingItem (WPCMK 8050) puts item id into the next inventory slot and recounts the
// load.
func addStartingItem(id int) {
	b := make([]byte, dbs.Size[ItemRec]())
	accessScenarioRecord(dbs.RecItem, id, b, false)
	rec, _ := dbs.Decode[ItemRec](b)
	it := &newPC.Items[newPC.ItemCount]
	newPC.ItemCount++
	*it = Item{
		ID:     int16(id),
		Weight: rec.Weight,
		Type:   rec.Type,
		Icon:   rec.Icon,
		Qty:    uint8(int(rec.Charges.Count)*int(rec.Charges.Sides) + int(rec.Charges.Base)),
	}
	for _, f := range [][2]uint8{{8, 4}, {4, 8}, {16, 16}, {32, 32}, {2, 64}} {
		if rec.Flags&f[0] != 0 {
			it.Flags |= f[1]
		}
	}
	newPC.Carried = 0
	for i := range int(newPC.ItemCount) {
		q := int16(max(newPC.Items[i].Qty, 1))
		newPC.Carried += newPC.Items[i].Weight * q
	}
}

// addClassRating (WPCMK 837b) adds delta to resistance i, kept in 0..125.
func addClassRating(i, delta int) {
	newPC.Resist[i] = uint8(min(max(int(newPC.Resist[i])+delta, 0), 125))
}

// initClassRaceModifiers (WPCMK 83f0) sets the resistances by class, race and stats, the
// bare hands, and the armour class.
func initClassRaceModifiers() {
	if s := classResists[newPC.Class]; s != "" {
		for i := range 14 {
			newPC.Resist[i] = 0
			addClassRating(i, int(s[i])<<2-260)
		}
	}
	vit := int(newPC.Stats[3])
	switch newPC.Race {
	case 0:
		addClassRating(11, 20)
	case 2, 4:
		addClassRating(6, vit/5*5)
		addClassRating(12, vit/5*5)
	case 3:
		addClassRating(12, vit/5*5)
	case 1:
		addClassRating(9, 20)
	case 5:
		for _, i := range []int{12, 4, 7, 10, 11} {
			addClassRating(i, 25)
		}
	case 6, 7:
		addClassRating(10, 25)
		addClassRating(8, 10)
		addClassRating(3, -20)
	case 8:
		addClassRating(7, 15)
		addClassRating(4, 20)
		addClassRating(3, 20)
	case 9:
		addClassRating(3, 20)
	case 10:
		addClassRating(12, 25)
		addClassRating(3, 15)
	}
	if vit > 15 {
		addClassRating(6, 5)
		addClassRating(11, 5)
	}
	if newPC.Stats[1] > 15 {
		addClassRating(9, 5)
	}
	if newPC.Stats[5] > 15 {
		addClassRating(7, 5)
		addClassRating(4, 5)
	}
	for h := range newPC.Hands {
		hd := &newPC.Hands[h]
		hd.B0D, hd.B0C = 0, 0
		hd.Data = [8]uint8{}
		hd.ToHit = 0
		hd.Damage = Dice{Base: 0, Count: 2, Sides: 2}
		hd.Flags = 32 // bare hand
	}
	hands := int(newPC.Skills[9]) / 5 // hands & feet
	kiri := int(newPC.Skills[29]) / 5 // kirijutsu
	h0 := &newPC.Hands[0]
	if newPC.Class == classNinja || newPC.Class == classMonk {
		h0.Damage.Sides = uint8(hands/3 + 2)
		h0.Flags |= 64
		h0.Data[6] = uint8(kiri + 5)
	} else {
		if hands > 0 {
			h0.Damage.Sides = uint8(hands/4 + 1)
			h0.Flags |= 64
		}
		if kiri > 0 {
			h0.Data[6] = uint8(kiri)
		}
	}
	if newPC.Stats[0] > 15 {
		h0.Damage.Base++
	}
	if newPC.Stats[0] > 17 {
		h0.Damage.Base++
	}
	if newPC.Stats[4] > 15 {
		h0.ToHit -= 5
	}
	if newPC.Stats[4] > 17 {
		h0.ToHit -= 5
		newPC.Hands[1].ToHit -= 5
	}
	newPC.AC[0] = 10
	if newPC.Stats[5] > 15 {
		newPC.AC[0]--
	}
	if newPC.Stats[5] > 17 {
		newPC.AC[0]--
	}
	if newPC.Race == raceFaerie {
		newPC.AC[0] -= 2
	}
	newPC.AC[2] = 0
	for i := 3; i < 8; i++ {
		newPC.AC[i] = newPC.AC[0]
	}
}

// initClassSkills (WPCMK 8786) rolls the skill points and the class's starting skills,
// which come out of the skill points.
func initClassSkills() {
	in, pi, vi := int(newPC.Stats[1]), int(newPC.Stats[2]), int(newPC.Stats[3])
	dx, sp := int(newPC.Stats[4]), int(newPC.Stats[5])
	newPC.SkillPoints = uint8(random(9) + 10)
	sk := &newPC.Skills
	roll := 0
	set := func(skill, n int) { roll = n; sk[skill] = uint8(n) }
	first := func(skill, n int) { set(skill, n); newPC.SkillPoints -= uint8(n) }
	switch newPC.Class {
	case 1:
		set(28, random(4)+in/3+3)
	case 2:
		set(26, random(4)+pi/3+3)
	case 3:
		set(15, random(4)+(in+dx)/2/3+3)
	case 4:
		first(11, random(3)+(in+vi)/2/5+2)
		set(7, random(3)+(dx+sp)/2/5+2)
	case 6:
		set(12, random(4)+(in+dx)/2/3+3)
	case 5:
		set(25, random(4)+in/3+3)
	case 7:
		set(27, random(4)+(in+pi)/2/3+3)
	case 9:
		first(28, random(3)+in/5+2)
		set(26, random(3)+pi/5+2)
	case 8:
		set(4, random(4)+(dx+sp)/2/3+3)
	case 10, 11:
		set(1, random(4)+(dx+sp)/2/3+3)
	case 12:
		first(9, random(3)+(dx+sp)/2/5+2)
		set(27, random(3)+(in+pi)/2/5+2)
	case 13:
		first(9, random(3)+(dx+sp)/2/5+2)
		set(16, random(3)+(dx+sp)/2/5+2)
	}
	if int(newPC.SkillPoints) > roll {
		newPC.SkillPoints -= uint8(roll)
	} else {
		newPC.SkillPoints = 0
	}
}

// initHPStamina (WPCMK 8aed) rolls age, hit points, stamina and carrying capacity.
func initHPStamina() {
	newPC.Unk1A6 = 1
	newPC.Unk1A7 = 10
	newPC.AgeDays = int32(random(1000) + 6570)
	str, vit := int(newPC.Stats[0]), int(newPC.Stats[3])
	var hp int
	switch newPC.Class {
	case 0:
		hp = random(5) + 6
	case 1:
		hp = random(3) + 2
	case 2, 12:
		hp = random(4) + 4
	case 3, 5, 9:
		hp = random(4) + 3
	case 4, 11:
		hp = random(4) + random(4) + 6
	case 6:
		hp = random(4) + 2
	case 7:
		hp = random(3) + 3
	case 8:
		hp = random(5) + 5
	case 10:
		hp = random(7) + 8
	case 13:
		hp = random(5) + 4
	}
	initClassSkills()
	stm := (vit<<1 + str) * 3
	load := (str<<1 + vit) * 3
	if vit < 8 {
		hp--
	}
	if vit > 15 {
		hp++
		stm += vit
	}
	if vit > 17 {
		hp++
		stm += vit
	}
	if str > 15 {
		load += str
	}
	if str > 17 {
		load += str
	}
	newPC.HP, newPC.HPMax = int16(hp), int16(hp)
	newPC.Stamina, newPC.StaminaMax = int16(stm), int16(stm)
	newPC.Carried = 0
	newPC.MaxLoad = int16(load * 15)
	if newPC.Race == raceFaerie {
		newPC.MaxLoad = newPC.MaxLoad << 1 / 3
	}
}

// setSixRatings (WPCMK 8de0) sets the six realms' spell regeneration, at least 1 each.
func setSixRatings(r [6]int) {
	for i, v := range r {
		newPC.SpellRegen[i] = uint8(max(v, 1))
	}
}

// initClassRatings (WPCMK 8e47) sets the spell regeneration and the spells to pick per
// spell book (mage, priest, alchemist, psionic).
func initClassRatings() {
	b := 0
	if newPC.Stats[3] > 15 {
		b++
	}
	if newPC.Race == raceFaerie {
		b++
	}
	if newPC.Race == raceLizard {
		b--
	}
	r := func(a ...int) {
		var v [6]int
		for i := range v {
			v[i] = b + a[i]
		}
		setSixRatings(v)
	}
	switch newPC.Class {
	case 1:
		newPC.SpellsLearn[0] = 2
		r(3, 3, 2, 2, 2, 3)
	case 2:
		newPC.SpellsLearn[1] = 2
		r(2, 2, 3, 3, 2, 3)
	case 5:
		newPC.SpellsLearn[2] = 2
		r(3, 2, 3, 3, 2, 2)
	case 7:
		newPC.SpellsLearn[3] = 2
		r(2, 2, 2, 2, 4, 2)
	case 9:
		newPC.SpellsLearn[0] = 1
		newPC.SpellsLearn[1] = 1
		r(1, 1, 1, 1, 3, 3)
	default:
		r(1, 1, 1, 1, 1, 1)
	}
}

// choosePortrait (WPCMK 9111) lets the player page through the 42 portraits.
func choosePortrait() {
	fillWindow(menuWin, 32, 3, false)
	refreshWindow(menuWin)
	fillWindow(promptWin, 32, 3, false)
	refreshWindow(promptWin)
	setCursor(promptWin, 9, 1)
	printMessage(1112, promptWin, 18)
	setCursor(promptWin, 12, 2)
	printMessage(1113, promptWin, 18)
	shown := -1
	printStatusCentered(1119)
	x, y := promptWin.X, promptWin.Y
	setClickRegion(0, x+9, y+1, x+9, y+1)
	setClickRegion(1, x+10, y+1, x+10, y+1)
	setClickRegion(2, x+18, y+2, x+18, y+2)
	setClickRegion(3, 255, 0, 0, 0)
	for {
		if int(newPC.Portrait) != shown {
			loadPortrait(int(newPC.Portrait), 0)
			shown = int(newPC.Portrait)
		}
		drawPortrait(menuWin, 0, 8, 3)
		k := getMenuInput(menuKeys, clickRegions[:])
		if menuClicked == 1 && k >= 0 && k <= 2 {
			k = [3]int{1, 3, 5}[k]
		}
		switch k {
		case 1:
			newPC.Portrait = uint8((int(newPC.Portrait) + 41) % 42)
		case 3:
			newPC.Portrait = uint8((int(newPC.Portrait) + 1) % 42)
		}
		if k == 5 {
			break
		}
	}
	clearStatusLine()
	fillWindow(menuWin, 32, 3, false)
	refreshWindow(menuWin)
	fillWindow(promptWin, 32, 3, false)
	refreshWindow(promptWin)
	drawPortrait(sheetWin, 0, 1, 1)
}

// initNewCharacter (WPCMK 9341) sets the new character's starting values.
func initNewCharacter() {
	newPC.Unk1A9 = 1
	newPC.Unk1AA = 1
	newPC.Unk1AB = newPC.Stats[5] + 1
	newPC.ToHit = 100
	initHPStamina()
	initClassRatings()
	newPC.Regen, newPC.Drain, newPC.LoadCat, newPC.Rank = 0, 0, 0, 0
	newPC.Level, newPC.Unk026 = 1, 1
	newPC.ItemCount = 0
	for i := range newPC.Equipped {
		newPC.Equipped[i] = 255
	}
	initClassRaceModifiers()
	drawStats()
}

// createCharacter (WPCMK 93ab) makes a character called name and, if the player keeps it,
// saves it into roster slot.
func createCharacter(name string, slot int) {
	newPC = PC{}
	copy(newPC.Name[:], name)
	clearNameRows()
	drawCharacterSheet()
	bonusPoints = -1
	drawStats()
	chooseRace()
	chooseSex()
	bonusPoints = random(6) + 5
	if random(20) == 0 {
		bonusPoints += 8
	}
	if random(20) == 0 {
		bonusPoints += 8
	}
	if maxBonus {
		bonusPoints = 21
	}
	drawStats()
	chooseClass()
	distributeBonusPoints()
	initNewCharacter()
	rollKarma()
	initNPCAttitudes()
	newPC.Portrait = 0
	choosePortrait()
	for _, id := range classItems[newPC.Class] {
		if id > 0 {
			addStartingItem(id)
		}
	}
	drawCharacterSheet()
	if newPC.SkillPoints > 0 {
		skillPointsScreen(promptWin)
	}
	for book := 0; book < 4; book++ {
		for newPC.SpellsLearn[book] > 0 {
			chooseStartingSpell(sheetWin, promptWin, book)
		}
	}
	fillWindow(promptWin, 32, 3, false)
	refreshWindow(promptWin)
	setCursor(promptWin, 6, 1)
	printMessage(1103, promptWin, 18)
	if yesNo(1114, 0) {
		pcfileRecordIO(&newPC, slot, true)
		rosterSlotUsed[slot] = 1
		stopSound()
		writePcfileHeader()
	}
}

// pcmkSkillShown (WPCMK set_class_skill_masks' table) marks the skills the new character
// can learn.
var pcmkSkillShown [30]bool

// putCharAt (WPCMK 611b) puts ch at x, y of win in color 5.
func putCharAt(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putChar(win, ch, 5)
}

// unhighlightRealm (WPCMK 6144) puts back realm's glyph on the sheet, or the "done"
// glyph in the spell window for -1.
func unhighlightRealm(sheet, win *Window, realm int) {
	if realm == -1 {
		putGlyphAt(win, 1, 12, 34)
		return
	}
	putGlyphAt(sheet, realm/3*10+1, realm%3<<1+14, byte(realm+35))
}

// drawSpellWindow (WPCMK 6216) draws the frame of the spell chooser.
func drawSpellWindow(w *Window) {
	g := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
	deferWindowDraw(w)
	fillWindow(w, 0, 1, false)
	g(0, 0, 14)
	g(19, 0, 10)
	g(0, 2, 21)
	g(19, 2, 8)
	for i := 1; i < 19; i++ {
		g(i, 0, 12)
		g(i, 2, 7)
	}
	g(0, 1, 13)
	g(19, 1, 5)
	setCursor(w, 1, 1)
	printMessage(700, w, 18)
	putGlyphAtC4(w, 2, 1, 34)
	putGlyphAtC4(w, 17, 1, 34)
	for i := 3; i < 9; i++ {
		g(0, i, 5)
	}
	for _, c := range [][3]int{{0, 9, 10}, {0, 10, 5}, {0, 11, 22}, {0, 12, 13}, {0, 13, 23}, {0, 14, 5},
		{0, 15, 8}, {1, 11, 18}, {1, 13, 7}, {2, 11, 34}, {2, 12, 13}, {2, 13, 11}} {
		g(c[0], c[1], byte(c[2]))
	}
	for i := 3; i < 6; i++ {
		g(i, 11, 18)
		g(i, 13, 7)
	}
	for i := 6; i < 9; i++ {
		g(i, 11, 18)
		g(i, 13, 12)
		g(i, 15, 7)
	}
	for i := 11; i < 19; i++ {
		g(i, 11, 18)
		g(i, 13, 7)
	}
	for _, c := range [][3]int{{5, 13, 24}, {5, 14, 4}, {5, 15, 6}, {9, 11, 19}, {9, 12, 5}, {9, 13, 10},
		{9, 14, 5}, {9, 15, 8}, {10, 11, 17}, {10, 12, 4}, {10, 13, 6}, {19, 11, 19}, {19, 12, 5}, {19, 13, 8}} {
		g(c[0], c[1], byte(c[2]))
	}
	putGlyphAt(w, 1, 12, 34)
	for i := range 6 {
		putCharAt(w, i+3, 12, byte(i+24))
	}
	setCursor(w, 1, 14)
	printMessage(3957, w, 18)
	blank := func(x, y, n int) {
		setCursor(w, x, y)
		for range n {
			putCharHi(w, 32, 3)
		}
	}
	blank(10, 14, 10)
	blank(1, 15, 4)
	blank(10, 15, 10)
	resumeWindowDraw(w)
	refreshWindow(w)
}

// listRealmSpells (WPCMK 66ea) is the known spells of realm.
func listRealmSpells(realm int) []int {
	var l []int
	for s := range 82 {
		if int(wpcvwSpells[s].Realm) == realm && testBit(newPC.KnownSpells[:], s) {
			l = append(l, s)
		}
	}
	return l
}

// drawSpellList (WPCMK 673f) shows up to three spells either side of sel; with cursor the
// selected one is highlighted in the realm's color.
func drawSpellList(w *Window, realm, sel int, cursor bool, spells []int) {
	drawListBox(w, false)
	deferWindowDraw(w)
	for d := -3; d <= 3; d++ {
		i := sel + d
		if i < 0 || i >= len(spells) {
			continue
		}
		setCursor(w, 1, d+3)
		color := 18
		if cursor && d == 0 {
			color = wpcvwRealmColors[realm]
		}
		printMessage(spells[i]+4000, w, color)
		if cursor && d == 0 {
			for w.CurX < 18 {
				putChar(w, 32, wpcvwRealmColors[realm])
			}
		}
	}
	resumeWindowDraw(w)
	refreshWindow(w)
}

// selectSpell (WPCMK 6800) lets the player pick a realm on the sheet and then one of its
// known spells; the spell, or -1.
func selectSpell(sheet, win *Window) int {
	w := createWindow(20, 4, 20, 16, 22, -1, 0, false)
	list := createWindow(21, 7, 19, 8, 23, -1, 0, false)
	defer destroyWindow(list, false)
	defer destroyWindow(w, false)
	fillWindow(list, 32, 3, false)
	drawSpellWindow(w)
	drawListBox(list, true)
	refreshWindow(win)
	realm := 0
	for r := range 6 {
		x, y := sheet.X+byte(r/3*10+1), sheet.Y+byte(r%3<<1+14)
		setClickRegion(r, x, y, x, y)
	}
	setClickRegion(6, w.X+1, w.Y+12, w.X+1, w.Y+12)
	var spells []int
	for {
		setClickRegion(7, 255, 0, 0, 0)
		redraw := true
		none := false // l10 == 0: an empty realm cannot be entered
		var key int
		for {
			if redraw {
				clearWindowArea(w, 11, 12, 18, 12)
				setCursor(w, 11, 12)
				if realm == -1 {
					putCharAt(w, 1, 12, 99)
					printMessage(3956, w, 9)
					spells, none = nil, false
					clearWindowArea(w, 6, 14, 8, 14)
					drawListBox(list, true)
				} else {
					putCharAt(sheet, realm/3*10+1, realm%3<<1+14, 99)
					printMessage(realm+3950, w, wpcvwRealmColors[realm])
					clearWindowArea(w, 6, 14, 8, 14)
					spells = listRealmSpells(realm)
					none = len(spells) == 0
					drawSpellList(list, realm, 0, false, spells)
				}
				redraw = false
			}
			key = getMenuInput(menuKeys, clickRegions[:])
			if menuClicked == 1 {
				switch {
				case key == realm:
					key = 5
				case key < 6:
					unhighlightRealm(sheet, w, realm)
					realm, key = key, 0
				default:
					unhighlightRealm(sheet, w, realm)
					realm, key, none = -1, 5, false
				}
				redraw = true
			} else {
				switch key {
				case 1:
					if realm == -1 {
						unhighlightRealm(sheet, w, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightRealm(sheet, w, realm)
						realm, redraw = realm-3, true
					}
				case 2:
					if realm != -1 && realm%3 > 0 {
						unhighlightRealm(sheet, w, realm)
						realm, redraw = realm-1, true
					}
				case 3:
					if realm != -1 {
						unhighlightRealm(sheet, w, realm)
						redraw = true
						if realm+3 < 6 {
							realm += 3
						} else {
							realm = -1
						}
					}
				case 4:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightRealm(sheet, w, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if key == 5 && !none {
				break
			}
		}
		if realm == -1 {
			return -1
		}
		unhighlightRealm(sheet, w, realm)
		sel, back := 0, false
		setClickRegion(7, list.X, list.Y, list.X, list.Y)
		setClickRegion(8, list.X, list.Y+6, list.X, list.Y+6)
		setClickRegion(9, list.X+1, list.Y+3, list.X+18, list.Y+3)
		setClickRegion(10, 255, 0, 0, 0)
		for {
			drawSpellList(list, realm, sel, true, spells)
			cost := int(wpcvwSpells[spells[sel]].Cost)
			setCursor(w, 6, 14)
			if int(newPC.SP[realm].Cur) < cost {
				printString(w, wpcvwNoPoints, wpcvwRealmColors[realm])
			} else {
				printInt(w, cost, 3, wpcvwRealmColors[realm])
			}
			key = getMenuInput(menuKeys, clickRegions[:])
			if menuClicked == 1 {
				switch {
				case key <= 5:
					realm, key = key, 1
				case key == 6:
					return -1
				case key == 7:
					key = 2
				case key == 8:
					key = 4
				case key == 9:
					key = 5
				}
			}
			switch key {
			case 1:
				back = true
			case 2:
				if sel > 0 {
					sel--
				}
			case 4:
				if sel+1 < len(spells) {
					sel++
				}
			}
			if key == 5 || back {
				break
			}
		}
		if !back {
			return spells[sel]
		}
	}
}

// chooseStartingSpell (WPCMK 6e38) lets the player learn one first-level spell of spell
// book book (0 mage, 1 priest, 2 alchemist, 3 psionic) and raises its realm's points.
func chooseStartingSpell(sheet, win *Window, book int) {
	bit := [4]uint8{8, 4, 1, 2}[book]
	saved := newPC.KnownSpells
	newPC.KnownSpells = [12]uint8{}
	n := 0
	for s := range 82 {
		sp := wpcvwSpells[s]
		if sp.Books&bit != 0 && sp.Level <= 1 && !testBit(saved[:], s) {
			setBit(newPC.KnownSpells[:], s, true)
			n++
		}
	}
	if n == 0 {
		newPC.KnownSpells = saved
		return // ponytail: as in the original, the pick is never used up, so this loops forever
	}
	fillWindow(win, 32, 3, false)
	deferWindowDraw(win)
	setCursor(win, 1, 1)
	printMessage(703, win, 18)
	resumeWindowDraw(win)
	var savedSP [6]int16
	for r := range 6 {
		savedSP[r] = newPC.SP[r].Cur
		newPC.SP[r].Cur = 999
	}
	spell := -1
	for spell == -1 {
		spell = selectSpell(sheet, win)
	}
	newPC.KnownSpells = saved
	for r := range 6 {
		newPC.SP[r].Cur = savedSP[r]
	}
	setBit(newPC.KnownSpells[:], spell, true)
	sp := wpcvwSpells[spell]
	add := int16(sp.Level)<<1 + 1
	r := sp.Realm
	if int16(sp.Cost) > newPC.SP[r].Max+add {
		add = int16(sp.Cost) - newPC.SP[r].Max
	}
	newPC.SP[r].Max += add
	newPC.SP[r].Cur += add
	newPC.SpellsLearn[book]--
	drawSpellPoints(sheetWin, &newPC)
}
