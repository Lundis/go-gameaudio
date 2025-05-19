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
	"errors"
	"io"
	"runtime"
	"sync"
	"time"
)

// Mux is a low-level multiplexer of audio players.
type Mux struct {
	sampleRate   int
	channelCount int

	players map[*playerImpl]struct{}
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
	var players []*playerImpl
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

func (m *Mux) addPlayer(player *playerImpl) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	if m.players == nil {
		m.players = map[*playerImpl]struct{}{}
	}
	m.players[player] = struct{}{}
	m.cond.Signal()
}

func (m *Mux) removePlayer(player *playerImpl) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	delete(m.players, player)
	m.cond.Signal()
}

// ReadFloat32s fills buf with the multiplexed data of the players as float32 values.
func (m *Mux) ReadFloat32s(buf []float32) {
	m.cond.L.Lock()
	players := make([]*playerImpl, 0, len(m.players))
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

type MuxPlayer struct {
	p *playerImpl
}

type playerState int

const (
	playerPaused playerState = iota
	playerPlay
	playerClosed
)

type playerImpl struct {
	mux        *Mux
	src        AudioStream
	prevVolume float64
	volume     float64
	err        error
	state      playerState
	bufPool    *sync.Pool
	buf        []float32
	eof        bool
	bufferSize int

	m sync.Mutex
}

func (m *Mux) NewPlayer(src AudioStream) *MuxPlayer {
	pl := &MuxPlayer{
		p: &playerImpl{
			mux:        m,
			src:        src,
			prevVolume: 1,
			volume:     1,
			bufferSize: m.defaultBufferSize(),
		},
	}
	runtime.SetFinalizer(pl, (*MuxPlayer).Close)
	return pl
}

func (p *MuxPlayer) Err() error {
	return p.p.Err()
}

func (p *playerImpl) Err() error {
	p.m.Lock()
	defer p.m.Unlock()

	return p.err
}

func (p *MuxPlayer) Play() {
	p.p.Play()
}

func (p *playerImpl) Play() {
	// Goroutines don't work effiently on Windows. Avoid using them (hajimehoshi/ebiten#1768).
	if runtime.GOOS == "windows" {
		p.m.Lock()
		defer p.m.Unlock()

		p.playImpl()
	} else {
		ch := make(chan struct{})
		go func() {
			p.m.Lock()
			defer p.m.Unlock()

			close(ch)
			p.playImpl()
		}()
		<-ch
	}
}

func (p *MuxPlayer) SetBufferSize(bufferSize int) {
	p.p.setBufferSize(bufferSize)
}

func (p *playerImpl) setBufferSize(bufferSize int) {
	p.m.Lock()
	defer p.m.Unlock()

	orig := p.bufferSize
	p.bufferSize = bufferSize
	if bufferSize == 0 {
		p.bufferSize = p.mux.defaultBufferSize()
	}
	if orig != p.bufferSize {
		p.bufPool = nil
	}
}

func (p *playerImpl) getTmpBuf() ([]float32, func()) {
	// The returned buffer could be accessed regardless of the mutex m (#254).
	// In order to avoid races, use a sync.Pool.
	// On the other hand, the calls of getTmpBuf itself should be protected by the mutex m,
	// then accessing p.bufPool doesn't cause races.
	if p.bufPool == nil {
		p.bufPool = &sync.Pool{
			New: func() interface{} {
				buf := make([]float32, p.bufferSize)
				return &buf
			},
		}
	}
	buf := p.bufPool.Get().(*[]float32)
	return *buf, func() {
		// p.bufPool could be nil when setBufferSize is called (#258).
		// buf doesn't have to (or cannot) be put back to the pool, as the size of the buffer could be changed.
		if p.bufPool == nil {
			return
		}
		if len(*buf) != p.bufferSize {
			return
		}
		p.bufPool.Put(buf)
	}
}

// read reads the source to buf.
// read unlocks the mutex temporarily and locks when reading finishes.
// This avoids locking during an external function call Read (#188).
//
// When read is called, the mutex m must be locked.
func (p *playerImpl) read(buf []float32) (int, error) {
	p.m.Unlock()
	defer p.m.Lock()
	return p.src.Read(buf)
}

// addToPlayers adds p to the players set.
//
// When addToPlayers is called, the mutex m must be locked.
func (p *playerImpl) addToPlayers() {
	p.m.Unlock()
	defer p.m.Lock()
	p.mux.addPlayer(p)
}

// removeFromPlayers removes p from the players set.
//
// When removeFromPlayers is called, the mutex m must be locked.
func (p *playerImpl) removeFromPlayers() {
	p.m.Unlock()
	defer p.m.Lock()
	p.mux.removePlayer(p)
}

func (p *playerImpl) playImpl() {
	if p.err != nil {
		return
	}
	if p.state != playerPaused {
		return
	}
	p.state = playerPlay

	if !p.eof {
		buf, free := p.getTmpBuf()
		defer free()
		for len(p.buf) < p.bufferSize {
			n, err := p.read(buf)
			if err != nil && err != io.EOF {
				p.setErrorImpl(err)
				return
			}
			p.buf = append(p.buf, buf[:n]...)
			if err == io.EOF {
				p.eof = true
				break
			}
		}
	}

	if p.eof && len(p.buf) == 0 {
		p.state = playerPaused
	}

	p.addToPlayers()
}

func (p *MuxPlayer) Pause() {
	p.p.Pause()
}

func (p *playerImpl) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state != playerPlay {
		return
	}
	p.state = playerPaused
}

