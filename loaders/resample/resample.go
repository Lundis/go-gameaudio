package resample

// Stereo resamples stereo interleaved float32 audio from srcRate to dstRate using linear interpolation.
func Stereo(src []float32, srcRate, dstRate int) []float32 {
	if srcRate == dstRate {
		return src
	}
	srcFrames := len(src) / 2
	dstFrames := int(int64(srcFrames) * int64(dstRate) / int64(srcRate))
	dst := make([]float32, dstFrames*2)
	for i := 0; i < dstFrames; i++ {
		// position in source frames (fractional)
		srcPos := float64(i) * float64(srcRate) / float64(dstRate)
		lo := int(srcPos)
		hi := lo + 1
		frac := float32(srcPos - float64(lo))
		if hi >= srcFrames {
			hi = srcFrames - 1
		}
		dst[i*2] = src[lo*2]*(1-frac) + src[hi*2]*frac
		dst[i*2+1] = src[lo*2+1]*(1-frac) + src[hi*2+1]*frac
	}
	return dst
}
