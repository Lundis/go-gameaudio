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
	Path   string
	Name   string
	Album  string
	Author string
	Volume float32
	sound  *audio.Sound
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
	if !track.sound.IsPlaying() {
		if len(pl.Tracks) > 1 {
			track.sound.Play()
			track.sound.OnEndCallback(pl.PlayNext)
		} else {
			track.sound.PlayLoop(time.Second)
		}
	}

}

func (pl *PlayList) stop() {
	track := pl.Tracks[pl.currentTrack]
	if track.sound.IsPlaying() {
		track.sound.Stop()
	}
}

func (pl *PlayList) PlayNext() {
	pl.stop()
	pl.currentTrack = (pl.currentTrack + 1) % len(pl.Tracks)
	log.Println("INFO: playing next track", pl.Tracks[pl.currentTrack].Name)
	pl.play()
}
