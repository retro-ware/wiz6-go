package dbs

import "fmt"

// Huffman is a decode tree: node i is Huffman[i] = {left, right}. A child >= 0
// is a leaf (its low byte is the output); a child < 0 points at node -child.
// The text tree is MISC.HDR (load_misc_table, ROOT 0x2985, reads its 1024
// bytes); each SOUNDnn.SND carries its own tree.
type Huffman [][2]int16

const (
	huffmanNodeSize = 4    // two int16 children
	miscTableSize   = 1024 // MISC.HDR: 256 nodes
	bitsPerByte     = 8
)

func parseHuffman(b []byte) Huffman {
	h := make(Huffman, len(b)/huffmanNodeSize)
	for i := range h {
		h[i], _ = Decode[[2]int16](b[huffmanNodeSize*i : huffmanNodeSize*(i+1)])
	}
	return h
}

// ReadMiscTable reads MISC.HDR, the text Huffman tree (load_misc_table).
func ReadMiscTable(dir string) (Huffman, error) {
	b, err := readFile(dir, "MISC.HDR")
	if err != nil {
		return nil, err
	}
	if len(b) < miscTableSize {
		return nil, fmt.Errorf("MISC.HDR: %d bytes, want %d", len(b), miscTableSize)
	}
	return parseHuffman(b[:miscTableSize]), nil
}

// decode emits n leaf bytes from the MSB-first bit stream src (ROOT f2a18 and
// the identical sound decoder f34d5). Bits past the end of src read as 0.
func (h Huffman) decode(src []byte, n int) []byte {
	out := make([]byte, 0, n)
	pos, bit := 0, 0
	next := func() int {
		var v byte
		if pos < len(src) {
			v = src[pos]
		}
		b := int(v>>(bitsPerByte-1-bit)) & 1
		if bit++; bit == bitsPerByte {
			bit, pos = 0, pos+1
		}
		return b
	}
	for len(out) < n {
		node := 0
		for {
			c := h[node][next()]
			if c >= 0 {
				out = append(out, byte(c))
				break
			}
			node = int(-c)
			if node >= len(h) {
				return out // corrupt tree
			}
		}
	}
	return out
}

// DecodeText is decode_text (ROOT 0x29d9): src[0] is the decoded length, the
// bit stream follows. The asm then stores two zero bytes; the result here
// omits the terminator.
func (h Huffman) DecodeText(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	return h.decode(src[1:], int(src[0]))
}
