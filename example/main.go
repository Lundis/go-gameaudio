// Copyright 2019 The Oto Authors
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

package main

import (
	"flag"
	"io"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/Lundis/oto/v3"
)

var (
	sampleRate   = flag.Int("samplerate", 44100, "sample rate")
	channelCount = flag.Int("channelcount", 2, "number of channel")
)

type SineWave struct {
	freq   float64
	length int64
	pos    int64

	channelCount int

	remaining []float32
}

func NewSineWave(freq float64, duration time.Duration, channelCount int) *SineWave {
	l := int64(channelCount) * int64(*sampleRate) * int64(duration) / int64(time.Second)
	return &SineWave{
		freq:         freq,
		length:       l,
		channelCount: channelCount,
	}
}

func (s *SineWave) Read(buf []float32) (int, error) {
	if len(s.remaining) > 0 {
		n := copy(buf, s.remaining)
		copy(s.remaining, s.remaining[n:])
		s.remaining = s.remaining[:len(s.remaining)-n]
		return n, nil
	}

	if s.pos == s.length {
		return 0, io.EOF
	}

	eof := false
	if s.pos+int64(len(buf)) > s.length {
		buf = buf[:s.length-s.pos]
		eof = true
	}

	samplesPerSecond := float64(*sampleRate) / s.freq

	p := s.pos / int64(s.channelCount)
	for i := 0; i < len(buf)/s.channelCount; i++ {
		bs := float32(math.Sin(2*math.Pi*float64(p)/samplesPerSecond) * 0.3)

		for ch := 0; ch < s.channelCount; ch++ {
			buf[s.channelCount*i+ch] = bs
		}
		p++
	}

	s.pos += int64(len(buf))

	n := len(buf)
	s.remaining = buf[n:]

	if eof {
		return n, io.EOF
	}
	return n, nil
}

func play(context *oto.Context, freq float64, duration time.Duration, channelCount int) *oto.Player {
	p := context.NewPlayer(NewSineWave(freq, duration, channelCount))
	p.Play()
	return p
}

func run() error {
	const (
		freqC = 523.3
		freqE = 659.3
		freqG = 784.0
	)

	op := &oto.NewContextOptions{}
	op.SampleRate = *sampleRate
	op.ChannelCount = *channelCount

	c, ready, err := oto.NewContext(op)
	if err != nil {
		return err
	}
	<-ready

	var wg sync.WaitGroup
	var players []*oto.Player
	var m sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		p := play(c, freqC, 3*time.Second, op.ChannelCount)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		p := play(c, freqE, 3*time.Second, op.ChannelCount)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		p := play(c, freqG, 3*time.Second, op.ChannelCount)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Wait()

	// Pin the players not to GC the players.
	runtime.KeepAlive(players)

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		panic(err)
	}
}
