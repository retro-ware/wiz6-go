package game

// charFuncs: WPCMK, WPCVW, WBASE and root mechanics functions.
var charFuncs = map[string]func(a []int) int{
	"pcmk_9341": func(a []int) int { initNewCharacter(); return 0 },
	"pcmk_7274": func(a []int) int { computeEligibleClasses(); return 0 },
	"pcmk_73e9": func(a []int) int { checkClassEligible(a[0], classReqs[a[0]], true); return 0 },
	"pcmk_7532": func(a []int) int { raiseStatsToMinimum(raceMinStats[a[0]]); return 0 },
	"pcmk_75d2": func(a []int) int { applySexModifiers(a[0]); return 0 },
	"pcmk_7e92": func(a []int) int { initNPCAttitudes(); return 0 },
	"pcmk_81ad": func(a []int) int {
		for _, id := range classItems[newPC.Class] {
			if id > 0 {
				addStartingItem(id)
			}
		}
		return 0
	},
}

// charRoots: the globals their cases name.
var charRoots = map[string]any{
	"party": &party, "partyCount": &partyCount,
	"newPC": &newPC, "bonusPoints": &bonusPoints, "classEligible": &classEligible,
}

func init() {
	for k, v := range charFuncs {
		diffFuncs[k] = v
	}
	for k, v := range charRoots {
		diffRoots[k] = v
	}
}
