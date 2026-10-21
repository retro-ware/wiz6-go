package game

import (
	"encoding/binary"
	"fmt"

	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/host"
)

// soundRec is a sound's SCENARIO type-9 record (sound_table, cs:3344 + 12*n).
type soundRec struct {
	Fallback int   // +0: the slot to play when this sound has no data of its own
	Size     int32 // +4: SOUNDnn.SND's size, 0 for none
	Rate     int   // +8: PIT divisor of the sample rate
	Level    int   // +10: loudness; 0 waits for the sound playing to end
	Flags    int   // +11: bit 1 leaves a sound that is playing alone
}

var (
	sounds     [15]soundRec // records of sounds 0..13 and, in 14, of load_sound's sound
	soundSlots [15][]byte   // their samples (load_sound_slot)
	curSound   int          // the sound in slot 14 (cur_sound)
	voice      = -1         // the host voice playing
	audioOK    bool
)

func readSoundRec(n int) soundRec {
	b := make([]byte, 12)
	accessScenarioRecord(9, n, b, false)
	return soundRec{
		Fallback: int(int16(binary.LittleEndian.Uint16(b))),
		Size:     int32(binary.LittleEndian.Uint32(b[4:])),
		Rate:     le16(b, 8),
		Level:    int(b[10]),
		Flags:    int(b[11]),
	}
}

// loadSoundSlot (ROOT 33e9) reads SOUNDnn.SND for sound n into slot; fatal code if missing.
func loadSoundSlot(n, slot, code int) {
	s, err := dbs.LoadSound(dbs.Path(dataDir, fmt.Sprintf("SOUND%02d.SND", n)))
	if err != nil {
		exitGame(code)
	}
	soundSlots[slot] = s.Data
}

// soundPlaying is whether the host voice is still going.
func soundPlaying() bool {
	for _, h := range host.AudioEnded() {
		if h == voice {
			host.AudioStop(voice)
			voice = -1
		}
	}
	return voice >= 0
}

// startSound (ROOT 35fd) plays slot at 1193182/rate Hz. Level 0 first waits for the sound
// playing to end; flags bit 1 leaves a playing sound alone.
//
// ponytail: the samples go out as 8-bit PCM; the PC-speaker driver (ROOT 1462) maps them
// through a level table to pulse widths, port it when the sound must match.
func startSound(slot, rate, level, flags int) {
	if slot < 0 || slot >= len(soundSlots) || rate == 0 || len(soundSlots[slot]) == 0 || !audioOK {
		return
	}
	if level == 0 {
		for soundPlaying() {
			host.Sleep()
		}
	}
	if flags&2 != 0 && soundPlaying() {
		return
	}
	stopSound()
	voice = host.AudioStart(host.Voice{Data: soundSlots[slot], Rate: 1193182 / rate, Volume: 0x7f, Pan: 0x8000})
}

// stopSound (ROOT 3640) stops the sound playing.
func stopSound() {
	if voice >= 0 {
		host.AudioStop(voice)
		voice = -1
	}
}

// playSound (ROOT 0aaa) plays sound n as the sound setting allows.
func playSound(n int) {
	if dtLog != nil {
		dtLog("snd", n)
		return
	}
	r := sounds[n]
	level := r.Level
	step := n >= 10 && n <= 12 // is_sound_10_to_12: footsteps and the like
	switch hdr.SoundMode {
	case 1:
		if step {
			level /= 2
		}
	case 2:
		if step {
			return
		}
	case 3:
		level /= 2
	case 4:
		if step {
			return
		}
		level /= 2
	case 5:
		return
	}
	slot := n
	if r.Size == 0 {
		slot = r.Fallback
	}
	startSound(slot, sounds[slot].Rate, level, sounds[slot].Flags)
}
