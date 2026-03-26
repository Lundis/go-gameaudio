package audio

import (
	"time"
)

type PlayingSound struct {
	sound           *Sound
	loop            bool
	loopedOnce      bool
	pos             int
	seekTo          float32
	fadeInEndsAt    int
	fadeOutStartsAt int
	endAt           int
	onEndCallback   func()
}

// OnEndCallback can be used to register a callback that will be called once when the sound has finished playing
func (ps *PlayingSound) OnEndCallback(onEndCallback func()) {
	ps.onEndCallback = onEndCallback
}

// Seek a playing sound to a given percentage
// if the sound already finished, this will do nothing
func (ps *PlayingSound) Seek(percentage float32) {
	ps.seekTo = percentage
}

// Seconds returns the current position and total length of the first currently playing instance of this sound
func (ps *PlayingSound) Seconds() (current, total float32) {
	samplesPerSecond := float32(mux.channelCount * mux.sampleRate)
	current = float32(ps.pos) / samplesPerSecond
	total = float32(len(ps.sound.data)) / samplesPerSecond
	return
}

func (ps *PlayingSound) Stop() {
	ps.endAt = ps.pos
	ps.loop = false
}

func (ps *PlayingSound) StopFadeOut(fadeIn time.Duration) {
	ps.endAt = min(ps.endAt, ps.pos+int(float64(mux.channelCount*mux.sampleRate)*fadeIn.Seconds()))
	ps.fadeOutStartsAt = ps.pos
}

func (ps *PlayingSound) IsPlaying() bool {
	return ps.pos < ps.endAt
}
