package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// WPCVW: the skills pages, levelling up and the modify menu (rename, portrait, class).

// markShownSkills (WPCVW dd93) marks the skills the skills pages list: those member m's
// class always shows (t, a '1' per skill of each group) and those it has points in.
func markShownSkills(m int, t [4]string) {
	p := &party[m]
	for g, s := range t {
		for i := 0; i < len(s); i++ {
			k := skillGroupStart[g] + i
			pvSkillShown[k] = s[i] == '1' || p.Skills[k] > 0
		}
	}
}

// drawSkillsPage (WPCVW e35f) draws skills page pg (weaponry, physical, personal,
// academia) of member m in w, listing in shown the skills of each row. It returns how
// many rows it shows.
func drawSkillsPage(w *Window, m, pg int, shown *[10]int) int {
	p := &party[m]
	markShownSkills(m, classSkills[p.Class])
	f := func(x, y int, ch byte) { setFrameGlyph(w, x, y, ch) }
	fillWindow(w, 0, 1)
	f(0, 0, 14)
	f(19, 0, 10)
	f(0, 15, 11)
	f(19, 15, 8)
	for x := 1; x < 19; x++ {
		f(x, 0, 12)
		f(x, 2, 12)
		f(x, 13, 12)
		f(x, 15, 7)
	}
	for y := 1; y < 15; y++ {
		f(0, y, 13)
		f(19, y, 5)
	}
	f(0, 2, 33)
	f(0, 9, 15)
	f(0, 11, 15)
	f(0, 13, 16)
	f(19, 2, 10)
	f(19, 13, 10)
	deferWindowDraw(w)
	setCursor(w, 1, 1)
	printStringHi(w, "                    ", 3)
	setCursor(w, 6, 1)
	printMessage(pg+600, w, 18)
	pages := [4][5]int{ // icon left, icon right, first skill, last skill, colour
		{1, 1, 0, firstPhysicalSkill - 1, 2}, {36, 37, firstPhysicalSkill, firstPersonalSkill - 1, 14},
		{36, 37, firstPersonalSkill, firstAcademiaSkill - 1, 12}, {33, 33, firstAcademiaSkill, skillCount - 1, 11},
	}
	pd := pages[pg]
	putIconAt(w, 2, 1, byte(pd[0]+1))
	putIconAt(w, 17, 1, byte(pd[1]+1))
	n := 0
	for k := pd[2]; k <= pd[3]; k++ {
		if pvSkillShown[k] {
			setCursor(w, 1, n+3)
			printMessage(k+5500, w, pd[4])
			setCursor(w, 16, n+3)
			printInt(w, int(p.Skills[k]), 3, 1)
			shown[n] = k
			n++
		}
	}
	setCursor(w, 1, 14)
	printMessage(5530, w, 9)
	setCursor(w, 16, 14)
	printInt(w, int(p.SkillPoints), 3, 1)
	resumeWindowDraw(w)
	refreshWindow(w)
	return n
}

// hasPersonalSkills reports whether member m has points in a personal skill.
func hasPersonalSkills(m int) bool {
	for k := firstPersonalSkill; k < firstAcademiaSkill; k++ {
		if party[m].Skills[k] > 0 {
			return true
		}
	}
	return false
}

// doSkillsMenu (do_skills_menu, WPCVW 929a) shows member m's skill pages until exit.
func doSkillsMenu(menu *Window, m int) {
	personal := hasPersonalSkills(m)
	w := createWindow(20, 4, 20, 16, 25, borderNone, 0, false)
	var shown [10]int
	for pg := skillPageWeaponry; pg != skillPageCount; {
		drawSkillsPage(w, m, pg, &shown)
		enabled := []int{1, 1, 1, 1, 1, -1}
		enabled[pg] = 0
		if !personal {
			enabled[skillPagePersonal] = 0
		}
		sel := countEnabledItems(enabled, pg+1)
		if pg+1 == skillPagePersonal && !personal {
			sel = countEnabledItems(enabled, pg+2)
		}
		fillWindow(menu, ' ', 3)
		refreshWindow(menu)
		pg = menuSelect(menu, 600, enabled, 2, 1, 10, 2, 5, sel, getMenuInputBlink)
	}
	fillWindow(menu, ' ', 3)
	destroyWindow(w, true)
}

