package sfx

import (
	"log"
	"math/rand/v2"
	"slices"
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

var alreadyLoggedMissingSfx = make([]Id, 0, 100)

func (id Id) logMissing() {
	if slices.Index(alreadyLoggedMissingSfx, id) >= 0 {
		return
	}
	log.Printf("sfx: %s not loaded", id)
	alreadyLoggedMissingSfx = append(alreadyLoggedMissingSfx, id)
}

func (id Id) PlayFadeIn(fadeIn time.Duration) bool {
	lock.RLock()
	defer lock.RUnlock()
	if loadedSfx, ok := loadedSfx[id]; ok {
		return loadedSfx.play(fadeIn)
	} else {
		id.logMissing()

		return false
	}
}
