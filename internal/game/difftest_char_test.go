package game

// charFuncs: WPCMK, WPCVW, WBASE and root mechanics functions.
var charFuncs = map[string]func(a []int) int{
	"pcmk_9341": func(a []int) int { initNewCharacter(); return 0 },
	"pcmk_7274": func(a []int) int { computeEligibleClasses(); return 0 },
	"pcmk_73e9": func(a []int) int { checkClassEligible(a[0], classReqs[a[0]], true); return 0 },
	"pcmk_7532": func(a []int) int { raiseStatsToMinimum(raceMinStats[a[0]]); return 0 },
	"pcmk_75d2": func(a []int) int { applySexModifiers(a[0]); return 0 },
	"pcmk_7e92": func(a []int) int { initNPCAttitudes(); return 0 },
	"pcmk_55ad": func(a []int) int {
		var m [4]string
		for g, r := range [4][2]int{{0, 10}, {10, 17}, {17, 22}, {22, 30}} {
			for _, d := range a[r[0]:r[1]] {
				m[g] += string(rune('0' + d))
			}
		}
		setClassSkillMasks(m)
		return 0
	},
	"pcmk_66ea": func(a []int) int { return len(listRealmSpells(a[0])) },
	"pcmk_81ad": func(a []int) int {
		for _, id := range classItems[newPC.Class] {
			if id > 0 {
				addStartingItem(id)
			}
		}
		return 0
	},
	"pcvw_f6e6": func(a []int) int { updateRank(a[0]); return 0 },
	"pcvw_f568": func(a []int) int { levelUpStats(dtWin(), dtWin(), a[0]); return 0 },
	"pcvw_ed21": func(a []int) int { levelUpHPStamina(dtWin(), dtWin(), a[0]); return 0 },
	"pcvw_eff8": func(a []int) int { levelUpCombat(a[0]); return 0 },
	"pcvw_ea8d": func(a []int) int { rollSkillPoints(a[0]); return 0 },
	"pcvw_ea25": func(a []int) int { return raiseSkill(a[0], a[1], a[2], a[3], a[4]) },
	"pcvw_f201": func(a []int) int { levelUpSpellPoints(a[0]); return 0 },
	"pcvw_c8c2": func(a []int) int { var out [10]int; return canEquipItem(a[0], a[1], &out) },
	"pcvw_cdb3": func(a []int) int { resetCombatValues(a[0]); return 0 },
	"pcvw_c726": func(a []int) int { sortEquippedFirst(a[0]); return 0 },
	"pcvw_5370": func(a []int) int { calcCarriedWeight(a[0]); return 0 },
	"pcvw_a1f9": func(a []int) int { calcAllowedClasses(a[0]); return 0 },
	"pcvw_76ee": func(a []int) int { return countKnownSpells(a[0], a[1], make([]int, 82)) },
	"pcvw_5c24": func(a []int) int { clearHandAttack(a[0], a[1]); return 0 },
	"pcvw_5d5b": func(a []int) int { removeItemShown(a[0], a[1]); return 0 },
	"pcvw_5ec7": func(a []int) int { removeSwagItem(a[0], a[1]); return 0 },
	"pcvw_5f53": func(a []int) int { useItemChargeShown(a[0], a[1]); return 0 },
	"pcvw_6aec": func(a []int) int { return clampToward(a[0], a[1], a[2]) },
	"pcvw_6b3c": func(a []int) int { castSpell(a[0], a[1], a[2], a[3], dtWin(), dtWin()); return 0 },
	"pcvw_8353": func(a []int) int { learnNewSpell(dtWin(), dtWin(), a[0], a[1]); return 0 },
	"pcvw_a4b1": func(a []int) int {
		if a[1] < len(raceMinStats) {
			limitStats(a[0], raceMinStats[a[1]], true)
		}
		return 0
	},
	"pcvw_a368": func(a []int) int { limitStats(a[0], classReqs[a[1]], false); return 0 },
	"pcvw_9c8e": func(a []int) int {
		if tryCombineRecipe(dtWin(), a[0], a[1], a[2]) {
			return 1
		}
		return 0
	},
	"base_6b30": func(a []int) int { removePartyMember(a[0]); return 0 },
	"base_6fd4": func(a []int) int { removeAllMembers(); return 0 },
	"base_5190": func(a []int) int { return findFreePortraitSlot(a[0]) },
	"root_09e2": func(a []int) int { return random(a[0]) },
	"root_2556": func(a []int) int { return randomWord() },
	"root_2881": func(a []int) int { setBit(party[0].Skills[:], a[0], a[1] != 0); return 0 },
	"root_28af": func(a []int) int { return b2i(testBit(party[0].Skills[:], a[0])) },
	"root_28da": func(a []int) int { setBitfield(party[0].Skills[:], a[0], a[1], a[2]); return 0 },
	"root_2925": func(a []int) int { return getBitfield(party[0].Skills[:], a[0], a[1]) },
	"root_2962": func(a []int) int { return extractBits(a[0], a[1], a[2]) },
	"pcvw_8fbf": func(a []int) int { useItemOnSheet(dtWin(), dtWin(), a[0], a[1]); return 0 },
	"pcvw_f784": func(a []int) int { doLevelUp(dtWin(), dtWin(), a[0]); return 0 },
	"pcvw_f8c6": func(a []int) int { checkLevelUp(a[0]); return 0 },
	"pcvw_a11b": func(a []int) int {
		t := ""
		for _, c := range a[2:] {
			t += string(rune('A' + c))
		}
		limitStats(a[0], t, a[1] == 1)
		return 0
	},
}

// dtWin is a window for functions that print on the sheet (one, made once).
var dtWinW *Window

func dtWin() *Window {
	if dtWinW == nil {
		dtWinW = createWindow(0, 0, 40, 20, 20, -1, 0, false)
	}
	return dtWinW
}

// charRoots: the globals their cases name.
var charRoots = map[string]any{
	"party": &party, "partyCount": &partyCount,
	"newPC": &newPC, "bonusPoints": &bonusPoints, "classEligible": &classEligible,
	"pcmkSkillShown": &pcmkSkillShown, "pvClassAllowed": &pvClassAllowed, "partyCounters": &partyCounters,
	"memberPcfileIndex": &memberPcfileIndex, "memberPortrait": &memberPortrait, "rosterSlotUsed": &rosterSlotUsed,
	"levelFlags": &levelFlags, "scenarioHdr": &scenarioHdr, "gameStateArg": &gameStateArg, "pvSpellTarget": &pvSpellTarget,
}

func init() {
	for k, v := range charFuncs {
		diffFuncs[k] = v
	}
	for k, v := range charRoots {
		diffRoots[k] = v
	}
}
