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
	"fmt"
	"sync"
	"time"
)

var (
	contextCreationMutex sync.Mutex
)

const ChannelCount = 2

// NewContextOptions represents options for NewContext.
type NewContextOptions struct {
	// SampleRate specifies the number of samples that should be played during one second.
	// Usual numbers are 44100 or 48000. One context has only one sample rate. You cannot play multiple audio
	// sources with different sample rates at the same time.
	SampleRate int

	// BufferSize specifies a buffer size in the underlying device.
	//
	// If 0 is specified, the driver's default buffer size is used.
	// Set BufferSize to adjust the buffer size if you want to adjust latency or reduce noises.
	// Too big buffer size can increase the latency time.
	// On the other hand, too small buffer size can cause glitch noises due to buffer shortage.
	BufferSize time.Duration
}

// InitContext creates a new context with given options.
// A context creates and holds ready-to-use Sound objects.
// InitContext returns a channel that is closed when the context is ready, and an error if it exists.
//
// Creating multiple contexts is NOT supported.
func InitContext(options *NewContextOptions) (chan struct{}, error) {
	contextCreationMutex.Lock()
	defer contextCreationMutex.Unlock()

	if mux != nil {
		return nil, fmt.Errorf("context was already created")
	}

	var bufferSizeInBytes int
	if options.BufferSize != 0 {
		// The underlying driver always uses 32bit floats.
		bytesPerSample := ChannelCount * 4
		bytesPerSecond := options.SampleRate * bytesPerSample
		bufferSizeInBytes = int(int64(options.BufferSize) * int64(bytesPerSecond) / int64(time.Second))
		bufferSizeInBytes = bufferSizeInBytes / bytesPerSample * bytesPerSample
	}
	initMux(options.SampleRate, ChannelCount)
	ready, err := newContext(bufferSizeInBytes)
	if err != nil {
		return nil, err
	}
	return ready, nil
}

func SampleRate() int {
	return mux.sampleRate
}
