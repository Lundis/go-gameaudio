package oggvorbis

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Lundis/go-gameaudio/loaders/resample"
	"github.com/jfreymuth/oggvorbis"
)

func LoadFile(path string, expectedSampleRate int) ([]float32, error) {
	rawData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to open: %w", path, err)
	}

	data, err := Load(rawData, expectedSampleRate)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return data, nil
}

func Load(oggData []byte, expectedSampleRate int) ([]float32, error) {

	data, format, err := oggvorbis.ReadAll(bytes.NewReader(oggData))

	if err != nil {
		return nil, err
	}
	if format.Channels != 2 {
		return nil, fmt.Errorf("number of channels must be 2 but was %d", format.Channels)
	}
	return resample.Stereo(data, format.SampleRate, expectedSampleRate), nil
}
