package drv

import (
	"os"
	"testing"
)

const data = "../../../wiz6-re/gog/x/app/"

func TestCreditsPic(t *testing.T) {
	b, err := os.ReadFile(data + "CREDITS.PIC")
	if err != nil {
		t.Skip("no data:", err)
	}
	LoadPic(0, b)
	FillWorkBuffer(Rect{0, 0, Cols, Rows}, 0)
	DrawFrames(0, 84, 52, 0, 319, 0, false, []byte{10})
	n := 0
	for y := range WorkBuffer {
		for _, c := range WorkBuffer[y] {
			if c != 0 {
				n++
			}
		}
	}
	if n < 500 {
		t.Fatalf("the Sir-Tech logo has %d pixels", n)
	}
}
