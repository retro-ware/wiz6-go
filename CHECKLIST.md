# wiz6-go checklist

## Foundation
- [x] Scaffold: gpu module and host package (from dagger-go), PORTING.md
- [x] Typed records: PC, items, monster groups, scenario records, windows, maze (state.go, dbs/records.go, LAYOUTS.md)
- [x] dbs: MSG, SCENARIO, DISK/MISC.HDR, fonts, EGA pictures, portraits, MON*.PIC, MAZEDATA, SND
- [x] drv: EGA.DRV behaviour (glyphs, sprite blits, maze tiles, picture slots)
- [x] Root runtime: windows, text, messages, random, bits, keys, mouse, click regions, delays, sound
- [x] main: game_state machine over the overlays
- [x] cmd/wiz6: -data, -import (gameimport), COVER.PNG, -shot, -input

## Overlays (Go functions / C functions)
- [x] WINIT 16/16 (scripted-run tested)
- [x] WBASE 46/46 (scripted-run tested)
- [x] WPCMK 75/75 (scripted-run tested)
- [x] WPCVW 101/101 (scripted-run tested)
- [x] WMAZE 128/128 (scripted-run tested)
- [x] WMELE 61/61 (scripted-run tested)
- [x] WMEXE 93/93 (scripted-run tested)
- [x] WTREA 85/85 (scripted-run tested)
- [x] WMNPC 140/140 (scripted-run tested)
- [x] WPOPS 62/62 (scripted-run tested)
- [x] WDOPT 69/69 (scripted-run tested)

## Parity
- [ ] Title and menus match DOSBox-X screenshots
- [ ] Character creation, save/load round trip with the original's files
- [ ] Exploration, combat, NPCs

## Cleanup
- [ ] Merge duplicated/prefixed functions (wpcvw*/wpops* variants), TODO(merge) globals into state.go
- [ ] PC.Items is 22 slots (20 + 2 in Unk0E0); spriteList has 12 entries
- [ ] fillWindow hi flag misuse in WBASE / test hooks (attribute 0x30 vs 3)
- [ ] DOSBox-X capture harness for side-by-side comparison
