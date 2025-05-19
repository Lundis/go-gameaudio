package wav_test

import (
	"github.com/Lundis/oto/v3/loaders/wav"
	"testing"
)

func TestLoadMono(t *testing.T) {
	_, err := wav.LoadWav("test_mono.wav", 44100)
	if err == nil {
		t.Fatalf("should not load mono tracks without error")
	}
}

func TestLoadStereo(t *testing.T) {
	data, err := wav.LoadWav("test_stereo.wav", 44100)
	if err != nil {
		t.Fatalf("error loading ogg: %s", err.Error())
	}
	if len(data) == 0 {
		t.Fatalf("no data")
	}
}

func TestLoad8khz(t *testing.T) {
	_, err := wav.LoadWav("test_8khz.wav", 44100)
	if err == nil {
		t.Fatalf("should not load tracks in unexpected sampling rate without error")
	}
}

func TestLoad8bit(t *testing.T) {
	_, err := wav.LoadWav("test_8bit.wav", 44100)
	if err == nil {
		t.Fatalf("should not load non-16bit PCM tracks without error")
	}
}
