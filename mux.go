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
)

// Mux is a low-level multiplexer of audio sounds.
type Mux struct {
	sampleRate   int
	channelCount int

	sounds map[*Sound]struct{}
	cond   *sync.Cond
}

// NewMux creates a new Mux.
func NewMux(sampleRate int, channelCount int) *Mux {
	m := &Mux{
		sampleRate:   sampleRate,
		channelCount: channelCount,
		cond:         sync.NewCond(&sync.Mutex{}),
	}
	return m
}

func (m *Mux) addSound(sound *Sound) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	if m.sounds == nil {
		m.sounds = map[*Sound]struct{}{}
	}
	m.sounds[sound] = struct{}{}
	m.cond.Signal()
}

func (m *Mux) removeSound(sound *Sound) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	delete(m.sounds, sound)
	m.cond.Signal()
}

// ReadFloat32s fills buf with the multiplexed data of the sounds as float32 values.
func (m *Mux) ReadFloat32s(buf []float32) {
	m.cond.L.Lock()
	sounds := make([]*Sound, 0, len(m.sounds))
	for p := range m.sounds {
		sounds = append(sounds, p)
	}
	m.cond.L.Unlock()

	clear(buf)
	for _, p := range sounds {
		p.readBufferAndAdd(buf)
	}
	m.cond.Signal()
}
