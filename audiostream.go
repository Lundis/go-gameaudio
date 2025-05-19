package oto

type AudioStream interface {
	Read(p []float32) (n int, err error)
	//Seek(offset int64, whence int) (int64, error)
}
