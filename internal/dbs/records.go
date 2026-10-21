package dbs

// Typed file records. Every type here is a packed little-endian image of its
// on-disk (and in-memory, DS-resident) C layout: Decode/Encode go through
// encoding/binary, so field order and widths ARE the layout. Offsets in the
// comments are from the start of the record. UnkXX fields are not understood
// yet; they exist so records round-trip byte for byte. See LAYOUTS.md.

import (
	"encoding/binary"
	"fmt"
)

// Decode decodes a packed record from b (which must be exactly its size).
func Decode[T any](b []byte) (T, error) {
	var v T
	if n := binary.Size(v); n != len(b) {
		return v, fmt.Errorf("dbs: %T is %d bytes, got %d", v, n, len(b))
	}
	_, err := binary.Decode(b, binary.LittleEndian, &v)
	return v, err
}

// Encode encodes a packed record.
func Encode[T any](v *T) []byte {
	b, err := binary.Append(nil, binary.LittleEndian, v)
	if err != nil {
		panic(err) // only fixed-size types live here
	}
	return b
}

// Size is the encoded size of a record type.
func Size[T any]() int { var v T; return binary.Size(v) }

// Dice is roll_dice's argument: Base + Count rolls of random(Sides)+1.
type Dice struct {
	Base  int16 // +0
	Count uint8 // +2
	Sides uint8 // +3
}

// ---------------------------------------------------------------- PCFILE.DBS

// PCFileHeader is the first 24 bytes of PCFILE.DBS (cs:0x4fd0).
type PCFileHeader struct {
	RecSize int16     // +0  432
	Count   int16     // +2  16
	Offset  int32     // +4  24, file offset of record 0
	Used    [16]uint8 // +8  roster slot in use
}

// Item is one inventory slot of a PC (8 bytes, party+0x40+8*i).
type Item struct {
	ID     int16 // +0 scenario item record (type 1)
	Weight int16 // +2 tenths
	Type   uint8 // +4 item record +60
	Icon   uint8 // +5 item record +61
	Qty    uint8 // +6 charges / count
	Flags  uint8 // +7 bit0 equipped, bit3 two-handed (wpcvw itemTwoHanded)
}

// Hand is one weapon hand (16 bytes, party+0xf0+16*h), rebuilt from the
// equipped weapon's item record on equip.
type Hand struct {
	Damage Dice     // +0x0 item +24; bare hand {0,2,2}
	Data   [8]uint8 // +0x4 item +46..53; Data[6] (+0xa) is crit chance
	B0C    uint8    // +0xc item +65
	B0D    uint8    // +0xd item +64
	ToHit  int8     // +0xe -5*item +71
	Flags  uint8    // +0xf item +72; 32 bare, |64 martial arts
}

// SpellPoints is one realm's pool (party+0x28+4*realm).
type SpellPoints struct {
	Cur int16 // +0
	Max int16 // +2
}

