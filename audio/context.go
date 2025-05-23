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
	currentContext       *Context
	contextCreationMutex sync.Mutex
)

var ErrContextNotCreated error = fmt.Errorf("context not created")

// Context is the main object in Oto. It interacts with the audio drivers.
//
// To play sound with Oto, first create a context. Then use the context to create
// an arbitrary number of sounds. Then use the sounds to play sound.
//
// Creating multiple contexts is NOT supported.
type Context struct {
	context *context
}

// NewContextOptions represents options for NewContext.
type NewContextOptions struct {
	// SampleRate specifies the number of samples that should be played during one second.
	// Usual numbers are 44100 or 48000. One context has only one sample rate. You cannot play multiple audio
	// sources with different sample rates at the same time.
	SampleRate int

	// ChannelCount specifies the number of channels. One channel is mono playback. Two
	// channels are stereo playback. No other values are supported.
	ChannelCount int

	// BufferSize specifies a buffer size in the underlying device.
	//
	// If 0 is specified, the driver's default buffer size is used.
	// Set BufferSize to adjust the buffer size if you want to adjust latency or reduce noises.
	// Too big buffer size can increase the latency time.
	// On the other hand, too small buffer size can cause glitch noises due to buffer shortage.
	BufferSize time.Duration
}

// NewContext creates a new context with given options.
// A context creates and holds ready-to-use Sound objects.
// NewContext returns a context, a channel that is closed when the context is ready, and an error if it exists.
//
// Creating multiple contexts is NOT supported.
func InitContext(options *NewContextOptions) (chan struct{}, error) {
	contextCreationMutex.Lock()
	defer contextCreationMutex.Unlock()

	if currentContext != nil {
		return nil, fmt.Errorf("context was already created")
	}

	var bufferSizeInBytes int
	if options.BufferSize != 0 {
		// The underying driver always uses 32bit floats.
		bytesPerSample := options.ChannelCount * 4
		bytesPerSecond := options.SampleRate * bytesPerSample
		bufferSizeInBytes = int(int64(options.BufferSize) * int64(bytesPerSecond) / int64(time.Second))
		bufferSizeInBytes = bufferSizeInBytes / bytesPerSample * bytesPerSample
	}
	ctx, ready, err := newContext(options.SampleRate, options.ChannelCount, bufferSizeInBytes)
	if err != nil {
		return nil, err
	}
	currentContext = &Context{context: ctx}
	return ready, nil
}

// Suspend suspends the entire audio play.
//
// Suspend is concurrent-safe.
func Suspend() error {
	if currentContext == nil {
		return ErrContextNotCreated
	}
	return currentContext.context.Suspend()
}

// Resume resumes the entire audio play, which was suspended by Suspend.
//
// Resume is concurrent-safe.
func Resume() error {
	if currentContext == nil {
		return ErrContextNotCreated
	}
	return currentContext.context.Resume()
}

// Err returns the current error.
//
// Err is concurrent-safe.
func Err() error {
	if currentContext == nil {
		return ErrContextNotCreated
	}
	return currentContext.context.Err()
}
