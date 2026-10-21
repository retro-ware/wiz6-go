# Wizardry VI in Go: porting plan

A native desktop port of Wizardry VI: Bane of the Cosmic Forge (WROOT.EXE and its 11
overlays). It is written in idiomatic Go from the byte-exact C decompilation in
../wiz6-re. It is also a goplay engine (see ../goplay/ADDING-GAMES.md and INTEGRATION.md).

## Stack

- Window, input and presentation: the `gpu` module in ./gpu. It is gogpu plus WebGPU, in
  pure Go, and is shared with ../dagger-go. Build with
  `CGO_ENABLED=0 go build -tags nofakecgo ./cmd/wiz6`.
- Sound output: `github.com/ebitengine/oto/v3`, through internal/host.
- `-import SRC` finds the game through ../gameimport. It accepts folders, zips, CD images
  (ISO/BIN/CUE), GOG installers (setup_*.exe plus .bin) and loose files. It copies the data
  files into -data and writes COVER.PNG, the title page.
- Run with no flags in a goplay folder, wiz6 reads ./data and writes only ./save.

## Packages

- `internal/host` is the desktop around the game. It shows the 320x200 frame of palette
  indices, turns the keyboard into PC scan codes, and handles the mouse, the clock and audio.
  The game runs on its own goroutine and calls host.Present whenever the screen changes.
- `internal/drv` reimplements EGA.DRV, the original's graphics driver, as a 320x200
  16-colour framebuffer. It covers glyph and sprite blits, maze tiles from MAZEDATA, and
  picture loading. It holds behaviour only. There is no x86, no planes and no ports.
- `internal/dbs` holds the data files: MSG.DBS/HDR (messages, decode_text),
  SCENARIO.DBS/HDR (records), PCFILE.DBS (the roster), NEWGAME.DBS/SAVEGAME.DBS,
  MISC.HDR, DISK.HDR, fonts (WFONT*.EGA), pictures (*.PIC, WPORT*.EGA, TITLEPAG.EGA,
  MAZEDATA.EGA, DRAGONSC.EGA, GRAVEYRD.EGA) and sounds (*.SND). Each record has a typed Go
  struct with Decode/Encode. Unknown bytes keep their place and round-trip unchanged.
- `internal/game` is all game code: the root runtime and the 11 overlays. They share one
  package, so overlays call each other and the root directly.
  - Files are named after the overlay and the subsystem, e.g. `root_window.go`,
    `root_text.go`, `winit.go`, `wmaze_view.go`, `wmele_combat.go`.
  - Functions take the C name in CamelCase: `draw_member_list` becomes `drawMemberList`.
    Unnamed C functions (`fXXXX`) get a descriptive name once their purpose is clear.
    Until then they are `ovlXXXX`, e.g. `wmazeF606f`.
  - A function copied into several overlays (run_menu, choose_party_member, ...) exists
    once. Compare the copies first. If they differ, keep one per overlay with the overlay
    as a suffix.
  - Every function starts with a comment naming its C origin, e.g.
    `// draw_member_list (WBASE).`
- `cmd/wiz6` holds the flags and the import.

## Rules

- Every value is a typed variable. Game code has no raw memory blocks, no byte offsets
  into records and no pointer arithmetic. `*(int *)((char *)pc_hp + 432*m)` becomes
  `party[m].HP`. Record bytes are decoded only at the file boundary (dbs).
- Behaviour matches the C exactly: the same random number sequence (root `random`,
  `random_word`, `step_random`), the same messages, the same rules, the same order of
  screen updates.
  - Integer widths matter where the C relies on them. A `char` field wraps at 8 bits, and
    unsigned-char comparisons stay unsigned. Use int8/uint8/int16/uint16 for record fields
    and plain `int` for locals unless wrap-around matters.
- The game is a blocking program, like the C. Input waits poll host and call host.Sleep.
  Every screen change reaches host through drv's present call.
- `goto` from the C becomes structured Go: loops, switch, early return. Only keep a Go
  `goto` where restructuring would obscure the logic.
- No cgo, no JavaScript, no web UI, no CPU emulation, no exe tables at runtime. Static
  tables in the overlays' data segments become Go literals (from ../wiz6-re/src/*/data.c
  and the names in ../wiz6-re/names).
- Native desktop only.

## Source of truth

- `../wiz6-re/src/IMAGE/fXXXX.c`: one C function per file. The doc comment is above the
  function. Names come from `../wiz6-re/names/IMAGE.json`.
- Root functions that stay asm in the decomp (drivers, the window refresh, put_char,
  decode_text, random, bitfields, sound, keyboard) are read from their asm
  (`src/ROOT/fXXXX.c` #asm blocks; `../wiz6-re/.venv/bin/python ../wiz6-re/tools/x86dis.py`
  helps). They are ported for their behaviour.
- EGA.DRV is in `../wiz6-re/gog/x/app/EGA.DRV`. Data formats are documented in
  `../wiz6-re/ref/martydill/file_format_docs`, and martydill's Python loaders are a
  cross-check.
- The game data is in `../wiz6-re/gog/x/app` (`-data` for testing).

## Phases

1. Foundation: host, drv, dbs, the root runtime (windows, text, messages, scenario
   records, random, bits, input, click regions, delays, sound), the game state types and
   main's overlay state machine. The milestone is booting to WINIT's title screen.
2. Overlays, in play order:
   - WINIT: boot and title.
   - WBASE: the start menu and party.
   - WPCMK: character creation.
   - WPCVW: character view.
   - WMAZE: exploration and the maze view.
   - WMELE and WMEXE: combat.
   - WTREA: treasure.
   - WMNPC: NPCs.
   - WPOPS.
   - WDOPT: options and disk.
3. Parity: compare screenshots and runs with DOSBox-X running the original.

After each milestone, update CHECKLIST.md and commit.
