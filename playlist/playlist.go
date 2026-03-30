package playlist

import (
	"log"
	"math/rand"
)

var playLists map[Id]*PlayList
var currentPlayList *PlayList

type Id string

type PlayList struct {
	Id           Id
	Tracks       []*Track
	currentTrack int
}

type trackCommon struct {
	Path   string
	Name   string
	Album  string
	Author string
	Volume float32
}

func Pause() {
	pauseMusic()
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
	if track := CurrentTrack(); track != nil {
		trackSeek(track, percentage)
	}
}

func Seconds() (current, total float32) {
	if track := CurrentTrack(); track != nil {
		return trackSeconds(track)
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
		playlistStop(currentPlayList)
		currentPlayList = nil
	}
	if pl, ok := playLists[playListId]; ok {
		currentPlayList = pl
		if shuffle {
			currentPlayList.currentTrack = rand.Intn(len(currentPlayList.Tracks))
		}
		playlistPlay(currentPlayList)
	}
}

func (pl *PlayList) PlayNext() {
	playlistStop(pl)
	pl.currentTrack = (pl.currentTrack + 1) % len(pl.Tracks)
	log.Println("INFO: playing next track", pl.Tracks[pl.currentTrack].Name)
	playlistPlay(pl)
}

func (pl *PlayList) PlayPrevious() {
	playlistStop(pl)
	pl.currentTrack = (pl.currentTrack - 1 + len(pl.Tracks)) % len(pl.Tracks)
	log.Println("INFO: playing previous track", pl.Tracks[pl.currentTrack].Name)
	playlistPlay(pl)
}
