package oto

import "io"

type AudioStream interface {
	Read(p []float32) (n int, err error)
}
type SeekableAudioStream interface {
	AudioStream
	Seek(offset int64, whence int) (int64, error)
}

type MemoryReader struct {
	data []float32
	pos  int
}

func NewMemoryReader(data []float32) *MemoryReader {
	return &MemoryReader{data, 0}
}

func (r *MemoryReader) Read(p []float32) (n int, err error) {
	n = min(len(r.data)-r.pos, len(p))
	for i := 0; i < n; i++ {
		p[i] = r.data[r.pos]
		r.pos++
	}
	if r.pos >= len(r.data) {
		err = io.EOF
	}
	return
}

func (r *MemoryReader) Seek(offset int, whence int) (int, error) {
	switch whence {
	case io.SeekStart:
		r.pos = offset
	case io.SeekCurrent:
		r.pos += offset
	case io.SeekEnd:
		r.pos = len(r.data) + offset
	}
	return r.pos, nil
}
