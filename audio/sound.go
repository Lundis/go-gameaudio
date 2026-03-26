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
		data:      data,
		volume:    volume,
		channelId: channel,
	}
	return pl
}

// OnEndCallback can be used to register a callback that will be called once when the sound has finished playing
func (p *Sound) OnEndCallback(onEndCallback func()) {
	p.m.Lock()
	p.onEndCallback = onEndCallback
	p.m.Unlock()
}

type Sound struct {
	data          []float32
	channelId     ChannelId
	volume        float32
	m             sync.Mutex
	onEndCallback func()
}

type PlayingSound struct {
	sound           *Sound
	loop            bool
	loopedOnce      bool
	pos             int
	seekTo          float32
	fadeInEndsAt    int
	fadeOutStartsAt int
	endAt           int
}

var playRequests = make(chan *PlayingSound, soundPoolSize)

func (p *Sound) Play() *PlayingSound {
	// TODO avoid allocations here
	ps := getFreePlayingSound(p)

	case playRequests <- playingSound:

	return playingSound
}

// Seek a playing sound to a given percentage
// if the sound already finished, this will do nothing
func (p *PlayingSound) Seek(percentage float32) {
	p.seekTo = percentage
}

// Seconds returns the current position and total length of the first currently playing instance of this sound
func (p *PlayingSound) Seconds() (current, total float32) {
	samplesPerSecond := float32(mux.channelCount * mux.sampleRate)
	current = float32(p.pos) / samplesPerSecond
	total = float32(len(p.sound.data)) / samplesPerSecond
	return
}

// PlayLoop starts playing this sound in an infinite loop.
// If the sound is already playing, it will not reset it.
// If it's playing multiple instances right now, this will cause all of them to loop.
func (p *Sound) PlayLoop(crossFade time.Duration) *PlayingSound {
	fadeDuration := int(float64(mux.channelCount*mux.sampleRate) * crossFade.Seconds())
	ps := &PlayingSound{
		sound:           p,
		loop:            true,
		fadeInEndsAt:    fadeDuration,
		fadeOutStartsAt: len(p.data) - fadeDuration,
		endAt:           len(p.data),
	}
	select {
	case playRequests <- ps:
	default:
		return nil
	}
	return ps
}

func (p *Sound) PlayFadeIn(fadeIn time.Duration) *PlayingSound {
	ps := &PlayingSound{
		sound:           p,
		fadeInEndsAt:    min(len(p.data), int(float64(mux.channelCount*mux.sampleRate)*fadeIn.Seconds())),
		fadeOutStartsAt: len(p.data),
		endAt:           len(p.data),
	}
	select {
	case playRequests <- ps:
	default:
		return nil
	}
	return ps
}

func (p *PlayingSound) Stop() {
	p.endAt = p.pos
	p.loop = false
}

func (p *PlayingSound) StopFadeOut(fadeIn time.Duration) {
	p.endAt = min(p.endAt, p.pos+int(float64(mux.channelCount*mux.sampleRate)*fadeIn.Seconds()))
	p.fadeOutStartsAt = p.pos
}

func (p *Sound) playImpl(fadeInEndsAt int, fadeOutStartsAt int) {
	// re-use an existing play slot if possible
	var freeInstance *PlayingSound
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
		freeInstance = &PlayingSound{}
		p.players = append(p.players, freeInstance)
	}
	// when not looping, reset the currently playing instance
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
	for _, i := range p.players {
		if i.pos < len(p.data) {
			return true
		}
	}
	return false
}

func (p *PlayingSound) readBufferAndAdd(buf []float32) {
	channelSettings := getChannelSettings(p.sound.channelId)
	if channelSettings.paused {
		return
	}

	volumeMultiplier := p.sound.volume * channelSettings.volume
	finishedPlaying := true
	available := p.endAt - p.pos
	if p.loop {
		available = len(buf)
	}
	n := min(len(buf), available)
	loopAdjustment := 0
	for i := 0; i < n; i++ {
		di := (p.pos + i + loopAdjustment) % p.endAt
		v := p.sound.data[di] * volumeMultiplier
		fadeInMultiplier := float32(1)
		fadeOutMultiplier := float32(1)
		if p.loop && di == p.fadeOutStartsAt {
			// crossfade: seek to start
			loopAdjustment += p.endAt - p.fadeOutStartsAt
			di = (p.pos + i + loopAdjustment) % p.endAt
			v = p.sound.data[di] * volumeMultiplier
			p.loopedOnce = true
		} else if di > p.fadeOutStartsAt && p.fadeOutStartsAt < p.endAt {
			fadeoutLength := p.endAt - p.fadeOutStartsAt
			fadeOutMultiplier = 1 - float32(di-p.fadeOutStartsAt)/float32(fadeoutLength)
		}
		if di < p.fadeInEndsAt {
			fadeInMultiplier = float32(di) / float32(p.fadeInEndsAt)
			if p.loopedOnce {
				// crossfade: mix in the "fadeout" value
				endValue := p.sound.data[di+p.fadeOutStartsAt] * volumeMultiplier
				buf[i] += v*fadeInMultiplier + (1-fadeInMultiplier)*endValue

				continue
			}
		}
		buf[i] += v * fadeInMultiplier * fadeOutMultiplier
	}
	p.pos += n
	if p.loop {
		p.pos %= p.endAt
	}
	if p.pos < p.endAt {
		finishedPlaying = false
	}
	if finishedPlaying {
		mux.removeSound(p)
		if p.onEndCallback != nil {
			p.onEndCallback()

			p.m.Lock()
			p.onEndCallback = nil
			p.m.Unlock()
		}
	}
}
