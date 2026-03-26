// Copyright 2021 The Oto Authors
// Copyright 2025 Lundis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audio

import (
	"log"
	"sync"
)

// Mux is a low-level multiplexer of audio sounds.
type Mux struct {
	sampleRate   int
	channelCount int

	sounds []*PlayingSound
}

var mux *Mux

const soundPoolSize = 500

var soundPool = make([]PlayingSound, soundPoolSize)
var nextFreeSoundPoolIndex = 0
var soundPoolLock sync.Mutex

func getFreePlayingSound(s *Sound) (ps *PlayingSound) {
	soundPoolLock.Lock()
	if nextFreeSoundPoolIndex < soundPoolSize {
		ps = &soundPool[nextFreeSoundPoolIndex]
		nextFreeSoundPoolIndex++
		ps.sound = s
		ps.pos = 0
		ps.endAt = len(s.data)
		ps.fadeInEndsAt = 0
		ps.fadeOutStartsAt = len(s.data)
		ps.seekTo = -1
		ps.loop = false
		ps.loopedOnce = false
	} else {
		log.Println("WARNING: sound pool is full. Throttle your SFX!")
	}
	soundPoolLock.Unlock()
	return ps
}

func initMux(sampleRate int, channelCount int) {
	mux = &Mux{
		sampleRate:   sampleRate,
		channelCount: channelCount,
	}
}

// ReadFloat32s fills buf with the multiplexed data of the sounds as float32 values.
func (m *Mux) ReadFloat32s(buf []float32) {
	soundPoolLock.Lock()
	i := 0
	// doing cleanup outside the play loop to avoid lock sync overhead in the loop
	for i < nextFreeSoundPoolIndex {
		ps := &soundPool[i]
		if ps.pos >= ps.endAt {
			soundPool[i] = soundPool[nextFreeSoundPoolIndex]
			nextFreeSoundPoolIndex--
		} else {
			i++
		}

	}
	currentLength := nextFreeSoundPoolIndex
	soundPoolLock.Unlock()

	clear(buf)
	for i := 0; i < currentLength; i++ {
		ps := &soundPool[i]
		if ps.seekTo >= 0 {
			ps.Seek(ps.seekTo)
			ps.pos = int(ps.seekTo * float32(ps.endAt))
			// align to frame
			ps.pos = ps.pos - ps.pos%(mux.channelCount)
			ps.seekTo = -1
		}
		ps.readBufferAndAdd(buf)
		if ps.pos >= ps.endAt && ps.onEndCallback != nil {
			ps.onEndCallback()
			ps.onEndCallback = nil
		}
	}

}

func (ps *PlayingSound) readBufferAndAdd(buf []float32) {
	channelSettings := getChannelSettings(ps.sound.channelId)
	if channelSettings.paused {
		return
	}

	volumeMultiplier := ps.sound.volume * channelSettings.volume
	available := ps.endAt - ps.pos
	if ps.loop {
		available = len(buf)
	}
	n := min(len(buf), available)
	loopAdjustment := 0
	for i := 0; i < n; i++ {
		di := (ps.pos + i + loopAdjustment) % ps.endAt
		v := ps.sound.data[di] * volumeMultiplier
		fadeInMultiplier := float32(1)
		fadeOutMultiplier := float32(1)
		if ps.loop && di == ps.fadeOutStartsAt {
			// crossfade: seek to start
			loopAdjustment += ps.endAt - ps.fadeOutStartsAt
			di = (ps.pos + i + loopAdjustment) % ps.endAt
			v = ps.sound.data[di] * volumeMultiplier
			ps.loopedOnce = true
		} else if di > ps.fadeOutStartsAt && ps.fadeOutStartsAt < ps.endAt {
			fadeoutLength := ps.endAt - ps.fadeOutStartsAt
			fadeOutMultiplier = 1 - float32(di-ps.fadeOutStartsAt)/float32(fadeoutLength)
		}
		if di < ps.fadeInEndsAt {
			fadeInMultiplier = float32(di) / float32(ps.fadeInEndsAt)
			if ps.loopedOnce {
				// crossfade: mix in the "fadeout" value
				endValue := ps.sound.data[di+ps.fadeOutStartsAt] * volumeMultiplier
				buf[i] += v*fadeInMultiplier + (1-fadeInMultiplier)*endValue

				continue
			}
		}
		buf[i] += v * fadeInMultiplier * fadeOutMultiplier
	}
	ps.pos += n
	if ps.loop {
		ps.pos %= ps.endAt
	}
}
