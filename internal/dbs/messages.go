package dbs

import "fmt"

// msgEntry is one 6-byte MSG.HDR index entry: messages First..First+Count
// start at byte Pos of 1K block Block, each as a length byte plus that many
// encoded bytes.
type msgEntry struct {
	First int16
	Pos   uint16
	Count uint8
	Block uint8
}

// Messages is MSG.HDR (the index) plus MSG.DBS (1K blocks of encoded text)
// and the MISC.HDR tree.
type Messages struct {
	index []msgEntry
	dbs   []byte
	Tree  Huffman
}

// OpenMessages reads MSG.HDR, MSG.DBS and MISC.HDR from dir.
func OpenMessages(dir string) (*Messages, error) {
	hdr, err := readFile(dir, "MSG.HDR")
	if err != nil {
		return nil, err
	}
	dbs, err := readFile(dir, "MSG.DBS")
	if err != nil {
		return nil, err
	}
	tree, err := ReadMiscTable(dir)
	if err != nil {
		return nil, err
	}
	n := le16(hdr, 0)
	if 2+6*n > len(hdr) {
		return nil, fmt.Errorf("MSG.HDR: %d entries do not fit", n)
	}
	m := &Messages{dbs: dbs, Tree: tree}
	for i := 0; i < n; i++ {
		e := hdr[2+6*i:]
		m.index = append(m.index, msgEntry{int16(le16(e, 0)), uint16(le16(e, 2)), e[4], e[5]})
	}
	return m, nil
}

// find is find_message_index (ROOT 0x063a): a binary search for the entry
// with First <= msg <= First+Count.
func (m *Messages) find(msg int) *msgEntry {
	lo, hi := 0, len(m.index)-1
	for lo <= hi {
		mid := (hi-lo)/2 + lo
		e := &m.index[mid]
		switch {
		case int(e.First) > msg:
			hi = mid - 1
		case int(e.Count)+int(e.First) < msg:
			lo = mid + 1
		default:
			return e
		}
	}
	return nil
}

// Get is get_message (ROOT 0x075b): the decoded text of message n, control
// bytes kept, without the terminator. It returns nil when n is not indexed;
// the C then sets msg_not_found and leaves the caller's buffer untouched.
// load_message_block's 4-block cache is unobservable and not modelled; a
// message running past its block continues in block+1, as the file does.
func (m *Messages) Get(n int) []byte {
	e := m.find(n)
	if e == nil {
		return nil
	}
	base := int(e.Block) << 10
	pos := int(e.Pos)
	for i := int(e.First); i != n; i++ {
		pos += int(m.at(base+pos)) + 1
	}
	l := int(m.at(base + pos))
	// The l stored bytes are decode_text's input: its own count byte, then bits.
	raw := make([]byte, 0, l)
	for i := 0; i < l; i++ {
		raw = append(raw, m.at(base+pos+1+i))
	}
	return m.Tree.DecodeText(raw)
}

func (m *Messages) at(i int) byte {
	if i < len(m.dbs) {
		return m.dbs[i]
	}
	return 0
}
