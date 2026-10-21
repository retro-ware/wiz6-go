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

## internal/drv

### Hex/binary literals in code

None left. Hex stays only in comments (driver addresses, cs: offsets, and the hex form of the slot table next to its decimal values at pic.go:18-19).

- mouse.go:91 `pointerImage`: the pointer bitmap as strings of hex digits (one colour index per character). It is image data copied from captures; each digit is parsed with `strconv.ParseUint(..., 16, 8)` (mouse.go:121).
- drv.go:72 `Palette`: the EGA RGB palette as decimal hardware data (85/170/255 levels). Not hex, listed as hardware data.

### Unclear names kept

- maze.go:11 `MazeTile(draw, or, mirror int)`: `or` is a 0/1 flag (OR the tile into the buffer). Kept to keep the exported signature; callers pass ints.
- pic.go:127 `DrawFrames(n, x, y, clipL, clipR, flags, ...)` and the short locals in DrawFrames, LoadPic and MazeTile (`d`, `t`, `tw`, `th`, `pw`, `ph`, `di`, `si`, `r0`): kept as is. They are local and described by the doc comments; `di`/`si` mirror the original's registers in the RLE loop.
- pic.go:52 `drawFramesBottom` (144): the first row DrawFrames leaves clear. The value is from the driver; why 144 (presumably the bottom of the picture windows) is not confirmed.
- pic.go:28 `rleChunkSlack` (256): the size of the zero padding after a chunk. The original reads past the chunk into whatever follows; 256 is this port's choice, not the original's.

### Low-level code kept

- pic.go:17 `slotArea` and pic.go:19 `slotStartParagraph`: the picture slot area is a flat byte array addressed in paragraphs, as in the driver. A picture larger than its slot runs on into the next slots, and LoadScreen's pristine copy lives in slot area 5 onwards, so `LoadScreen(nil)` shows whatever the slots there hold now. Typed per-slot values would lose that, and the parity goldens may depend on it.
- pic.go:62 `LoadPic`: RLE decoding with the original's chunk quirk (control bytes only below index 0xfff of each 4096-byte chunk).
- pic.go:99 `LoadScreen`, pic.go:152-156 (frame record offset and tile bitmap), pic.go:216 `tilePixel`, maze.go:62 `putPlanarByte`: EGA bit-plane decoding with bit masks (`pixelBit`, drv.go:192). This is the file format of the pictures and MAZEDATA. Frame records are read straight from `slotArea` bytes (offset as little-endian uint16, tile bitmap LSB first) because they live inside the slot memory above.
- maze.go:62 `putPlanarByte` takes a byte offset into a 40-bytes-a-row buffer (`o`), as MAZEDATA records address it; offsets outside the screen are dropped, as the original wrote them to memory the screen does not show.
- `compose` (the unused 0x1400-byte draw_text scratch buffer in pic.go) was deleted, not kept.

## internal/host

### Hex/binary literals in code

- window.go:16-39 `scanCodes`: the PC set-1 scan code of each key, in hex. These are external hardware codes, conventionally written in hex and checked against scan code tables that way.
- window.go:12 `extendedKeyPrefix = 0xE0`: the scan code prefix byte of grey (extended) keys. Hardware value.
- host.go:37 `keyBreakBit = 0x80`: the bit set in a key-up scan code. Hardware value.

### Unclear names kept

- audio.go:65 `mix`, pace.go:29 `pace`, host.go `state`: package state structs with short names; their fields are documented.
- audio.go:59 `voice.pos`: 32.32 fixed-point position (documented by `positionShift`).
- audio.go:29-31 `maxVolume`, `maxPan`: only named for the doc comments; the code takes volume and pan as given. The game passes `0x7f`/`0x8000` literals (root_sound.go:78), left for the internal/game pass.

### Low-level code kept

- audio.go:59, 189 and the mixer loop: fixed-point sample positions and shifts (`positionShift`, `interpolationBits`, `panShift`, `volumeShift`). This is the mixer's integer arithmetic; switching to float would change the output samples.
- host.go:247 `dacTo8Bits`: widens a 6-bit VGA DAC value to 8 bits with shifts (`v<<2 | v>>4`), the standard bit-replication.

## gpu

