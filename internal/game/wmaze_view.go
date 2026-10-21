package game

import "github.com/retro-ware/wiz6-go/internal/drv"

// WMAZE: the maze view. The party looks along its facing; each of up to four rows
// ahead has a left, front and right cell (column 0, 1, 2) whose walls, doors and
// features become MAZEDATA tiles (drv_op_c6) or monster-picture frames (view objects).

// The view tables (bss 0x5040..0x50da, 0x521a..0x5228). [row][col] tables are the
// C's row*3+col arrays; the eight-byte tables are per row, and the C's g5073-style
// aliases are index +1, +2, +3 of them.
var (
	viewRow   int         // g5040: the row being drawn
	mzG5042   [4][3]uint8 // g5042: no floor/ceiling tile at the cell
	mzG504e   [4][3]int16 // g504e: animated floor at the cell
	mzG5066   [4][3]uint8 // g5066: the cell is open (no wall drawn)
	mzG5072   [8]uint8    // g5072: draw the left-front cell of the row
	viewDepth [8]uint8    // view_depth (0x507a): draw the left side of the row
	mzG5082   [8]uint8    // g5082: draw the left floor of the row
	mzG508a   [8]uint8    // g508a: draw the front of the row
	mzG5092   [8]uint8    // g5092: draw the right-front cell of the row
	mzG509a   [8]uint8    // g509a: draw the right side of the row
	mzG50a2   [8]uint8    // g50a2: draw the right floor of the row
	mzG50aa   [4][3]uint8 // g50aa: no ceiling tile at the cell
	mzG50b6   [4][3]int16 // g50b6: a stair/hole marker at the cell

	viewObjCount int            // g50ce
	viewObjs     [30]viewObject // 0x50d0

	mzParity     int // g521a: (party_x+party_y+facing)%2, flips the tile pairs
	mzRowParity  int // g521c: the same plus the row
	viewRows     int // g521e: rows to draw (4, less behind a closed front)
	mzFrontWall  int // g5220
	mzLeftWall   int // g5222
	mzRightWall  int // g5224
	mzLeftFront  int // g5226
	mzRightFront int // g5228
)

// viewObject is one 11-byte view object (0x50d0 + 11*i): a pair of MAZEDATA tiles
// (Kind 255) or a frame of the monster picture in slot Kind.
type viewObject struct {
	A, B  int16 // +0 +2: tiles (-1 none), or the frame's x, y
	C, D  int16 // +4 +6: mirrored tiles (-1 none), or the frame's clip left/right
	Kind  uint8 // +8: 255 tiles, else a picture slot
	Frame uint8 // +9: the frame, or the tile flag
	Row   uint8 // +10
}

// ROOT data: the view's x clips per row (g0042, g004a) and object x offsets (g0052).
var (
	viewClipL   = [4]int{0, 104, 128, 144}
	viewClipR   = [4]int{0, 216, 192, 176}
	viewObjXOff = [4][3]int{{0, 0, 0}, {-64, 0, 64}, {-32, 0, 32}, {-16, 0, 16}}
)

// cellIndex is the maze cell index of (cx, cy) in block.
func cellIndex(block, cx, cy int) int { return block<<6 + cy<<3 + cx }

// getWallInDirection (WMAZE 548e) is the wall code on side dir of cell (cx, cy) of
// block; (x, y) is the cell's map position. 2 and 3 look at the neighbour's wall.
func getWallInDirection(dir, block, cx, cy, x, y int) int {
	switch dir {
	case 0:
		return getBitfield(mazeA.Walls0[:], cellIndex(block, cx, cy), 2)
	case 1:
		return getBitfield(mazeA.Walls1[:], cellIndex(block, cx, cy), 2)
	case 2:
		return wallsSouthOf(x, y, cx, cy, block)
	case 3:
		return wallsWestOf(x, y, cx, cy, block)
	}
	return 0
}

// blockAt (WMAZE 7ade) is block if map position (x, y) lies in it, else -1.
func blockAt(block, x, y int) int {
	bx, by := int(mazeA.BlockX[block]), int(mazeA.BlockY[block])
	if bx > x || bx+7 < x || by > y || by+7 < y {
		return -1
	}
	return block
}

