package internal

import (
	"github.com/Lundis/go-gameaudio/audio"
	"math"
	"time"
)

func GenerateSineWave(freq float64, duration time.Duration) []float32 {
	n := int(float64(ChannelCount*SampleRate) * duration.Seconds())
	samples := make([]float32, n)
	for i := 0; i < n; i += 2 {
		angle := 2 * math.Pi * float64(i/2) * freq / SampleRate
		value := float32(math.Sin(angle)) * 0.3
		samples[i] = value   // left
		samples[i+1] = value // right
	}
	return samples

}

func PlaySineWave(context *audio.oto, freq float64, duration time.Duration) *audio.oto {
	p := context.NewSound(GenerateSineWave(freq, duration), 1, audio.oto.ChannelIdDefault)
	p.Play()
	return p
}