`gpu` is its own module (gpu/go.mod, `replace gpu => ./gpu`), not part of the wiz6-go module. Its exported API copies Ebiten's (`GeoM`, `ColorScale`, `SetTPS`, `KeyKP0`, `Vertex.DstX`, `MeshVertex.NX`, `Mat4`, `Ident`, ...), so exported names were not renamed. Only unexported names changed.

### Hex/binary literals in code

None left. 16-bit colour channel maxima use `math.MaxUint16`, the multisample mask `math.MaxUint32`.

- pipelines.go:45 `blitWGSL`: the WGSL full-screen-triangle trick `(id << 1u) & 2u`, `id & 2u` in shader source. It is the standard way to make the three corners from the vertex index; WGSL constants would not make it clearer.

### Unclear names kept

- image.go:258 `GeoM` fields `b`, `c`, `tx`, `ty` (and `aMinusOne`, `dMinusOne`): the conventional names of affine matrix entries, as in Ebiten.
- device.go:22 `dev`, run.go:23 `win`, image.go:391 `batch`, input.go:167 `input`: package state structs; their fields are documented.
- mesh.go:49 `Ident`: Ebiten-style exported name, kept.
- shader.go `wgslSize`: WGSL's size/alignment table, as numbers from the WGSL spec.

### Low-level code kept

- device.go:314 `bytesOf`: `unsafe.Slice` over a vertex slice to upload it to the GPU without copying. The GPU API takes bytes; this is the usual Go way to hand it a typed slice.
- pipelines.go:142-152: vertex layouts use `unsafe.Sizeof` / `unsafe.Offsetof` of `gpuVertex` and `MeshVertex` (instead of the former literals 48, 8, 12, 16, 20, 32), so the layout follows the structs.
- device.go `alignUp` and the stream chunk / readback row alignment: byte offsets and sizes for GPU buffers, as WebGPU requires (`uniformOffsetAlignment`, `bytesPerRowAlignment`, `writeBufferAlignment`).

## internal/game: core (root, state, winit, animpace, difftest_hook)

Shared names for the other groups: `gameStateID` and its `state...` constants (state.go), the key codes `keyLeft`/`keyUp`/`keyRight`/`keyDown`/`keyEnter`/`keyEscape` and key classes `keyClassAny`/`keyClassMenu`/`keyClassDigits` (root_sys.go), the border styles `borderNone`..`borderTopEdge` (root_window.go), `viewAnimShow`/`viewAnimMaze`/`viewAnimQueue` (state.go), `videoEGA`/`videoMouse` (state.go), the sound settings `soundQuietSteps`..`soundOff` and `loadedSound` (root_sound.go). Other groups' files still use the literals; only edits forced by renames and the `gameState` type were made there (`gameStateID(returnCmd)` in wbase.go, wmele.go, wpcvw_sheet.go; wdopt_test.go).

### Hex/binary literals in code

