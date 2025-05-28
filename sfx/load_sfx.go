package sfx

import (
	"encoding/json"
	"fmt"
	"github.com/Lundis/go-gameaudio/audio"
	"github.com/Lundis/go-gameaudio/loaders/wav"
	"golang.org/x/tools/godoc/vfs"
	"io"
	"log"
	"sync"
	"time"
)

var lock sync.RWMutex

// LoadFolder loads sound effects from a regular folder.
// See Load for more information.
func LoadFolder(folder string) error {
	fs := vfs.OS(folder)
	return Load(fs)
}

// Load loads sound effects from a virtual filesystem.
// At the root of the filesystem there must be a "sfx.json" file, which references any files to be loaded
func Load(fileSystem vfs.Opener) error {
	lock.Lock()
	defer lock.Unlock()
	cachedDiskReads := make(map[string][]float32)
	start := time.Now()
	soundEffects, err := loadRegistry(fileSystem, "sfx.json")
	if err != nil {
		return err
	}
	effects := make(map[Id]*Sfx, len(soundEffects))
	for _, e := range soundEffects {
		for _, v := range e.Variations {
			mem, ok := cachedDiskReads[v.Path]
			if !ok {
				raw, err := readFile(fileSystem, v.Path)
				if err != nil {
					log.Println("Failed to read sound effect from disk", v.Path, ":", err.Error())
					continue
				}
				mem, err = wav.LoadWav(raw, audio.SampleRate())
				if err != nil {
					log.Println("Failed to decompress sound effect", v.Path, ":", err.Error())
					continue
				}
				cachedDiskReads[v.Path] = mem
			}
			v.sound = audio.NewSound(mem, e.Volume*v.Volume, audio.ChannelIdSfx)
		}
		effects[e.Id] = e
	}
	loadedSfx = effects

	log.Printf("Loaded %d sound effects in %.2fs\n", len(loadedSfx),
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

func loadRegistry(fs vfs.Opener, path string) (registry []*Sfx, err error) {
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
