# Parity harness

Runs the original Wizardry VI (DOSBox-X) and the Go port with the same key script, grabs one
frame per step from each, and diffs them.

    tools/parity/parity.sh tools/parity/party.keys /tmp/out

Needs `dosbox-x`, `ffmpeg`/`ffprobe`, ImageMagick (`compare`, `magick`), `bc`, Go.

## Key script

One AUTOTYPE key name per line (`enter esc up down left right space a..z 0..9 f1..f10 ...`,
see `scan` in parity.sh for the list). `#` comments. Optional `WAIT n` (seconds before the
first key, default 8) and `PACE n` (seconds between keys, default 3). Key i goes in at
WAIT+i*PACE; step i's frame is taken 0.3 s before key i+1.

## How it works

- Original: a fresh copy of `$ORIG` (default ~/Projects/wiz6-re/gog/x/app) under
  OUTDIR/work/dos, so saves never touch the source. DOSBox-X runs headless
  (`SDL_VIDEODRIVER=dummy`, `-time-limit`, `timeout -s KILL` as a backstop). Keys are fed by
  DOSBox-X's own `AUTOTYPE -w WAIT -p PACE`; the screen is recorded by `DX-CAPTURE /V`
  (ZMBV AVI, one file per video mode) and ffmpeg pulls the frame at each step time.
  Nothing touches the host keyboard or mouse.
- Port: built from this repo, run on another data copy with `-input T:kXX` and `-shot T:FILE`
  at the same times. `-input` sets host.Scripted, so the window ignores the real keyboard and
  mouse. The port still opens a window (it has no headless mode).
- Diff: `compare -metric AE` per step; `FUZZ=n` (percent) tolerates colour noise.

No palette mapping is needed: DOSBox-X's EGA output and the port's DAC give the same 16 RGB
values (#000000, #0000AA, ... #FFFFFF).

## Output

OUTDIR/dos/NNN.png, OUTDIR/go/NNN.png, OUTDIR/diff/NNN.png (red = differing pixel),
OUTDIR/report.txt (step, key, differing pixels), OUTDIR/work (logs, AVIs, data copies).

## Limits

Timing is approximate: AUTOTYPE runs on emulated time, the port on wall time, and load
times differ. Keep PACE generous (3 s) so each screen settles; keep the default
`CYCLES=fixed 20000`: with `max` the original drops or misreads keys and drifts from the port. Animated screens (the camp scene, the gate) differ
frame to frame even when correct. The original shows its mouse arrow; the port under
-input does not draw one, so expect ~33 pixels of difference on otherwise equal screens.
