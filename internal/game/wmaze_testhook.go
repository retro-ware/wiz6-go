package game

import "github.com/retro-ware/wiz6-go/internal/dbs"

// The testing entries (the -maze, -fight, -pcview, -chest and -npc flags) boot like
// WINIT without the title, start NEWGAME.DBS's map and position with the first six
// characters of PCFILE.DBS, as WBASE's start_game would, and run main's game_state loop
// from the state they set.

// MainMaze starts in the maze (game state 6).
func MainMaze(dir string) int {
	return run(dir, startTestMaze)
}

// MainFight starts the map's encounter (monster mon instead when mon > 0) in the maze.
func MainFight(dir string, mon int) int {
	return run(dir, func() {
		startTestView()
		returnCmd = int16(stateExplore)
		startEncounter(0)
		if mon > 0 {
			encounterID = mon
		}
	})
}

// PCView opens the character view of member m (game state 17).
func PCView(dir string, m int) int {
	return run(dir, func() {
		startTestMaze()
		menuKeys = getMessage(msgSheetKeys)
		curMember = int16(min(m, max(partyCount-1, 0)))
		returnCmd, gameState = int16(stateExplore), stateViewCharacter
	})
}

// MainChest opens treasure record id as a found chest (game state 21).
func MainChest(dir string, id int) int {
	return run(dir, func() {
		startTestView()
		treasureID = int16(id)
		gameState = stateTreasureFound
	})
}

// MainNPC meets NPC record npc (game state 18).
func MainNPC(dir string, npc int) int {
	return run(dir, func() {
		startTestView()
		encounterID = npc
		gameState = stateNPC
	})
}

// startTestView is startTestMaze with the maze screen drawn, for the entries that start
// from inside the maze.
func startTestView() {
	startTestMaze()
	loadMazeLevel(gameStateArg)
	initMazeScreen()
	refreshAllWindows()
}

// startTestMaze is load_game(0), the roster and setup_game_windows (WBASE), cut down.
func startTestMaze() {
	initGameData()
	initMainScreen()
	g, err := dbs.DecodeSaveGame(openData("NEWGAME.DBS", 9))
	if err != nil {
		exitGame(9)
	}
	scenarioHdr.LevelFlags = g.Hdr.LevelFlags
	for i := range 14 {
		mapCacheA[i], mapCacheB[i] = g.MapA[i], g.MapB[i]
	}
	s := g.State
	partyX, partyY, cellX, cellY, partyBlock, partyFacing = s.PartyX, s.PartyY, s.CellX, s.CellY, s.Block, s.Facing
	gameStateArg = int(s.Map)
	saveState = s
	partyCounters = s.Counters
	roster := openData("PCFILE.DBS", 6)
	partyCount = 0
	for i := range 6 {
		pc, err := dbs.Decode[PC](roster[24+432*i : 24+432*(i+1)])
		if err != nil || pc.Name[0] == 0 {
			break
		}
		memberPortrait[partyCount] = uint8(findFreePortraitSlot(partyCount))
		party[partyCount] = pc
		memberPcfileIndex[partyCount] = i
		partyCount++
	}
	for i := 0; i < partyCount; i++ {
		loadPortrait(int(party[i].Portrait), int(memberPortrait[i]))
	}
	delaySetting = int16(scenarioHdr.Delay) // as load_game does
	if delaySetting == 0 {
		delaySetting = 5
	}
	padSidestep = 0
	eraseWindow(mainWindow, false)
	eraseCursorMark(bottomLineWin, false)
	movePadWin = createWindow(20, 19, 20, 5, 8, borderNone, 0, false)
	fillWindow(movePadWin, 32, 3)
	curEvent = [4]int16{}
	secretSpotted = 0
	mazeA, mazeB = nil, nil
	gameState = stateLoadMap
}