// PC is a character record: PCFILE.DBS / SAVEGAME.DBS / party[6] at DS:0x43e8.
type PC struct {
	Name                   [8]uint8       // +0x000 NUL-padded
	AgeDays                int32          // +0x008
	Exp                    int32          // +0x00c
	Kills                  int32          // +0x010 "MKS"
	Gold                   int32          // +0x014
	HP                     int16          // +0x018
	HPMax                  int16          // +0x01a
	Stamina                int16          // +0x01c
	StaminaMax             int16          // +0x01e
	Carried                int16          // +0x020 tenths of a pound
	MaxLoad                int16          // +0x022
	Level                  int16          // +0x024 (pc_level)
	Unk026                 int16          // +0x026 ++ on resurrection; shown with glyph 31 (age-like)
	SP                     [6]SpellPoints // +0x028 fire water air earth mental magic
	Items                  [22]Item       // +0x040: 10 inventory, then up to 12 swag from 10 on
	Hands                  [2]Hand        // +0x0f0 primary, secondary
	Equipped               [8]uint8       // +0x110 inventory index per equip position, 255 none
	SpellsLearn            [4]uint8       // +0x118 spells still to learn per book
	SpellRegen             [6]uint8       // +0x11c per realm
	Status                 [10]uint8      // +0x122 status durations; [2] sleep, [5] poison
	Stats                  [8]uint8       // +0x12c STR INT PIE VIT DEX SPD PER KAR
	Skills                 [30]uint8      // +0x134 skill levels (10,17..21 not shown as skills)
	Resist                 [14]uint8      // +0x152
	AC                     [8]uint8       // +0x160 per body area
	NPCAttitude            [32]int8       // +0x168 indexed by NPC id
	KnownSpells            [12]uint8      // +0x188 bitset (test_bit)
	Unk194                 [4]uint8       // +0x194
	Flags                  [4]uint8       // +0x198 bitset
	Portrait               uint8          // +0x19c
	Race                   uint8          // +0x19d
	Sex                    uint8          // +0x19e
	Class                  uint8          // +0x19f
	Rank                   uint8          // +0x1a0
	Condition              uint8          // +0x1a1
	Regen                  uint8          // +0x1a2
	Drain                  uint8          // +0x1a3
	PoisonStr              uint8          // +0x1a4
	LoadCat                uint8          // +0x1a5 cached load category
	RestHPRoll             uint8          // +0x1a6 rest_party: hit point roll - 1 per hour
	RestStaminaPct         uint8          // +0x1a7 rest_party: stamina percent per hour
	SkillPoints            uint8          // +0x1a8
	PrimaryAttacks         uint8          // +0x1a9 attacks per round, primary hand; load cap
	SecondaryAttacks       uint8          // +0x1aa attacks per round, secondary hand; load cap
	AttackPower            uint8          // +0x1ab SPD + level
	ItemCount              uint8          // +0x1ac
	SwagCount              uint8          // +0x1ad
	ToHit                  int8           // +0x1ae
	LevelBeforeClassChange uint8          // +0x1af level before a class change, 0 once reached again
}

// ------------------------------------------------------------------- headers

// MasterHeader is MASTER.HDR (66 bytes, DS:0x3300).
type MasterHeader struct {
	RecordSizes  [10]int16 // +0  record_sizes
	RecordCounts [10]int16 // +20
	Unk40        [16]uint8 // +40
	RecordPacked [10]uint8 // +56 record_packed (all 0 in the shipped data)
}

// DiskHeader is DISK.HDR (700 bytes, DS:0x3044).
type DiskHeader struct {
	Unk0            [4]uint8   // +0
	Offsets         [10]int32  // +4   SCENARIO.DBS offset of each record type
	MapIndex        [16]uint8  // +44  map -> record for types 2/3 (g3070)
	NPCIndex        [64]uint8  // +60  NPC (g3080)
	TreasureIndex   [256]uint8 // +124 treasure (g30c0)
	MonsterIndex    [256]uint8 // +380 monster (g31c0)
	MonsterPicIndex [64]uint8  // +636 monster pic (g32c0)
}

// ScenarioHeader is SCENARIO.HDR and the first 414 bytes of NEWGAME/SAVEGAME
// (DS:0x33f8, scenario_hdr).
type ScenarioHeader struct {
	LevelFlags [14][10]uint8 // +0   per-map event bitsets (runtime copy at 0x4eec)
	Unk140     [188]uint8    // +140
	Paths      [4][20]uint8  // +328 (0x3540); Paths[2][0..2] = sound port cfg
	SoundMode  uint8         // +408
	Delay      uint8         // +409 cfg_delay
	Disks      uint8         // +410 g3592
	Level      uint8         // +411 cfg level
	Input      uint8         // +412 cfg_input
	Gmode      uint8         // +413 cfg_gmode
}