// spendSkillPage (WPCVW deca) lets member m spend skill points on the n skills shown on
// page w (cursor up/down picks, left/right lowers and raises what this page added).
func spendSkillPage(menu, w *Window, m int, shown *[10]int, n int) {
	p := &party[m]
	left := int(p.SkillPoints)
	fillWindow(menu, ' ', 3)
	refreshWindow(menu)
	setCursor(menu, 2, 1)
	printMessage(606, menu, 18)
	menu.CurX += 3
	printMessage(607, menu, 18)
	var added [10]int
	row := 0
	mx, my := int(menu.X), int(menu.Y)
	for i, x := range []int{2, 3, 22, 23} {
		setRegion(i, mx+x, my+1, mx+x, my+1)
	}
	setRegion(4, mx+13, my+2, mx+13, my+2)
	endRegions(5)
	mark := func(ch byte, c int) {
		setCursor(w, 15, row+3)
		putChar(w, ch, c)
	}
	mark(100, 4)
	for {
		setCursor(menu, 7, 2)
		x := 18
		if left > 0 {
			printMessage(608, menu, 18)
			x = 13
		} else {
			printMessage(609, menu, 18)
		}
		setRegion(4, mx+x, my+2, mx+x, my+2)
		k := getMenuInputBlink()
		if menuClicked == 1 && k >= 0 && k < 5 {
			k = [5]int{menuKeyLeft, menuKeyRight, menuKeyUp, menuKeyDown, menuKeyEnter}[k]
		}
		sk := &p.Skills[shown[row]]
		switch k {
		case menuKeyLeft:
			if added[row] > 0 {
				*sk--
				added[row]--
				left++
				playSound(1)
				printNumberCol16(w, row+3, int(*sk))
				printNumberCol16(w, 14, left)
			} else {
				playSound(soundBeep)
			}
		case menuKeyUp:
			mark(' ', 7)
			if row > 0 {
				row--
			} else {
				row = n - 1
			}
			mark(100, 4)
		case menuKeyRight:
			if *sk < maxSkill && left > 0 {
				*sk++
				added[row]++
				left--
				playSound(1)
				printNumberCol16(w, row+3, int(*sk))
				printNumberCol16(w, 14, left)
			} else {
				playSound(soundBeep)
			}
		case menuKeyDown:
			mark(' ', 7)
			if row+1 < n {
				row++
			} else {
				row = 0
			}
			mark(100, 4)
		}
		if k == menuKeyEnter {
			break
		}
	}
	mark(' ', 7)
	p.SkillPoints = uint8(left)
}

// spendSkillPoints (WPCVW e922) cycles member m through its skill pages until all its
// skill points are spent.
func spendSkillPoints(menu *Window, m int) {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	fillWindow(menu, ' ', 3)
	refreshWindow(menu)
	showCenteredMsg(menu, getMessage(605), true)
	w := createWindow(20, 4, 20, 16, 25, borderNone, 0, false)
	personal := hasPersonalSkills(m)
	var shown [10]int
	for pg := skillPageWeaponry; ; {
		n := drawSkillsPage(w, m, pg, &shown)
		spendSkillPage(menu, w, m, &shown, n)
		if pg = (pg + 1) % skillPageCount; pg == skillPagePersonal && !personal {
			pg++
		}
		if party[m].SkillPoints == 0 {
			break
		}
	}
	destroyWindow(w, false)
}

