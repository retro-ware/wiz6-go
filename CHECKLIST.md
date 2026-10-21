# wiz6-go checklist

## Foundation
- [x] Scaffold: gpu module and host package (from dagger-go), PORTING.md
- [ ] Typed records: PC, items, monster groups, scenario records, windows, maze (state.go, dbs/records.go, LAYOUTS.md)
- [ ] dbs: MSG, SCENARIO, DISK/MISC.HDR, fonts, EGA pictures, portraits, MON*.PIC, MAZEDATA, SND
- [ ] drv: EGA.DRV behaviour (glyphs, sprite blits, maze tiles, picture slots)
- [ ] Root runtime: windows, text, messages, random, bits, keys, mouse, click regions, delays, sound
- [ ] main: game_state machine over the overlays
- [ ] cmd/wiz6: -data, -import (gameimport), COVER.PNG, -shot, -input

## Overlays (Go functions / C functions)
- [ ] WINIT 0/16
- [ ] WBASE 0/46
- [ ] WPCMK 0/75
- [ ] WPCVW 0/101
- [ ] WMAZE 0/128
- [ ] WMELE 0/61
- [ ] WMEXE 0/93
- [ ] WTREA 0/85
- [ ] WMNPC 0/140
- [ ] WPOPS 0/62
- [ ] WDOPT 0/69

## Parity
- [ ] Title and menus match DOSBox-X screenshots
- [ ] Character creation, save/load round trip with the original's files
- [ ] Exploration, combat, NPCs