// SaveState is the 66-byte position block of SAVEGAME/NEWGAME, after the maps.
type SaveState struct {
	Map             int16           // +0  game_state_arg
	PartyX          int16           // +2
	PartyY          int16           // +4
	CellX           int16           // +6
	CellY           int16           // +8
	Block           int16           // +10 party_level (8x8 block index)
	Facing          int16           // +12
	SavedMap        int16           // +14 g4f98, save_position
	SavedPartyX     int16           // +16 g4f96
	SavedPartyY     int16           // +18 g4f94
	SavedCellX      int16           // +20 g4f92
	SavedCellY      int16           // +22 g4f90
	SavedBlock      int16           // +24 g4f8e
	SavedFacing     int16           // +26 g4f8c
	Minutes         int32           // +28 g4f80, game clock: ++ per step, +60 per hour rested
	LastRestMinutes int32           // +32 g4f7c, Minutes at the last rest
	RestBankMinutes int32           // +36 g4f78, minutes banked towards full-rest hours
	Counters        [6]PartyCounter // +40 g4ec8
	PartyCount      int16           // +64
}

// PartyCounter is one g4ec8 entry (party-wide timed effect: value, duration).
type PartyCounter struct {
	A int16 // +0 g4ec8[i]
	B int16 // +2 g4eca[i]
}

// SaveGame is NEWGAME.DBS / SAVEGAME.DBS (49856 + 432*PartyCount bytes).
type SaveGame struct {
	Hdr   ScenarioHeader
	MapA  [16]MapA // 14,15 zero
	MapB  [16]MapB
	State SaveState
	Party []PC
}

// saveFixedSize is the size of a save file before its PC records.
var saveFixedSize = Size[ScenarioHeader]() + len(SaveGame{}.MapA)*(Size[MapA]()+Size[MapB]()) + Size[SaveState]()

func DecodeSaveGame(b []byte) (*SaveGame, error) {
	pcSize := Size[PC]()
	if len(b) < saveFixedSize || (len(b)-saveFixedSize)%pcSize != 0 {
		return nil, fmt.Errorf("dbs: save file is %d bytes", len(b))
	}
	g := &SaveGame{}
	o := 0
	next := func(n int) []byte { o += n; return b[o-n : o] }
	var err error
	if g.Hdr, err = Decode[ScenarioHeader](next(Size[ScenarioHeader]())); err != nil {
		return nil, err
	}
	for i := range g.MapA {
		g.MapA[i], _ = Decode[MapA](next(Size[MapA]()))
		g.MapB[i], _ = Decode[MapB](next(Size[MapB]()))
	}
	g.State, _ = Decode[SaveState](next(Size[SaveState]()))
	for o < len(b) {
		pc, _ := Decode[PC](next(pcSize))
		g.Party = append(g.Party, pc)
	}
	return g, nil
}

func (g *SaveGame) Encode() []byte {
	b := Encode(&g.Hdr)
	for i := range g.MapA {
		b = append(b, Encode(&g.MapA[i])...)
		b = append(b, Encode(&g.MapB[i])...)
	}
	b = append(b, Encode(&g.State)...)
	for i := range g.Party {
		b = append(b, Encode(&g.Party[i])...)
	}
	return b
}

// ------------------------------------------------- SCENARIO.DBS record types

// Record type numbers (access_scenario_record's first argument).
const (
	RecClassExp   = 0
	RecItem       = 1
	RecMapA       = 2
	RecMapB       = 3
	RecMonster    = 4
	RecNPC        = 5
	RecEvent      = 6
	RecTreasure   = 7
	RecMonsterPic = 8
	RecSound      = 9
)

// ClassExp (type 0, 64 bytes): experience needed per level, one per class.
// T[L-1] is the exp for level L; past 16: (T[15]-T[14])*(L-16)+T[15].
type ClassExp struct {
	Exp [16]int32 // +0
}

