package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// WPCMK: making a character: race, sex, class, bonus points, karma, portrait, skills,
// starting spells and items.

// WPCMK data: the class and race tables (ds:4ffd..5459). Stat strings are 8 letters
// STR..KAR, 'A' = 0.
var (
	// classResists are the resistances, 'A' = 0 in steps of 4, by class.
	classResists = [classCount]string{
		0: "ACBCBCCCCHCHBA", 4: "ACCCCDCCCHHKDA", 8: "ACCCCDCCCHHKDA", 10: "ACCCCDCCCIFKEA",
		11: "ACHCHDCCCHFIDA", 13: "ACKCHHGCCIECBA", 1: "ACBCCDECCEEEGA", 5: "ACBCCDECCEEEGA",
		7: "ACICIBHCCFKFFA", 2: "ACECCEGCCIFKFA", 9: "ACECCEGCCIFKFA", 12: "ACKCGDHCCFKIDA",
		3: "ACHCHHGCCCECHA", 6: "ACHCHHGCCCECHA",
	}
	// classItems are the items each class starts with (give_class_items).
	classItems = [classCount][5]int{
		{8, 135, 132, 130, 141}, {18, 122, 123, 130, 335}, {24, 122, 123, 130, 316},
		{6, 120, 121, 131, 27}, {31, 33, 126, 127, 131}, {18, 122, 123, 130, 326},
		{29, 30, 120, 121, 55}, {1, 122, 123, 130, 241}, {22, 124, 125, 130, 138},
		{24, 122, 123, 130, 163}, {9, 133, 134, 131, 143}, {10, 4, 122, 123, 130},
		{25, 47, 122, 123, 130}, {47, 160, 158, 159, 161},
	}
)

// Race (message 100+), sex (140+) and class (120+) numbers; the other classes are in
// wmaze.go.
const (
	raceHuman     = 0
	raceElf       = 1
	raceDwarf     = 2
	raceGnome     = 3
	raceHobbit    = 4
	raceFaerie    = 5
	raceLizardman = 6
	raceDracon    = 7
	raceFelpurr   = 8
	raceRawulf    = 9
	raceMook      = 10
	raceCount     = 11
	sexMale       = 0
	sexFemale     = 1
	classValkyrie = 8
	classMonk     = 12
	classNinja    = 13
)

// Character creation values.
const (
	maxBonusPoints = 21          // the bonus points with maxBonus
	adultAge       = 18          // years: a new character's age, the youth power's floor
	startAgeSpread = 1000        // days added at random to the starting age
	neutralKarma   = 9           // initNPCAttitudes: the karma NPCs like best
	baseAttitude   = 40          // an NPC's attitude to a character of neutral karma
	baseAC         = 10          // the armour class before speed, race and class
	bareHandDamage = 2           // a bare hand does Count d Sides of this
	minSpellRegen  = 1           // the least spell regeneration of a realm
	loadPerStat    = 15          // MaxLoad per point of (2 STR + VIT) * 3
	karmaRoll      = maxStat + 1 // karma is rolled 0..maxStat
)

