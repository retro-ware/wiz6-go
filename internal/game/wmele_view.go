package game

// WMELE's encounter view. The view tick (animate_view_step, 6d4d), its full cycle
// (redraw_view_background, 6e01), the sprite list (add_view_sprite 65bf,
// draw_view_sprite 6913) and the monster frames (get_monster_anim_chars 6326) are the
// same code as WPOPS's: animateCombatStep, animateUntilReset, insertSprite,
// drawSprite and buildMonsterAnimFrames.

func animateViewStep() {
	if dtLog == nil {
		animateCombatStep()
	}
}

func redrawViewBackground() {
	if dtLog == nil {
		animateUntilReset()
	}
}

// queueMonsterSprite (queue_monster_sprite, WMELE 66bf) queues the monster at view row,
// col shifted dx pixels (the encounter flash); WPOPS's copy has no shift.
func queueMonsterSprite(row, col, dx int) {
	if dtLog != nil {
		return
	}
	g := int(viewPosGroup[row][col])
	if g < 1 || g > 3 || row > int(viewRowsOpen) {
		return
	}
	slot := int(monsterGroups[g].PicSlot)
	p := &monsterPics[slot]
	var frames []byte
	if row == 0 {
		frames = buildMonsterAnimFrames(g, col)
	} else if p.Unk306[0] != 0 {
		frames = []byte{p.Unk306[0]}
	}
	if len(frames) == 0 {
		return
	}
	mirror := col % 2
	if random(10) == 9 {
		monsterJitterX[row][col] = uint8(random(int(p.Unk309[1])))
		monsterJitterY[row][col] = uint8(random(int(p.Unk309[2])))
	}
	f := int(frames[0])
	if p.Unk309[0] != 0 {
		dx = 0
	}
	depth := int(p.Quads[f][0])
	if depth == 0 {
		depth = int(viewDepthBase[row*5+col]) - g
	}
	x := int(p.Offsets[row][col][0]) + int(monsterJitterX[row][col]) + dx + int(p.FrameX[f][mirror])
	if x < 0 {
		x = 0
	} else if x >= 248 {
		return
	}
	y := int(p.Offsets[row][col][1]) + int(monsterJitterY[row][col]) + int(p.FrameY[f])
	insertSprite(depth, slot, x, y, 72, 248, mirror, frames)
}

// placeGroupAt (place_group_at, WMELE 6af9) puts a not yet placed member of group g at
// view row, col to flash in; false if none can go there.
func placeGroupAt(g, row, col int) bool {
	if g > monsterGroupCount {
		return false
	}
	grp := monsterGroups[g]
	if row == 1 && monsterPics[grp.PicSlot].Unk306[0] == 0 {
		return false
	}
	for i := 0; int(grp.Count) > i; i++ {
		c := &grp.Members[i]
		if c.Pos != -1 {
			continue
		}
		meleFlash[row][col] = 1
		viewPosGroup[row][col] = int16(g)
		c.Pos = int8(row*10 + col)
		if row == 0 {
			chooseMonsterFrames(g, col)
		}
		return true
	}
	return false
}

// placeFirstGroupFrom (place_first_group_from, WMELE 6bcf) places the first of groups
// g..3 that fits at row, col.
func placeFirstGroupFrom(g, row, col int) {
	for ; g <= 3; g++ {
		if placeGroupAt(g, row, col) {
			return
		}
	}
}

// arrangeMonsterPositions (arrange_monster_positions, WMELE 6bf6) fills the free view
// positions in random order: the near columns with the first groups, the far ones with
// the groups behind.
func arrangeMonsterPositions() {
	for row := 0; row <= int(viewRowsOpen); row++ {
		var used [5]bool
		meleFlash[row] = [5]int16{}
		for n := 0; n < 5; n++ {
			col := random(5)
			for used[col] {
				col = random(5)
			}
			if !used[0] && meleNPCFirst == 1 {
				col = 0
			}
			used[col] = true
			if viewPosGroup[row][col] != 0 {
				continue
			}
			switch {
			case row == 0 && (col < 3 || monsterGroupCount == 1):
				placeFirstGroupFrom(1, row, col)
			case row == 0 && (col == 3 || viewRowsOpen == 1 || monsterGroupCount == 2):
				placeFirstGroupFrom(2, row, col)
			case row == 0:
				placeFirstGroupFrom(3, row, col)
			case monsterGroupCount == 1:
				placeFirstGroupFrom(1, row, col)
			case col == 1 || col == 2 || monsterGroupCount == 2:
				placeFirstGroupFrom(2, row, col)
			default:
				placeFirstGroupFrom(3, row, col)
			}
		}
	}
}

// showEncounterView (show_encounter_view, WMELE 6e12) flashes the newly placed monsters
// in over four frames; wide (1) swings them in from further out.
func showEncounterView(wide int) {
	if dtLog != nil {
		return
	}
	if meleAnnounce == 1 {
		eraseWindow(mainWindow, true)
		printCentered(mainWindow, getMessage(2025), 0, 18)
		meleAnnounce = 0
	}
	loud := random(4)
	for flash := 0; flash < 4; flash++ {
		if (random(8) == 0 || flash == loud) && groupMonsterRec[1].Sound > 0 {
			playSound(14)
		}
		viewAnimState = -2
		animateViewStep()
		for row := range 2 {
			for col := range 5 {
				dx := int(flashShift[wide*4+flash])
				if col%2 == 1 || col == 0 {
					dx = -dx
				}
				if meleFlash[row][col] != 1 || row == 0 && col == 0 && meleNPCFirst == 1 {
					dx = 0
				}
				queueMonsterSprite(row, col, dx)
			}
		}
		viewAnimState = 0
		for {
			animateViewStep()
			if viewAnimState == -2 {
				break
			}
		}
	}
	meleFlash = [2][5]int16{}
	meleNPCFirst = 0
}