// moveToMazeCell (WMAZE 7b1b) finds the block holding map position (x, y), trying
// *block, then the last found, then the others in turn, and sets the cell in it;
// 1 if no block holds it.
func moveToMazeCell(cx, cy, block *int16, x, y int) int {
	b := blockAt(int(*block), x, y)
	if b == -1 {
		b = blockAt(int(g4fa6), x, y)
	}
	if b == -1 {
		for i := (int(*block) + 1) % 12; ; {
			if b = blockAt(i, x, y); b != -1 {
				break
			}
			if i = (i + 1) % 12; i == int(*block) {
				return 1
			}
		}
	}
	*cx = int16(x - int(mazeA.BlockX[b]))
	*cy = int16(y - int(mazeA.BlockY[b]))
	*block = int16(b)
	g4fa6 = int16(b)
	return 0
}

// findCellEvent (WMAZE 7bd2) is the first active, not yet flagged event of block at
// (cx, cy) from event index start on, or -1.
func findCellEvent(cx, cy, block, start int) int {
	for i := start; i < 144; i++ {
		if int(mazeB.Unk1296[i]) != block {
			return -1
		}
		if int(mazeB.X[i]) == cx && int(mazeB.Y[i]) == cy && mazeB.Kind[i] > 0 &&
			!testBit(levelFlags[gameStateArg][:], int(mazeB.FlagBit[i])) {
			return i
		}
	}
	return -1
}

// openEdge is the wall code at a map edge: open in maps 10 and 12, else a wall.
func openEdge() int {
	if gameStateArg != 10 && gameStateArg != 12 {
		return 2
	}
	return 0
}

// wallsSouthOf (WMAZE 7c41) is the Walls0 code of the cell south of map position (x, y).
func wallsSouthOf(x, y, cx, cy, block int) int {
	c, r, b := int16(cx), int16(cy), int16(block)
	if moveToMazeCell(&c, &r, &b, x, y-1) != 0 {
		return openEdge()
	}
	return getBitfield(mazeA.Walls0[:], cellIndex(int(b), int(c), int(r)), 2)
}

// wallsWestOf (WMAZE 7ca6) is the Walls1 code of the cell west of map position (x, y).
func wallsWestOf(x, y, cx, cy, block int) int {
	c, r, b := int16(cx), int16(cy), int16(block)
	if moveToMazeCell(&c, &r, &b, x-1, y) != 0 {
		return openEdge()
	}
	return getBitfield(mazeA.Walls1[:], cellIndex(int(b), int(c), int(r)), 2)
}

// stepRelative (WMAZE 7d0b) moves map position (x, y) by (right, ahead) relative to
// the party's facing and sets the cell; moveToMazeCell's result.
func stepRelative(x, y, cx, cy, block *int16, right, ahead int) int {
	switch partyFacing {
	case 0:
		*x += int16(right)
		*y += int16(ahead)
	case 1:
		*x += int16(ahead)
		*y -= int16(right)
	case 2:
		*x -= int16(right)
		*y -= int16(ahead)
	case 3:
		*x -= int16(ahead)
		*y += int16(right)
	}
	return moveToMazeCell(cx, cy, block, int(*x), int(*y))
}

