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
	"github.com/Lundis/oto/v3/example/internal"
	"runtime"
	"sync"
	"time"

	"github.com/Lundis/oto/v3"
)

func main() {
	op := &oto.NewContextOptions{}
	op.SampleRate = internal.SampleRate
	op.ChannelCount = internal.ChannelCount
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	c, ready, err := oto.NewContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	var wg sync.WaitGroup
	var players []*oto.Player
	var m sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		p := internal.PlaySineWave(c, internal.FreqC, 3*time.Second)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		p := internal.PlaySineWave(c, internal.FreqE, 3*time.Second)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		p := internal.PlaySineWave(c, internal.FreqG, 3*time.Second)
		m.Lock()
		players = append(players, p)
		m.Unlock()
		time.Sleep(3 * time.Second)
	}()

	wg.Wait()

	// Pin the players not to GC the players.
	runtime.KeepAlive(players)

}
