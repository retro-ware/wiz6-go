package game

// npcFuncs: WTREA, WMNPC and WPOPS functions.
var npcFuncs = map[string]func(a []int) int{
	"f8a45": func(a []int) int { chooseMonsterActions(); return 0 },
	"f5dbb": func(a []int) int { return pickTargetMonster(a[0], a[1]) },
}

// npcRoots: the globals their cases name (beyond combat's).
var npcRoots = map[string]any{
	"wpopsTargetTaken": &wpopsTargetTaken,
}

func init() {
	for k, v := range npcFuncs {
		diffFuncs[k] = v
	}
	for k, v := range npcRoots {
		diffRoots[k] = v
	}
}
