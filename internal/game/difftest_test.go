package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
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

// diffFuncs runs the Go port of each tested original function, by case file name;
// diffRoots are the globals the cases name. Each area's difftest_<area>_test.go fills
// them in its init.
var (
	diffFuncs = map[string]func(a []int) int{}
	diffRoots = map[string]any{}
)

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
