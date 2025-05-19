package vorbis

import (
	"fmt"
	"os"

	"github.com/jfreymuth/oggvorbis"
)

func LoadOggVorbis(path string, expectedSampleRate int) ([]float32, error) {
	f, err := os.Open(path)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	data, format, err := oggvorbis.ReadAll(f)

	if err != nil {
		return nil, err
	}
	if format.Channels != 2 {
		return nil, fmt.Errorf("%s: number of channels must be 2 but was %d", path, format.Channels)
	}
	if format.SampleRate != expectedSampleRate {
		return nil, fmt.Errorf("%s: sample rate must be %d but was %d", path, expectedSampleRate, format.SampleRate)
	}
	return data, nil
}
