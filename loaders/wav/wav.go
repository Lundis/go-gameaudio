// Copyright 2016 Hajime Hoshi
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

// Package wav provides WAV (RIFF) decoder.
package wav

import (
	"bytes"
	"fmt"
	"os"
)

func LoadWav(path string, wantedSampleRate int) ([]float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data[0:4], []byte("RIFF")) {
		return nil, fmt.Errorf("wav: invalid header: 'RIFF' not found")
	}
	if !bytes.Equal(data[8:12], []byte("WAVE")) {
		return nil, fmt.Errorf("wav: invalid header: 'WAVE' not found")
	}

	// Read chunks
	headerSize := 12
	for {
		buf := data[headerSize : headerSize+8]
		headerSize += 8
		size := int(buf[4]) | int(buf[5])<<8 | int(buf[6])<<16 | int(buf[7])<<24
		switch {
		case bytes.Equal(buf[0:4], []byte("fmt ")):
			// Size of 'fmt' header is usually 16, but can be more than 16.
			if size < 16 {
				return nil, fmt.Errorf("wav: invalid header: maybe non-PCM file?")
			}
			buf := data[headerSize : headerSize+size]
			format := int(buf[0]) | int(buf[1])<<8
			if format != 1 {
				return nil, fmt.Errorf("wav: format must be linear PCM")
			}
			channelCount := int(buf[2]) | int(buf[3])<<8
			switch channelCount {
			case 1:
				return nil, fmt.Errorf("WAV must be stereo")
			case 2:
				// OK
			default:
				return nil, fmt.Errorf("wav: number of channels must be 2 but was %d", channelCount)
			}
			bitsPerSample := int(buf[14]) | int(buf[15])<<8
			if bitsPerSample != 16 {
				return nil, fmt.Errorf("wav: bits per sample must be 16 but was %d", bitsPerSample)
			}
			sampleRate := int(buf[4]) | int(buf[5])<<8 | int(buf[6])<<16 | int(buf[7])<<24
			if sampleRate != wantedSampleRate {
				return nil, fmt.Errorf("wav: sample rate must be %d but was %d", wantedSampleRate, sampleRate)
			}
			headerSize += size
		case bytes.Equal(buf[0:4], []byte("data")):
			return convertInt16ToFloat32(data[headerSize : headerSize+size]), nil
		default:
			headerSize += size
		}
	}
}

func convertInt16ToFloat32(i16Buf []byte) []float32 {
	f32 := make([]float32, len(i16Buf)/2)
	for i := 0; i < len(i16Buf); i += 2 {
		vi16l := i16Buf[i]
		vi16h := i16Buf[i+1]
		f32[i/2] = float32(int16(vi16l)|int16(vi16h)<<8) / (1 << 15)
	}
	return f32
}
