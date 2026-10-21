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
	spriteList      [11]*Sprite      // 0x4390
	spriteCount     int16            // 0x438e

	treasureID     int16             // treasure_id
	combatGold     int32             // combat_gold
	rosterSlotUsed [16]uint8         // PCFILE header Used
	monsterPics    [10]MonsterPicRec // DS:0x36de, pic slots
)

// Root globals: the game state machine, the data files, the input and the windows.
var (
	gameState    int // which overlay main runs next, and what it does (game_state)
	gameStateArg int // the map for state 25 (game_state_arg)

	dataDir   string             // where the data files are (the "C:\BANE\" prefix)
	hdr       dbs.ScenarioHeader // SCENARIO.HDR (cs:33f8, scenario_hdr)
	scen      *dbs.Scenario      // SCENARIO.DBS and its headers (scenario_fd)
	msgs      *dbs.Messages      // MSG.DBS (msg_fd)
	misc      dbs.Huffman        // MISC.HDR, the text decoding tree
	pcfileHdr dbs.PCFileHeader   // the first bytes of PCFILE.DBS (cs:4fd0); Used is rosterSlotUsed

	videoFlags   int  // init_video's result: 1 EGA, 0x1000 a mouse
	mousePresent bool // the mouse is in use (mouse_present)
	clickFlag    bool // the last poll saw a click (click_flag)
	messageDelay int  // (message_delay)
	msgNotFound  bool // the last getMessage found nothing (msg_not_found)

	copyProtectionOK int16 // 0x4fc0 copy_protection_ok: the magic word was given
	g4f84            int16 // 0x4f84: set to 17210 once the magic word is typed
	g4fc8            int16 // 0x4fc8: the delay setting in use (cfg_delay)

	mainWindow      *Window // the one-line message window (main_window)
	fullscreenWin   *Window // a full-screen picture window (fullscreen_win)
	bottomLineWin   *Window // (bottom_line_win)
	partyPictureWin *Window // (party_picture_win)
	pictureWin      *Window // the maze view (picture_win)
	g4fac           *Window // the title strip above the view
	g4fb0           *Window
	g4fb8           *Window // the right-hand side panel
	g4fba           *Window // the left-hand side panel
)

// Party state (WINIT/WBASE).
var (
	partyCount        int    // party_count
	memberPcfileIndex [6]int // PCFILE slot of each member (member_pcfile_index)
	g4330             [3]int8
	g4326             [3]byte
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
