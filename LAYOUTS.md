# Record layouts: the translator's table

The authoritative field list, with offset, width and meaning, is the Go source:

- `internal/dbs/records.go` holds the file records.
- `internal/game/state.go` holds the runtime structures and root globals.

Every field has a `// +0xNN` comment. This file covers what you need to turn C offset arithmetic into a typed field: indexes by offset, the C idioms and the traps.

Widths follow C: `char` is int8, `unsigned char` is uint8, `int` is int16, `unsigned` is uint16 and `long` is int32. When the C casts a field, for example `(unsigned char)x[1584]`, keep the cast in Go (`uint8(...)`) if the field type differs.

## Records and where they live

| Record | Size | Go type | C home |
|---|---|---|---|
| PC | 432 | `PC` | `party` at DS:0x43e8 (6 records); PCFILE.DBS at 24+432*i; save after 49856 |
| PCFILE header | 24 | `PCFileHeader` | cs:0x4fd0 |
| MASTER.HDR | 66 | `MasterHeader` | 0x3300 (`record_sizes`) |
| DISK.HDR | 700 | `DiskHeader` | 0x3044 (`disk_header`) |
| SCENARIO.HDR | 414 | `ScenarioHeader` | 0x33f8 (`scenario_hdr`) |
| save position | 66 | `SaveState` | globals 0x4f78..0x4fa4, `g4ec8`, `party_count` |
| NEWGAME/SAVEGAME | 49856+432n | `SaveGame` | hdr, 16×(MapA, MapB), SaveState, PCs |
| type 0 | 64 | `ClassExp` | local buffer in check_level_up |
| type 1 | 74 | `ItemRec` | locals (buy, equip, identify) |
| type 2 | 1346 | `MapA` | `maze_data` (far) |
| type 3 | 1740 | `MapB` | `g4fa8` |
| type 4 | 222 | `MonsterRec` | `group_monster_rec[g]` (malloc) |
| type 5 | 142 | `NPCRec` | WMNPC DS:0x5146 |
| type 6 | 32 | `EventRec` | WMNPC locals |
| type 7 | 40 | `TreasureRec` | locals |
| type 8 | 314 | `MonsterPicRec` | 10 slots at 0x36de (`monsterPics`) |
| type 9 | 12 | `dbs.SoundRec` | [14] at 0x3344; the game copy is `soundRec` in root_sound.go |
| combatant | 44 | `Combatant` | `monster_groups[g]+44*i` |
| group | 416 | `MonsterGroup` | `monster_groups[7]` at 0x43a8 |
| sprite | 24 | `Sprite` | `sprite_list[11]` at 0x4390 |
| click region | 4 | `ClickRegion` | `click_regions[40]` at 0x3598; X1==255 ends it |
| window | 20+2wh | `Window` (root_window.go) | header bytes: w, h, x, y, color, flags, curx, cury, border[8], unk[4]; then cells |

Scenario record *r* of type *T* is at `Disk.Offsets[T] + Master.RecordSizes[T]*idx`, where `idx` is the DISK.HDR remap of *r* for types 2, 3, 4, 5, 7 and 8. Use `Decode[ItemRec](scen.Record(...))`.

## PC quick index (stride 432)

The C names fields after PC 0, as `pc_xxx`, `party + K` or `0x43e8 + K`. The member index enters through `432*m`.

| C | Go |
|---|---|
| 0x00 name | `.Name` |
| 0x08 | `.AgeDays` |
| 0x0c | `.Exp` |
| 0x10 | `.Kills` |
| 0x14 | `.Gold` |
| 0x18 | `.HP` |
| 0x1a | `.HPMax` |
| 0x1c | `.Stamina` |
| 0x1e | `.StaminaMax` |
| 0x20 | `.Carried` |
| 0x22 | `.MaxLoad` |
| 0x24 | `.Level` |
| 0x26 | `.Unk026` |
| 0x28+4r | `.SP[r].Cur` |
| 0x2a+4r | `.SP[r].Max` |
| 0x40+8i +0 | `.Items[i].ID` |
| 0x40+8i +2 | `.Items[i].Weight` |
| 0x40+8i +4 | `.Items[i].Type` |
| 0x40+8i +5 | `.Items[i].Icon` |
| 0x40+8i +6 | `.Items[i].Qty` |
| 0x40+8i +7 | `.Items[i].Flags` |
| 0xf0+16h | `.Hands[h]` (+0 Damage, +4 Data, +0xc B0C, +0xd B0D, +0xe ToHit, +0xf Flags) |
| 0x110 | `.Equipped` |
| 0x118 | `.SpellsLearn` |
| 0x11c | `.SpellRegen` |
| 0x122 | `.Status` |
| 0x12c | `.Stats` |
| 0x134 | `.Skills` (0x13d is Skills[9]; 0x151 is Skills[29], kirijutsu) |
| 0x152 | `.Resist` |
| 0x160 | `.AC` |
| 0x168 | `.NPCAttitude` |
| 0x188 | `.KnownSpells` |
| 0x198 | `.Flags` |
| 0x19c | `.Portrait` |
| 0x19d | `.Race` |
| 0x19e | `.Sex` |
| 0x19f | `.Class` |
| 0x1a0 | `.Rank` |
| 0x1a1 | `.Condition` |
| 0x1a2 | `.Regen` |
| 0x1a3 | `.Drain` |
| 0x1a4 | `.PoisonStr` |
| 0x1a5 | `.LoadCat` |
| 0x1a8 | `.SkillPoints` |
| 0x1ac | `.ItemCount` |
| 0x1ad | `.SwagCount` |
| 0x1ae | `.ToHit` |

