package main

import (
	"github.com/Lundis/go-gameaudio/audio"
	"github.com/Lundis/go-gameaudio/examples/internal"
	"github.com/Lundis/go-gameaudio/loaders/wav"
	"time"
)

func main() {
	op := &audio.NewContextOptions{}
	op.SampleRate = internal.SampleRate
	op.ChannelCount = internal.ChannelCount
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	ready, err := audio.InitContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	data, err := wav.LoadWav("loaders/wav/test_stereo.wav", internal.SampleRate)
	if err != nil {
		panic(err)
	}
	p := audio.NewSound(data, 1, audio.ChannelIdDefault)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)

	time.Sleep(1 * time.Second)

}
