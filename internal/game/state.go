package game

// Typed game state: the record types (from dbs) and the runtime structures
// and root globals that the C keeps in the data segment. Offsets are the C
// layout, for translating; see LAYOUTS.md. Declared elsewhere already (not
// repeated here): Window/Cell (root_window.go), partyCount,
// memberPcfileIndex, picSlotMonster/picSlotLoaded (winit.go), gameState,
// gameStateArg, the *_win windows (root.go), sounds (root_sound.go).

import "github.com/retro-ware/wiz6-go/internal/dbs"

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
