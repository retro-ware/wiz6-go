package dbs

import (
	"os"
	"strings"
	"testing"
)

const data = "/Users/felix/Projects/wiz6-re/gog/x/app"

func need(t *testing.T) {
	if _, err := os.Stat(data); err != nil {
		t.Skip("no game data")
	}
}

func TestMessagesAndScenario(t *testing.T) {
	need(t)
	m, err := OpenMessages(data)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for n := 0; n < 3000 && found < 3; n++ {
		if s := m.Get(n); len(s) > 0 {
			found++
			t.Logf("msg %d: %q", n, s)
			if strings.Trim(string(s), " ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.,!?'-:;()\"\r\n/") == string(s) {
				t.Errorf("msg %d does not look like text", n)
			}
		}
	}
	if found == 0 {
		t.Fatal("no messages decoded")
	}
	if m.Get(-30000) != nil {
		t.Error("missing message should be nil")
	}
	s, err := OpenScenario(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.RecordSize(0) != 0x40 {
		t.Errorf("size 0 = %d", s.RecordSize(0))
	}
	r, err := s.Record(9, 0) // sound 0: martydill reports phase 0x7e, level 0x49
	if err != nil || len(r) != s.RecordSize(9) {
		t.Fatal(err, len(r))
	}
	if r[8] != 0x7e || r[10] != 0x49 {
		t.Errorf("sound record 0 = % x", r)
	}
}

func TestPictures(t *testing.T) {
	need(t)
	px, err := LoadScreen(Path(data, "TITLEPAG.EGA"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[uint8]bool{}
	for _, c := range px {
		seen[c] = true
	}
	if len(seen) < 4 {
		t.Errorf("title has %d colours", len(seen))
	}
	for _, name := range []string{"MON03.PIC", "CREDITS.PIC"} {
		p, err := LoadPic(Path(data, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Frames) == 0 || p.Frames[0].W == 0 {
			t.Errorf("%s: frames %d", name, len(p.Frames))
		}
	}
	for i := 0; i < 5; i++ {
		f, err := LoadFont(Path(data, "WFONT"+string(rune('0'+i))+".EGA"))
		if err != nil || f.Mono != (i == 0) {
			t.Fatal(i, err)
		}
	}
	if f, _ := LoadFont(Path(data, "WFONT0.EGA")); f.Glyphs['A'] == f.Glyphs[' '] {
		t.Error("font 0: A blank")
	}
	ps, err := LoadPortraits(Path(data, "WPORT1.EGA"))
	if err != nil || len(ps) < 14 {
		t.Fatal(err, len(ps))
	}
	md, err := LoadMazeData(Path(data, "MAZEDATA.EGA"))
	if err != nil {
		t.Fatal(err)
	}
	if len(md.Tiles) != 153 || len(md.Draws) != 366 {
		t.Errorf("maze %d tiles %d draws", len(md.Tiles), len(md.Draws))
	}
	last := md.Tiles[len(md.Tiles)-1]
	if last.Off+4*last.W*last.H > len(md.Raw) {
		t.Error("last tile past end of file")
	}
	snd, err := LoadSound(Path(data, "SOUND00.SND"))
	if err != nil || len(snd.Data) == 0 {
		t.Fatal(err)
	}
}
