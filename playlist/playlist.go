package playlist

import (
	"log"
	"math/rand"
	"time"

	"github.com/Lundis/go-gameaudio/audio"
)

var playLists map[Id]*PlayList
var currentPlayList *PlayList

type Id string

type PlayList struct {
	Id           Id
	Tracks       []*Track
	currentTrack int
}

type Track struct {
	Path         string
	Name         string
	Album        string
	Author       string
	Volume       float32
	sound        *audio.Sound
	playingSound *audio.PlayingSound
}

func Pause() {
	audio.ChannelIdMusic.Pause()
}

func CurrentPlaylist() *PlayList {
	return currentPlayList
}

func CurrentTrack() *Track {
	if currentPlayList == nil {
		return nil
	}
	return currentPlayList.Tracks[currentPlayList.currentTrack]
}

func Seek(percentage float32) {
	if track := CurrentTrack(); track != nil && track.playingSound != nil {
		track.playingSound.Seek(percentage)
	}
}

func Seconds() (current, total float32) {
	if track := CurrentTrack(); track != nil && track.playingSound != nil {
		return track.playingSound.Seconds()
	}
	return
}

func (playListId Id) Play(shuffle bool) {
	lock.RLock()
	defer lock.RUnlock()
	if currentPlayList != nil {
		if currentPlayList.Id == playListId {
			return
		}
		currentPlayList.stop()
		currentPlayList = nil
	}
	if pl, ok := playLists[playListId]; ok {
		currentPlayList = pl
		if shuffle {
			currentPlayList.currentTrack = rand.Intn(len(currentPlayList.Tracks))
		}
		currentPlayList.play()
	}
}

func (pl *PlayList) play() {
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

func (pl *PlayList) stop() {
	track := pl.Tracks[pl.currentTrack]
	if track.playingSound != nil && track.playingSound.IsPlaying() {
		track.playingSound.StopFadeOut(time.Second)
	}
	track.playingSound = nil
}

func (pl *PlayList) PlayNext() {
	pl.stop()
	pl.currentTrack = (pl.currentTrack + 1) % len(pl.Tracks)
	log.Println("INFO: playing next track", pl.Tracks[pl.currentTrack].Name)
	pl.play()
}

func (pl *PlayList) PlayPrevious() {
	pl.stop()
	pl.currentTrack = (pl.currentTrack - 1 + len(pl.Tracks)) % len(pl.Tracks)
	log.Println("INFO: playing previous track", pl.Tracks[pl.currentTrack].Name)
	pl.play()
}
