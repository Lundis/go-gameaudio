// Copyright 2022 The Oto Authors
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
	"errors"
	"fmt"
	"time"
)

var errDeviceNotFound = errors.New("oto: device not found")

var windowsContext struct {
	wasapiContext *wasapiContext
	winmmContext  *winmmContext
	nullContext   *nullContext

	ready chan struct{}
	err   atomicError
}

func newContext(bufferSizeInBytes int) (chan struct{}, error) {
	windowsContext.ready = make(chan struct{})

	// Initializing drivers might take some time. Do this asynchronously.
	go func() {
		defer close(windowsContext.ready)

		xc, err0 := newWASAPIContext(bufferSizeInBytes)
		if err0 == nil {
			windowsContext.wasapiContext = xc
			return
		}

		wc, err1 := newWinMMContext(bufferSizeInBytes)
		if err1 == nil {
			windowsContext.winmmContext = wc
			return
		}

		if errors.Is(err0, errDeviceNotFound) && errors.Is(err1, errDeviceNotFound) {
			windowsContext.nullContext = newNullContext()
			return
		}

		windowsContext.err.TryStore(fmt.Errorf("oto: initialization failed: WASAPI: %v, WinMM: %v", err0, err1))
	}()

	return windowsContext.ready, nil
}

type nullContext struct {
	suspended bool
}

func newNullContext() *nullContext {
	c := &nullContext{}
	go c.loop()
	return c
}

func (c *nullContext) loop() {
	var buf32 [4096]float32
	sleep := time.Duration(float64(time.Second) * float64(len(buf32)) / float64(ChannelCount) / float64(mux.sampleRate))
	for {
		if c.suspended {
			time.Sleep(time.Second)
			continue
		}

		mux.ReadFloat32s(buf32[:])
		time.Sleep(sleep)
	}
}