// raiseSkill (raise_skill, WPCVW ea25) raises skill k of member m by a level-up roll
// (random(r) + stat/div - level/3, none past maxRaisedSkill) and returns the rise.
func raiseSkill(m, stat, div, r, k int) int {
	p := &party[m]
	v := int(int16(random(r) + stat/div - int(p.Level)/3)) // C int wraps
	if v < 0 || p.Skills[k] > maxRaisedSkill {
		v = 0
	}
	p.Skills[k] += uint8(v)
	return v
}

// rollSkillPoints (roll_skill_points, WPCVW ea8d) gives member m 5..10 skill points,
// less what its class's free skill rises take.
func rollSkillPoints(m int) {
	p := &party[m]
	in, pi, vi := int(p.Stats[statIntelligence]), int(p.Stats[statPiety]), int(p.Stats[statVitality])
	dx, sp := int(p.Stats[statDexterity]), int(p.Stats[statSpeed])
	p.SkillPoints = uint8(random(6) + 5)
	used := 0
	switch p.Class {
	case classMage:
		used = raiseSkill(m, in, 3, 3, skillThaumaturgy)
	case classPriest:
		used = raiseSkill(m, pi, 3, 3, skillTheology)
	case classThief:
		used = raiseSkill(m, (in+dx)/2, 4, 3, skillPickLock)
		used += raiseSkill(m, (dx+sp)/2, 4, 2, skillPilfer)
	case classRanger:
		used = raiseSkill(m, (dx+sp)/2, 4, 2, skillBows)
		used += raiseSkill(m, (in+vi)/2, 4, 2, skillNoticeSecret)
	case classBard:
		used = raiseSkill(m, (in+dx)/2, 3, 3, skillInstrument)
	case classAlchemist:
		used = raiseSkill(m, in, 3, 3, skillAlchemy)
	case classPsionic:
		used = raiseSkill(m, (in+pi)/2, 3, 3, skillTheosophy)
	case classBishop:
		used = raiseSkill(m, in, 4, 2, skillThaumaturgy)
		used += raiseSkill(m, pi, 4, 2, skillTheology)
	case classSamurai:
		used = raiseSkill(m, (dx+sp)/2, 3, 3, skillSword)
	case classNinja:
		used = raiseSkill(m, (dx+sp)/2, 4, 3, skillUnarmed)
		used += raiseSkill(m, (dx+sp)/2, 4, 2, skillHide)
	}
	if int(p.SkillPoints) > used {
		p.SkillPoints -= uint8(used)
	} else {
		p.SkillPoints = 0
	}
}

// updateRank (WPCVW f6e6) raises member m's rank to the number of the levels 2, 4, 7,
// 10, 14, 18 and 24 it has reached.
func updateRank(m int) {
	p := &party[m]
	r := uint8(0)
	for _, l := range []int16{2, 4, 7, 10, 14, 18, 24} {
		if p.Level >= l {
			r++
		}
	}
	p.Rank = max(p.Rank, r)
}

// levelUpStats (WPCVW f568) raises random stats of member m but karma, each at most once,
// below maxStat, showing each on the sheet.
func levelUpStats(sheet, menu *Window, m int) {
	p := &party[m]
	var done [statCount]bool
	for {
		s, tries := 0, 0
		for {
			s = random(statKarma)
			tries++
			if !(p.Stats[s] >= maxStat || done[s]) || tries >= 3 {
				break
			}
		}
		if p.Stats[s] < maxStat && !done[s] {
			done[s] = true
			p.Stats[s]++
			msg := replaceCharWithString(getMessage(5014), getMessage(s+5050), '$')
			printMsgAt(s+204, sheet, 1, s+5, -5)
			sheet.CurX++
			printInt(sheet, int(p.Stats[s]), 2, -1)
			showCenteredMsg(menu, msg, true)
			printMsgAt(s+204, sheet, 1, s+5, 5)
			sheet.CurX++
			printInt(sheet, int(p.Stats[s]), 2, 1)
		}
		if random(2) == 0 {
			break
		}
	}
}

