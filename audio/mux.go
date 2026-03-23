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

// Mux is a low-level multiplexer of audio sounds.
type Mux struct {
	sampleRate   int
	channelCount int

	sounds map[*Sound]struct{}
}

var mux *Mux

func initMux(sampleRate int, channelCount int) {
	mux = &Mux{
		sampleRate:   sampleRate,
		channelCount: channelCount,
	}
}

func (m *Mux) addSound(sound *Sound) {

	if m.sounds == nil {
		m.sounds = map[*Sound]struct{}{}
	}
	m.sounds[sound] = struct{}{}
}

func (m *Mux) removeSound(sound *Sound) {
	delete(m.sounds, sound)
}

// ReadFloat32s fills buf with the multiplexed data of the sounds as float32 values.
func (m *Mux) ReadFloat32s(buf []float32) {

loop:
	for {
		select {
		case request := <-playRequests:
			request.sound.playImpl(request.fadeInEndsAt, request.fadeOutStartsAt)
		default:
			break loop
		}
	}

	clear(buf)
	for p := range m.sounds {
		p.readBufferAndAdd(buf)
	}

}
