package oggvorbis_test

import (
	"github.com/Lundis/go-gameaudio/loaders/oggvorbis"
	"testing"
)

func TestLoadMono(t *testing.T) {
	_, err := oggvorbis.LoadFile("test.ogg", 44100)
	if err == nil {
		t.Fatalf("should not load mono tracks without error")
	}
}

func TestLoadStereo(t *testing.T) {
	data, err := oggvorbis.LoadFile("test_stereo.ogg", 44100)
	if err != nil {
		t.Fatalf("error loading ogg: %s", err.Error())
	}
	if len(data) == 0 {
		t.Fatalf("no data")
	}
}

func TestLoad8khz(t *testing.T) {
	_, err := oggvorbis.LoadFile("test_stereo_8khz.ogg", 44100)
	if err == nil {
		t.Fatalf("should not load tracks in unexpected sampling rate without error")
	}
}
