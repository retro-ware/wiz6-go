package game

// Typed game state: the record types (from dbs) and the runtime structures
// and root globals that the C keeps in the data segment. Offsets are the C
// layout, for translating; see LAYOUTS.md. Declared elsewhere already (not
// repeated here): Window/Cell (root_window.go), sounds (root_sound.go).

import (
	"strings"

	"github.com/retro-ware/wiz6-go/internal/dbs"
)

type (
	Dice          = dbs.Dice
	PC            = dbs.PC
	Item          = dbs.Item
	Hand          = dbs.Hand
	SpellPoints   = dbs.SpellPoints
	ClassExp      = dbs.ClassExp
	ItemRec       = dbs.ItemRec
	MapA          = dbs.MapA
	MapB          = dbs.MapB
	MonsterRec    = dbs.MonsterRec
	MonsterAttack = dbs.MonsterAttack
	NPCRec        = dbs.NPCRec
	EventRec      = dbs.EventRec
	TreasureRec   = dbs.TreasureRec
	MonsterPicRec = dbs.MonsterPicRec
	PartyCounter  = dbs.PartyCounter
)

// Combatant is one fighter of a MonsterGroup (44 bytes; for group 0, the
// party member with the same index).
type Combatant struct {
	Level      int16     // +0
	HP         int16     // +2
	HPMax      int16     // +4
	Stamina    int16     // +6
	StaminaMax int16     // +8
	W10        int16     // +10
	W12        int16     // +12
	Status     [10]uint8 // +14 mirrors PC.Status
	B24        [7]int8   // +24 indexed
	B31        int8      // +31 target group
	B32        int8      // +32 target index
	B33        int8      // +33
	Condition  int8      // +34
	B35        [3]int8   // +35
	B38        int8      // +38 difficulty mod
	B39        int8      // +39 difficulty mod
	B40        int8      // +40
	B41        int8      // +41 difficulty mod
	Pos        int8      // +42 -1 = none
	Toggle     int8      // +43 toggled +-1
}

// MonsterGroup is one combat group (416 bytes, malloc'd; monster_groups[g]).
type MonsterGroup struct {
	Members   [9]Combatant // +0
	B396      [6]int8      // +396
	B402      [4]uint8     // +402
	B406      uint8        // +406
	B407      [3]uint8     // +407
	MonsterID uint8        // +410
	PicSlot   uint8        // +411
	B412      uint8        // +412 =1
	Count     uint8        // +413 1..9
	B414      uint8        // +414
	_         uint8        // +415
}

// Sprite is one maze-view sprite (24 bytes; sprite_list[i]).
type Sprite struct {
	X, Y, W, H int16    // +0 +2 +4 +6
	Layers     [12]int8 // +8
	Depth      uint8    // +20 255 = free
	Pic        uint8    // +21
	Mirror     uint8    // +22
	_          uint8    // +23
}

// ClickRegion is a 4-byte click_regions entry; X1 == 255 ends a table.
type ClickRegion struct{ X1, Y1, X2, Y2 uint8 }

var (
	party          [6]PC    // DS:0x43e8
	curMember      int16    // 0x43cc cur_member
	memberPortrait [6]uint8 // 0x43d0

	partyX, partyY int16 // 0x4fa4, 0x4fa2
	cellX, cellY   int16 // 0x4f9e.. party_cell_x/y (within the block)
	partyBlock     int16 // party_level: 8x8 block index 0..11 (NOT the dungeon level)
	partyFacing    int16 // 0x4f9a

	mazeA          *MapA           // maze_data: current map geometry
	mazeB          *MapB           // g4fa8: current map events
	cellEventIndex int16           // 0x4ee8
	levelFlags     [14][10]uint8   // 0x4eec, map*10
	curEvent       [4]int16        // g4ee0, per-direction values of the current event
	partyCounters  [6]PartyCounter // g4ec8 (scalars g4ecc..g4edc alias its elements)
	saveState      dbs.SaveState   // the rest of the save block's globals (g4f78..g4f98)

	monsterGroups   [7]*MonsterGroup // 0x43a8; [0] is the party side
	groupMonsterRec [7]*MonsterRec   // 0x43b6
	spriteList      [12]Sprite       // 0x4390 sprite_list
	spriteCount     int16            // 0x438e

	treasureID     int16             // treasure_id
	combatGold     int32             // combat_gold
	rosterSlotUsed [16]uint8         // PCFILE header Used
	monsterPics    [10]MonsterPicRec // DS:0x36de, pic slots
)