// getFrontWall (WMAZE 7d8c) is the code of the wall facing the party in the cell side
// cells to the right of (x, y), and marks the view tables for its features.
func getFrontWall(x, y, cx, cy, block int16, side int) int {
	col := side + 1
	if side != 0 && stepRelative(&x, &y, &cx, &cy, &block, side, 0) != 0 {
		if viewRow == 0 && side == 0 {
			viewRowsOpen = 0
		}
		if gameStateArg != 10 && gameStateArg != 12 {
			return 2
		}
		mzG5042[viewRow][col] = 1
		mzG504e[viewRow][col] = 1
		return 0
	}
	cell := cellIndex(int(block), int(cx), int(cy))
	if testBit(mazeA.Bits1082[:], cell) {
		switch gameStateArg {
		case 0, 4, 5, 12:
			if gameStateArg == 12 && block < 9 {
				mzG504e[viewRow][col] = 1
				break
			}
			mzG5066[viewRow][col] = 1
			switch side {
			case -1:
				mzG5072[viewRow] = 0
				viewDepth[viewRow] = 0
				mzG5082[viewRow] = 0
			case 0:
				mzG508a[viewRow] = 0
			case 1:
				mzG5092[viewRow] = 0
				mzG509a[viewRow] = 0
				mzG50a2[viewRow] = 0
			}
			return 0
		case 1:
			mzG50aa[viewRow][col] = 1
			mzG5042[viewRow][col] = 1
		case 2, 3, 6, 11, 13:
			mzG50aa[viewRow][col] = 1
		case 8, 10:
			mzG504e[viewRow][col] = 1
		}
	}
	if testBit(mazeA.Bits1178[:], cell) {
		switch gameStateArg {
		case 0, 1, 3, 4, 6, 8, 10, 11, 12, 13:
			mzG5042[viewRow][col] = 1
		}
	}
	w := 0
	switch partyFacing {
	case 0:
		w = getBitfield(mazeA.Walls0[:], cell, 2)
	case 1:
		w = getBitfield(mazeA.Walls1[:], cell, 2)
	case 2:
		w = wallsSouthOf(int(x), int(y), int(cx), int(cy), int(block))
	case 3:
		w = wallsWestOf(int(x), int(y), int(cx), int(cy), int(block))
	}
	floor := getBitfield(mazeA.Floor[:], cell, 4)
	feat := getBitfield(mazeA.Feature[:], cell, 2)
	if feat == int(partyFacing) || floor == 6 || floor > 12 {
		switch floor {
		case 1:
			w = 5
		case 2:
			w = 6
		case 3:
			w = 8
		case 4:
			w = 9
		case 5:
			w = 14
		case 6:
			mzG50b6[viewRow][col] = 1
		case 7:
			w = 4
		case 8:
			w = 7
		case 9:
			w = 10
		case 10:
			w = 11
		case 11:
			w = 12
		case 12:
			w = 13
		case 13:
			mzG504e[viewRow][col] = 1
		case 14:
			mzG50aa[viewRow][col] = 1
		case 15:
			mzG5042[viewRow][col] = 1
		}
	}
	if w != 0 && viewRow == 0 && side == 0 {
		viewRowsOpen = 0
	}
	return w
}

// sideFeature maps a floor feature facing the side to its wall code.
var sideFeature = map[int]int{1: 5, 2: 6, 3: 8, 4: 9, 7: 4, 8: 7, 9: 10, 10: 11, 11: 12, 12: 13}

// getSideWall is get_left_wall_m / get_right_wall_m: the wall on the left (turn 1) or
// right (turn 3) of the cell side cells to the right of (x, y).
func getSideWall(x, y, cx, cy, block int16, side, turn int) int {
	if side != 0 && stepRelative(&x, &y, &cx, &cy, &block, side, 0) != 0 {
		if gameStateArg != 10 && gameStateArg != 12 {
			return 2
		}
		return 0
	}
	cell := cellIndex(int(block), int(cx), int(cy))
	if testBit(mazeA.Bits1082[:], cell) {
		switch gameStateArg {
		case 0, 4, 5, 12:
			if gameStateArg != 12 || block > 8 {
				return 0
			}
		}
	}
	w := 0
	// The wall on side dir of the cell: 0 Walls0 here, 1 Walls1 here, 2 and 3 the
	// neighbour's.
	dir := (int(partyFacing) + 4 - turn) % 4
	switch dir {
	case 0:
		w = getBitfield(mazeA.Walls0[:], cell, 2)
	case 1:
		w = getBitfield(mazeA.Walls1[:], cell, 2)
	case 2:
		w = wallsSouthOf(int(x), int(y), int(cx), int(cy), int(block))
	case 3:
		w = wallsWestOf(int(x), int(y), int(cx), int(cy), int(block))
	}
	floor := getBitfield(mazeA.Floor[:], cell, 4)
	if (getBitfield(mazeA.Feature[:], cell, 2)+turn)%4 == int(partyFacing) {
		if v, ok := sideFeature[floor]; ok {
			w = v
		}
	}
	return w
}

