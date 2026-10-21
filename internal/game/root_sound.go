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
	Flags    int   // +11: soundFlagNoInterrupt
}

// soundFlagNoInterrupt in a soundRec's Flags leaves a sound that is playing alone.
const soundFlagNoInterrupt = 2

const (
	soundRecSize = 12      // bytes of a SCENARIO type-9 record
	pitClockHz   = 1193182 // the PC timer's input clock, which Rate divides
	soundVolume  = 127     // host.Voice's full volume
	soundPanMid  = 1 << 15 // host.Voice's centre pan
	loadedSound  = 14      // the slot load_sound fills (cur_sound)
)

// The sound settings (scenario_hdr's SoundMode): which sounds play, and how loud.
const (
	soundAll          = 0
	soundQuietSteps   = 1 // footsteps at half level
	soundNoSteps      = 2 // no footsteps
	soundAllQuiet     = 3 // everything at half level
	soundQuietNoSteps = 4 // no footsteps, the rest at half level
	soundOff          = 5
	firstStepSound    = 10 // sounds 10..12: footsteps and the like (is_sound_10_to_12)
	lastStepSound     = 12
)

var (
	sounds     [loadedSound + 1]soundRec // records of sounds 0..13 and, in loadedSound, of load_sound's sound
	soundSlots [loadedSound + 1][]byte   // their samples (load_sound_slot)
	curSound   int                       // the sound in slot loadedSound (cur_sound)
	voice      = -1                      // the host voice playing
	audioOK    bool
)

func readSoundRec(n int) soundRec {
	b := make([]byte, soundRecSize)
	accessScenarioRecord(dbs.RecSound, n, b, false)
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

// startSound (ROOT 35fd) plays slot at pitClockHz/rate Hz. Level 0 first waits for the
// sound playing to end; soundFlagNoInterrupt in flags leaves a playing sound alone.
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
	if flags&soundFlagNoInterrupt != 0 && soundPlaying() {
		return
	}
	stopSound()
	voice = host.AudioStart(host.Voice{Data: soundSlots[slot], Rate: pitClockHz / rate, Volume: soundVolume, Pan: soundPanMid})
}

// stopSound (ROOT 3640) stops the sound playing.
func stopSound() {
	if dtLog != nil {
		return
	}
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
	step := n >= firstStepSound && n <= lastStepSound
	switch scenarioHdr.SoundMode {
	case soundQuietSteps:
		if step {
			level /= 2
		}
	case soundNoSteps:
		if step {
			return
		}
	case soundAllQuiet:
		level /= 2
	case soundQuietNoSteps:
		if step {
			return
		}
		level /= 2
	case soundOff:
		return
	}
	slot := n
	if r.Size == 0 {
		slot = r.Fallback
	}
	startSound(slot, sounds[slot].Rate, level, sounds[slot].Flags)
}
