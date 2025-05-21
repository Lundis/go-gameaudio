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

package oto

import (
	"sync"
	"time"
)

type playInstance struct {
	pos             int
	fadeInEndsAt    int
	fadeOutStartsAt int
}

type Player struct {
	mux           *Mux
	data          []float32
	playInstances []*playInstance
	channelId     ChannelId
	volume        float32
	m             sync.Mutex
	throttlingMs  int
	loop          bool
	loopedOnce    bool
}

func (m *Mux) NewPlayer(data []float32, volume float32, channel ChannelId) *Player {
	pl := &Player{
		mux:          m,
		data:         data,
		volume:       volume,
		channelId:    channel,
		throttlingMs: 50,
	}
	return pl
}

func (p *Player) Play() {
	p.m.Lock()
	p.playImpl(0, len(p.data))
	p.m.Unlock()
}

func (p *Player) Stop() {
	p.m.Lock()
	p.loop = false
	p.playInstances = p.playInstances[:0]
	p.m.Unlock()
}

func (p *Player) PlayLoop(crossFade time.Duration) {
	if p.loop {
		return
	}
	p.m.Lock()
	p.loop = true
	fadeDuration := int(float64(p.mux.channelCount*p.mux.sampleRate) * crossFade.Seconds())
	p.playImpl(fadeDuration, len(p.data)-fadeDuration)
	p.m.Unlock()
}

func (p *Player) PlayFadeIn(fadeIn time.Duration) {
	p.m.Lock()
	p.playImpl(int(float64(p.mux.channelCount*p.mux.sampleRate)*fadeIn.Seconds()), len(p.data))
	p.m.Unlock()
}

func (p *Player) SetThrottlingMs(ms int) {
	p.m.Lock()
	p.throttlingMs = ms
	p.m.Unlock()
}

func (p *Player) playImpl(fadeInEndsAt int, fadeOutStartsAt int) {
	// re-use an existing play slot if possible
	var freeInstance *playInstance
	for _, pi := range p.playInstances {
		if pi.pos < p.mux.sampleRate*p.mux.channelCount*p.throttlingMs/1000 && !p.loop {
			// don't start playing again until throttlingMs has passed
			return
		}
		if pi.pos >= len(p.data) || p.loop {
			freeInstance = pi
			break
		}
	}
	if freeInstance == nil {
		freeInstance = &playInstance{}
		p.playInstances = append(p.playInstances, freeInstance)
	}
	// when looping, don't reset the currently playing instance
	if !p.loop {
		freeInstance.pos = 0
	}
	freeInstance.fadeInEndsAt = fadeInEndsAt
	freeInstance.fadeOutStartsAt = fadeOutStartsAt

	p.mux.addPlayer(p)
}

func (p *Player) Reset() {
	p.m.Lock()
	p.playInstances = p.playInstances[:0]
	p.m.Unlock()
}

func (p *Player) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()

	for _, i := range p.playInstances {
		if i.pos < len(p.data) {
			return true
		}
	}
	return false
}

func (p *Player) readBufferAndAdd(buf []float32) {
	channelSettings := getChannelSettings(p.channelId)
	if channelSettings.paused {
		return
	}

	p.m.Lock()

	volumeMultiplier := p.volume * channelSettings.volume
	finishedPlaying := true
	for _, playInstance := range p.playInstances {
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
		p.mux.removePlayer(p)
	}

	p.m.Unlock()
}