// getLeftWall (WMAZE 8175).
func getLeftWall(x, y, cx, cy, block int16, side int) int {
	return getSideWall(x, y, cx, cy, block, side, 1)
}

// getRightWall (WMAZE 8332).
func getRightWall(x, y, cx, cy, block int16, side int) int {
	return getSideWall(x, y, cx, cy, block, side, 3)
}

// addViewObject (WMAZE 84f1) queues view object kind (255 tiles, else a picture slot)
// for row; a picture frame's position takes the frame's offset in the slot's record.
func addViewObject(kind int, a, b int, frame uint8, c, d int, row uint8) {
	o := &viewObjs[viewObjCount]
	o.Kind = uint8(kind)
	o.A, o.B = int16(a), int16(b)
	if kind != 255 {
		o.A += int16(monsterPics[kind].Pairs[frame][0])
		o.B += int16(monsterPics[kind].Unk57[frame])
	}
	o.Frame = frame
	o.C, o.D = int16(c), int16(d)
	o.Row = row
	viewObjCount++
	if viewObjCount >= 30 {
		for {
			playSound(0) // the C hangs here
		}
	}
}

// pairTile draws tile a, or with flip tile b mirrored from a (the C's
// drv_op_c6(flip ? b : a, flag, flip ? a : -1)).
func pairTile(flip bool, a, b, flag int) {
	if flip {
		drvMazeTile(b, flag, a)
	} else {
		drvMazeTile(a, flag, -1)
	}
}

// drawFrontCell (WMAZE 85d0) draws cell col of row with front wall code wall: floor,
// wall tiles and the objects for doors, stairs and features. clipL/clipR bound the
// objects; t holds the 30 tile bases (C p14..p72), offset by the row.
func drawFrontCell(row, wall, col, clipL, clipR int, t ...int) {
	p := func(i int) int { return t[i] + row }
	if mzG5042[row][col] == 0 {
		pairTile(mzParity != 0, p(0), p(1), 1)
	}
	if row < 3 {
		rp := mzRowParity != 0
		if wall == 2 || wall >= 7 {
			pairTile(rp, p(2), p(3), 1)
		}
		switch wall {
		case 1, 3, 4:
			if wall == 3 {
				if col == 1 {
					if row == 0 && g363e > 0 {
						addViewObject(2, 0, 0, uint8(g363e+15), clipL, clipR, uint8(row))
					} else if row > 0 || g363e == 0 {
						addViewObject(255, p(4), -1, 0, -1, -1, uint8(row))
					}
				} else if row > 0 {
					addViewObject(255, p(4), -1, 0, -1, -1, uint8(row))
				}
			} else if wall == 4 && (row > 0 || col == 1) {
				addViewObject(2, viewObjXOff[row][col], 0, uint8(row+10), clipL, clipR, uint8(row))
			}
			addViewObject(255, p(5), p(7), 0, p(6), p(8), uint8(row))
			addViewObject(255, p(9), p(11), 0, p(10), p(12), uint8(row))
			if row > 0 || col == 1 {
				addViewObject(2, viewObjXOff[row][col], 0, uint8(row+13), clipL, clipR, uint8(row+1))
			}
		case 5:
			pairTile(rp, p(13), p(14), 1)
		case 6:
			pairTile(rp, p(15), p(16), 1)
		case 7:
			if rp {
				drvMazeTile(p(18), 0, -1)
			} else {
				drvMazeTile(p(17), 0, -1)
			}
		case 8:
			if row > 0 || col == 1 {
				f := row<<1 + int(animPhase) + 1
				if rp && row < 2 {
					f += 18
				}
				addViewObject(1, viewObjXOff[row][col], 0, uint8(f), clipL, clipR, uint8(row))
			}
		case 9:
			if row > 0 || col == 1 {
				addViewObject(2, viewObjXOff[row][col], 0, uint8(row+1), clipL, clipR, uint8(row))
			}
		case 10, 11, 12, 13:
			pairTile(rp, p(19), p(20), 0)
			switch wall {
			case 11:
				pairTile(rp, p(21), p(22), 0)
			case 12:
				pairTile(rp, p(23), p(24), 0)
			case 13:
				drvMazeTile(p(25), 0, -1)
			}
		case 14:
			if row == 0 && col == 1 {
				pairTile(rp, 344, 344, 0)
			}
		}
	}
	if mzG50aa[row][col] != 0 {
		return
	}
	if mzG504e[row][col] == 0 {
		pairTile(mzParity != 0, p(26), p(27), 1)
		if row == 1 && col == 1 && mzG50b6[row][col] == 1 {
			pairTile(mzParity != 0, 345, 345, 0)
		}
		return
	}
	pairTile((mzParity+int(animPhase))%2 != 0, p(28), p(29), 1)
}

