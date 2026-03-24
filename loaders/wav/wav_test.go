package wav_test

import (
	"testing"

	"github.com/Lundis/go-gameaudio/loaders/wav"
)

func TestLoadMono(t *testing.T) {
	_, err := wav.LoadWavFile("test_mono.wav", 44100)
	if err == nil {
		t.Fatalf("should not load mono tracks without error")
	}
}

func TestLoadStereo(t *testing.T) {
	data, err := wav.LoadWavFile("test_stereo.wav", 44100)
	if err != nil {
		t.Fatalf("error loading ogg: %s", err.Error())
	}
	if len(data) == 0 {
		t.Fatalf("no data")
	}
}

func TestLoad8khz(t *testing.T) {
	_, err := wav.LoadWavFile("test_8khz.wav", 44100)
	if err != nil {
		t.Fatalf("Error resampling 8khz wav: %s", err.Error())
	}
}

func TestLoad8bit(t *testing.T) {
	_, err := wav.LoadWavFile("test_8bit.wav", 44100)
	if err == nil {
		t.Fatalf("should not load non-16bit PCM tracks without error")
	}
}
