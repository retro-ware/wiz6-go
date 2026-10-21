package game

import "github.com/retro-ware/wiz6-go/internal/drv"

// WMAZE: exploration, the maze view and the cell events (game states 5, 6, 23; main
// turns 25 into 5).

// ovmain (WMAZE 4572): flee (23), load the map game_state_arg (6), explore (5).
func wmazeOvmain() {
	if gameState == stateFlee {
		fleeParty(random(10) + 4)
	}
	if gameState == stateLoadMap {
		loadMazeLevel(gameStateArg)
	}
	if gameState == stateExplore {
		mazeMainLoop()
	}
}

// WMAZE data segment (bss 0x501e..0x5240). The view's own tables (0x5040..0x5228) live
// in wmaze_view.go.
var (
	explorePrompt, menuPrompt string // g501e, g5028: messages 2000 and 2001 (get_maze_input prompts)

	mzRedrawView     int16 // g5032: redraw the view at the top of the main loop
	mzEnteredCell    int16 // g5034: the party entered the cell (its events may run)
	mzAnimCounter    int16 // g5036: counts to 10 between animation frames
	mzWaitCut        int16 // g5038: a click or Enter cut wait_animating_m short
	mzBumpMessage    int16 // g503a: the bump message id (2020, 2053)
	mzRefreshPending int16 // g503c: refresh all windows at the next main loop turn
	mzClearMessage   int16 // g503e: erase the main window at the next main loop turn

	textWin          *Window  // g522a: the event text window (rows 19..23)
	textLine         int16    // g522c: the next line on the text page
	textLastColumn   int16    // g522e: the last line's column ('^')
	eventScreenDirty int16    // event_screen_dirty
	mzNextEvent      int16    // g5232: step to the next event record after the current one
	lockTumblers     [6]int16 // lock_tumblers (12 bytes)
)

// drv entry points as the root calls them.
func drvMazeTile(tile, flag, mirror int) { drv.MazeTile(tile, flag, mirror) }         // drv_op_c6
func drvShowWindow(w *Window)            { drv.ShowWindow(winRect(w)) }               // drv_op_c2
func drvFillBack(w *Window, v int)       { drv.FillWorkBuffer(winRect(w), uint8(v)) } // drv_op_be

// Directions as party_facing and the walls count them (the compass names take y as
// growing northwards).
const (
	dirNorth = iota // y+1
	dirEast         // x+1
	dirSouth        // y-1
	dirWest         // x-1
	dirCount
)

// The places of the keys in explorePrompt (message 2000), as get_maze_input returns them.
const (
	exploreKeyForward    = iota
	exploreKeyTurnRight  // or step right
	exploreKeyToggleTurn // the arrow pad turns or side-steps
	exploreKeyTurnLeft   // or step left
	exploreKeyMenu
)

// Cell event kinds (mazeB.Kind), as run_cell_events dispatches them.
const (
	eventTeleport      = 1
	eventSpinner       = 2
	eventMessage       = 3 // also a locked chest
	eventEncounter     = 4
	eventNPC           = 5
	eventCheckItem     = 6
	eventDirValues     = 7 // also hidden doors and locks
	eventSecret        = 8
	eventTrap          = 9
	eventChangeWalls   = 10
	eventSetFlag       = 11
	eventPuzzle        = 12
	eventRiddle        = 13
	eventText          = 16
	eventNoop17        = 17
	eventNoop18        = 18
	eventUseItem       = 19 // an item used on the cell (WDOPT check_item_location_event)
	eventWallChange    = 21
	eventClearFlag     = 22
	eventChangeFloor   = 23
	eventSwapFlags     = 24
	eventMemberEffects = 25
)

// Map layout: 12 blocks of 8x8 cells, 144 event records, 5 start points per block.
const (
	mapBlocks      = 12
	blockSize      = 8
	mapEvents      = 144
	blockStarts    = 5
	animFrameSteps = 10 // idle steps between animation frames (mzAnimCounter)
)

// Cell codes: mazeA.Floor's 4-bit floor features and the 2-bit wall codes.
const (
	floorNone = 0
	floorDoor = 7 // a closed door, facing its Feature direction
	wallOpen  = 0
	wallSolid = 2
	wallDoor  = 3 // opens with the door animation
)

