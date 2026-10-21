#!/bin/bash
# parity.sh SCRIPT OUTDIR -- run Wizardry VI in DOSBox-X and the Go port with the same keys,
# capture one frame per step from each and diff them. See README.md.
set -e
SCRIPT=$(cd "$(dirname "$1")" && pwd)/$(basename "$1"); OUT=$2
HERE=$(cd "$(dirname "$0")" && pwd); GO=$(cd "$HERE/../.." && pwd)
ORIG=${ORIG:-$HOME/Projects/wiz6-re/gog/x/app}
WAIT=8; PACE=3; FUZZ=${FUZZ:-0}
n=0; KEYS=(); SLOT=(); AT=() # SLOT[i]: key i's PACE slot (',' lines add an empty slot); AT: the autotype words
while read -r a b; do
  case "$a" in
  ''|'#'*) ;; WAIT|PACE) eval "$a=$b" ;;
  ,) AT+=(","); n=$((n+1)) ;;
  *) KEYS+=("$a"); SLOT+=($((n+0))); AT+=("$a"); n=$((n+1)) ;;
  esac
done < "$SCRIPT"
N=${#KEYS[@]}; SLOTS=$n
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd); rm -rf "$OUT"/{dos,go,diff,work}; mkdir -p "$OUT"/{dos,go,diff,work/cap}

# PC set-1 scan code of an AUTOTYPE key name (the port's -input kXX).
scan() {
  local s=" esc:01 1:02 2:03 3:04 4:05 5:06 6:07 7:08 8:09 9:0a 0:0b minus:0c equals:0d bksp:0e tab:0f
  q:10 w:11 e:12 r:13 t:14 y:15 u:16 i:17 o:18 p:19 enter:1c a:1e s:1f d:20 f:21 g:22 h:23 j:24
  k:25 l:26 z:2c x:2d c:2e v:2f b:30 n:31 m:32 comma:33 period:34 space:39 f1:3b f2:3c f3:3d
  f4:3e f5:3f f6:40 f7:41 f8:42 f9:43 f10:44 home:47 up:48 pageup:49 left:4b right:4d end:4f
  down:50 pagedown:51 insert:52 delete:53 "
  local r=${s#* $1:}; [ "$r" != "$s" ] || { echo "parity: unknown key $1" >&2; exit 1; }; echo "${r%% *}"
}
# The ASCII code BIOS reports with a key (hex, 2 digits): 00 for arrows and function keys.
ascii() {
  case "$1" in
  enter) echo 0d ;; esc) echo 1b ;; space) echo 20 ;; bksp) echo 08 ;; tab) echo 09 ;;
  minus) echo 2d ;; equals) echo 3d ;; comma) echo 2c ;; period) echo 2e ;;
  [a-z0-9]) printf %02x "'$1" ;; *) echo 00 ;;
  esac
}
# Step i's key goes in at WAIT+SLOT[i]*PACE; its frame is taken just before key i+1 (last: one PACE after).
shot_t() { local s=$SLOTS; [ $(($1+1)) -lt $N ] && s=${SLOT[$(($1+1))]}; echo "$WAIT + $s*$PACE - 0.3" | bc; }

# --- original: AUTOTYPE feeds keys inside DOSBox-X, DX-CAPTURE records a ZMBV AVI ---
cp -R "$ORIG/" "$OUT/work/dos/"
# Deterministic random numbers (DETRNG=0 to turn off): seed_random reads 0 instead of the BIOS
# tick count (26 8B 07 -> 31 C0 90 at 0x1F4A) and key_available no longer calls step_random
# (E8 66 FF -> 90 90 90 at 0x2850). The port does the same under -detrng.
DET=${DETRNG:-1}
# Sound off (SCENARIO.HDR byte 408 = 5) in both copies: the original's sound player reprograms
# the PIT and hooks IRQ0, which speeds up INT 1Ch and so KEYFEED's clock.
nosound() { printf '\005' | dd of="$1/SCENARIO.HDR" bs=1 seek=408 conv=notrunc 2>/dev/null; }
nosound "$OUT/work/dos"
if [ "$DET" = 1 ]; then
  python3 - "$OUT/work/dos/WROOT.EXE" <<'P'
import sys
p = sys.argv[1]; b = bytearray(open(p, 'rb').read())
for off, old, new in ((0x1F4A, '268b07', '31c090'), (0x2850, 'e866ff', '909090')):
    assert b[off:off+3] == bytes.fromhex(old), hex(off)
    b[off:off+3] = bytes.fromhex(new)
