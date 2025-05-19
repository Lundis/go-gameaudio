// Copyright 2022 The Oto Authors
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

package vorbis_test

import (
	vorbis "github.com/Lundis/oto/v3/loaders/oggvorbis"
	"testing"
)

func TestLoadMono(t *testing.T) {
	_, err := vorbis.LoadOggVorbis("test.ogg", 44100)
	if err == nil {
		t.Fatalf("should not load mono tracks without error")
	}
}

func TestLoadStereo(t *testing.T) {
	data, err := vorbis.LoadOggVorbis("test_stereo.ogg", 44100)
	if err != nil {
		t.Fatalf("error loading ogg: %s", err.Error())
	}
	if len(data) == 0 {
		t.Fatalf("no data")
	}
}

func TestLoad8khz(t *testing.T) {
	_, err := vorbis.LoadOggVorbis("test_stereo_8khz.ogg", 44100)
	if err == nil {
		t.Fatalf("should not load tracks in unexpected sampling rate without error")
	}
}