// drawSideWall (WMAZE 8b18) draws the side wall wall of row (f1, f2 frame bases of the
// side's objects); t holds the 12 tile bases (C p12..p34).
func drawSideWall(row, wall, f1, f2 int, t ...int) {
	p := func(i int) int { return t[i] + row }
	fl := mzParity != 0
	if row == 3 && wall != 0 || wall == 2 || wall >= 7 {
		pairTile(fl, p(0), p(1), 1)
	}
	if row >= 3 {
		return
	}
	flag := 1
	if row < 3 {
		flag = 0
	}
	switch wall {
	case 1, 3, 4:
		pairTile(fl, p(2), p(3), 1)
		if wall == 3 {
			drvMazeTile(p(4), flag, -1)
		} else if wall == 4 {
			drvMazeTile(p(5), flag, -1)
		}
	case 5, 6:
		pairTile(fl, p(6), p(7), 1)
	case 7:
		pairTile(fl, p(8), p(9), 0)
	case 8:
		addViewObject(1, 0, 0, uint8(row<<1+f1+int(animPhase)), 72, 248, uint8(row))
	case 9:
		addViewObject(2, 0, 0, uint8(row+f2), 72, 248, uint8(row))
	case 10, 11, 12, 13:
		pairTile(fl, p(10), p(11), 0)
	}
}

// drawFloorCeiling (WMAZE 8d07) draws the floor and ceiling of cell col of row.
func drawFloorCeiling(row, col, a, b, c, d, e, f int) {
	fl := mzParity != 0
	if mzG5042[row][col] == 0 {
		pairTile(fl, a+row, b+row, 1)
	}
	if mzG50aa[row][col] != 0 {
		return
	}
	if mzG504e[row][col] == 0 {
		pairTile(fl, c+row, d+row, 1)
		return
	}
	pairTile((mzParity+int(animPhase))%2 != 0, e+row, f+row, 1)
}

// hideBehindFront (WMAZE 8df6): a closed front hides the rows behind it.
func hideBehindFront(row, wall int) {
	if wall != 2 && wall < 5 && mzG5066[row][1] != 1 {
		return
	}
	mzG508a[row+1], mzG508a[row+2], mzG508a[row+3] = 0, 0, 0
	viewDepth[row+1] = 0
	mzG5092[row+1] = 0
	mzG5072[row+2] = 0
	viewDepth[row+2] = 0
	mzG5092[row+2] = 0
	mzG509a[row+2] = 0
	if row+3 < viewRows {
		viewRows = row + 3
	}
}

// hideBehindLeft (WMAZE 8e59): a left wall hides the left cells behind it.
func hideBehindLeft(row, wall int) {
	if wall != 0 || mzG5066[row][1] == 1 {
		viewDepth[row] = 0
		mzG5072[row+1] = 0
		mzG5082[row+2] = 0
	}
}

// hideBehindRight (WMAZE 8e8a): a right wall hides the right cells behind it.
func hideBehindRight(row, wall int) {
	if wall != 0 || mzG5066[row][1] == 1 {
		mzG5092[row] = 0
		mzG509a[row+1] = 0
		mzG50a2[row+2] = 0
	}
}

// wmazeF8ebb (WMAZE 8ebb).
func wmazeF8ebb(row, wall int) {
	if wall == 2 || wall >= 5 || mzG5066[row][0] == 1 {
		mzG5082[row+1] = 0
	}
}

// wmazeF8ee8 (WMAZE 8ee8).
func wmazeF8ee8(row, wall int) {
	if wall == 2 || wall >= 5 || mzG5066[row][0] == 1 {
		mzG5072[row+1] = 0
		mzG5082[row+2] = 0
	}
}

