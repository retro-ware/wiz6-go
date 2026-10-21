package dbs

import (
	"fmt"
	"os"
)

// Scenario gives raw record access to SCENARIO.DBS, laid out by MASTER.HDR
// (record_sizes, record_packed) and DISK.HDR (per-type file offsets and index
// remap tables), as access_scenario_record (ROOT 0x0882) uses them.
type Scenario struct {
	Hdr    []byte // SCENARIO.HDR, 414 bytes (scenario_hdr); typed elsewhere
	Disk   DiskHeader
	Master MasterHeader
	// remapTables is DISK.HDR from MapIndex on, as read: an index past the
	// end of its table reads on into the next one, as the C does.
	remapTables []byte
	path        string
	dbs         []byte
	tree        Huffman
}

// OpenScenario reads SCENARIO.HDR, MASTER.HDR, DISK.HDR, MISC.HDR and
// SCENARIO.DBS from dir.
func OpenScenario(dir string) (*Scenario, error) {
	s := &Scenario{path: Path(dir, "SCENARIO.DBS")}
	var disk, master []byte
	var err error
	for _, f := range []struct {
		name string
		dst  *[]byte
		min  int
	}{{"SCENARIO.HDR", &s.Hdr, Size[ScenarioHeader]()}, {"DISK.HDR", &disk, Size[DiskHeader]()}, {"MASTER.HDR", &master, Size[MasterHeader]()}, {"SCENARIO.DBS", &s.dbs, 0}} {
		if *f.dst, err = readFile(dir, f.name); err != nil {
			return nil, err
		}
		if len(*f.dst) < f.min {
			return nil, fmt.Errorf("%s: %d bytes, want %d", f.name, len(*f.dst), f.min)
		}
	}
	if s.Disk, err = Decode[DiskHeader](disk[:Size[DiskHeader]()]); err != nil {
		return nil, err
	}
	s.remapTables = disk[Size[DiskHeader]()-len(s.Disk.MapIndex)-len(s.Disk.NPCIndex)-len(s.Disk.TreasureIndex)-len(s.Disk.MonsterIndex)-len(s.Disk.MonsterPicIndex):]
	if s.Master, err = Decode[MasterHeader](master[:Size[MasterHeader]()]); err != nil {
		return nil, err
	}
	if s.tree, err = ReadMiscTable(dir); err != nil {
		return nil, err
	}
	return s, nil
}

// ReadDiskHeader returns DISK.HDR's 700 bytes.
func ReadDiskHeader(dir string) ([]byte, error) { return readFile(dir, "DISK.HDR") }

// RecordSize is record_sizes[typ] (MASTER.HDR word typ).
func (s *Scenario) RecordSize(typ int) int { return int(s.Master.RecordSizes[typ]) }

func (s *Scenario) packed(typ int) bool { return s.Master.RecordPacked[typ] != 0 }

// indexTableStart is where the DISK.HDR table that remaps a record index of
// type typ starts in remapTables, or -1 for the types read by their own index.
func (s *Scenario) indexTableStart(typ int) int {
	d := &s.Disk
	mapStart := 0
	npcStart := mapStart + len(d.MapIndex)
	treasureStart := npcStart + len(d.NPCIndex)
	monsterStart := treasureStart + len(d.TreasureIndex)
	monsterPicStart := monsterStart + len(d.MonsterIndex)
	switch typ {
	case RecMapA, RecMapB:
		return mapStart
	case RecNPC:
		return npcStart
	case RecTreasure:
		return treasureStart
	case RecMonster:
		return monsterStart
	case RecMonsterPic:
		return monsterPicStart
	}
	return -1
}

// offset maps (typ, index) to a file offset: types 2,3,4,5,7,8 first remap the
// index through DISK.HDR tables, then base = DISK.HDR Offsets[typ].
func (s *Scenario) offset(typ, index int) int64 {
	if t := s.indexTableStart(typ); t >= 0 {
		index = int(s.remapTables[t+index])
	}
	return int64(index)*int64(s.RecordSize(typ)) + int64(uint32(s.Disk.Offsets[typ]))
}

// Record is access_scenario_record mode 0: the record's bytes, or for a packed
// type its decode_text output. Bytes past the file end read as 0 (DOS read
// stops short and leaves the buffer as it was).
func (s *Scenario) Record(typ, index int) ([]byte, error) {
	if typ < RecClassExp || typ > RecSound {
		return nil, fmt.Errorf("scenario: bad record type %d", typ)
	}
	off, n := s.offset(typ, index), s.RecordSize(typ)
	b := make([]byte, n)
	if off < int64(len(s.dbs)) {
		copy(b, s.dbs[off:])
	}
	if s.packed(typ) {
		return s.tree.DecodeText(b), nil
	}
	return b, nil
}

// WriteRecord is mode 1: it writes b over the record in memory and in the
// file. Packed types are not written, as in the C.
func (s *Scenario) WriteRecord(typ, index int, b []byte) error {
	if s.packed(typ) {
		return nil
	}
	off, n := s.offset(typ, index), s.RecordSize(typ)
	if len(b) < n {
		return fmt.Errorf("scenario: record %d/%d needs %d bytes", typ, index, n)
	}
	if end := int(off) + n; end > len(s.dbs) {
		s.dbs = append(s.dbs, make([]byte, end-len(s.dbs))...)
	}
	copy(s.dbs[off:], b[:n])
	f, err := os.OpenFile(s.path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteAt(b[:n], off)
	return err
}
