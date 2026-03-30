//go:build js

package playlist

import (
	"syscall/js"

	"github.com/Lundis/go-gameaudio/audio"
)

// Track holds a reference to the browser's HTMLAudioElement.
// The browser owns decoding and playback; Go only drives control flow.
type Track struct {
	trackCommon
	audioEl js.Value
	blobURL string
	endedFn *js.Func // kept alive to allow removeEventListener
}

func pauseMusic() {
	if track := CurrentTrack(); track != nil {
		track.audioEl.Call("pause")
	}
}

func trackSeek(track *Track, percentage float32) {
	duration := track.audioEl.Get("duration").Float()
	if duration > 0 {
		track.audioEl.Set("currentTime", float64(percentage)*duration)
	}
}

func trackSeconds(track *Track) (current, total float32) {
	current = float32(track.audioEl.Get("currentTime").Float())
	total = float32(track.audioEl.Get("duration").Float())
	return
}

func playlistPlay(pl *PlayList) {
	track := pl.Tracks[pl.currentTrack]
	if !track.audioEl.Get("paused").Bool() {
		return
	}
	track.audioEl.Set("volume", track.Volume*audio.ChannelIdMusic.Volume())
	if len(pl.Tracks) > 1 {
		track.audioEl.Set("loop", false)
		releaseEndedFn(track)
		fn := js.FuncOf(func(this js.Value, args []js.Value) any {
			pl.PlayNext()
			return nil
		})
		track.endedFn = &fn
		track.audioEl.Call("addEventListener", "ended", fn)
	} else {
		track.audioEl.Set("loop", true)
	}
	track.audioEl.Call("play")
}

func playlistStop(pl *PlayList) {
	track := pl.Tracks[pl.currentTrack]
	track.audioEl.Call("pause")
	track.audioEl.Set("currentTime", 0)
	releaseEndedFn(track)
}

func releaseEndedFn(track *Track) {
	if track.endedFn != nil {
		track.audioEl.Call("removeEventListener", "ended", *track.endedFn)
		track.endedFn.Release()
		track.endedFn = nil
	}
}
