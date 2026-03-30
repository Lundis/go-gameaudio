//go:build !js

package playlist

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/Lundis/go-gameaudio/audio"
	"github.com/Lundis/go-gameaudio/loaders/oggvorbis"
	"golang.org/x/tools/godoc/vfs"
)

// LoadFolder loads playlists from a regular folder.
// See Load for more information.
func LoadFolder(folder string) error {
	fs := vfs.OS(folder)
	return Load(fs)
}

// Load loads playlists from a virtual filesystem.
// At the root of the filesystem there must be a "playlist.json" file, which references any files to be loaded
func Load(fileSystem vfs.Opener) error {
	lock.Lock()
	defer lock.Unlock()
	start := time.Now()
	playlists, err := loadRegistry(fileSystem, "playlist.json")
	if err != nil {
		return err
	}
	workers := runtime.GOMAXPROCS(0) - 1
	if workers < 1 {
		workers = 1
	}

	type loadResult struct {
		plIdx int
		track *Track
		sound *audio.Sound
		err   error
	}

	sem := make(chan struct{}, workers)
	resultCh := make(chan loadResult)
	var wg sync.WaitGroup

	for i, pl := range playlists {
		for _, track := range pl.Tracks {
			wg.Add(1)
			go func(plIdx int, track *Track) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				raw, err := readFile(fileSystem, track.Path)
				if err != nil {
					log.Println("Failed to read music track from disk", track.Path, ":", err.Error())
					resultCh <- loadResult{plIdx: plIdx, err: err}
					return
				}
				mem, err := oggvorbis.Load(raw, audio.SampleRate())
				if err != nil {
					log.Println("Failed to decompress music", track.Path, ":", err.Error())
					resultCh <- loadResult{plIdx: plIdx, err: err}
					return
				}
				resultCh <- loadResult{
					plIdx: plIdx,
					track: track,
					sound: audio.NewSound(mem, track.Volume, audio.ChannelIdMusic),
				}
			}(i, track)
		}
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	failedPlaylists := make(map[int]bool)
	for result := range resultCh {
		if result.err != nil {
			failedPlaylists[result.plIdx] = true
		} else {
			result.track.sound = result.sound
		}
	}

	effects := make(map[Id]*PlayList, len(playlists))
	for i, pl := range playlists {
		if !failedPlaylists[i] {
			effects[pl.Id] = pl
		}
	}
	playLists = effects

	log.Printf("Loaded %d playlists in %.2fs\n", len(playLists),
		time.Since(start).Seconds())
	return nil
}