// ItemRec (type 1, 74 bytes).
type ItemRec struct {
	Name       [16]uint8 // +0  "unidentified\0identified"
	Price      int32     // +16
	Charges    Dice      // +20 quantity roll
	Damage     Dice      // +24 -> Hand.Damage
	Spell      int16     // +28 power<<8 | spell
	Weight     int16     // +30 tenths
	SkillBonus [14]uint8 // +32 add_skill_bonus, [1..13]
	HandData   [8]uint8  // +46 -> Hand.Data
	ClassMask  uint16    // +54
	RaceMask   uint16    // +56
	SexMask    uint8     // +58
	Flags      uint8     // +59
	Type       uint8     // +60
	Icon       uint8     // +61
	Unk62      uint8     // +62
	Unk63      uint8     // +63
	HandD      uint8     // +64 -> Hand.B0D
	HandC      uint8     // +65 -> Hand.B0C
	Regen      uint8     // +66 -> PC.Regen (max)
	Drain      uint8     // +67 -> PC.Drain
	Unk68      uint8     // +68
	Unk69      uint8     // +69
	AC         uint8     // +70
	ToHitPen   uint8     // +71 Hand.ToHit -= 5*x
	HandFlags  uint8     // +72 -> Hand.Flags (sometimes read as int16 with +73)
	Unk73      uint8     // +73
}

// MapA (type 2, 1346 bytes, maze_data): geometry and encounters of one map.
// Packed sub-arrays are read with get_bitfield/test_bit; index = cell index
// (block<<6 | y<<3 | x) unless noted.
type MapA struct {
	Encounters           [8][4][3]uint8 // +0    [p4][l2][l6]
	Walls0               [192]uint8     // +96   2 bits x 768
	Walls1               [192]uint8     // +288  2 bits x 768
	BlockX               [12]uint8      // +480  map position of each 8x8 block
	BlockY               [12]uint8      // +492
	Floor                [384]uint8     // +504  4 bits x 768 (1,2,7,13,14 seen)
	Feature              [192]uint8     // +888  2 bits x 768
	Unk1080              [2]uint8       // +1080
	SpecialCells         [96]uint8      // +1082 1 bit x 768, pit / special cells
	SolidCells           [96]uint8      // +1178 1 bit x 768, solid on the automap
	StartX               [24]uint8      // +1274 3 bits x 64, index p4*5+i, block start points
	StartY               [24]uint8      // +1298 3 bits x 64, index p4*5+i, block start points
	EncounterTables      [12]uint8      // +1322 4 bits x 24, index p4<<1|l2, two per block
	EncounterTableChance [12]uint8      // +1334 percent chance of a block's table 0
}

// MapB (type 3, 1740 bytes, g4fa8): the 144 cell events of one map.
type MapB struct {
	Data       [144]int16 // +0
	Unk288     [144]int16 // +288
	DirBits    [144]int16 // +576 2 bits per facing: extract_bits(v, facing<<1, 2)
	Kind       [144]uint8 // +864 7 = lock
	X          [144]uint8 // +1008
	Y          [144]uint8 // +1152
	Block      [144]uint8 // +1296 8x8 block of the event
	FlagBit    [144]uint8 // +1440 bit in LevelFlags
	Param      [144]uint8 // +1584 99 special
	FirstEvent [12]uint8  // +1728 per block
}

// MonsterAttack is one of a monster's three attacks (16 bytes, mon+68+16*i).
type MonsterAttack struct {
	Damage Dice     // +0
	Unk4   [9]uint8 // +4
	Range  uint8    // +13 reach; <=1 melee only
	ToHit  int8     // +14
	Effect uint8    // +15 bitset
}

