package sfx

import (
	"log"
	"math/rand/v2"
	"time"
)

// Id is used to identify a specific sound effect
// Use Play to play the sounds after loading them
type Id string

func (id Id) Play() bool {
	return id.PlayFadeIn(0)
}

func (id Id) PlayRandomFadeIn(maxFadeIn time.Duration) bool {
	return id.PlayFadeIn(time.Duration(rand.Int64N(int64(maxFadeIn))))
}

func (id Id) PlayFadeIn(fadeIn time.Duration) bool {
	lock.RLock()
	defer lock.RUnlock()
	if loadedSfx, ok := loadedSfx[id]; ok {
		return loadedSfx.play(fadeIn)
	} else {
		log.Printf("sfx: %s not loaded", id)
	}
	return false
}