// PC fields as WMAZE, WMELE and WMEXE read them.
const (
	conditionOK     = 0
	conditionDown   = 2 // and worse: cannot act
	conditionStoned = 2 // cured by spell 46 (CURE STONE)
	conditionDead   = 3 // and worse
	statusSilenced  = 0
	statusAfraid    = 1
	statusAsleep    = 2
	statusParalyzed = 3 // cured by spell 15 (CURE PARALYSIS)
	statusPoisoned  = 5 // with PoisonStr
	statusBlinded   = 4
	statusInsane    = 7 // cured by spell 57 (SANE MIND)
	statusNauseated = 8
	statusForever   = 255
	// Resist indexes (PC/Combatant.Resist, the r argument of hitDamage/hitEffect).
	resistFire        = 1
	resistCold        = 3
	resistPoison      = 6
	resistMental      = 10
	statStrength      = 0
	statIntelligence  = 1
	statPiety         = 2
	statVitality      = 3
	statDexterity     = 4
	statSpeed         = 5
	statPersonality   = 6
	statKarma         = 7
	statCount         = 8
	skillWandDagger   = 0
	skillSword        = 1
	skillAxe          = 2
	skillMaceFlail    = 3
	skillPoleStaff    = 4
	skillThrowing     = 5
	skillSling        = 6
	skillBows         = 7
	skillNoticeSecret = 11 // check_secret_detect's skill
	skillPickLock     = 15
	skillShield       = 8  // trained by blocking and shield bashes
	skillUnarmed      = 9  // the skill of an empty hand
	skillInstrument   = 12 // a bard's skill with an item's spell
	skillOratory      = 13 // sets the cast chance
	skillHide         = 16
	skillPilfer       = 14 // npcPilfer's skill
	skillArtifacts    = 22 // invoking an item's spell
	skillSpotMonsters = 23 // identifies monster groups
	skillScribe       = 24 // reading a scroll
	skillAlchemy      = 25
	skillTheology     = 26
	skillTheosophy    = 27
	skillThaumaturgy  = 28
	skillKirijutsu    = 29 // critical hits
	skillCount        = 30
	classFighter      = 0
	classMage         = 1
	classPriest       = 2
	classThief        = 3
	classRanger       = 4
	classAlchemist    = 5
	classBard         = 6
	classPsionic      = 7
	classBishop       = 9
	classLord         = 10
	classSamurai      = 11
	classUndead       = 20 // a monster class (MonsterRec.Class): DISPEL UNDEAD
	classCount        = 14
	soundBeep         = 0
	soundClick        = 1 // a point moved in a skill/ability adjuster
	soundHurt         = 2 // a member hurt but alive
	soundHit          = 7 // a melee hit
	soundDeathBase    = 8 // plus Sex
)

// MSG.DBS message ids.
const (
	msgDrinkResultBase = 2048 // plus the result: no effect, refreshing, poison, unusual
	msgForceResultBase = 2113 // plus the result: success, failure, jammed
	msgConfigItemBase  = 2300 // plus the item: SOUND, DELAY, LEVEL, ..
	msgSoundModeBase   = 2350 // plus SoundMode
	msgLevelBase       = 2357 // plus Level: NORMAL, EASY, EXPERT
	msgInputBase       = 2360 // plus Input: KEYBOARD, MOUSE, JOYSTICK
	msgGmodeBase       = 2363 // plus Gmode: VGA, ..

	msgExploreKeys             = 2000
	msgMenuKeys                = 2001
	msgPartyOptions            = 2002
	msgPartyMenu               = 2005 // SEARCH
	msgLocked                  = 2021
	msgYouAreInRock            = 2022
	msgDark                    = 2023
	msgEncounter               = 2025 // ^_an_encounter_^
	msgMazeReviewWho           = 2026 // review_who}
	msgExit                    = 2027
	msgExitHi                  = 2028 // EXIT
	msgWhoCasts                = 2029
	msgCastOnWho               = 2030
	msgWhoWillUse              = 2031
	msgUseOnWho                = 2032
	msgMemberOptions           = 2033 // $`s_options
	msgMazeUseWhichItem        = 2034 // USE WHICH ITEM?
	msgUseCancel               = 2035 // CANCEL
	msgResting                 = 2036
	msgMarchingOrder           = 2037
	msgYouMay                  = 2038
	msgAnswer                  = 2039
	msgDetectsSomethingUnusual = 2040
	msgRunning                 = 2041
	msgSearching               = 2043
	msgPressButton             = 2044
	msgSearchNothing           = 2045 // ^_nothing_^
	msgWhoDrinks               = 2047
	msgChute                   = 2052 // a chute
	msgFlying                  = 2054
	msgProtectedByLevitation   = 2056
	msgLockMenu                = 2100 // FORCE
	msgLockWhoWillTry          = 2110 // who_will_try}
	msgTumblingPress           = 2111
	msgStrainingPress          = 2112
	msgDiskOptions             = 2200
	msgDiskMenu                = 2201 // SAVE GAME & RESUME
	msgSavingGame              = 2257
	msgDiskError               = 2258
	msgLoadingGame             = 2259
	msgSavegameDrive           = 2305
	msgScenarioDrive           = 2306
	msgGameConfiguration       = 2307
	msgConfigExit              = 2308  // exit
	msgConfigExitHi            = 2309  // EXIT
	msgYouFellYourDeath        = 10000 // you_fell_to_your_death
)
