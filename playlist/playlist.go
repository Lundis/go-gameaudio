package playlist

import (
	"github.com/Lundis/go-gameaudio/audio"
	"time"
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
	Author string
	Volume float32
	sound  *audio.Sound
}

func Pause() {
	audio.ChannelIdMusic.Pause()
}

func (playListId Id) Play() {
	lock.RLock()
	defer lock.RUnlock()
	audio.ChannelIdMusic.Resume()
	if currentPlayList != nil && currentPlayList.Id == playListId {
		return
	}
	currentPlayList = nil
	if pl, ok := playLists[playListId]; ok {
		currentPlayList = pl
		currentPlayList.play()
	}
}

func (pl *PlayList) play() {
	track := pl.Tracks[pl.currentTrack]
	if !track.sound.IsPlaying() {
		if len(pl.Tracks) > 1 {
			// TODO on-end callback
			track.sound.Play()
			track.sound.OnEndCallback(pl.playNext)
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

func (pl *PlayList) playNext() {
	pl.stop()
	pl.currentTrack = (pl.currentTrack + 1) % len(pl.Tracks)
	pl.play()
}
