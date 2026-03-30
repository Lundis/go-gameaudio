//go:build js

package playlist

import (
	"log"
	"syscall/js"
	"time"

	"golang.org/x/tools/godoc/vfs"
)

// LoadFolder loads playlists from a regular folder.
// See Load for more information.
func LoadFolder(folder string) error {
	fs := vfs.OS(folder)
	return Load(fs)
}

// Load loads playlists from a virtual filesystem.
// At the root of the filesystem there must be a "playlist.json" file, which references any files to be loaded.
//
// On JS/WASM, each track's raw bytes are placed in a Blob and a URL is created via
// URL.createObjectURL. An HTMLAudioElement is then created for each track so that the
// browser's native audio engine handles decoding and playback, keeping the single-threaded
// JS runtime free from that work.
func Load(fileSystem vfs.Opener) error {
	lock.Lock()
	defer lock.Unlock()
	start := time.Now()
	playlists, err := loadRegistry(fileSystem, "playlist.json")
	if err != nil {
		return err
	}

	failedPlaylists := make(map[int]bool)
	for i, pl := range playlists {
		for _, track := range pl.Tracks {
			raw, err := readFile(fileSystem, track.Path)
			if err != nil {
				log.Println("Failed to read music track from disk", track.Path, ":", err.Error())
				failedPlaylists[i] = true
				continue
			}
			if err := initTrackAudioElement(track, raw); err != nil {
				log.Println("Failed to create audio element for track", track.Path, ":", err.Error())
				failedPlaylists[i] = true
			}
		}
	}

	effects := make(map[Id]*PlayList, len(playlists))
	for i, pl := range playlists {
		if !failedPlaylists[i] {
			effects[pl.Id] = pl
		}
	}
	playLists = effects

	log.Printf("Loaded %d playlists in %.2fs\n", len(playLists), time.Since(start).Seconds())
	return nil
}

// initTrackAudioElement stores the raw ogg bytes in a Blob, registers a blob URL,
// and attaches an HTMLAudioElement to the track.
func initTrackAudioElement(track *Track, raw []byte) error {
	uint8Array := js.Global().Get("Uint8Array").New(len(raw))
	js.CopyBytesToJS(uint8Array, raw)

	blob := js.Global().Get("Blob").New(
		[]any{uint8Array},
		map[string]any{"type": "audio/ogg"},
	)
	urlVal := js.Global().Get("URL").Call("createObjectURL", blob)
	blobURL := urlVal.String()

	audioEl := js.Global().Get("Audio").New(blobURL)
	audioEl.Set("preload", "auto")

	track.audioEl = audioEl
	track.blobURL = blobURL
	return nil
}
