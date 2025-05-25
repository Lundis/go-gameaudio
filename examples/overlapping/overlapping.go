// Copyright 2019 The Oto Authors
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

package main

import (
	"github.com/Lundis/go-gameaudio/audio"
	"github.com/Lundis/go-gameaudio/examples/internal"
	"runtime"
	"sync"
	"time"
)

func main() {
	op := &audio.NewContextOptions{}
	op.SampleRate = internal.SampleRate
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	ready, err := audio.InitContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	var wg sync.WaitGroup
	var sounds []*audio.Sound
	var m sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		p := internal.PlaySineWave(internal.FreqC, 3*time.Second)
		m.Lock()
		sounds = append(sounds, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		p := internal.PlaySineWave(internal.FreqE, 3*time.Second)
		m.Lock()
		sounds = append(sounds, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		p := internal.PlaySineWave(internal.FreqG, 3*time.Second)
		m.Lock()
		sounds = append(sounds, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Wait()

	// Pin the sounds not to GC the sounds.
	runtime.KeepAlive(sounds)

}
