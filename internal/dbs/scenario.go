package dbs

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Scenario gives raw record access to SCENARIO.DBS, laid out by MASTER.HDR
// (record_sizes, record_packed) and DISK.HDR (per-type file offsets and index
// remap tables), as access_scenario_record (ROOT 0x0882) uses them.
type Scenario struct {
	Hdr    []byte // SCENARIO.HDR, 414 bytes (scenario_hdr); typed elsewhere
	Disk   []byte // DISK.HDR, 700 bytes (disk_header)
	Master []byte // MASTER.HDR, 66 bytes
	path   string
	dbs    []byte
	tree   Huffman
}

// OpenScenario reads SCENARIO.HDR, MASTER.HDR, DISK.HDR, MISC.HDR and
// SCENARIO.DBS from dir.
func OpenScenario(dir string) (*Scenario, error) {
	s := &Scenario{path: Path(dir, "SCENARIO.DBS")}
	var err error
	for _, f := range []struct {
		name string
		dst  *[]byte
		min  int
	}{{"SCENARIO.HDR", &s.Hdr, 414}, {"DISK.HDR", &s.Disk, 700}, {"MASTER.HDR", &s.Master, 66}, {"SCENARIO.DBS", &s.dbs, 0}} {
		if *f.dst, err = readFile(dir, f.name); err != nil {
			return nil, err
		}
		if len(*f.dst) < f.min {
			return nil, fmt.Errorf("%s: %d bytes, want %d", f.name, len(*f.dst), f.min)
		}
	}
	if s.tree, err = ReadMiscTable(dir); err != nil {
		return nil, err
	}
	return s, nil
}

// ReadDiskHeader returns DISK.HDR's 700 bytes.
func ReadDiskHeader(dir string) ([]byte, error) { return readFile(dir, "DISK.HDR") }

// RecordSize is record_sizes[typ] (MASTER.HDR word typ).
func (s *Scenario) RecordSize(typ int) int { return le16(s.Master, 2*typ) }

func (s *Scenario) packed(typ int) bool { return s.Master[0x38+typ] != 0 }

// offset maps (typ, index) to a file offset: types 2,3,4,5,7,8 first remap the
// index through DISK.HDR tables, then base = DISK.HDR long at 4+4*typ.
func (s *Scenario) offset(typ, index int) int64 {
	remap := map[int]int{2: 0x2c, 3: 0x2c, 5: 0x3c, 7: 0x7c, 4: 0x17c, 8: 0x27c}
	if t, ok := remap[typ]; ok {
		index = int(s.Disk[t+index])
	}
	return int64(index)*int64(s.RecordSize(typ)) + int64(binary.LittleEndian.Uint32(s.Disk[4+4*typ:]))
}

// Record is access_scenario_record mode 0: the record's bytes, or for a packed
// type its decode_text output. Bytes past the file end read as 0 (DOS read
// stops short and leaves the buffer as it was).
func (s *Scenario) Record(typ, index int) ([]byte, error) {
	if typ < 0 || typ > 9 {
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
