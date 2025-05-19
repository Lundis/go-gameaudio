// Copyright 2021 The Oto Authors
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

// Package mux offers APIs for a low-level multiplexer of audio players.
// Usually you don't have to use this package directly.
package oto

import (
	"sync"
	"time"
)

// Mux is a low-level multiplexer of audio players.
type Mux struct {
	sampleRate   int
	channelCount int

	players map[*Player]struct{}
	cond    *sync.Cond
}

// NewMux creates a new Mux.
func NewMux(sampleRate int, channelCount int) *Mux {
	m := &Mux{
		sampleRate:   sampleRate,
		channelCount: channelCount,
		cond:         sync.NewCond(&sync.Mutex{}),
	}
	go m.loop()
	return m
}

func (m *Mux) shouldWait() bool {
	for p := range m.players {
		if p.canReadSourceToBuffer() {
			return false
		}
	}
	return true
}

func (m *Mux) wait() {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	for m.shouldWait() {
		m.cond.Wait()
	}
}

func (m *Mux) loop() {
	var players []*Player
	for {
		m.wait()

		m.cond.L.Lock()
		for i := range players {
			players[i] = nil
		}
		players = players[:0]
		for p := range m.players {
			players = append(players, p)
		}
		m.cond.L.Unlock()

		allZero := true
		for _, p := range players {
			n := p.readSourceToBuffer()
			if n != 0 {
				allZero = false
			}
		}

		// Sleeping is necessary especially on browsers.
		// Sometimes a player continues to read 0 bytes from the source and this loop can be a busy loop in such case.
		if allZero {
			time.Sleep(time.Millisecond)
		}
	}
}

func (m *Mux) addPlayer(player *Player) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	if m.players == nil {
		m.players = map[*Player]struct{}{}
	}
	m.players[player] = struct{}{}
	m.cond.Signal()
}

func (m *Mux) removePlayer(player *Player) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	delete(m.players, player)
	m.cond.Signal()
}

// ReadFloat32s fills buf with the multiplexed data of the players as float32 values.
func (m *Mux) ReadFloat32s(buf []float32) {
	m.cond.L.Lock()
	players := make([]*Player, 0, len(m.players))
	for p := range m.players {
		players = append(players, p)
	}
	m.cond.L.Unlock()

	for i := range buf {
		buf[i] = 0
	}
	for _, p := range players {
		p.readBufferAndAdd(buf)
	}
	m.cond.Signal()
}

// TODO: The term 'buffer' is confusing. Name each buffer with good terms.

// defaultBufferSize returns the default size of the buffer for the audio source.
// This buffer is used when unreading on pausing the player.
func (m *Mux) defaultBufferSize() int {
	s := m.sampleRate * m.channelCount / 2 // 0.5[s]
	return s
}
