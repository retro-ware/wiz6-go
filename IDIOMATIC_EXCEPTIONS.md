# Idiomatic Go exceptions

Rules left unmet on purpose, for review. See ~/notes/idiomatic-go.md.

## internal/dbs

### Hex/binary literals in code

None left. Hex stays only in comments (original-binary offsets and addresses).

### Kept Unk fields (meaning not known yet, or one field holds several unrelated values)

Fields shown as `[n]` are arrays whose elements mean different things. Splitting them would change the field types, which the binary layout rules forbid.

- records.go:94 `PC.Unk026`: incremented on resurrection and shown on the sheet like an age. Meaning unclear.
- records.go:108 `PC.Unk194`: not used.
- records.go:138 `MasterHeader.Unk40`: not used.
- records.go:144 `DiskHeader.Unk0`: not used.
- records.go:157 `ScenarioHeader.Unk140`: mixed. Bytes 20.. are a global flag bitset (DS:0x3498), and 28+2*npc holds NPC flags. The rest is unknown.
- records.go:283-294 `ItemRec.Unk62, Unk63, Unk68, Unk69, Unk73`: each is used (an id threshold, a printed cipher, a power, a use-action switch, the high byte of HandFlags) but has no clear single name.
- records.go:308 `MapA.Unk1080`: not used.
- records.go:320 `MapB.Unk288`: per-event-kind argument (side<<8|face, a flag bit, a sound).
- records.go:334 `MonsterAttack.Unk4`: copied whole into the attack effect array.
- records.go:351-367 `MonsterRec.Unk134, Unk135, Unk138, Unk168, Unk172, Unk198, Unk208, Unk211, Unk221`: mixed arrays (attack stats, spells, chances, hit-area weights, a kind code compared with spell kinds, a 9-byte block of unrelated combat values) or unused.
- records.go:381 `NPCRec.Unk68 [9]int16`: message bases, interjections, price percents, accepted item ids.
- records.go:384 `NPCRec.Unk86 [5]uint8`: [0] a game global, [1] pic, [3] NPC level.
- records.go:387 `NPCRec.Unk8D`: trade/talk flag bits 2/4/8.
- records.go:419-435 `MonsterPicRec.Unk0, Unk56, Unk114, Unk283, Unk304, Unk306, Unk309, Unk313`: unused or mixed (Unk309: [0] flag, [1],[2] jitter range, [3] layer count; Unk114 run lengths that continue into Quads).

### Non-Unk names that are still not descriptive (left as is: the meaning is a guess, see LAYOUTS.md "These names are guesses")

- records.go:67 `Hand.Data [8]uint8`: an opaque copy of item +46..53. Only [6] (crit chance) is known.
- records.go:68-69 `Hand.B0C`, `Hand.B0D` and records.go:285-286 `ItemRec.HandD`, `HandC`: named after their offset or destination. Use is unclear (B0C goes into attack effects, and B0D==0 gates the back rows).
- records.go:192-193 `PartyCounter.A`, `B`: `[0].B` is also used as a flag (levitation-like), so "value, duration" does not hold for every entry.
- records.go:319 `MapB.Data`: per-event-kind payload.
- records.go:372-373 `NPCEntry.A`, `B`, records.go:404-405 `TreasureEntry.A`, `B`: A is an item id for some kinds, B a count. Meaning by kind not fully known.
- records.go:420 `MonsterPicRec.W4`, records.go:428 `Quads`: W4 is unused. Quads is [depth, next, min ticks, random ticks] in its use, but [0][...] is also read as a continuation of Unk114.

### Low-level code kept

- path.go:42 `planarPixel`, pic.go:63, huffman.go:39-50 `decode`: bit shifts and masks to read EGA bit planes and an MSB-first Huffman bit stream. This is the file format itself, and there is no higher-level equivalent.
- pic.go:144 `UnRLE` loop bound `rleChunkSize-1`: copies op_ae's quirk of reading control bytes only while the chunk index is below 0xfff.
- pic.go:158 `-int(int8(c))`: an RLE run length is the control byte read as a negative int8.
- scenario.go:90 `offset`: indexes a flat `remapTables` byte slice (DISK.HDR from MapIndex on), not the typed `DiskHeader` arrays. The original reads past the end of a table into the next one (treasure indexes 256+ land in MonsterIndex), and the parity goldens rely on it. Table starts are derived from the array lengths.
- Scenario.Hdr stays raw `[]byte` (typed by the game through Decode[ScenarioHeader]). This is unchanged.
