#!/bin/bash
# parity.sh SCRIPT OUTDIR -- run Wizardry VI in DOSBox-X and the Go port with the same keys,
# capture one frame per step from each and diff them. See README.md.
set -e
SCRIPT=$(cd "$(dirname "$1")" && pwd)/$(basename "$1"); OUT=$2
HERE=$(cd "$(dirname "$0")" && pwd); GO=$(cd "$HERE/../.." && pwd)
ORIG=${ORIG:-$HOME/Projects/wiz6-re/gog/x/app}
WAIT=8; PACE=3; FUZZ=${FUZZ:-0}
KEYS=()
while read -r a b; do
  case "$a" in ''|'#'*) ;; WAIT|PACE) eval "$a=$b" ;; *) KEYS+=("$a") ;; esac
done < "$SCRIPT"
N=${#KEYS[@]}
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
# Step i's key goes in at WAIT+i*PACE; its frame is taken just before key i+1 (step N: one PACE after).
shot_t() { echo "$WAIT + ($1+1)*$PACE - 0.3" | bc; }

# --- original: AUTOTYPE feeds keys inside DOSBox-X, DX-CAPTURE records a ZMBV AVI ---
cp -R "$ORIG/" "$OUT/work/dos/"
LIMIT=$(echo "$WAIT + ($N+1)*$PACE + 2" | bc | cut -d. -f1)
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
autotype -w $WAIT -p $PACE ${KEYS[*]}
dx-capture /V wroot.exe
exit
C
SDL_VIDEODRIVER=dummy timeout -s KILL $((LIMIT+30)) dosbox-x -conf "$OUT/work/dos.conf" \
  -nogui -nomenu -exit -fastlaunch -time-limit $LIMIT >"$OUT/work/dos.log" 2>&1 || true
# The capture splits into one AVI per video mode; walk them to map program time to file time.
for i in $(seq 0 $((N-1))); do
  t=$(shot_t $i); off=0
  for f in "$OUT"/work/cap/*.avi; do
    d=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$f")
    if [ "$(echo "$t < $off + $d" | bc)" = 1 ]; then
      ffmpeg -v error -y -ss "$(echo "$t - $off" | bc)" -i "$f" -frames:v 1 -vf scale=320:200:flags=neighbor \
        "$OUT/dos/$(printf %03d $i).png"; break
    fi
    off=$(echo "$off + $d" | bc)
  done
done

# --- port: same keys and times through -input / -shot ---
(cd "$GO" && CGO_ENABLED=0 go build -tags nofakecgo -o "$OUT/work/wiz6" ./cmd/wiz6)
cp -R "$ORIG/" "$OUT/work/godata/"
in=""; sh=""
for i in $(seq 0 $((N-1))); do
  in+="$(echo "$WAIT + $i*$PACE" | bc):k$(scan "${KEYS[$i]}"),"
  sh+="$(shot_t $i):$OUT/go/$(printf %03d $i).png,"
done
(cd "$OUT/work" && timeout -s KILL $((LIMIT+10)) ./wiz6 -data godata -input "${in%,}" -shot "${sh%,}" \
  >go.log 2>&1) || true

# --- diff ---
printf "step key pixels\n" > "$OUT/report.txt"
for i in $(seq 0 $((N-1))); do
  p=$(printf %03d $i)
  [ -f "$OUT/dos/$p.png" ] && [ -f "$OUT/go/$p.png" ] || { echo "$p ${KEYS[$i]} missing" >> "$OUT/report.txt"; continue; }
  n=$(compare -metric AE -fuzz "$FUZZ%" "$OUT/dos/$p.png" "$OUT/go/$p.png" "$OUT/diff/$p.png" 2>&1 >/dev/null || true)
  echo "$p ${KEYS[$i]} $n" >> "$OUT/report.txt"
done
cat "$OUT/report.txt"