// setClassSkillMasks (WPCMK 55ad) marks the skills the class can learn, or already has.
func setClassSkillMasks(masks [4]string) {
	for g, m := range masks {
		for i := range len(m) {
			s := skillGroupStart[g] + i
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
	setClassSkillMasks(classSkills[newPC.Class])
	fillWindow(win, 0, 1)
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
	}{{1, 1, 0, firstPhysicalSkill - 1, 2}, {36, 37, firstPhysicalSkill, firstPersonalSkill - 1, 14},
		{36, 37, firstPersonalSkill, firstAcademiaSkill - 1, 12}, {33, 33, firstAcademiaSkill, skillCount - 1, 11}}
	p := pages[page]
	putIconAt(win, 2, 1, p.left+1)
	putIconAt(win, 17, 1, p.right+1)
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
	fillWindow(win, ' ', 3)
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
	setClickRegion(5, clickRegionsEnd, 0, 0, 0)
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
		key := menuInput()
		if menuClicked == 1 && key >= 0 && key <= 4 {
			key = [5]int{menuKeyLeft, menuKeyRight, menuKeyUp, menuKeyDown, menuKeyEnter}[key]
		}
		s := &newPC.Skills[skills[row]]
		switch key {
		case menuKeyLeft:
			if added[row] > 0 {
				*s--
				added[row]--
				points++
				playSound(1)
				printSkillValue(skillWin, row+3, int(*s))
				printSkillValue(skillWin, 14, points)
			} else {
				playSound(soundBeep)
			}
		case menuKeyUp:
			mark(' ', 7)
			if row > 0 {
				row--
			} else {
				row = count - 1
			}
			mark(100, 4)
		case menuKeyRight:
			if *s < maxSkill && points > 0 {
				*s++
				added[row]++
				points--
				playSound(1)
				printSkillValue(skillWin, row+3, int(*s))
				printSkillValue(skillWin, 14, points)
			} else {
				playSound(soundBeep)
			}
		case menuKeyDown:
			mark(' ', 7)
			if row+1 < count {
				row++
			} else {
				row = 0
			}
			mark(100, 4)
		}
		if key == menuKeyEnter {
			break
		}
	}
	mark(' ', 7)
	newPC.SkillPoints = uint8(points)
}

