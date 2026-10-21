package dbs

import "os"

// Sound is a decoded SOUNDnn.SND as load_sound_slot (ROOT 0x33e9) leaves it.
// The file is either word 0 then raw bytes, or word treeLen, the tree, word
// count, and a bit stream decoded like text (f34bc). Data is not PCM: the
// IRQ0 sound driver (ROOT 0x1462) maps each byte through a 256-entry table
// built from the sound's SCENARIO type-9 record (level byte +0x0A, rate word
// +0x08) and writes it as a PC-speaker PIT channel 2 reload, i.e. each byte
// is one pulse width sample. Treat it as 8-bit unsigned samples at the rate
// the type-9 record gives until that driver is ported.
type Sound struct {
	Data []byte
}

// LoadSound reads and decodes a .SND file.
func LoadSound(path string) (*Sound, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 2 {
		return &Sound{}, nil
	}
	tl := le16(b, 0)
	if tl == 0 {
		return &Sound{Data: b[2:]}, nil
	}
	if 2+tl+2 > len(b) {
		return &Sound{}, nil
	}
	tree := parseHuffman(b[2 : 2+tl])
	return &Sound{Data: tree.decode(b[4+tl:], le16(b, 2+tl))}, nil
}
