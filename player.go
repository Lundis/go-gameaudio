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

package oto

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
)

type Player struct {
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

type playerState int

const (
	playerPaused playerState = iota
	playerPlay
	playerClosed
)

func (m *Mux) NewPlayer(src AudioStream) *Player {
	pl := &Player{
		mux:        m,
		src:        src,
		prevVolume: 1,
		volume:     1,
		bufferSize: m.defaultBufferSize(),
	}
	runtime.SetFinalizer(pl, (*Player).Close)
	return pl
}

func (p *Player) Err() error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.err != nil {
		return fmt.Errorf("oto: audio error: %w", p.err)
	}
	return nil
}

func (p *Player) Play() {
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

func (p *Player) SetBufferSize(bufferSize int) {
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

func (p *Player) getTmpBuf() ([]float32, func()) {
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
func (p *Player) read(buf []float32) (int, error) {
	p.m.Unlock()
	defer p.m.Lock()
	return p.src.Read(buf)
}

// addToPlayers adds p to the players set.
//
// When addToPlayers is called, the mutex m must be locked.
func (p *Player) addToPlayers() {
	p.m.Unlock()
	defer p.m.Lock()
	p.mux.addPlayer(p)
}

// removeFromPlayers removes p from the players set.
//
// When removeFromPlayers is called, the mutex m must be locked.
func (p *Player) removeFromPlayers() {
	p.m.Unlock()
	defer p.m.Lock()
	p.mux.removePlayer(p)
}

func (p *Player) playImpl() {
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

func (p *Player) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state != playerPlay {
		return
	}
	p.state = playerPaused
}

func (p *Player) Seek(offset int64, whence int) (int64, error) {
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

func (p *Player) Reset() {
	p.m.Lock()
	defer p.m.Unlock()
	p.resetImpl()
}

func (p *Player) resetImpl() {
	if p.state == playerClosed {
		return
	}
	p.state = playerPaused
	p.buf = p.buf[:0]
	p.eof = false
}

func (p *Player) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.state == playerPlay
}

func (p *Player) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()
	return p.volume
}

func (p *Player) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()
	p.volume = volume
	if p.state != playerPlay {
		p.prevVolume = volume
	}
}

func (p *Player) BufferedSize() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.buf)
}

func (p *Player) Close() error {
	runtime.SetFinalizer(p, nil)
	p.m.Lock()
	defer p.m.Unlock()
	return p.closeImpl()
}

func (p *Player) closeImpl() error {
	p.removeFromPlayers()

	if p.state == playerClosed {
		return p.err
	}
	p.state = playerClosed
	p.buf = nil
	return p.err
}

func (p *Player) readBufferAndAdd(buf []float32) int {
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

func (p *Player) canReadSourceToBuffer() bool {
	p.m.Lock()
	defer p.m.Unlock()

	if p.eof {
		return false
	}
	return len(p.buf) < p.bufferSize
}

func (p *Player) readSourceToBuffer() int {
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

func (p *Player) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}