// levelUpHPStamina (WPCVW ed21) rolls member m's hit point and stamina gains and shows
// them on the sheet.
func levelUpHPStamina(sheet, menu *Window, m int) {
	p := &party[m]
	str, vit := int(p.Stats[statStrength]), int(p.Stats[statVitality])
	hp := 0
	switch p.Class {
	case classFighter, classRanger, classLord:
		hp = random(5) + 6
	case classMage:
		hp = random(3) + 2
	case classPriest, classAlchemist, classMonk:
		hp = random(4) + 4
	case classThief:
		hp = random(4) + 3
	case classBard, classPsionic, classBishop:
		hp = random(3) + 3
	case classValkyrie, classSamurai:
		hp = random(5) + 5
	case classNinja:
		hp = random(5) + 4
	}
	st := random(4) + (str+vit)/6
	if vit < lowStat {
		hp--
	}
	if vit > highStat {
		hp++
	}
	if vit > veryHighStat {
		hp++
	}
	hp = max(hp, 1)
	if int(p.Level) < int(p.LevelBeforeClassChange) {
		hp, st = 1, 1
	}
	p.HP += int16(hp)
	p.HPMax += int16(hp)
	p.Stamina += int16(st)
	p.StaminaMax += int16(st)
	setCursor(sheet, 15, 8)
	pct := int(int16(int(p.Stamina)*100) / p.StaminaMax) // 16-bit, as in the C
	if pct < 1 && p.Stamina > 0 {
		pct = 1
	}
	printInt(sheet, pct, 3, -6)
	showCenteredMsg(menu, getMessage(5013), true)
	setCursor(sheet, 15, 8)
	printInt(sheet, pct, 3, 6)
	msg := 5012
	if hp == 1 {
		msg = 5011
	}
	s := replaceCharWithString(getMessage(msg), itoa(hp), '$')
	showHP := func(c int) {
		setCursor(sheet, 14, 5)
		printInt(sheet, int(p.HP), 4, c)
		setCursor(sheet, 14, 6)
		printInt(sheet, int(p.HPMax), 4, c)
	}
	showHP(-6)
	showCenteredMsg(menu, s, true)
	showHP(6)
}

// levelUpCombat (WPCVW eff8) improves member m's to-hit, attacks and AC, up to level
// maxCombatLevel.
func levelUpCombat(m int) {
	p := &party[m]
	if p.Level > maxCombatLevel {
		return
	}
	th := 0
	switch p.Class {
	case classFighter, classRanger, classValkyrie, classLord, classMonk:
		th = random(4) + 2
	case classMage, classAlchemist, classPsionic, classBishop:
		th = random(3) + 1
	case classPriest, classThief, classBard:
		th = random(4) + 1
	case classSamurai, classNinja:
		th = random(4) + 3
	}
	if p.Stats[statDexterity] < lowStat {
		th--
	}
	if int(p.Level) < int(p.LevelBeforeClassChange) {
		th = 0
	}
	if p.ToHit -= int8(th); p.ToHit < 0 {
		p.ToHit = 0
	}
	setClassAttackCounts(m)
	ac := naturalAC(p)
	if d := ac - int(int8(p.AC[acBase])); d != 0 {
		p.AC[acBase] = uint8(ac)
		for s := equipHelmet; s < equipSlots; s++ {
			p.AC[s] += uint8(d)
		}
	}
}