// wmazeF8f1a (WMAZE 8f1a).
func wmazeF8f1a(row, wall int) {
	if wall == 2 || wall >= 5 || mzG5066[row][2] == 1 {
		mzG509a[row+1] = 0
		mzG50a2[row+2] = 0
	}
}

// wmazeF8f4c (WMAZE 8f4c).
func wmazeF8f4c(row, wall int) {
	if wall == 2 || wall >= 5 || mzG5066[row][2] == 1 {
		mzG50a2[row+1] = 0
	}
}

// drawViewFrame (WMAZE 8f79) draws the six frame ornaments around the view (lit while
// the party counter of the same index runs; ornament 0 shows the facing) and shows
// the title strip.
func drawViewFrame() {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	for i := range 6 {
		var tile, frame int
		switch i {
		case 0:
			tile, frame = 361, int(partyFacing)+1
		case 1:
			tile, frame = 349, int(animPhase)+1
		case 2:
			tile, frame = 355, int(animPhase)*int(g4f86)+1
		case 3:
			tile, frame = 346, int(animPhase)+1
		case 4:
			tile, frame = 352, int(animPhase)+1
		case 5:
			tile, frame = 358, int(animPhase)+1
		}
		if partyCounters[i].A > 0 {
			tile += frame
		}
		drvMazeTile(tile, 0, -1)
	}
	drvShowWindow(g4fac)
}

