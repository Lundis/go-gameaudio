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

type Sound struct {
	data      []float32
	channelId ChannelId
	volume    float32
	m         sync.Mutex
}

func (s *Sound) Play() *PlayingSound {
	return getFreePlayingSound(s)
}

// PlayLoop starts playing this sound in an infinite loop.
// If the sound is already playing, it will not reset it.
// If it's playing multiple instances right now, this will cause all of them to loop.
func (s *Sound) PlayLoop(crossFade time.Duration) *PlayingSound {
	fadeDuration := int(float64(mux.channelCount*mux.sampleRate) * crossFade.Seconds())
	ps := getFreePlayingSound(s)
	if ps != nil {
		ps.loop = true
		ps.fadeInEndsAt = fadeDuration
		ps.fadeOutStartsAt = len(s.data) - fadeDuration
	}
	return ps
}

func (s *Sound) PlayFadeIn(fadeIn time.Duration) *PlayingSound {

	fadeDuration := int(float64(mux.channelCount*mux.sampleRate) * fadeIn.Seconds())
	ps := getFreePlayingSound(s)
	if ps != nil {
		ps.fadeInEndsAt = fadeDuration
	}
	return ps
}
