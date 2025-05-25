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
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	ready, err := audio.InitContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	data, err := wav.LoadWavFile("loaders/wav/test_stereo.wav", internal.SampleRate)
	if err != nil {
		panic(err)
	}
	p := audio.NewSound(data, 1, audio.ChannelIdDefault)
	// this crossfading sounds rather silly...
	p.PlayLoop(1000 * time.Millisecond)

	time.Sleep(5 * time.Second)

}