// levelUpSpellPoints (WPCVW f201) gives member m of a spell casting class a spell to
// learn and more spell points in each realm it knows spells of.
func levelUpSpellPoints(m int) {
	p := &party[m]
	in, pi := int(p.Stats[statIntelligence]), int(p.Stats[statPiety])
	vi, pe := int(p.Stats[statVitality]), int(p.Stats[statPersonality])
	var stat, book, skill, from int
	switch p.Class {
	case classMage:
		stat, book, skill, from = (in+pi)/2, bookMage, skillThaumaturgy, 1
	case classPriest:
		stat, book, skill, from = (pi+vi)/2, bookPriest, skillTheology, 1
	case classRanger:
		stat, book, skill, from = (in+pi)/2, bookAlchemist, skillAlchemy, 3
	case classAlchemist:
		stat, book, skill, from = (in+pi)/2, bookAlchemist, skillAlchemy, 1
	case classBard:
		stat, book, skill, from = (in+pi)/2, bookMage, skillThaumaturgy, 3
	case classPsionic:
		stat, book, skill, from = (in+pi)/2, bookPsionic, skillTheosophy, 1
	case classValkyrie:
		stat, book, skill, from = (pi+vi)/2, bookPriest, skillTheology, 3
	case classBishop: // mage and priest spells by turns
		stat, book, from = (in+pi)/2, int(p.Level)%2, 1
		skill = skillTheology
		if book == bookMage {
			skill = skillThaumaturgy
		}
	case classLord:
		stat, book, skill, from = (pe+pi)/2, bookPriest, skillTheology, 3
	case classSamurai:
		stat, book, skill, from = (in+pi)/2, bookMage, skillThaumaturgy, 3
	case classMonk:
		stat, book, skill, from = (in+pi)/2, bookPsionic, skillTheosophy, 3
	case classNinja:
		stat, book, skill, from = (in+pi)/2, bookAlchemist, skillAlchemy, 5
	default:
		return
	}
	if int(p.Level) < from {
		return
	}
	p.SpellsLearn[book]++
	for r := range p.SP {
		known := 0
		if p.SP[r].Max > 0 {
			for s := range spellTable {
				if int(spellTable[s].Realm) == r && testBit(p.KnownSpells[:], s) {
					known++
				}
			}
		}
		if known <= 0 {
			continue
		}
		bonus := 0
		if stat > 13 {
			bonus++
		}
		if stat > 16 {
			bonus++
		}
		if int(p.Level) < int(p.LevelBeforeClassChange) {
			p.SP[r].Max++
			continue
		}
		p.SP[r].Max = int16(random(int(p.Skills[skill])/20)+int(p.SP[r].Max)+int(p.SpellRegen[r])) + int16(known<<1+bonus)
	}
}

// doLevelUp (do_level_up, WPCVW f784) raises member m a level: rank, stats, hit points,
// combat values, skills and spells.
func doLevelUp(sheet, menu *Window, m int) {
	p := &party[m]
	p.Level++
	if int16(p.LevelBeforeClassChange) <= p.Level {
		p.LevelBeforeClassChange = 0
	}
	updateRank(m)
	drawSheet(sheet, menu, m)
	fillWindow(menu, ' ', 3)
	refreshWindow(menu)
	s := replaceCharWithString(getMessage(5010), memberName(m), '^')
	s = replaceCharWithString(s, getMessage(int(p.Class)+120), '$')
	showCenteredMsg(menu, s, true)
	levelUpStats(sheet, menu, m)
	levelUpHPStamina(sheet, menu, m)
	levelUpCombat(m)
	rollSkillPoints(m)
	if p.SkillPoints > 0 {
		spendSkillPoints(menu, m)
	}
	levelUpSpellPoints(m)
	for b := range bookCount {
		for p.SpellsLearn[b] > 0 {
			learnNewSpell(sheet, menu, m, b)
		}
	}
}

// checkLevelUp (check_level_up, WPCVW f8c6) levels member m up while its experience
// reaches the next level.
func checkLevelUp(m int) {
	p := &party[m]
	sheet := createWindow(0, 0, 40, 20, 20, borderNone, 0, false)
	menu := createWindow(0, 20, 40, 4, 15, borderNone, 0, false)
	resumeWindowDraw(bottomLineWin)
	eraseWindow(bottomLineWin, false)
	b := make([]byte, dbs.Size[dbs.ClassExp]())
	accessScenarioRecord(dbs.RecClassExp, int(p.Class), b, false)
	t, _ := dbs.Decode[dbs.ClassExp](b)
	pvNamesLoaded = false
	for {
		var need int32
		if p.Level < 16 {
			need = t.Exp[p.Level-1]
		} else {
			need = (t.Exp[15]-t.Exp[14])*int32(p.Level-16) + t.Exp[15]
		}
		if p.Exp < need {
			break
		}
		doLevelUp(sheet, menu, m)
	}
	destroyWindow(menu, false)
	destroyWindow(sheet, false)
}

