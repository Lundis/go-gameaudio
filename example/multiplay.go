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
	"github.com/Lundis/oto/v3/loaders/wav"
	"time"

	"github.com/Lundis/oto/v3"
)

func main() {
	op := &oto.NewContextOptions{}
	op.SampleRate = internal.SampleRate
	op.ChannelCount = internal.ChannelCount
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	context, ready, err := oto.NewContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	data, err := wav.LoadWav("loaders/wav/test_stereo.wav", internal.SampleRate)
	if err != nil {
		panic(err)
	}
	p := context.NewPlayer(data, 1, oto.ChannelIdDefault)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)

	time.Sleep(1 * time.Second)

}
