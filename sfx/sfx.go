package sfx

import (
	"github.com/Lundis/go-gameaudio/audio"
	"math/rand/v2"
	"time"
)

var loadedSfx map[Id]*Sfx

type Sfx struct {
	Id           Id
	Volume       float32
	ThrottlingMs int
	Variations   []*SfxVariant
	DebugMode    bool
	lastPlayed   time.Time
}

type SfxVariant struct {
	Path         string
	Probability  float64
	Volume       float32
	ThrottlingMs int
	sound        *audio.Sound
	lastPlayed   time.Time
}

func (e *Sfx) play(fadeInMs time.Duration) bool {
	if len(e.Variations) == 0 {
		return false
	}

	if time.Since(e.lastPlayed) <= time.Duration(e.ThrottlingMs)*time.Millisecond {
		//fmt.Println("throttling", e.Id)
		return false
	} else {
	}
	unThrottled := make([]*SfxVariant, 0, len(e.Variations))
	probabilitySum := 0.0
	for _, v := range e.Variations {
		if time.Since(v.lastPlayed) > time.Duration(v.ThrottlingMs)*time.Millisecond {
			unThrottled = append(unThrottled, v)
			probabilitySum += v.Probability
		}
	}
	random := rand.Float64() * probabilitySum

	if len(unThrottled) == 0 {
		return false
	}

	for _, v := range e.Variations {
		if random <= v.Probability+0.001 {
			v.play(fadeInMs)
			e.lastPlayed = time.Now()
			/*
				if e.DebugMode || config.DebugMode {
					fmt.Println("Playing sound effect", e.Id, "with variation", v.Path)
				}
			*/
			return true
		} else {
			random -= v.Probability
		}
	}
	return false
}

func (e *SfxVariant) play(fadeInMs time.Duration) {
	if (fadeInMs) > 0 {
		e.sound.PlayFadeIn(fadeInMs)
	} else {
		e.sound.Play()
	}
	e.lastPlayed = time.Now()
}
