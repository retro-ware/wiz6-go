package game

// npcFuncs: WTREA, WMNPC and WPOPS functions.
var npcFuncs = map[string]func(a []int) int{
	"pops_8a45": func(a []int) int { chooseMonsterActions(); return 0 },
	"pops_5dbb": func(a []int) int { return pickTargetMonster(a[0], a[1]) },
	"trea_5f49": func(a []int) int { springTrap(a[0], a[1]); return 0 },
	"trea_66b1": func(a []int) int {
		var c TreasureRec
		for k := range c.Traps {
			c.Traps[k] = uint8(a[k])
		}
		pickChestTrap(&c)
		return 0
	},
	"trea_8583": func(a []int) int { return giveItemToMember(a[0], a[1]) },
	"trea_9ab6": func(a []int) int { shareCombatGold(); return 0 },
	"trea_9601": func(a []int) int { openTreasure(); return 0 },
	"npc_b805": wareCase(func(a []int) int { npcStockWares(); return 0 }),
	"npc_a69b": wareCase(func(a []int) int { npcBuyWare(a[0], a[1]); return 0 }),
	"npc_b047": wareCase(func(a []int) int { npcPilfer(a[0]); return 0 }),
	"npc_7f76": wareCase(func(a []int) int { return npcReact(a[0], a[1], a[2], uint8(a[3]), uint8(a[4])) }),
	"npc_817e": wareCase(func(a []int) int { npcApplySpell(a[0], a[1], a[2], a[3]); return 0 }),
}

// npcRoots: the globals their cases name (beyond combat's).
var npcRoots = map[string]any{
	"wpopsTargetTaken": &wpopsTargetTaken, "chestTrap": &chestTrap, "lootItem": &lootItem,
	"lootQty": &lootQty, "lootIcon": &lootIcon, "lootWeight": &lootWeight, "lootImportant": &lootImportant,
	"lootCount": &lootCount, "trRow": &trRow, "trChestOpen": &trChestOpen, "trTrapAnim": &trTrapAnim, "trTrapMember": &trTrapMember, "treasureID": &treasureID,
	"npcRec": &npcRec, "npcID": &npcID, "npcCharm": &npcCharm, "npcPilfered": &npcPilfered,
	"npcGoldGiven": &npcGoldGiven, "npcShowGold": &npcShowGold, "npcLeaves": &npcLeaves, "npcTarget": &npcTarget, "npcFlags": &npcFlags,
	"npcWares": &npcWares, "dtWareCount": &dtWareCount, "dtWareList": &dtWareList,
	"returnCmd": &returnCmd, "gameState": &gameState, "gameStateArg": &gameStateArg,
}

func init() {
	for k, v := range npcFuncs {
		diffFuncs[k] = v
	}
	for k, v := range npcRoots {
		diffRoots[k] = v
	}
}

// npcWareList is a slice; the cases name the C's count and array (g54b2, g54b4).
var (
	dtWareCount int
	dtWareList  [30]int
)

func wareCase(f func(a []int) int) func(a []int) int {
	return func(a []int) int {
		n := dtWareCount
		if n == 0 {
			n = 30
		}
		npcWareList = append([]int(nil), dtWareList[:n]...)
		npcTextWin = nil // the C's windows are null (create_window is stubbed)
		r := f(a)
		if len(npcWareList) != n {
			dtWareCount = len(npcWareList)
		}
		copy(dtWareList[:], npcWareList)
		return r
	}
}