- root_sys.go:169-181 `scanExtendedPrefix`, `scanBreakBit`, `scanLeftShift`, ... `scanDown`: PC set-1 scan codes, in hex as in scan code tables (as host's `scanCodes`). Hardware values.
- root_sys.go `scanToASCII`, `scanToShiftedASCII`: the scan code to ASCII tables as strings with `\x00`/`\x1b`/`\x08` escapes. Keyboard layout data.
- state.go `videoMouse = 1 << 12`, root_window.go `cellIDMask`, `cellPointer`, `cellDirty`, `attrFontMask`: bit masks of the driver's / root's packed bytes, written as shifts.

### Unclear names kept

- state.go `g4fb0` (a window), `g433a` (read as WMNPC's npcID), `g363e`, `g4338`/`g4354` ("combat entry parameters"), `g4f86`, `g4f88`, `g4f8a`, `g4fa6`, `g4352` (set by an item of kind 7), `msgBlockNum`: meaning not clear from their use. Most are named in the parity goldens too.
- state.go `Combatant.W10`, `W12`, `B24`..`B41`, `MonsterGroup.B396`..`B414`, `Spell.B3`: offset names of struct fields whose meaning is not known (or not one meaning).
- state.go `magicWordTyped`: the value it gets, 17210, is the original's; why that number is unknown. It is set in wbase.go as a literal.
- state.go `conditionGlyph`, `statusGlyph`, `realmColors`, `stackable`, `recipes`, `spellTable`: root data tables, kept as numbers (glyph indexes, colours, item ids, spell fields).
- winit.go `gmodeInvalid = 255`: the Gmode that makes the game exit with code 2; what the original means by it is not known.
- root_sound.go `firstStepSound`..`lastStepSound`: "footsteps and the like" (is_sound_10_to_12); which sounds exactly is not checked.
- Sound numbers (`loadSound(39)`, `playSound(4)`, ...), exit codes (`exitGame(9)`, ...), window positions, priorities, fill glyphs and fonts in winit.go and root*.go stay literals: they are the original's arguments, and most have no name of their own (exit codes are reused for several failures).

### Low-level code kept

- root_sys.go:95-122 `setBitfield`, `getBitfield`, `extractBits`: packed bit fields read as little-endian 16-bit words with shifts and masks, as the original does (the save data and map flag layouts depend on it).
- root_sys.go `stepRandom`, `randomWord`: 16-bit wrap-around (`int16`/`uint16` conversions) and the `& randomMax` masks of the original's generator; the parity goldens record its numbers.
- root_sound.go:51 `readSoundRec` and root.go:292 `le16`: a 12-byte record decoded with encoding/binary by offset (not a dbs type).
- winit.go:46 `loadFontFile`: packs font 0's pixels into bytes MSB first (`<< (7 - x)`) for drv.SetMonoFont, the driver's glyph format.
- root.go:253 `clickRegions`: a flat byte table of 4-byte entries (x1 `clickRegionsEnd` ends it), passed as `[]byte` to drv.PollClick and the overlays.
- root_window.go `cellOwner` and `eachCell`: the ownership map is one 1000-byte line as in the C; a rectangle past the right edge wraps to the next row.

## internal/game: wbase (wbase, wbase_party)

New names in wbase.go: `menuKeyEscape`..`menuKeyEnter` (places in menuKeys, the menu loops' results), `campAdd`..`campQuit` (camp menu entries), `rosterSlotAvailable`/`rosterSlotInParty`, `inputPrintable`/`inputDigits` (inputText modes; wpcmk.go, wpcvw.go, wpcvw_levelup.go still pass 0/1), `noEquippedItem` (wdopt.go, wmaze_main.go still use 255), portrait layout constants and a few widths/glyph bases. Game states, key codes and classes, borders, `clickRegionsEnd`, `videoEGA`/`videoMouse` replace the literals.

### Hex/binary literals in code

- wbase.go `leftmostPixelBit = 0x80`: the top bit of a WPORTn.EGA glyph row byte is its leftmost pixel. File format.

### Unclear names kept

- `g4fb0`, `g4f88`, `g4f86` (set in setupGameWindows, resetPartyScreen): core state, meaning not clear.
- `magicWordTypedValue = 17210`: the original's value; why that number is unknown.
- Message numbers (1000, 1002, 1202, 2300, 2350, ...), sound numbers, window positions/priorities, colours, fonts, fill glyphs (26), glyph numbers of the camp/command window/list frame/gauges (69..71, 86, 99, drawGlyph2x2 arguments), the camp scene's maze tile and frame tables and clip columns, and the click region indexes in choosePcfileChar/doGameConfiguration stay literals: they are the original's arguments with no name of their own.
- doGameConfiguration's setting indexes 0..7 (sound, delay, level, input, gmode, path 3, 0, 1), their rows/colours and the `% 6`, `% 3`, `% 2` value counts; scenarioHdr.Paths indexes 0, 1, 3; `scenarioHdr.Input == 1` (mouse): kept as numbers, the path slots' meaning is not checked.
- `startGame(0/1)`, `loadGame(resume)` 0/1, `drawPcfileList` highlight 1, `menuClicked` 0/1: the original's int flags.
- `openData(..., 14)`, `loadMonsterPic(mon, slot, 21)`, `loadMonsterPicSlot(8, 4)`, picture slots 2 and 4, `mapCacheA` limit 14, `exitGame(19)`: original arguments, not named.

### Low-level code kept

- wbase.go loadPortrait: decodes EGA bit planes from the WPORTn.EGA bytes by offset and bit mask (file format).
- wbase.go drawGauge: `int16(int32(cur)*100/int32(max)/10) + 1`, the C's int casts and 16-bit wrap kept for parity.
- `int(byte(getKey(...)))`: the C's `& 0xff` of the key word (low byte), kept as a byte conversion.
- wbase_party.go loadGame: walks the save file's bytes record by record (`next`) and decodes them with dbs.Decode; pcfileRecordIO reads/writes a record at a byte offset of PCFILE.DBS.
