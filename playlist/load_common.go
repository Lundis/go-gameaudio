package playlist

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"golang.org/x/tools/godoc/vfs"
)

var lock sync.RWMutex

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
