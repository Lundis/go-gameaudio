//go:build !js

package playlist

import (
	"time"

	"github.com/Lundis/go-gameaudio/audio"
)

type Track struct {
	trackCommon
	sound        *audio.Sound
	playingSound *audio.PlayingSound
}

func pauseMusic() {
	audio.ChannelIdMusic.Pause()
}

func trackSeek(track *Track, percentage float32) {
	if track.playingSound != nil {
		track.playingSound.Seek(percentage)
	}
}

func trackSeconds(track *Track) (current, total float32) {
	if track.playingSound != nil {
		return track.playingSound.Seconds()
	}
	return
}

func playlistPlay(pl *PlayList) {
	track := pl.Tracks[pl.currentTrack]
	if track.playingSound == nil || !track.playingSound.IsPlaying() {
		if len(pl.Tracks) > 1 {
			track.playingSound = track.sound.PlayFadeIn(time.Second / 2)
			track.playingSound.OnEndCallback(pl.PlayNext)
		} else {
			track.playingSound = track.sound.PlayLoop(time.Second)
		}
	}
}

func playlistStop(pl *PlayList) {
	track := pl.Tracks[pl.currentTrack]
	if track.playingSound != nil && track.playingSound.IsPlaying() {
		track.playingSound.StopFadeOut(time.Second)
	}
	track.playingSound = nil
}
