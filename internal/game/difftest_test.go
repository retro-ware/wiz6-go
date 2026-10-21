package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

// Differential tests: each case in testdata/diff/FUNC.json.gz was produced by running the
// original function in a CPU emulator (../wiz6-re/tools/difftest). The Go function gets
// the same named input fields, args and RNG seeds, and must give the same return value,
// changed fields, random numbers and messages.

type diffCase struct {
	Args  []int
	Seeds [3]int
	State map[string]int64
	Ret   *int // nil: a void function
	Out   map[string]int64
	RNG   [][]int
	Log   [][]any
}

// diffFuncs runs the Go port of each tested original function.
var diffFuncs = map[string]func(a []int) int{
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

// diffRoots are the globals the cases name.
var diffRoots = map[string]any{
	"party": &party, "combatGold": &combatGold, "partyCount": &partyCount, "g4fc8": &g4fc8,
	"actMember": &actMember, "actGroup": &actGroup, "monsterGroupCount": &monsterGroupCount,
	"mexeMsgMode": &mexeMsgMode, "hdr": &hdr, "viewPosGroup": &viewPosGroup,
	"monsterJitterX": &monsterJitterX, "monsterJitterY": &monsterJitterY,
	"animCycle": &animCycle, "mexeEffect": &mexeEffect, "mexeBackfire": &mexeBackfire, "mexeDelayed": &mexeDelayed,
	"partyCounters": &partyCounters, "g4352": &g4352, "initiative": &initiative, "g4fcc": &g4fcc,
	"combatOver": &combatOver, "viewRowsOpen": &viewRowsOpen,
	"monsterGroups": &monsterGroups, "groupMonsterRec": &groupMonsterRec,
}

var pathPart = regexp.MustCompile(`\.?(\w+)|\[(\d+)\]`)

// field resolves a case path like "monsterGroups[1].Members[0].HP".
func field(t *testing.T, path string) reflect.Value {
	var v reflect.Value
	for i, m := range pathPart.FindAllStringSubmatch(path, -1) {
		switch {
		case i == 0:
			r, ok := diffRoots[m[1]]
			if !ok {
				t.Fatalf("unknown global %q", m[1])
			}
			v = reflect.ValueOf(r).Elem()
		case m[1] != "":
			v = v.FieldByName(m[1])
		default:
			n, _ := strconv.Atoi(m[2])
			v = v.Index(n)
		}
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		if !v.IsValid() {
			t.Fatalf("bad path %q", path)
		}
	}
	return v
}

func getInt(v reflect.Value) int64 {
	if v.CanInt() {
		return v.Int()
	}
	return int64(v.Uint())
}

func setInt(v reflect.Value, n int64) {
	if v.CanInt() {
		v.SetInt(n)
	} else {
		v.SetUint(uint64(n))
	}
}

// resetDiffState zeroes every global the cases can name.
func resetDiffState() {
	for _, r := range diffRoots {
		v := reflect.ValueOf(r).Elem()
		v.Set(reflect.Zero(v.Type()))
	}
	for g := range monsterGroups {
		monsterGroups[g] = &MonsterGroup{}
		groupMonsterRec[g] = &MonsterRec{}
	}
	if scen == nil {
		dir := os.Getenv("WIZ6_DATA")
		if dir == "" {
			dir = "../../../wiz6-re/gog/x/app"
		}
		scen, _ = dbs.OpenScenario(dir) // nil: functions that read records fail
	}
	if partyPictureWin == nil {
		partyPictureWin = createWindow(0, 0, 40, 5, 8, -1, 0, false)
	}
}

func TestDiff(t *testing.T) {
	files, _ := filepath.Glob("testdata/diff/*.json.gz")
	if len(files) == 0 {
		t.Skip("no cases")
	}
	defer func() { dtLog = nil }()
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json.gz")
		t.Run(name, func(t *testing.T) {
			run := diffFuncs[name]
			if run == nil {
				t.Skipf("no Go function for %s", name)
			}
			cases := loadDiffCases(t, f)
			fails := 0
			for k, c := range cases {
				if msg := runDiffCase(t, run, c); msg != "" {
					if fails++; fails <= 5 {
						t.Errorf("case %d %v: %s", k, c.Args, msg)
					}
				}
			}
			if fails > 0 {
				t.Errorf("%d of %d cases differ", fails, len(cases))
			}
		})
	}
}

func loadDiffCases(t *testing.T, f string) []diffCase {
	r, err := os.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	z, err := gzip.NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	var cases []diffCase
	if err := json.NewDecoder(z).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func runDiffCase(t *testing.T, run func([]int) int, c diffCase) (msg string) {
	resetDiffState()
	for p, n := range c.State {
		setInt(field(t, p), n)
	}
	for i, s := range c.Seeds {
		seeds[i] = s
	}
	var rng [][]int
	var log [][]any
	dtLog = func(kind string, v ...int) {
		if kind == "rng" {
			rng = append(rng, v)
			return
		}
		log = append(log, []any{kind, float64(v[0])})
	}
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint("panic: ", r, " ", panicSite())
		}
	}()
	ret := run(append([]int(nil), c.Args...))
	var diffs []string
	if c.Ret != nil && ret != *c.Ret {
		diffs = append(diffs, fmt.Sprintf("return %d, want %d", ret, *c.Ret))
	}
	for p, want := range c.Out {
		if got := getInt(field(t, p)); got != want {
			diffs = append(diffs, fmt.Sprintf("%s = %d, want %d", p, got, want))
		}
	}
	for p, in := range c.State {
		if _, ok := c.Out[p]; !ok {
			if got := getInt(field(t, p)); got != in {
				diffs = append(diffs, fmt.Sprintf("%s = %d, want unchanged %d", p, got, in))
			}
		}
	}
	if fmt.Sprint(rng) != fmt.Sprint(c.RNG) {
		diffs = append(diffs, fmt.Sprintf("rng %v, want %v", rng, c.RNG))
	}
	if fmt.Sprint(log) != fmt.Sprint(c.Log) {
		diffs = append(diffs, fmt.Sprintf("log %v, want %v", log, c.Log))
	}
	return strings.Join(diffs, "; ")
}

// panicSite is the innermost game-code line of a panic's stack.
func panicSite() string {
	var at []string
	for _, l := range strings.Split(string(debug.Stack()), "\n") {
		if strings.Contains(l, "internal/game/") && !strings.Contains(l, "_test.go") && len(at) < 4 {
			at = append(at, filepath.Base(strings.Fields(l)[0]))
		}
	}
	return strings.Join(at, " < ")
}
