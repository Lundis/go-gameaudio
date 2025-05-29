package sfx

import (
	"math/rand/v2"
	"time"
)

// Scheduler lets you register sounds that should play in the future.
//
// If you are making a simulation game, the time is likely virtual,
// and this lets you use any time notion.
// If you use real time, just pass time.Now().Seconds() as the time.
//
// Scheduler can be used to schedule sounds to match timed animations,
// without needing to worry about executing it at exactly the right time.
//
// Remember to call Scheduler.Process() from your game loop.
type Scheduler struct {
	sounds []queuedSound
}

type queuedSound struct {
	id         Id
	whenToPlay float64
	fadeIn     time.Duration
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		sounds: make([]queuedSound, 0, 100),
	}
}

func (fs *Scheduler) PlaySoundEffectAt(id Id, at float64) {
	fs.PlaySoundEffectAtFadeIn(id, at, 0)
}

func (fs *Scheduler) PlaySoundEffectAtFadeIn(id Id, at float64, fadeIn time.Duration) {
	fs.sounds = append(fs.sounds, queuedSound{
		whenToPlay: at,
		fadeIn:     fadeIn,
		id:         id,
	})
}

func (fs *Scheduler) PlaySoundEffectAtRandomFadeIn(id Id, at float64, maxFadeInMilliSeconds int) {
	fs.PlaySoundEffectAtFadeIn(id, at, time.Duration(rand.IntN(maxFadeInMilliSeconds))*time.Millisecond)
}

func (fs *Scheduler) Clear() {
	fs.sounds = fs.sounds[:0]
}

func (fs *Scheduler) Process(now float64) {
	i := 0
	for i < len(fs.sounds) {
		if fs.sounds[i].whenToPlay <= now {
			if fs.sounds[i].whenToPlay >= now-3 {
				fs.sounds[i].id.PlayFadeIn(fs.sounds[i].fadeIn)
			}
			// clean array by moving the last element to the now free position
			fs.sounds[i] = fs.sounds[len(fs.sounds)-1]
			fs.sounds = fs.sounds[:len(fs.sounds)-1]
			continue
		} else {
			i++
		}
	}
}
