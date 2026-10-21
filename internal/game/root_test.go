package game

import (
	"testing"

	"github.com/retro-ware/wiz6-go/internal/drv"
)

func TestBits(t *testing.T) {
	a := make([]byte, 8)
	setBitfield(a, 3, 5, 0x15)
	setBitfield(a, 4, 5, 0x0a)
	if getBitfield(a, 3, 5) != 0x15 || getBitfield(a, 4, 5) != 0x0a || getBitfield(a, 2, 5) != 0 {
		t.Fatalf("bitfields % x", a)
	}
	setBit(a, 63, true)
	if !testBit(a, 63) || a[7] != 0x80 {
		t.Fatal("setBit")
	}
	if extractBits(0xabcd, 4, 8) != 0xbc {
		t.Fatal("extractBits")
	}
}

func TestRandomRange(t *testing.T) {
	for range 1000 {
		if r := random(6); r < 0 || r >= 6 {
			t.Fatal(r)
		}
	}
	if random(0) != 0 {
		t.Fatal("random(0)")
	}
}

func TestWindowsOwnCells(t *testing.T) {
	windows, cellOwner = nil, [len(cellOwner)]uint8{}
	low := createWindow(0, 0, 10, 5, 1, -1, 0, false)
	high := createWindow(2, 2, 4, 2, 9, -1, 0, false)
	if cellOwner[2*drv.Cols+3]&cellIDMask != windowID(high) || cellOwner[0]&cellIDMask != windowID(low) {
		t.Fatal("ownership after link")
	}
	destroyWindow(high, false)
	if cellOwner[2*drv.Cols+3]&cellIDMask != windowID(low) {
		t.Fatal("cells not handed back")
	}
}
