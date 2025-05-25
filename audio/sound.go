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
	"sync"
	"time"
)

// NewSound creates a new, ready-to-use Sound belonging to the Context.
// It is safe to create multiple sounds.
//
//	[data]      = [sample 1] [sample 2] [sample 3] ...
//	[sample *]  = [channel 1] [channel 2] ...
//	[channel *] = [float32]
//
// NewSound is concurrent-safe.
//
// All the functions of a Sound returned by NewSound are concurrent-safe.
func NewSound(data []float32, volume float32, channel ChannelId) *Sound {
	if mux == nil {
		return nil
	}
	pl := &Sound{
		data:         data,
		volume:       volume,
		channelId:    channel,
		throttlingMs: 50,
	}
	return pl
}

type Sound struct {
	data         []float32
	players      []*player
	channelId    ChannelId
	volume       float32
	m            sync.Mutex
	throttlingMs int
	loop         bool
	loopedOnce   bool
}

type player struct {
	pos             int
	fadeInEndsAt    int
	fadeOutStartsAt int
}

func (p *Sound) Play() {
	p.m.Lock()
	p.playImpl(0, len(p.data))
	p.m.Unlock()
}

func (p *Sound) Stop() {
	p.m.Lock()
	p.loop = false
	p.players = p.players[:0]
	p.m.Unlock()
}

// PlayLoop starts playing this sound in an infinite loop.
// If the sound is already playing, it will not reset it.
// If it's playing multiple instances right now, this will cause all of them to loop.
func (p *Sound) PlayLoop(crossFade time.Duration) {
	if p.loop {
		return
	}
	p.m.Lock()
	p.loop = true
	fadeDuration := int(float64(mux.channelCount*mux.sampleRate) * crossFade.Seconds())
	p.playImpl(fadeDuration, len(p.data)-fadeDuration)
	p.m.Unlock()
}

func (p *Sound) PlayFadeIn(fadeIn time.Duration) {
	p.m.Lock()
	p.playImpl(int(float64(mux.channelCount*mux.sampleRate)*fadeIn.Seconds()), len(p.data))
	p.m.Unlock()
}

func (p *Sound) SetThrottlingMs(ms int) {
	p.m.Lock()
	p.throttlingMs = ms
	p.m.Unlock()
}

func (p *Sound) playImpl(fadeInEndsAt int, fadeOutStartsAt int) {
	// re-use an existing play slot if possible
	var freeInstance *player
	for _, pi := range p.players {
		if pi.pos < mux.sampleRate*mux.channelCount*p.throttlingMs/1000 && !p.loop {
			// don't start playing again until throttlingMs has passed
			return
		}
		if pi.pos >= len(p.data) || p.loop {
			freeInstance = pi
			break
		}
	}
	if freeInstance == nil {
		freeInstance = &player{}
		p.players = append(p.players, freeInstance)
	}
	// when looping, don't reset the currently playing instance
	if !p.loop {
		freeInstance.pos = 0
	}
	freeInstance.fadeInEndsAt = fadeInEndsAt
	freeInstance.fadeOutStartsAt = fadeOutStartsAt

	mux.addSound(p)
}

func (p *Sound) Reset() {
	p.m.Lock()
	p.players = p.players[:0]
	p.m.Unlock()
}

func (p *Sound) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()

	for _, i := range p.players {
		if i.pos < len(p.data) {
			return true
		}
	}
	return false
}

func (p *Sound) readBufferAndAdd(buf []float32) {
	channelSettings := getChannelSettings(p.channelId)
	if channelSettings.paused {
		return
	}

	p.m.Lock()

	volumeMultiplier := p.volume * channelSettings.volume
	finishedPlaying := true
	for _, playInstance := range p.players {
		available := len(p.data) - playInstance.pos
		if p.loop {
			available = len(buf)
		}
		n := min(len(buf), available)
		loopAdjustment := 0
		for i := 0; i < n; i++ {
			di := (playInstance.pos + i + loopAdjustment) % len(p.data)
			v := p.data[di] * volumeMultiplier
			fadeInMultiplier := float32(1)
			fadeOutMultiplier := float32(1)
			if p.loop && di == playInstance.fadeOutStartsAt {
				// crossfade: seek to start
				loopAdjustment += len(p.data) - playInstance.fadeOutStartsAt
				di = (playInstance.pos + i + loopAdjustment) % len(p.data)
				v = p.data[di] * volumeMultiplier
				p.loopedOnce = true
			} else if di > playInstance.fadeOutStartsAt && playInstance.fadeOutStartsAt < len(p.data) {
				fadeoutLength := len(p.data) - playInstance.fadeOutStartsAt
				fadeOutMultiplier = 1 - float32(di-playInstance.fadeOutStartsAt)/float32(fadeoutLength)
			}
			if di < playInstance.fadeInEndsAt {
				fadeInMultiplier = float32(di) / float32(playInstance.fadeInEndsAt)
				if p.loopedOnce {
					// crossfade: mix in the "fadeout" value
					endValue := p.data[di+playInstance.fadeOutStartsAt] * volumeMultiplier
					buf[i] += v*fadeInMultiplier + (1-fadeInMultiplier)*endValue

					continue
				}
			}
			buf[i] += v * fadeInMultiplier * fadeOutMultiplier
		}
		playInstance.pos += n
		if p.loop {
			playInstance.pos %= len(p.data)
		}
		if playInstance.pos < len(p.data) {
			finishedPlaying = false
		}
	}
	if finishedPlaying {
		mux.removeSound(p)
	}

	p.m.Unlock()
}
