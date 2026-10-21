package game

// combatFuncs: WMEXE and WMELE functions.
var combatFuncs = map[string]func(a []int) int{
	"f7ddc": func(a []int) int { return damageRoll(a[0], a[1], a[2], a[3], a[4], a[5], a[6]) },
	"f4c48": func(a []int) int { doAttack(actor(), a[0], a[1]); return 0 },
	"f47a3": func(a []int) int {
		var fx attackFx
		for k := range fx {
			fx[k] = uint8(a[2+k])
		}
		applyHitEffects(actor(), fx, a[0], a[1])
		return 0
	},
	"f4572": func(a []int) int { return fatigueMod(a[0], a[1], a[2]) },
	"f77d4": func(a []int) int { mexeRaiseSkill(a[0], a[1], a[2]); return 0 },
	"f7254": func(a []int) int { reduceStamina(fighter(a[0], a[1]), a[2]); return 0 },
	"f8861": func(a []int) int { return clampToward(a[0], a[1], a[2]) },
	"fc4e7": func(a []int) int { return nextLivingMember(a[0]) },
	"fc546": func(a []int) int { return nextTargetMonster(a[0], a[1]) },
	"f7816": func(a []int) int { return spellPower(a[0], a[1]) },
	"f7af7": func(a []int) int { return resistReduce(a[0], a[1], a[2], a[3]) },
	"f7b59": func(a []int) int { damageEffect(a[0], a[1], a[2], a[3], a[4], a[5], a[9]); return 0 },
	"f7c5a": func(a []int) int { statusEffect(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8], a[10]); return 0 },
	"f7848": func(a []int) int { killCombatant(a[0], a[1]); return 0 },
	"f7a16": func(a []int) int { effectMsg(a[0], a[1], a[2], a[3], a[4]); return 0 },
	"f8064": func(a []int) int { return groupWard(a[0], a[1], a[2], a[3]) },
	"f80cc": func(a []int) int { return wardedDice(a[0], a[1], a[2]) },
	"f8149": func(a []int) int { hitDamage(a[0], a[1], a[2], a[3], a[4], a[5], a[6]); return 0 },
	"f825f": func(a []int) int {
		hitEffect(a[0], a[1], a[2], a[3], a[4], a[5], a[6] == 1, a[7], a[8], a[9])
		return 0
	},
	"f83f3": func(a []int) int {
		hitDamageThen(a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8], a[9] == 1, a[10], a[11], a[12], a[13], a[14])
		return 0
	},
	"f86aa": func(a []int) int {
		groupStatus(a[0], a[1], a[2], a[3], a[4], a[5] == 1, a[6], a[7], a[8])
		return 0
	},
}

// combatRoots: the globals their cases name.
var combatRoots = map[string]any{
	"party": &party, "combatGold": &combatGold, "partyCount": &partyCount, "g4fc8": &g4fc8,
	"actMember": &actMember, "actGroup": &actGroup, "monsterGroupCount": &monsterGroupCount,
	"mexeMsgMode": &mexeMsgMode, "hdr": &hdr, "viewPosGroup": &viewPosGroup,
	"monsterJitterX": &monsterJitterX, "monsterJitterY": &monsterJitterY,
	"animCycle": &animCycle, "mexeEffect": &mexeEffect, "mexeBackfire": &mexeBackfire, "mexeDelayed": &mexeDelayed,
	"partyCounters": &partyCounters, "g4352": &g4352, "initiative": &initiative, "g4fcc": &g4fcc,
	"combatOver": &combatOver, "viewRowsOpen": &viewRowsOpen,
	"monsterGroups": &monsterGroups, "groupMonsterRec": &groupMonsterRec,
}

func init() {
	for k, v := range combatFuncs {
		diffFuncs[k] = v
	}
	for k, v := range combatRoots {
		diffRoots[k] = v
	}
}