// levelUpEntry (level_up_entry, WPCVW fa10): stateLevelUp, after combat or rest. Every
// member levels up as far as its experience goes, then the maze screen comes back.
func levelUpEntry() {
	returnCmd = int16(stateExplore)
	gameState = stateExplore
	menuKeys = getMessage(300)
	for m := 0; m < partyCount; m++ {
		checkLevelUp(m)
	}
	resumeWindowDraw(g4fb0)
	resumeWindowDraw(mainWindow)
	resumeWindowDraw(bottomLineWin)
	eraseWindow(mainWindow, false)
	fillWindow(partyPictureWin, 26, 3)
	deferWindowDraw(partyPictureWin)
	for y := 0; y < 5; y++ {
		setCursor(partyPictureWin, 19, y)
		putCharHi(partyPictureWin, 27, 3)
	}
	resumeWindowDraw(partyPictureWin)
	eraseCursorMark(bottomLineWin, false)
}

// limitStats (WPCVW a11b) lowers member m's stats to at most t (lower) or raises them to
// at least t, a letter per stat with 'A' 0.
func limitStats(m int, t string, lower bool) {
	p := &party[m]
	for s := range statKarma {
		v := t[s] - 'A'
		if lower && p.Stats[s] > v || !lower && p.Stats[s] < v {
			p.Stats[s] = v
		}
	}
}

// calcAllowedClasses (calc_allowed_classes, WPCVW a1f9, with check_class_reqs a1a7) marks
// in pvClassAllowed the classes member m's stats (and, for valkyries, sex) qualify for.
func calcAllowedClasses(m int) {
	p := &party[m]
	for c := range pvClassAllowed {
		ok := c != classValkyrie || p.Sex == sexFemale
		for s := 0; ok && s < statKarma; s++ {
			ok = p.Stats[s] >= classReqs[c][s]-'A'
		}
		pvClassAllowed[c] = ok
	}
}

// doChangeClass (do_change_class, WPCVW a5b8) lets member m take another class it
// qualifies for: back to level 1, stats to the race's and then the class's minimum.
func doChangeClass(sheet, menu *Window, m int) {
	p := &party[m]
	calcAllowedClasses(m)
	w := createWindow(20, 4, 20, 16, 30, borderNone, 0, false)
	fillWindow(w, ' ', 3)
	deferWindowDraw(w)
	drawPortraitFrame(w)
	resumeWindowDraw(w)
	fillWindow(menu, ' ', 3)
	refreshAllWindows()
	enabled := make([]int, classCount+2) // cancel (message 119), the classes, the end
	enabled[0], enabled[classCount+1] = 1, -1
	for c, ok := range pvClassAllowed {
		if ok {
			enabled[c+1] = 1
		}
	}
	enabled[p.Class+1] = 0 // not the class it has
	setCursor(menu, 4, 1)
	printMessage(660, menu, 18)
	setCursor(menu, 6, 2)
	printMessage(661, menu, 18)
	c := menuSelect(w, 119, enabled, 1, 1, 10, 11, 5, 0, getMenuInputBlink)
	destroyWindow(w, false)
	if c <= 0 {
		return
	}
	p.Class = uint8(c - 1)
	if p.Level < 250 {
		p.LevelBeforeClassChange = uint8(p.Level)
	} else {
		p.LevelBeforeClassChange = 0
	}
	p.Level, p.Rank, p.Exp = 1, 0, 0
	if int(p.Race) < len(raceMinStats) {
		limitStats(m, raceMinStats[p.Race], true)
	}
	limitStats(m, classReqs[p.Class], false)
	drawSheet(sheet, menu, m)
	doEquipItems(sheet, menu, m)
}