// gameStateID is a game_state value: which overlay main runs next, and what it does.
type gameStateID int

// The game states, by the overlay that runs them (9 is not used).
const (
	stateLoadData           gameStateID = 0  // WINIT: read the configuration and open the data files
	stateTitle              gameStateID = 1  // WINIT: the title sequence
	stateMainScreen         gameStateID = 2  // WINIT: load the pictures and make the game's windows
	stateQuit               gameStateID = 3  // main: shut down and leave
	stateMasterOptions      gameStateID = 4  // WBASE: the master options menu
	stateExplore            gameStateID = 5  // WMAZE: the maze main loop
	stateLoadMap            gameStateID = 6  // WMAZE: load map gameStateArg
	stateResetPartyScreen   gameStateID = 7  // WBASE: reset the party screen
	stateGraveyard          gameStateID = 8  // WINIT: the graveyard
	stateBeginCombat        gameStateID = 10 // WMELE: start an encounter
	stateRedrawCombat       gameStateID = 11 // WMELE: redraw the combat screen
	stateStartCombatRound   gameStateID = 12 // WPOPS: choose the round's actions
	stateRunCombatRound     gameStateID = 13 // WMEXE: carry out the round
	stateEndCombatRound     gameStateID = 14 // WMELE: end a combat round
	stateTreasureAfterFight gameStateID = 15 // WTREA: the treasure after combat
	stateCreateCharacter    gameStateID = 16 // WPCMK: make a character
	stateViewCharacter      gameStateID = 17 // WPCVW: view the current member
	stateNPC                gameStateID = 18 // WMNPC: an NPC encounter
	stateCastSpellInMaze    gameStateID = 19 // WDOPT: cast a spell outside combat
	stateUseItemInMaze      gameStateID = 20 // WDOPT: use an item outside combat
	stateTreasureFound      gameStateID = 21 // WTREA: a treasure found in the maze
	stateLevelUp            gameStateID = 22 // WPCVW: the level up after combat
	stateFlee               gameStateID = 23 // WMAZE: the party flees
	stateGameConfiguration  gameStateID = 24 // WBASE: the game configuration
	stateEnterMap           gameStateID = 25 // main: change to map gameStateArg, then stateExplore
)

// init_video's result bits (driver values).
const (
	videoEGA   = 1
	videoMouse = 1 << 12
)

// Root globals: the game state machine, the data files, the input and the windows.
var (
	gameState    gameStateID // which overlay main runs next, and what it does (game_state)
	gameStateArg int         // the map for stateEnterMap (game_state_arg)

	dataDir     string             // where the data files are (the "C:\BANE\" prefix)
	scenarioHdr dbs.ScenarioHeader // SCENARIO.HDR (cs:33f8, scenario_hdr)
	scenarioDBS *dbs.Scenario      // SCENARIO.DBS and its headers (scenario_fd)
	messageDBS  *dbs.Messages      // MSG.DBS (msg_fd)
	textTree    dbs.Huffman        // MISC.HDR, the text decoding tree
	pcfileHdr   dbs.PCFileHeader   // the first bytes of PCFILE.DBS (cs:4fd0); Used is rosterSlotUsed

	videoFlags   int  // init_video's result: videoEGA, videoMouse
	mousePresent bool // the mouse is in use (mouse_present)
	clickFlag    bool // the last poll saw a click (click_flag)
	messageDelay int  // (message_delay)
	msgNotFound  bool // the last getMessage found nothing (msg_not_found)

	magicWordTyped int16 // 0x4f84: set to 17210 once the magic word is typed
	delaySetting   int16 // 0x4fc8: the delay setting in use (cfg_delay)

	mainWindow      *Window // the one-line message window (main_window)
	fullscreenWin   *Window // a full-screen picture window (fullscreen_win)
	bottomLineWin   *Window // (bottom_line_win)
	partyPictureWin *Window // (party_picture_win)
	pictureWin      *Window // the maze view (picture_win)
	titleStripWin   *Window // g4fac: the title strip above the view
	g4fb0           *Window
	rightPanelWin   *Window // g4fb8: the right-hand side panel
	leftPanelWin    *Window // g4fba: the left-hand side panel
)