// MonsterRec (type 4, 222 bytes; group_monster_rec[g]).
type MonsterRec struct {
	Names      [4][16]uint8     // +0   unid/id singular/plural
	GoldLow    int16            // +64 gold carried, low word
	GoldHigh   int16            // +66 gold carried, high word
	Attacks    [3]MonsterAttack // +68
	GroupSize  Dice             // +116 hard: Base+1
	HP         Dice             // +120 HP.Count is the level (hard: +1)
	Stamina    Dice             // +124
	AttackKind [3]uint8         // +128 msg 3371+
	AttackVerb [3]uint8         // +131 msg 3799+
	Unk134     uint8            // +134
	Unk135     [3]uint8         // +135 summed in attack
	Unk138     [10]uint8        // +138
	Resist     [14]uint8        // +148
	Treasure   [2]uint8         // +162 treasure ids
	Companion  [2]uint8         // +164 monster id
	CompChance [2]uint8         // +166
	Unk168     [4]uint8         // +168
	Unk172     [18]uint8        // +172
	AC         [8]uint8         // +190
	Unk198     [10]uint8        // +198
	Unk208     uint8            // +208
	Pic        uint8            // +209 monster pic (via f4858)
	Sound      uint8            // +210
	Unk211     [9]uint8         // +211 [2] (+213) is a divisor
	Flags      uint8            // +220
	Unk221     uint8            // +221
}

// NPCEntry is one of an NPC's eight 6-byte entries (+0x10).
type NPCEntry struct {
	A int16    // +0
	B [4]uint8 // +2
}

// NPCRec (type 5, 142 bytes; WMNPC DS:0x5146).
type NPCRec struct {
	Name     [16]uint8   // +0x00
	Entries  [8]NPCEntry // +0x10
	Topics   [20]int16   // +0x40 message ids
	Unk68    [9]int16    // +0x68
	Gold     int32       // +0x7a g51c0
	Scripts  [8]uint8    // +0x7e event script ids (npcRunScript)
	Unk86    [5]uint8    // +0x86
	Treasure uint8       // +0x8b g51d1
	Disarm   int8        // +0x8c spell_disarm_result
	Unk8D    uint8       // +0x8d
}

// EventOp is one step of an event script.
type EventOp struct {
	Arg   int16 // +0
	Param uint8 // +2
	Op    uint8 // +3
}

// EventRec (type 6, 32 bytes): script run by WMNPC f7038.
type EventRec struct {
	Ops [8]EventOp
}

// TreasureEntry is one roll of a treasure table.
type TreasureEntry struct {
	A      int16 // +0
	B      int16 // +2
	Kind   uint8 // +4
	Chance uint8 // +5
}

// TreasureRec (type 7, 40 bytes).
type TreasureRec struct {
	Entries   [6]TreasureEntry // +0
	Traps     [3]uint8         // +36 bitset
	ChestType uint8            // +39
}

// MonsterPicRec (type 8, 314 bytes; 10 slots at DS:0x36de).
type MonsterPicRec struct {
	Unk0          [4]uint8      // +0
	W4            int16         // +4   g36e2
	FrameX        [25][2]uint8  // +6   g36e4 per-frame x: normal, mirrored
	Unk56         uint8         // +56
	FrameY        [27]uint8     // +57  g3717 per-frame y
	SeqFirst      [10]uint8     // +84 per layer: first sequence to pick
	SeqCount      [10]uint8     // +94 per layer: sequences to pick from
	SeqStartFrame [10]uint8     // +104 per sequence: first frame
	Unk114        [9]uint8      // +114
	Quads         [40][4]uint8  // +123
	Unk283        uint8         // +283
	Offsets       [2][5][2]int8 // +284 g37fa
	Unk304        [2]uint8      // +304
	Unk306        [2]uint8      // +306 g3810
	FrameCount    uint8         // +308 g3812
	Unk309        [4]uint8      // +309 g3813..g3816
	Unk313        uint8         // +313
}

// SoundRec (type 9, 12 bytes; table [14] at DS:0x3344).
type SoundRec struct {
	Fallback int32 // +0
	Size     int32 // +4
	Rate     int16 // +8
	Level    uint8 // +10
	Flags    uint8 // +11
}
