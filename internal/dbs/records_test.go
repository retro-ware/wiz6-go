package dbs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const appDir = "../../../wiz6-re/gog/x/app"

func load(t *testing.T, name string) []byte {
	b, err := os.ReadFile(filepath.Join(appDir, name))
	if err != nil {
		t.Skip(err)
	}
	return b
}

func roundTrip[T any](t *testing.T, what string, b []byte) T {
	t.Helper()
	v, err := Decode[T](b)
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if e := Encode(&v); !bytes.Equal(e, b) {
		t.Fatalf("%s: re-encode differs", what)
	}
	return v
}

func TestRecordSizes(t *testing.T) {
	for _, c := range []struct{ got, want int }{
		{Size[PC](), 432}, {Size[Item](), 8}, {Size[Hand](), 16}, {Size[PCFileHeader](), 24},
		{Size[MasterHeader](), 66}, {Size[DiskHeader](), 700}, {Size[ScenarioHeader](), 414},
		{Size[SaveState](), 66}, {Size[ClassExp](), 64}, {Size[ItemRec](), 74}, {Size[MapA](), 1346},
		{Size[MapB](), 1740}, {Size[MonsterRec](), 222}, {Size[NPCRec](), 142}, {Size[EventRec](), 32},
		{Size[TreasureRec](), 40}, {Size[MonsterPicRec](), 314}, {Size[SoundRec](), 12},
	} {
		if c.got != c.want {
			t.Errorf("size %d, want %d", c.got, c.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	pf := load(t, "PCFILE.DBS")
	h := roundTrip[PCFileHeader](t, "pcfile hdr", pf[:24])
	if h.RecSize != 432 {
		t.Fatalf("PC size %d", h.RecSize)
	}
	for i := 0; i < int(h.Count); i++ {
		o := int(h.Offset) + 432*i
		roundTrip[PC](t, "pc", pf[o:o+432])
	}

	m := roundTrip[MasterHeader](t, "master", load(t, "MASTER.HDR"))
	d := roundTrip[DiskHeader](t, "disk", load(t, "DISK.HDR"))
	roundTrip[ScenarioHeader](t, "scenario hdr", load(t, "SCENARIO.HDR"))
	sc := load(t, "SCENARIO.DBS")
	check := []func(string, []byte){
		func(w string, b []byte) { roundTrip[ClassExp](t, w, b) },
		func(w string, b []byte) { roundTrip[ItemRec](t, w, b) },
		func(w string, b []byte) { roundTrip[MapA](t, w, b) },
		func(w string, b []byte) { roundTrip[MapB](t, w, b) },
		func(w string, b []byte) { roundTrip[MonsterRec](t, w, b) },
		func(w string, b []byte) { roundTrip[NPCRec](t, w, b) },
		func(w string, b []byte) { roundTrip[EventRec](t, w, b) },
		func(w string, b []byte) { roundTrip[TreasureRec](t, w, b) },
		func(w string, b []byte) { roundTrip[MonsterPicRec](t, w, b) },
		func(w string, b []byte) { roundTrip[SoundRec](t, w, b) },
	}
	for ty, f := range check {
		sz, n := int(m.RecordSizes[ty]), int(m.RecordCounts[ty])
		for i := 0; i < n; i++ {
			o := int(d.Offsets[ty]) + sz*i
			if o+sz > len(sc) {
				break // counts are capacities; the file ends early for some types
			}
			f("scenario rec", sc[o:o+sz])
		}
	}

	ng := load(t, "NEWGAME.DBS")
	g, err := DecodeSaveGame(ng)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g.Encode(), ng) {
		t.Fatal("NEWGAME.DBS re-encode differs")
	}
}