// Party state (WINIT/WBASE).
var (
	partyCount        int    // party_count
	memberPcfileIndex [6]int // PCFILE slot of each member (member_pcfile_index)
	g433a             int
	msgBlockNum       int

	// Monster pictures in the picture slots (pic_slot_monster, pic_slot_loaded).
	picSlotMonster [10]int
	picSlotLoaded  [10]bool
)

// Root data g0526 and g0532: the condition glyph by condition, and the status glyph by
// status (-1 none); 37 is added to get the glyph.
var (
	conditionGlyph = [6]int{0, 1, 3, 4, 4, 0}
	statusGlyph    = [10]int{-1, -1, -1, -1, 6, 7, 8, 8, 7, 5}
)

// cstr is the NUL-terminated string at the start of b.
func cstr(b []byte) string {
	if i := strings.IndexByte(string(b), 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// Root data (ROOT data segment) shared by the overlays.
var (
	textColor   = 5                                          // g0002
	realmColors = [6]int{4, 2, 3, 6, 7, 5}                   // realm_colors: fire water air earth mental magic
	stackable   = [18]uint8{2: 1, 4: 1, 12: 1, 15: 1, 16: 1} // g00cc: by item type, the item has a quantity
	recipes     = [9][3]int16{                               // g0546: item a + item b make item c
		{390, 396, 397}, {407, 407, 408}, {408, 408, 409}, {393, 440, 442}, {393, 441, 443},
		{442, 441, 444}, {443, 440, 444}, {446, 447, 448}, {405, 480, 406},
	}
)

// Spell is one spell_table entry (ROOT 0x00de, 6 bytes).
type Spell struct {
	Realm  uint8 // +0
	Level  uint8 // +1 g00df
	Cost   uint8 // +2 g00e0: spell points per power level
	B3     uint8 // +3
	Target uint8 // +4 g00e2: 1 cast on a chosen member
	Books  uint8 // +5 g00e3: bit per spell book (8 mage 4 priest 2 alchemist 1 psionic)
}

// spellTable (ROOT spell_table): the 82 spells.
var spellTable = [82]Spell{
	{0, 1, 2, 3, 2, 8}, {0, 2, 3, 2, 2, 1}, {0, 3, 6, 3, 2, 8}, {0, 3, 8, 1, 0, 8}, {0, 4, 8, 3, 2, 1},
	{0, 5, 8, 3, 2, 4}, {0, 5, 9, 3, 2, 8}, {0, 6, 12, 3, 2, 8}, {0, 7, 16, 3, 0, 8}, {1, 1, 2, 3, 2, 8},
	{1, 1, 2, 1, 1, 7}, {1, 1, 3, 3, 2, 10}, {1, 2, 4, 3, 2, 10}, {1, 2, 4, 3, 2, 6}, {1, 3, 5, 1, 0, 6},
	{1, 3, 6, 1, 1, 7}, {1, 3, 8, 1, 0, 8}, {1, 4, 8, 3, 2, 8}, {1, 4, 5, 3, 2, 6}, {1, 5, 6, 3, 2, 8},
	{2, 1, 2, 3, 2, 1}, {2, 2, 5, 1, 0, 8}, {2, 3, 4, 3, 2, 9}, {2, 3, 8, 1, 3, 9}, {2, 3, 4, 3, 2, 6},
	{2, 4, 7, 3, 2, 1}, {2, 4, 8, 1, 1, 5}, {2, 4, 8, 3, 2, 4}, {2, 5, 10, 1, 0, 5}, {2, 5, 8, 3, 2, 1},
	{2, 5, 12, 1, 0, 8}, {2, 6, 8, 3, 2, 1}, {2, 6, 10, 3, 2, 9}, {2, 6, 12, 3, 0, 9}, {2, 7, 16, 3, 0, 1},
	{3, 1, 2, 2, 2, 1}, {3, 1, 2, 2, 2, 1}, {3, 1, 2, 1, 1, 8}, {3, 1, 3, 1, 0, 8}, {3, 2, 6, 1, 0, 10},
	{3, 3, 6, 3, 2, 6}, {3, 3, 6, 1, 0, 4}, {3, 3, 7, 3, 2, 9}, {3, 4, 8, 3, 2, 1}, {3, 4, 8, 3, 2, 10},
	{3, 5, 10, 1, 0, 1}, {3, 6, 18, 1, 1, 5}, {4, 1, 3, 1, 2, 2}, {4, 1, 3, 3, 2, 11}, {4, 1, 4, 1, 0, 6},
	{4, 1, 5, 3, 2, 7}, {4, 2, 4, 1, 1, 7}, {4, 2, 4, 1, 0, 6}, {4, 2, 5, 1, 0, 10}, {4, 2, 8, 1, 0, 6},
	{4, 3, 6, 3, 2, 6}, {4, 3, 8, 1, 2, 2}, {4, 3, 10, 1, 1, 6}, {4, 4, 8, 3, 2, 2}, {4, 4, 10, 1, 0, 2},
	{4, 4, 10, 1, 0, 10}, {4, 5, 10, 3, 2, 6}, {4, 6, 8, 1, 0, 6}, {4, 7, 18, 3, 0, 2}, {5, 1, 4, 1, 1, 7},
	{5, 1, 3, 3, 2, 4}, {5, 2, 4, 3, 2, 8}, {5, 2, 7, 1, 2, 4}, {5, 2, 4, 1, 0, 4}, {5, 3, 7, 1, 0, 10},
	{5, 4, 8, 1, 0, 8}, {5, 4, 10, 1, 0, 12}, {5, 5, 7, 1, 2, 8}, {5, 5, 10, 1, 1, 4}, {5, 6, 12, 1, 2, 6},
	{5, 6, 8, 1, 2, 12}, {5, 7, 18, 3, 0, 4}, {5, 7, 20, 1, 1, 14}, {5, 7, 20, 3, 0, 4}, {5, 0, 0, 0, 2, 0},
	{5, 0, 0, 0, 0, 0}, {5, 0, 0, 0, 0, 0},
}

// view_anim_state values below the sprite indexes.
const (
	viewAnimShow  = -3
	viewAnimMaze  = -2
	viewAnimQueue = -1
)

// Root combat state the combat overlays share.
var (
	monsterGroupCount int         // 0x4360 monster_group_count
	viewAnimState     int         // 0x4358 view_anim_state: viewAnimShow, viewAnimMaze, viewAnimQueue or 0..11 a sprite
	viewPosGroup      [2][5]int16 // 0x4362 view_pos_group: the group shown at view row, col (0 none)
	combatKeys        string      // 0x4376: message 3900, the combat menu keys
	allyStripWin      *Window     // g4fae: the ally strip under the view
)

// Root state WMAZE uses.
var (
	returnCmd    int16 // 0x4fce return_cmd
	menuClicked  int16 // 0x4fc4 menu_clicked
	animPhase    int16 // 0x3646 anim_phase
	g363e        int16 // 0x363e
	g4338, g4354 int16 // combat entry parameters
	viewRowsOpen int16 // 0x433c view_rows_open

	reachableCells [96]byte // 0x4e08: 768-bit cell set (flood_mark_cells)
	eventCells     [96]byte // 0x4e68: cells with an event record

	// The maps of the current game, 14 used (far_buffers_a/b, g07d4/g080c); WBASE fills
	// them from NEWGAME/SAVEGAME. mazeA/mazeB are working copies of map gameStateArg.
	mapCacheA              [16]MapA
	mapCacheB              [16]MapB
	reloadMapA, reloadMapB bool // g4eeb, g4eea

	g4f86, g4f88, g4f8a, g4fa6 int16
	curSoundSize               int32 // 0x33f0 cur_sound_size
)

// Combat state in the root data segment that the combat overlays share.
var (
	combatOver      int16           // 0x4356 combat_over
	monsterJitterX  [2][5]uint8     // g4348: random x offset of the monster at row, col
	monsterJitterY  [2][5]uint8     // g433e: random y offset
	combatActionWin *Window         // g438a: the action window (left)
	monsterListWin  *Window         // g438c: the monster list window (right)
	g4352           int16           // set by an item of kind 7 (do_use_item_action)
	initiative      int             // g435a: the initiative count of the round, 100 down to 1
	actMember       int             // g435c: the acting combatant's index
	actGroup        int             // g435e: the acting combatant's group
	animCycle       int16           // 0x3644 anim_cycle: 0..2, effect sprites step at 0
	monsterFrames   [5][10][3]uint8 // 0x3648: per view column and frame: frame, step, ticks left
	alwaysHit       int16           // 0x4fcc: nonzero, every swing hits (a debug switch)
)