// skillPointsScreen (WPCMK 604d) goes through the skill pages until the skill points are
// spent; the personal page only when the character has a personal skill.
func skillPointsScreen(win *Window) {
	fillWindow(win, ' ', 3)
	refreshWindow(win)
	skillWin := createWindow(20, 4, 20, 16, 25, -1, 0, false)
	personal := false
	for i := firstPersonalSkill; i < firstAcademiaSkill; i++ {
		if newPC.Skills[i] > 0 {
			personal = true
		}
	}
	for page := skillPageWeaponry; ; {
		list := drawSkillPage(skillWin, page)
		distributeSkillPoints(win, skillWin, list)
		page = (page + 1) % skillPageCount
		if page == skillPagePersonal && !personal {
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
	for i := range statCount {
		if v := min[i] - 'A'; newPC.Stats[i] < v {
			newPC.Stats[i] = v
		}
	}
}

// checkClassEligible (WPCMK 7212) sets whether the bonus points pay for the class's
// minimums; with apply it raises the stats, paying the points.
func checkClassEligible(class int, min string, apply bool) {
	points := bonusPoints
	for stat := range statCount {
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
	for c := range classCount {
		if c == classValkyrie && newPC.Sex != sexFemale {
			classEligible[c] = false
			continue
		}
		checkClassEligible(c, classReqs[c], false)
	}
}

// applySexModifiers (WPCMK 75d2): women have 2 less strength, 1 more personality and karma.
func applySexModifiers(sex int) {
	if sex == sexFemale {
		newPC.Stats[statStrength] -= 2
		newPC.Stats[statPersonality]++
		newPC.Stats[statKarma]++
	}
}

// chooseFromMenu shows the prompt msg at column x, the status title status and the menu
// of messages first.. in menuWin; the chosen one.
func chooseFromMenu(x, msg, status, first int, enabled []int) int {
	fillWindow(promptWin, ' ', 3)
	refreshWindow(promptWin)
	setCursor(promptWin, x, 1)
	printMessage(msg, promptWin, 18)
	fillWindow(menuWin, ' ', 3)
	refreshWindow(menuWin)
	printStatusCentered(status)
	c := menuSelect(menuWin, first, enabled, 1, 1, 10, 11, 5, 0, menuInput)
	clearStatusLine()
	fillWindow(menuWin, ' ', 3)
	refreshWindow(menuWin)
	return c
}

// chooseRace (WPCMK 75f1) asks for the race and sets its base stats.
func chooseRace() {
	r := chooseFromMenu(9, 1104, 1116, 100, []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1})
	newPC.Race = uint8(r)
	setCursor(sheetWin, 15, 1)
	printMsgColor(int(newPC.Race)+100, sheetWin, 1)
	raiseStatsToMinimum(raceMinStats[r])
	drawStats()
}

// chooseSex (WPCMK 770a) asks for the sex.
func chooseSex() {
	s := chooseFromMenu(10, 1105, 1117, 140, []int{1, 1, -1})
	newPC.Sex = uint8(s)
	setCursor(sheetWin, 13, 1)
	putChar(sheetWin, getMessage(int(newPC.Sex) + 140)[0], 1)
	putChar(sheetWin, '-', 1)
	applySexModifiers(s)
	drawStats()
}

// chooseClass (WPCMK 7845) asks for one of the classes the bonus points allow and raises
// the stats to its minimums.
func chooseClass() {
	computeEligibleClasses()
	enabled := make([]int, classCount+1)
	for c := range classCount {
		if classEligible[c] {
			enabled[c] = 1
		}
	}
	enabled[classCount] = -1
	c := chooseFromMenu(7, 1106, 1118, 120, enabled)
	newPC.Class = uint8(c)
	printMsgAt(int(newPC.Class)+120, sheetWin, 13, 2, 1)
	checkClassEligible(c, classReqs[c], true)
	drawStats()
}

// distributeBonusPoints (WPCMK 7969) lets the player spend the remaining bonus points on
// the stats. The cursor only reaches the first seven; karma is cast after.
func distributeBonusPoints() {
	w := promptWin
	fillWindow(w, ' ', 3)
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
	setClickRegion(4, clickRegionsEnd, 0, 0, 0)
	lastStat := statKarma - 1
	for {
		setCursor(w, 12, 3)
		if bonusPoints == 0 {
			printMessage(1110, w, 18)
			setClickRegion(4, x+18, y+3, x+18, y+3)
			setClickRegion(5, clickRegionsEnd, 0, 0, 0)
		} else {
			for w.CurX < 38 {
				putCharHi(w, ' ', 3)
			}
			setClickRegion(4, clickRegionsEnd, 0, 0, 0)
		}
		key := menuInput()
		if menuClicked == 1 && key >= 0 && key <= 4 {
			key = [5]int{menuKeyLeft, menuKeyRight, menuKeyUp, menuKeyDown, menuKeyEnter}[key]
		}
		switch key {
		case menuKeyLeft:
			if added[stat] > 0 {
				newPC.Stats[stat]--
				added[stat]--
				bonusPoints++
				playSound(1)
				drawStats()
			} else {
				playSound(soundBeep)
			}
		case menuKeyUp:
			mark(' ')
			if stat > 0 {
				stat--
			} else {
				stat = lastStat
			}
			mark(98)
		case menuKeyRight:
			if newPC.Stats[stat] < maxStat && bonusPoints > 0 {
				newPC.Stats[stat]++
				added[stat]++
				bonusPoints--
				playSound(1)
				drawStats()
			} else {
				playSound(soundBeep)
			}
		case menuKeyDown:
			mark(' ')
			if stat < lastStat {
				stat++
			} else {
				stat = 0
			}
			mark(98)
		}
		if key == menuKeyEnter && bonusPoints > 0 {
			playSound(soundBeep)
		}
		if key == menuKeyEnter && bonusPoints <= 0 {
			break
		}
	}
	mark(' ')
}

// rollKarma (WPCMK 7d9b) rolls karma until Enter or a click (women get one more).
func rollKarma() {
	fillWindow(promptWin, ' ', 3)
	refreshWindow(promptWin)
	setCursor(promptWin, 9, 1)
	printMessage(1111, promptWin, 18)
	for rolling := true; rolling; {
		newPC.Stats[statKarma] = uint8(random(karmaRoll))
		if newPC.Sex == sexFemale {
			newPC.Stats[statKarma]++
		}
		setCursor(sheetWin, 5, 12)
		printInt(sheetWin, int(newPC.Stats[statKarma]), 2, 1)
		if checkScreenClick() {
			rolling = false
		}
		if keyAvailable(keyClassMenu) && byte(getKey(keyClassMenu)) == keyEnter {
			rolling = false
		}
	}
	drawStats()
}

// initNPCAttitudes (WPCMK 7e92, with 7e5f) sets every NPC's attitude from karma: 40 less
// 4 per point away from 9.
func initNPCAttitudes() {
	for i := range newPC.NPCAttitude {
		d := int(newPC.Stats[statKarma]) - neutralKarma
		if d < 0 {
			d = -d
		}
		newPC.NPCAttitude[i] = int8(baseAttitude - d<<2)
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
	for _, f := range [][2]uint8{{recItemOffHand, itemOffHand}, {recItemTwoHanded, itemTwoHanded}, {16, 16}, {32, 32}, {itemImportant, itemKept}} {
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

// addNewPCResistance (WPCMK 837b) adds delta to resistance i, kept in 0..125.
func addNewPCResistance(i, delta int) {
	newPC.Resist[i] = uint8(min(max(int(int16(int(newPC.Resist[i])+delta)), 0), maxResist)) // C int sum wraps
}

// initResistsHandsAC (WPCMK 83f0) sets the resistances by class, race and stats, the
// bare hands, and the armour class.
func initResistsHandsAC() {
	if s := classResists[newPC.Class]; s != "" {
		for i := range newPC.Resist {
			newPC.Resist[i] = 0
			addNewPCResistance(i, int(s[i])<<2-260)
		}
	}
	vit := int(newPC.Stats[statVitality])
	switch newPC.Race {
	case raceHuman:
		addNewPCResistance(11, 20)
	case raceDwarf, raceHobbit:
		addNewPCResistance(6, vit/5*5)
		addNewPCResistance(12, vit/5*5)
	case raceGnome:
		addNewPCResistance(12, vit/5*5)
	case raceElf:
		addNewPCResistance(9, 20)
	case raceFaerie:
		for _, i := range []int{12, 4, 7, 10, 11} {
			addNewPCResistance(i, 25)
		}
	case raceLizardman, raceDracon:
		addNewPCResistance(10, 25)
		addNewPCResistance(8, 10)
		addNewPCResistance(3, -20)
	case raceFelpurr:
		addNewPCResistance(7, 15)
		addNewPCResistance(4, 20)
		addNewPCResistance(3, 20)
	case raceRawulf:
		addNewPCResistance(3, 20)
	case raceMook:
		addNewPCResistance(12, 25)
		addNewPCResistance(3, 15)
	}
	if vit > highStat {
		addNewPCResistance(6, 5)
		addNewPCResistance(11, 5)
	}
	if newPC.Stats[statIntelligence] > highStat {
		addNewPCResistance(9, 5)
	}
	if newPC.Stats[statSpeed] > highStat {
		addNewPCResistance(7, 5)
		addNewPCResistance(4, 5)
	}
	for h := range newPC.Hands {
		hd := &newPC.Hands[h]
		hd.B0D, hd.B0C = 0, 0
		hd.Data = [8]uint8{}
		hd.ToHit = 0
		hd.Damage = Dice{Base: 0, Count: bareHandDamage, Sides: bareHandDamage}
		hd.Flags = handPunch // bare hand
	}
	hands := int(newPC.Skills[skillUnarmed]) / 5 // hands & feet
	kiri := int(newPC.Skills[skillKirijutsu]) / 5
	h0 := &newPC.Hands[equipPrimary]
	if newPC.Class == classNinja || newPC.Class == classMonk {
		h0.Damage.Sides = uint8(hands/3 + 2)
		h0.Flags |= handKick
		h0.Data[handCritical] = uint8(kiri + 5)
	} else {
		if hands > 0 {
			h0.Damage.Sides = uint8(hands/4 + 1)
			h0.Flags |= handKick
		}
		if kiri > 0 {
			h0.Data[handCritical] = uint8(kiri)
		}
	}
	if newPC.Stats[statStrength] > highStat {
		h0.Damage.Base++
	}
	if newPC.Stats[statStrength] > veryHighStat {
		h0.Damage.Base++
	}
	if newPC.Stats[statDexterity] > highStat {
		h0.ToHit -= 5
	}
	if newPC.Stats[statDexterity] > veryHighStat {
		h0.ToHit -= 5
		newPC.Hands[equipSecondary].ToHit -= 5
	}
	newPC.AC[acBase] = baseAC
	if newPC.Stats[statSpeed] > highStat {
		newPC.AC[acBase]--
	}
	if newPC.Stats[statSpeed] > veryHighStat {
		newPC.AC[acBase]--
	}
	if newPC.Race == raceFaerie {
		newPC.AC[acBase] -= 2
	}
	newPC.AC[equipMisc] = 0
	for i := equipHelmet; i < equipSlots; i++ {
		newPC.AC[i] = newPC.AC[acBase]
	}
}

// initClassSkills (WPCMK 8786) rolls the skill points and the class's starting skills,
// which come out of the skill points.
func initClassSkills() {
	in, pi, vi := int(newPC.Stats[statIntelligence]), int(newPC.Stats[statPiety]), int(newPC.Stats[statVitality])
	dx, sp := int(newPC.Stats[statDexterity]), int(newPC.Stats[statSpeed])
	newPC.SkillPoints = uint8(random(9) + 10)
	sk := &newPC.Skills
	roll := 0
	set := func(skill, n int) { roll = n; sk[skill] = uint8(n) }
	first := func(skill, n int) { set(skill, n); newPC.SkillPoints -= uint8(n) }
	switch newPC.Class {
	case classMage:
		set(skillThaumaturgy, random(4)+in/3+3)
	case classPriest:
		set(skillTheology, random(4)+pi/3+3)
	case classThief:
		set(skillPickLock, random(4)+(in+dx)/2/3+3)
	case classRanger:
		first(skillNoticeSecret, random(3)+(in+vi)/2/5+2)
		set(skillBows, random(3)+(dx+sp)/2/5+2)
	case classBard:
		set(skillInstrument, random(4)+(in+dx)/2/3+3)
	case classAlchemist:
		set(skillAlchemy, random(4)+in/3+3)
	case classPsionic:
		set(skillTheosophy, random(4)+(in+pi)/2/3+3)
	case classBishop:
		first(skillThaumaturgy, random(3)+in/5+2)
		set(skillTheology, random(3)+pi/5+2)
	case classValkyrie:
		set(skillPoleStaff, random(4)+(dx+sp)/2/3+3)
	case classLord, classSamurai:
		set(skillSword, random(4)+(dx+sp)/2/3+3)
	case classMonk:
		first(skillUnarmed, random(3)+(dx+sp)/2/5+2)
		set(skillTheosophy, random(3)+(in+pi)/2/5+2)
	case classNinja:
		first(skillUnarmed, random(3)+(dx+sp)/2/5+2)
		set(skillHide, random(3)+(dx+sp)/2/5+2)
	}
	if int(newPC.SkillPoints) > roll {
		newPC.SkillPoints -= uint8(roll)
	} else {
		newPC.SkillPoints = 0
	}
}

// initHPStamina (WPCMK 8aed) rolls age, hit points, stamina and carrying capacity.
func initHPStamina() {
	newPC.RestHPRoll = 1
	newPC.RestStaminaPct = 10
	newPC.AgeDays = int32(random(startAgeSpread) + adultAge*daysPerYear)
	str, vit := int(newPC.Stats[statStrength]), int(newPC.Stats[statVitality])
	var hp int
	switch newPC.Class {
	case classFighter:
		hp = random(5) + 6
	case classMage:
		hp = random(3) + 2
	case classPriest, classMonk:
		hp = random(4) + 4
	case classThief, classAlchemist, classBishop:
		hp = random(4) + 3
	case classRanger, classSamurai:
		hp = random(4) + random(4) + 6
	case classBard:
		hp = random(4) + 2
	case classPsionic:
		hp = random(3) + 3
	case classValkyrie:
		hp = random(5) + 5
	case classLord:
		hp = random(7) + 8
	case classNinja:
		hp = random(5) + 4
	}
	initClassSkills()
	stm := (vit<<1 + str) * 3
	load := (str<<1 + vit) * 3
	if vit < lowStat {
		hp--
	}
	if vit > highStat {
		hp++
		stm += vit
	}
	if vit > veryHighStat {
		hp++
		stm += vit
	}
	if str > highStat {
		load += str
	}
	if str > veryHighStat {
		load += str
	}
	newPC.HP, newPC.HPMax = int16(hp), int16(hp)
	newPC.Stamina, newPC.StaminaMax = int16(stm), int16(stm)
	newPC.Carried = 0
	newPC.MaxLoad = int16(load * loadPerStat)
	if newPC.Race == raceFaerie {
		newPC.MaxLoad = newPC.MaxLoad << 1 / 3
	}
}

// setSpellRegen (WPCMK 8de0) sets the six realms' spell regeneration, at least 1 each.
func setSpellRegen(r [realmCount]int) {
	for i, v := range r {
		newPC.SpellRegen[i] = uint8(max(v, minSpellRegen))
	}
}

// initSpellRegen (WPCMK 8e47) sets the spell regeneration and the spells to pick per
// spell book (mage, priest, alchemist, psionic).
func initSpellRegen() {
	b := 0
	if newPC.Stats[statVitality] > highStat {
		b++
	}
	if newPC.Race == raceFaerie {
		b++
	}
	if newPC.Race == raceLizardman {
		b--
	}
	r := func(a ...int) {
		var v [realmCount]int
		for i := range v {
			v[i] = b + a[i]
		}
		setSpellRegen(v)
	}
	switch newPC.Class {
	case classMage:
		newPC.SpellsLearn[bookMage] = 2
		r(3, 3, 2, 2, 2, 3)
	case classPriest:
		newPC.SpellsLearn[bookPriest] = 2
		r(2, 2, 3, 3, 2, 3)
	case classAlchemist:
		newPC.SpellsLearn[bookAlchemist] = 2
		r(3, 2, 3, 3, 2, 2)
	case classPsionic:
		newPC.SpellsLearn[bookPsionic] = 2
		r(2, 2, 2, 2, 4, 2)
	case classBishop:
		newPC.SpellsLearn[bookMage] = 1
		newPC.SpellsLearn[bookPriest] = 1
		r(1, 1, 1, 1, 3, 3)
	default:
		r(1, 1, 1, 1, 1, 1)
	}
}

// choosePortrait (WPCMK 9111) lets the player page through the 42 portraits.
func choosePortrait() {
	fillWindow(menuWin, ' ', 3)
	refreshWindow(menuWin)
	fillWindow(promptWin, ' ', 3)
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
	setClickRegion(3, clickRegionsEnd, 0, 0, 0)
	for {
		if int(newPC.Portrait) != shown {
			loadPortrait(int(newPC.Portrait), 0)
			shown = int(newPC.Portrait)
		}
		drawPortrait(menuWin, 0, 8, 3)
		k := menuInput()
		if menuClicked == 1 && k >= 0 && k <= 2 {
			k = [3]int{menuKeyLeft, menuKeyRight, menuKeyEnter}[k]
		}
		switch k {
		case menuKeyLeft:
			newPC.Portrait = uint8((int(newPC.Portrait) + portraitCount - 1) % portraitCount)
		case menuKeyRight:
			newPC.Portrait = uint8((int(newPC.Portrait) + 1) % portraitCount)
		}
		if k == menuKeyEnter {
			break
		}
	}
	clearStatusLine()
	fillWindow(menuWin, ' ', 3)
	refreshWindow(menuWin)
	fillWindow(promptWin, ' ', 3)
	refreshWindow(promptWin)
	drawPortrait(sheetWin, 0, 1, 1)
}

// initNewCharacter (WPCMK 9341) sets the new character's starting values.
func initNewCharacter() {
	newPC.PrimaryAttacks = 1
	newPC.SecondaryAttacks = 1
	newPC.AttackPower = newPC.Stats[statSpeed] + 1
	newPC.ToHit = 100
	initHPStamina()
	initSpellRegen()
	newPC.Regen, newPC.Drain, newPC.LoadCat, newPC.Rank = 0, 0, 0, 0
	newPC.Level, newPC.Unk026 = 1, 1
	newPC.ItemCount = 0
	for i := range newPC.Equipped {
		newPC.Equipped[i] = noEquippedItem
	}
	initResistsHandsAC()
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
		bonusPoints = maxBonusPoints
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
	for book := range bookCount {
		for newPC.SpellsLearn[book] > 0 {
			chooseStartingSpell(sheetWin, promptWin, book)
		}
	}
	fillWindow(promptWin, ' ', 3)
	refreshWindow(promptWin)
	setCursor(promptWin, 6, 1)
	printMessage(1103, promptWin, 18)
	if yesNo(1114, 0) {
		pcfileRecordIO(&newPC, slot, true)
		rosterSlotUsed[slot] = rosterSlotAvailable
		stopSound()
		writePcfileHeader()
	}
}

// pcmkSkillShown (WPCMK set_class_skill_masks' table) marks the skills the new character
// can learn.
var pcmkSkillShown [skillCount]bool

// putCharAt (WPCMK 611b) puts ch at x, y of win in color 5.
func putCharAt(win *Window, x, y int, ch byte) {
	setCursor(win, x, y)
	putChar(win, ch, 5)
}

// unhighlightSheetRealm (WPCMK 6144, WPCVW 7148) puts back realm's glyph on the sheet, or the "done"
// glyph in the spell window for -1.
func unhighlightSheetRealm(sheet, win *Window, realm int) {
	if realm == -1 {
		putGlyphAt(win, 1, 12, 34)
		return
	}
	putGlyphAt(sheet, realm/3*10+1, realm%3<<1+14, byte(realm+35))
}

// drawSheetSpellWindow (WPCMK 6216, WPCVW 721a) draws the frame of the spell chooser.
func drawSheetSpellWindow(w *Window) {
	g := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
	deferWindowDraw(w)
	fillWindow(w, 0, 1)
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
	putIconAt(w, 2, 1, 34)
	putIconAt(w, 17, 1, 34)
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
			putCharHi(w, ' ', 3)
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
	for s := range spellTable {
		if int(spellTable[s].Realm) == realm && testBit(newPC.KnownSpells[:], s) {
			l = append(l, s)
		}
	}
	return l
}

// drawSheetSpellList (draw_spell_list, WPCMK 673f, WPCVW 7749) shows up to three spells either side of sel; with cursor the
// selected one is highlighted in the realm's color.
func drawSheetSpellList(w *Window, realm, sel int, cursor bool, spells []int) {
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
			color = realmColors[realm]
		}
		printMessage(spells[i]+4000, w, color)
		if cursor && d == 0 {
			for w.CurX < 18 {
				putChar(w, ' ', realmColors[realm])
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
	fillWindow(list, ' ', 3)
	drawSheetSpellWindow(w)
	drawListBox(list, true)
	refreshWindow(win)
	realm := 0
	for r := range realmCount {
		x, y := sheet.X+byte(r/3*10+1), sheet.Y+byte(r%3<<1+14)
		setClickRegion(r, x, y, x, y)
	}
	setClickRegion(6, w.X+1, w.Y+12, w.X+1, w.Y+12)
	var spells []int
	for {
		setClickRegion(7, clickRegionsEnd, 0, 0, 0)
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
					printMessage(realm+3950, w, realmColors[realm])
					clearWindowArea(w, 6, 14, 8, 14)
					spells = listRealmSpells(realm)
					none = len(spells) == 0
					drawSheetSpellList(list, realm, 0, false, spells)
				}
				redraw = false
			}
			key = menuInput()
			if menuClicked == 1 {
				switch {
				case key == realm:
					key = menuKeyEnter
				case key < realmCount:
					unhighlightSheetRealm(sheet, w, realm)
					realm, key = key, menuKeyEscape
				default:
					unhighlightSheetRealm(sheet, w, realm)
					realm, key, none = -1, menuKeyEnter, false
				}
				redraw = true
			} else {
				switch key {
				case menuKeyLeft:
					if realm == -1 {
						unhighlightSheetRealm(sheet, w, realm)
						realm, redraw = 4, true
					} else if realm-3 >= 0 {
						unhighlightSheetRealm(sheet, w, realm)
						realm, redraw = realm-3, true
					}
				case menuKeyUp:
					if realm != -1 && realm%3 > 0 {
						unhighlightSheetRealm(sheet, w, realm)
						realm, redraw = realm-1, true
					}
				case menuKeyRight:
					if realm != -1 {
						unhighlightSheetRealm(sheet, w, realm)
						redraw = true
						if realm+3 < realmCount {
							realm += 3
						} else {
							realm = -1
						}
					}
				case menuKeyDown:
					if realm != -1 && realm%3+1 < 3 {
						unhighlightSheetRealm(sheet, w, realm)
						realm, redraw = realm+1, true
					}
				}
			}
			if key == menuKeyEnter && !none {
				break
			}
		}
		if realm == -1 {
			return -1
		}
		unhighlightSheetRealm(sheet, w, realm)
		sel, back := 0, false
		setClickRegion(7, list.X, list.Y, list.X, list.Y)
		setClickRegion(8, list.X, list.Y+6, list.X, list.Y+6)
		setClickRegion(9, list.X+1, list.Y+3, list.X+18, list.Y+3)
		setClickRegion(10, clickRegionsEnd, 0, 0, 0)
		for {
			drawSheetSpellList(list, realm, sel, true, spells)
			cost := int(spellTable[spells[sel]].Cost)
			setCursor(w, 6, 14)
			if int(newPC.SP[realm].Cur) < cost {
				printString(w, wpcvwNoPoints, realmColors[realm])
			} else {
				printInt(w, cost, 3, realmColors[realm])
			}
			key = menuInput()
			if menuClicked == 1 {
				switch {
				case key < realmCount:
					realm, key = key, menuKeyLeft
				case key == 6:
					return -1
				case key == 7:
					key = menuKeyUp
				case key == 8:
					key = menuKeyDown
				case key == 9:
					key = menuKeyEnter
				}
			}
			switch key {
			case menuKeyLeft:
				back = true
			case menuKeyUp:
				if sel > 0 {
					sel--
				}
			case menuKeyDown:
				if sel+1 < len(spells) {
					sel++
				}
			}
			if key == menuKeyEnter || back {
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
	bit := spellBookBits[book]
	saved := newPC.KnownSpells
	newPC.KnownSpells = [12]uint8{}
	n := 0
	for s := range spellTable {
		sp := spellTable[s]
		if sp.Books&bit != 0 && sp.Level <= 1 && !testBit(saved[:], s) {
			setBit(newPC.KnownSpells[:], s, true)
			n++
		}
	}
	if n == 0 {
		newPC.KnownSpells = saved
		return // ponytail: as in the original, the pick is never used up, so this loops forever
	}
	fillWindow(win, ' ', 3)
	deferWindowDraw(win)
	setCursor(win, 1, 1)
	printMessage(703, win, 18)
	resumeWindowDraw(win)
	var savedSP [realmCount]int16
	for r := range realmCount {
		savedSP[r] = newPC.SP[r].Cur
		newPC.SP[r].Cur = unlimitedSP
	}
	spell := -1
	for spell == -1 {
		spell = selectSpell(sheet, win)
	}
	newPC.KnownSpells = saved
	for r := range realmCount {
		newPC.SP[r].Cur = savedSP[r]
	}
	setBit(newPC.KnownSpells[:], spell, true)
	sp := spellTable[spell]
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
