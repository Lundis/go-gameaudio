package main

import (
	"github.com/Lundis/oto/v3/example/internal"
	"github.com/Lundis/oto/v3/loaders/wav"
	"time"

	"github.com/Lundis/oto/v3"
)

func main() {
	op := &oto.NewContextOptions{}
	op.SampleRate = internal.SampleRate
	op.ChannelCount = internal.ChannelCount
	op.BufferSize = 10 * time.Millisecond // this is actually ignored in windows (WASAPI)

	context, ready, err := oto.NewContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	data, err := wav.LoadWav("loaders/wav/test_stereo.wav", internal.SampleRate)
	if err != nil {
		panic(err)
	}
	p := context.NewSound(data, 1, oto.ChannelIdDefault)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)
	p.Play()
	time.Sleep(300 * time.Millisecond)

	time.Sleep(1 * time.Second)

}