func (p *MuxPlayer) Seek(offset int64, whence int) (int64, error) {
	return p.p.Seek(offset, whence)
}

func (p *playerImpl) Seek(offset int64, whence int) (int64, error) {
	p.m.Lock()
	defer p.m.Unlock()

	// If a player is playing, keep playing even after this seeking.
	if p.state == playerPlay {
		defer p.playImpl()
	}

	// Reset the internal buffer.
	p.resetImpl()

	// Check if the source implements io.Seeker.
	s, ok := p.src.(io.Seeker)
	if !ok {
		return 0, errors.New("mux: the source must implement io.Seeker")
	}
	return s.Seek(offset, whence)
}

func (p *MuxPlayer) Reset() {
	p.p.Reset()
}

func (p *playerImpl) Reset() {
	p.m.Lock()
	defer p.m.Unlock()
	p.resetImpl()
}

func (p *playerImpl) resetImpl() {
	if p.state == playerClosed {
		return
	}
	p.state = playerPaused
	p.buf = p.buf[:0]
	p.eof = false
}

func (p *MuxPlayer) IsPlaying() bool {
	return p.p.IsPlaying()
}

func (p *playerImpl) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.state == playerPlay
}

func (p *MuxPlayer) Volume() float64 {
	return p.p.Volume()
}

func (p *playerImpl) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()
	return p.volume
}

func (p *MuxPlayer) SetVolume(volume float64) {
	p.p.SetVolume(volume)
}

func (p *playerImpl) SetVolume(volume float64) {
}

func (p *MuxPlayer) BufferedSize() int {
	return p.p.BufferedSize()
}

func (p *playerImpl) BufferedSize() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.buf)
}

func (p *MuxPlayer) Close() error {
	runtime.SetFinalizer(p, nil)
	return p.p.Close()
}

func (p *playerImpl) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	return p.closeImpl()
}

func (p *playerImpl) closeImpl() error {
	p.removeFromPlayers()

	if p.state == playerClosed {
		return p.err
	}
	p.state = playerClosed
	p.buf = nil
	return p.err
}

func (p *playerImpl) readBufferAndAdd(buf []float32) int {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state != playerPlay {
		return 0
	}

	n := len(p.buf)
	if n > len(buf) {
		n = len(buf)
	}

	prevVolume := float32(p.prevVolume)
	volume := float32(p.volume)

	src := p.buf[:n]

	for i := 0; i < n; i++ {
		var v = src[i]
		if volume == prevVolume {
			buf[i] += v * volume
		} else {
			rate := float32(i) / float32(n)
			if rate > 1 {
				rate = 1
			}
			buf[i] += v * (volume*rate + prevVolume*(1-rate))
		}
	}

	p.prevVolume = p.volume

	copy(p.buf, p.buf[n:])
	p.buf = p.buf[:len(p.buf)-n]

	if p.eof && len(p.buf) == 0 {
		p.state = playerPaused
	}

	return n
}

func (p *playerImpl) canReadSourceToBuffer() bool {
	p.m.Lock()
	defer p.m.Unlock()

	if p.eof {
		return false
	}
	return len(p.buf) < p.bufferSize
}

func (p *playerImpl) readSourceToBuffer() int {
	p.m.Lock()
	defer p.m.Unlock()

	if p.err != nil {
		return 0
	}
	if p.state == playerClosed {
		return 0
	}

	if len(p.buf) >= p.bufferSize {
		return 0
	}

	buf, free := p.getTmpBuf()
	defer free()
	n, err := p.read(buf)

	if err != nil && err != io.EOF {
		p.setErrorImpl(err)
		return 0
	}

	p.buf = append(p.buf, buf[:n]...)
	if err == io.EOF {
		p.eof = true
		if len(p.buf) == 0 {
			p.state = playerPaused
		}
	}
	return n
}

func (p *playerImpl) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}

// TODO: The term 'buffer' is confusing. Name each buffer with good terms.

// defaultBufferSize returns the default size of the buffer for the audio source.
// This buffer is used when unreading on pausing the player.
func (m *Mux) defaultBufferSize() int {
	s := m.sampleRate * m.channelCount / 2 // 0.5[s]
	return s
}
