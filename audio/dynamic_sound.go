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
)

// NewDynamicSound creates a new, ready-to-use DynamicSound belonging to the Context.
// It is safe to create multiple sounds.
//
//	[data]      = [sample 1] [sample 2] [sample 3] ...
//	[sample *]  = [channel 1] [channel 2] ...
//	[channel *] = [float32]
//
// NewDynamicSound is concurrent-safe.
//
// All the functions of a DynamicSound returned by NewDynamicSound are concurrent-safe.
func NewDynamicSound(fillFunc func(buf []float32), volume float32, channel ChannelId) *DynamicSound {
	if mux == nil {
		return nil
	}
	pl := &DynamicSound{
		fillFunc:  fillFunc,
		volume:    volume,
		channelId: channel,
	}
	return pl
}

type DynamicSound struct {
	fillFunc  func(buf []float32)
	tmp       []float32
	channelId ChannelId
	volume    float32
	m         sync.Mutex
}

func (ds *DynamicSound) Play() {
	for i, existing := range dynamicSounds {
		if existing == nil {
			dynamicSounds[i] = ds
			return
		}
	}
	dynamicSounds = append(dynamicSounds, ds)
}

func (ds *DynamicSound) Stop() {
	for i, existing := range dynamicSounds {
		if ds == existing {
			dynamicSounds[i] = nil
		}
	}
}

func (ds *DynamicSound) readBufferAndAdd(buf []float32) {
	if len(ds.tmp) < len(buf) {
		ds.tmp = make([]float32, len(buf))
	}
	clear(ds.tmp)
	ds.fillFunc(ds.tmp)
	volume := ds.volume * ds.channelId.Volume()
	for i := 0; i < len(ds.tmp); i++ {
		buf[i] += volume * ds.tmp[i]
	}
}