// doRename (do_rename, WPCVW abd8) asks for a new name of up to nameLength letters for
// member m.
func doRename(w *Window, m int) {
	fillWindow(w, ' ', 3)
	refreshAllWindows()
	setCursor(w, 1, 1)
	printMessage(1128, w, 18)
	x, y := int(w.CurX), int(w.CurY)
	clearWindowArea(w, x, y, x+nameLength, y)
	setCursor(w, x, y)
	if s := inputText(w, nameLength, 5, inputPrintable); s != "" {
		party[m].Name = [8]uint8{}
		copy(party[m].Name[:nameLength], s)
	}
}

// doChoosePortrait (do_choose_portrait, WPCVW a920) cycles member m through the
// portraitCount portraits (left/right) until Enter.
func doChoosePortrait(w *Window, m int) {
	p := &party[m]
	pw := createWindow(20, 4, 20, 16, 30, borderNone, 0, false)
	fillWindow(pw, ' ', 3)
	deferWindowDraw(pw)
	drawPortraitFrame(pw)
	resumeWindowDraw(pw)
	fillWindow(w, ' ', 3)
	refreshAllWindows()
	setCursor(w, 9, 1)
	printMessage(1112, w, 18)
	setCursor(w, 12, 2)
	printMessage(1113, w, 18)
	wx, wy := int(w.X), int(w.Y)
	setRegion(0, wx+9, wy+1, wx+9, wy+1)
	setRegion(1, wx+10, wy+1, wx+10, wy+1)
	setRegion(2, wx+18, wy+2, wx+18, wy+2)
	endRegions(3)
	shown := -1
	for {
		if int(p.Portrait) != shown {
			loadPortrait(int(p.Portrait), int(memberPortrait[m]))
			shown = int(p.Portrait)
		}
		drawPortrait(pw, int(memberPortrait[m]), 8, 6)
		k := getMenuInputBlink()
		if menuClicked == 1 && k >= 0 && k < 3 {
			k = [3]int{menuKeyLeft, menuKeyRight, menuKeyEnter}[k]
		}
		switch k {
		case menuKeyLeft:
			p.Portrait = (p.Portrait + portraitCount - 1) % portraitCount
		case menuKeyRight:
			p.Portrait = (p.Portrait + 1) % portraitCount
		}
		if k == menuKeyEnter {
			break
		}
	}
	destroyWindow(pw, false)
	// The disk prompts around this (prompt_insert_disk) are no-ops here.
}

// doModifyMenu's entries (messages 650+).
const (
	editMenuRename = iota
	editMenuPortrait
	editMenuClass
	editMenuReplace // REPLACE CHARACTER, always off
	editMenuExit
)

// maxRaisedSkill is the skill above which raiseSkill adds nothing.
const maxRaisedSkill = 50

// doModifyMenu (do_modify_menu, WPCVW ac83): rename, portrait, change class, exit.
func doModifyMenu(sheet, menu *Window, m int) {
	for {
		fillWindow(menu, ' ', 3)
		refreshAllWindows()
		// Entry 3 is off: the C's enabled array overlaps a local it zeroes.
		enabled := []int{1, 1, 1, 0, 1, -1}
		k := menuSelect(menu, 650, enabled, 2, 1, 18, 2, 5, countEnabledItems(enabled, editMenuExit), getMenuInputBlink)
		switch k {
		case editMenuRename:
			doRename(menu, m)
		case editMenuPortrait:
			doChoosePortrait(menu, m)
		case editMenuClass:
			doChangeClass(sheet, menu, m)
		}
		drawSheet(sheet, menu, m)
		if k == editMenuExit {
			return
		}
	}
}