Martydill's documentation has Level and Rank wrong. Trust the C.

## C patterns and their Go forms

| C | Go |
|---|---|
| `*(int *)((char *)pc_item_id + 432*p4 + (l12<<3))` | `party[p4].Items[l12].ID` |
| `pc_hp[216*m]` (an int array, so the stride is halved) | `party[m].HP` |
| `*(char *)(m*432 + 0x45b5)` | subtract 0x43e8 and split by 432 to get 0x1cd. Split again by 0x134: `party[m].Skills[0x1cd-0x134]`. Name the field after the first offset that is ≤ the rest, and index into it. |
| `*(long *)(l72 + level*4)` in check_level_up | `exp.Exp[level-1]`. The buffer is &l68, so l72 is element -1. |
| `test_bit(pc + 0x188, s)` | `testBit(party[m].KnownSpells[:], s)` |
| `get_bitfield(maze_data + 504, cell, 4)` | `getBitfield(mazeA.Floor[:], cell, 4)` |
| `(g4fa8 + i)[1584]` | `mazeB.Param[i]` |
| `*(int *)(g4fa8 + (i<<1) + 576)` | `mazeB.DirBits[i]` |
| `*(char *)(monster_groups[g] + 44*i + 34)` | `monsterGroups[g].Members[i].Condition` |
| `monster_groups + 44*member` (in WMEXE/WPOPS) | `monsterGroups[0].Members[member]`. This is the party side. |
| `g43a6[l2]` | `monsterGroups[l2-1]`. The array sits 2 bytes lower. |
| `g43b4[l2]` | `groupMonsterRec[l2-1]` |
| `*(int *)(sprite_count + (l4<<1))` | `spriteList[l4-1]` |
| `*(char *)(group_monster_rec[g] + 122)` | `groupMonsterRec[g].HP.Count`. This is the monster's level. |
| `roll_dice(&rec[24])` | `rollDice(rec.Damage)` |
| `win[0]`, `win[1]` ... on a `char *` window | the `Window` header fields |

### Scenario record buffers held in locals

The C declares a record buffer as a local array lB (at bp-B) and `access_scenario_record(T, &lB, n)` fills it. Fields then appear in three forms, and each maps to offset **B-K** in the record:

- separately named locals lK, at bp-K
- `((char *)&lB)[k]`, at offset k
- `*(int *)((char *)&lB + k)`, at offset k

Example: an item buffer at &l76 with a use of `l46` is ItemRec offset 30, which is `Weight`. Declare a single `var rec dbs.ItemRec`, or decode it, and replace every such local with a field. Locals inside [B-size+1, B] belong to the buffer. Anything else is an ordinary local.

Where a pointer to the buffer is passed on (`f1234(&l76)`), the callee's `pN` is a typed `*ItemRec`. `MapA` and `MapB` arrive as `maze_data` and `g4fa8`.

## Watch-outs

- **party_level is the 8×8 block index**, from 0 to 11, inside the current map. In Go it is `partyBlock`. The dungeon map number is `game_state_arg`. A cell index is `block<<6 | cell_y<<3 | cell_x`.
- **Signedness:**
  - C `char` is signed. `Combatant` and `NPCAttitude` are int8 because they are compared with < 0.
  - PC bytes are uint8, except `ToHit`.
  - Keep any explicit `(unsigned char)` cast from the C, because some fields are read both ways.
- **Aliasing:**
  - `g4ecc` to `g4edc` are fields of `g4ec8[i]` (`partyCounters`).
  - `g36e2` and the related `g36e4`… are fields of `monsterPics[0]`. For slot s, add 314*s.
  - `g51c0` and `g51c2` are the low and high halves of `NPCRec.Gold`.
- **Double reads:** `ItemRec.HandFlags` (+72) and `MonsterPicRec.FrameX` (+6) are sometimes read as int16 across two fields. In Go, write `int16(a) | int16(b)<<8` explicitly.
- **SaveState:** `party_count` is saved as int16. Maps 14 and 15 are always zero, and only 14 maps are cached.
- **MASTER.HDR:** `RecordCounts` are capacities. For types 2 and 3 there are 16 (and 14 are used).

## Uncertain fields

- These names are guesses: `PC.Unk026`, `Hand.B0C` and `Hand.B0D`, `ItemRec.HandD` and `HandC`, `MapA.Feature`, `SpecialCells` and `SolidCells`, `MapB.Data` and `Unk288`, `MonsterAttack.Range`, `NPCRec.Entries`, `TreasureEntry.A` and `B`, and the `Combatant` fields `B24` to `B41`.
- Every `UnkXX` field is unknown. Name it when a translated function makes its meaning clear.
- The race and spell tables are static data in the ROOT or overlay data segments, not scenario records. They are not typed here. Their layout is in the per-image `data.c`. The spell table has 6-byte entries at 0xde.