// drawMazeView (WMAZE 903b) draws the view from the party's position into the
// working buffer, row by row from the front, then the queued objects from the back.
func drawMazeView() {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	x, y, cx, cy, block := partyX, partyY, cellX, cellY, partyBlock
	g4fa6 = 0
	viewRowsOpen = 1
	drvFillBack(pictureWin, 0)
	if testBit(mazeA.Bits1082[:], cellIndex(int(block), int(cx), int(cy))) {
		switch gameStateArg {
		case 0, 4, 5:
			drawViewFrame()
			return
		}
	}
	for i := range 4 {
		mzG508a[i], mzG5072[i], viewDepth[i] = 1, 1, 1
		mzG5092[i], mzG509a[i], mzG5082[i], mzG50a2[i] = 1, 1, 1, 1
	}
	mzG5066, mzG5042, mzG50aa = [4][3]uint8{}, [4][3]uint8{}, [4][3]uint8{}
	mzG504e, mzG50b6 = [4][3]int16{}, [4][3]int16{}
	viewObjs = [30]viewObject{}
	viewObjCount = 0
	mzG5072[0], mzG509a[0], mzG5082[0], mzG5082[1], mzG50a2[0], mzG50a2[1] = 0, 0, 0, 0, 0, 0
	b2 := block
	stepRelative(&x, &y, &cx, &cy, &b2, 0, -1)
	viewRows = 4
	mzParity = int(partyX+partyY+partyFacing) % 2
	for viewRow = 0; viewRow < viewRows; viewRow++ {
		r := viewRow
		mzRowParity = (int(partyX+partyY+partyFacing) + r) % 2
		if stepRelative(&x, &y, &cx, &cy, &block, 0, 1) != 0 {
			if gameStateArg == 10 || gameStateArg == 12 {
				mzG5042[r][1] = 1
				// The C indexes g5050 here as a byte array: byte 2+3r of the int16
				// table g504e, which sets the low or high byte of one entry.
				b := 2 + 3*r
				mzG504e[b/2/3][b/2%3] |= 1 << (8 * (b % 2))
				mzFrontWall, mzLeftWall, mzRightWall = 0, 0, 0
			} else {
				mzFrontWall, mzLeftWall, mzRightWall = 2, 2, 2
			}
		} else {
			mzFrontWall = getFrontWall(x, y, cx, cy, block, 0)
			mzLeftWall = getLeftWall(x, y, cx, cy, block, 0)
			mzRightWall = getRightWall(x, y, cx, cy, block, 0)
		}
		mzLeftFront = getFrontWall(x, y, cx, cy, block, -1)
		mzRightFront = getFrontWall(x, y, cx, cy, block, 1)
		if mzG508a[r] == 1 {
			drawFrontCell(r, mzFrontWall, 1, 72, 248, 122, 122, 0, 0, 91, 23, 23, 26, 26, 29, 32, 32, 29, 178, 178, 199, 199, 242, 257, 278, 278, 299, 299, 314, 314, 329, 150, 150, 214, 214)
		}
		hideBehindFront(r, mzFrontWall)
		if mzG508a[r] == 1 {
			drawSideWall(r, mzLeftWall, 7, 4, 15, 19, 83, 87, 106, 114, 193, 196, 272, 275, 293, 296)
		}
		hideBehindLeft(r, mzLeftWall)
		if mzG508a[r] == 1 {
			drawSideWall(r, mzRightWall, 13, 7, 19, 15, 87, 83, 110, 118, 196, 193, 275, 272, 296, 293)
		}
		hideBehindRight(r, mzRightWall)
		if mzG5082[r] == 1 {
			drawFloorCeiling(r, 0, 126, 146, 154, 174, 218, 238)
		}
		if mzG5072[r] == 1 {
			drawFrontCell(r, mzLeftFront, 0, 72, viewClipL[r], 130, 142, 3, 12, 94, 35, 62, 41, 68, 47, 80, 53, 74, 181, 190, 202, 211, 245, 260, 281, 290, 302, 311, 317, 326, 332, 158, 170, 222, 234)
		}
		wmazeF8ebb(r, mzLeftFront)
		if viewDepth[r] == 1 {
			drawFrontCell(r, mzLeftFront, 0, viewClipL[r], 248, 134, 138, 6, 9, 97, 38, 59, 44, 65, 50, 77, 56, 71, 184, 187, 205, 208, 248, 263, 284, 287, 305, 308, 320, 323, 335, 162, 166, 226, 230)
		}
		wmazeF8ee8(r, mzLeftFront)
		if mzG5092[r] == 1 {
			drawFrontCell(r, mzRightFront, 2, 72, viewClipR[r], 138, 134, 9, 6, 100, 59, 38, 65, 44, 71, 56, 77, 50, 187, 184, 208, 205, 251, 266, 287, 284, 308, 305, 323, 320, 338, 166, 162, 230, 226)
		}
		wmazeF8f1a(r, mzRightFront)
		if mzG509a[r] == 1 {
			drawFrontCell(r, mzRightFront, 2, viewClipR[r], 248, 142, 130, 12, 3, 103, 62, 35, 68, 41, 74, 53, 80, 47, 190, 181, 211, 202, 254, 269, 290, 281, 311, 302, 326, 317, 341, 170, 158, 234, 222)
		}
		wmazeF8f4c(r, mzRightFront)
		if mzG50a2[r] == 1 {
			drawFloorCeiling(r, 2, 146, 126, 174, 154, 238, 218)
		}
	}
	for viewRow = viewRows; viewRow >= 0; viewRow-- {
		for i := viewObjCount - 1; i >= 0; i-- {
			o := &viewObjs[i]
			mzRowParity = (int(o.Row) + int(partyX+partyY+partyFacing)) % 2
			if int(o.Row) != viewRow || o.Kind != 255 {
				continue
			}
			flip := mzRowParity != 0
			if o.A != -1 {
				if o.C != -1 {
					pairTile(flip, int(o.A), int(o.C), int(o.Frame))
				} else {
					drvMazeTile(int(o.A), int(o.Frame), -1)
				}
			}
			if o.B != -1 {
				if o.D != -1 {
					pairTile(flip, int(o.B), int(o.D), int(o.Frame))
				} else {
					drvMazeTile(int(o.B), int(o.Frame), -1)
				}
			}
		}
		for i := viewObjCount - 1; i >= 0; i-- {
			o := &viewObjs[i]
			if int(o.Row) == viewRow && o.Kind != 255 {
				drv.DrawFrames(int(o.Kind), int(o.A), int(o.B), int(o.C), int(o.D), 0, false, []byte{o.Frame})
			}
		}
	}
	drawViewFrame()
}

// redrawMazeView (WMAZE 98cb) draws the view and shows it.
func redrawMazeView() {
	if dtLog != nil { // stubbed in the difftest
		return
	}
	drawMazeView()
	drvShowWindow(pictureWin)
}
