package playlist

import (
	"encoding/json"
	"fmt"
	"github.com/Lundis/go-gameaudio/audio"
	"github.com/Lundis/go-gameaudio/loaders/oggvorbis"
	"golang.org/x/tools/godoc/vfs"
	"io"
	"log"
	"sync"
	"time"
)

var lock sync.RWMutex

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
	effects := make(map[Id]*PlayList, len(playlists))
playlistLoop:
	for _, pl := range playlists {
		for _, track := range pl.Tracks {
			raw, err := readFile(fileSystem, track.Path)
			if err != nil {
				log.Println("Failed to read music track from disk", track.Path, ":", err.Error())
				continue playlistLoop
			}
			mem, err := oggvorbis.Load(raw, audio.SampleRate())
			if err != nil {
				log.Println("Failed to decompress music", track.Path, ":", err.Error())
				continue playlistLoop
			}
			track.sound = audio.NewSound(mem, track.Volume, audio.ChannelIdMusic)
		}
		effects[pl.Id] = pl
	}
	playLists = effects

	log.Printf("Loaded %d playlists in %.2fs\n", len(playLists),
		time.Since(start).Seconds())
	return nil
}

func readFile(fs vfs.Opener, path string) (data []byte, err error) {
	file, err := fs.Open(path)
	if err != nil {
		return
	}
	data, err = io.ReadAll(file)
	_ = file.Close()
	return
}

func loadRegistry(fs vfs.Opener, path string) (registry []*PlayList, err error) {
	data, err := readFile(fs, path)
	if err != nil {
		err = fmt.Errorf("failed to open %s: %w", path, err)
		return
	}
	err = json.Unmarshal(data, &registry)
	if err != nil {
		err = fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return
}