open(p, 'wb').write(b)
P
  # The ambient-sound idle step (WMAZE f4655, WDOPT 4688, WTREA 45b8) rolls random(40) on every
  # idle step: its call random becomes mov ax,1 (the port: -detrng skips the roll).
  python3 - "$OUT"/work/dos/*.OVR <<'P'
import re, sys
for p in sys.argv[1:]:
    b = open(p, 'rb').read()
    n = re.sub(rb'(\xb8\x28\x00\x50)\xe8..(\x59\x3d\x00\x00\x7e\x08)', lambda m: m.group(1) + bytes.fromhex('b80100') + m.group(2), b, flags=re.S)
    if n != b:
        open(p, 'wb').write(n)
P
fi
LIMIT=$(echo "$WAIT + ($SLOTS+1)*$PACE + 2" | bc | cut -d. -f1)
# Keys: KEYFEED.COM (keyfeed.asm), a TSR that puts each key in the BIOS buffer at its time on
# the emulated clock (18.2 ticks/s), so key and frame times line up exactly. (DOSBox-X's
# AUTOTYPE runs on host time and aborts at random with "MAPPER: Couldn't find a button".)
# TPS: INT 1Ch ticks per second (18.2065 with the PIT left alone; see nosound).
TPS=${TPS:-18.2065}
{ cat "$HERE/keyfeed.asm"
  prev=0
  for i in $(seq 0 $((N-1))); do
    t=$(echo "($WAIT + ${SLOT[$i]}*$PACE)*$TPS/1" | bc)
    echo "	dw $((t-prev)), 0x$(scan "${KEYS[$i]}")$(ascii "${KEYS[$i]}")"; prev=$t
  done
  echo "	dw 0FFFFh"; echo "tend:"
} > "$OUT/work/keyfeed.asm"
nasm -f bin -o "$OUT/work/dos/KEYFEED.COM" "$OUT/work/keyfeed.asm"
cat > "$OUT/work/dos.conf" <<C
[sdl]
output=surface
[dosbox]
machine=vga
memsize=16
quit warning=false
captures=$OUT/work/cap
[cpu]
cycles=${CYCLES:-fixed 20000}
[autoexec]
mount c "$OUT/work/dos"
c:
keyfeed
dx-capture /V wroot.exe
exit
C
SDL_VIDEODRIVER=dummy timeout -s KILL $((LIMIT*3+60)) dosbox-x -conf "$OUT/work/dos.conf" \
  -nogui -nomenu -exit -fastlaunch -time-limit $LIMIT >"$OUT/work/dos.log" 2>&1 || true
# The capture splits into one AVI per video mode; walk them to map program time to file time.
for i in $(seq 0 $((N-1))); do
  # The frame just before the next key, and one a second earlier: pixels that differ between
  # the two (0.35 PACE apart) are animating (water, the camp scene) and are masked out of the diff.
  for v in "0:$OUT/dos" "$(echo "$PACE*0.35" | bc):$OUT/work/dosb"; do
    t=$(echo "$(shot_t $i) - ${v%%:*}" | bc); off=0; mkdir -p "${v#*:}"
    for f in "$OUT"/work/cap/*.avi; do
      d=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$f")
      if [ "$(echo "$t < $off + $d" | bc)" = 1 ]; then
        ffmpeg -v error -y -ss "$(echo "$t - $off" | bc)" -i "$f" -frames:v 1 -vf scale=320:200:flags=neighbor \
          "${v#*:}/$(printf %03d $i).png"; break
      fi
      off=$(echo "$off + $d" | bc)
    done
  done
done

# --- port: same keys, each sent once the screen has settled (-steps), frames saved there ---
[ -n "$DOSONLY" ] && exit 0
(cd "$GO" && CGO_ENABLED=0 go build -tags nofakecgo -o "$OUT/work/wiz6" ./cmd/wiz6)
cp -R "$ORIG/" "$OUT/work/godata/"; nosound "$OUT/work/godata"
in=""
# Each key waits at least as long after the previous one as in the original's run, then for
# the screen to settle.
for i in $(seq 0 $((N-1))); do
  d=$PACE; [ $i -gt 0 ] && d=$(( (SLOT[i]-SLOT[i-1])*PACE ))
  in+="k$(scan "${KEYS[$i]}")/$d,"
done
(cd "$OUT/work" && timeout -s KILL $((WAIT + N*10 + 30)) ./wiz6 -data godata -steps "${in%,}" -stepdir "$OUT/go" \
  -stepwait "$WAIT" $([ "$DET" = 1 ] && echo -detrng) >go.log 2>&1) || true

# --- diff ---
printf "step key pixels still-pixels\n" > "$OUT/report.txt"
for i in $(seq 0 $((N-1))); do
  p=$(printf %03d $i)
  [ -f "$OUT/dos/$p.png" ] && [ -f "$OUT/go/$p.png" ] || { echo "$p ${KEYS[$i]} missing" >> "$OUT/report.txt"; continue; }
  n=$(compare -metric AE -fuzz "$FUZZ%" "$OUT/dos/$p.png" "$OUT/go/$p.png" "$OUT/diff/$p.png" 2>&1 >/dev/null || true)
  m=$OUT/work/mask$p.png; s="-"
  if [ -f "$OUT/work/dosb/$p.png" ]; then
    magick "$OUT/dos/$p.png" "$OUT/work/dosb/$p.png" -compose difference -composite -threshold 0 -negate "$m"
    for x in dos go; do magick "$OUT/$x/$p.png" "$m" -compose multiply -composite "$OUT/work/m$x$p.png"; done
    s=$(compare -metric AE -fuzz "$FUZZ%" "$OUT/work/mdos$p.png" "$OUT/work/mgo$p.png" null: 2>&1 || true)
  fi
  echo "$p ${KEYS[$i]} ${n%% *} ${s%% *}" >> "$OUT/report.txt"
done
cat "$OUT/report.txt"
for f in SAVEGAME.DBS PCFILE.DBS; do
  [ -f "$OUT/work/dos/$f" ] && [ -f "$OUT/work/godata/$f" ] && \
    echo "$f: $(cmp -l "$OUT/work/dos/$f" "$OUT/work/godata/$f" 2>&1 | wc -l | tr -d ' ') bytes differ" | tee -a "$OUT/report.txt"
done
