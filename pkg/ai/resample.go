package ai

import (
	"encoding/binary"
	"math"
	"sync"
)

// CalculateRMS computes the Root Mean Square energy of 16-bit PCM little-endian audio.
func CalculateRMS(pcmData []byte) float64 {
	numSamples := len(pcmData) / 2
	if numSamples == 0 {
		return 0
	}

	var sumSq float64
	for i := 0; i < len(pcmData)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(pcmData[i : i+2]))
		sumSq += float64(sample) * float64(sample)
	}

	return math.Sqrt(sumSq / float64(numSamples))
}

// CalculateDBFS computes decibels relative to full scale (-inf to 0 dBFS) for 16-bit PCM.
func CalculateDBFS(rms float64) float64 {
	if rms <= 0 {
		return -100.0
	}
	db := 20 * math.Log10(rms/32767.0)
	if db < -100.0 {
		return -100.0
	}
	return db
}

// ResamplePCM converts 16-bit mono little-endian PCM between fromRate and toRate using linear interpolation.
func ResamplePCM(pcmData []byte, fromRate, toRate int) []byte {
	if fromRate == toRate || len(pcmData) < 2 {
		return pcmData
	}

	numInSamples := len(pcmData) / 2
	if numInSamples == 0 {
		return nil
	}

	// Fast paths for common telephony and AI sample rates
	if fromRate == 8000 && toRate == 16000 {
		return Resample8kTo16k(pcmData)
	}
	if fromRate == 16000 && toRate == 8000 {
		return Resample16kTo8k(pcmData)
	}
	if fromRate == 24000 && toRate == 8000 {
		return Resample24kTo8k(pcmData)
	}
	if fromRate == 24000 && toRate == 16000 {
		return Resample24kTo16k(pcmData)
	}
	if fromRate == 16000 && toRate == 24000 {
		return Resample16kTo24k(pcmData)
	}

	// Generic linear interpolation
	numOutSamples := int(float64(numInSamples) * float64(toRate) / float64(fromRate))
	if numOutSamples == 0 {
		return nil
	}

	inSamples := make([]int16, numInSamples)
	for i := 0; i < numInSamples; i++ {
		inSamples[i] = int16(binary.LittleEndian.Uint16(pcmData[i*2 : i*2+2]))
	}

	outData := make([]byte, numOutSamples*2)
	ratio := float64(fromRate) / float64(toRate)

	for i := 0; i < numOutSamples; i++ {
		srcPos := float64(i) * ratio
		srcIdx := int(srcPos)
		frac := srcPos - float64(srcIdx)

		var s1, s2 int16
		if srcIdx < numInSamples {
			s1 = inSamples[srcIdx]
		}
		if srcIdx+1 < numInSamples {
			s2 = inSamples[srcIdx+1]
		} else {
			s2 = s1
		}

		interpVal := float64(s1)*(1.0-frac) + float64(s2)*frac
		// Clamp to int16
		if interpVal > 32767 {
			interpVal = 32767
		} else if interpVal < -32768 {
			interpVal = -32768
		}

		binary.LittleEndian.PutUint16(outData[i*2:i*2+2], uint16(int16(interpVal)))
	}

	return outData
}

// Resample8kTo16k upsamples 8kHz L16 PCM to 16kHz L16 PCM.
func Resample8kTo16k(pcm8k []byte) []byte {
	numInSamples := len(pcm8k) / 2
	if numInSamples == 0 {
		return nil
	}

	outData := make([]byte, numInSamples*4)
	for i := 0; i < numInSamples; i++ {
		s1 := int16(binary.LittleEndian.Uint16(pcm8k[i*2 : i*2+2]))
		var s2 int16
		if i+1 < numInSamples {
			s2 = int16(binary.LittleEndian.Uint16(pcm8k[(i+1)*2 : (i+1)*2+2]))
		} else {
			s2 = s1
		}

		mid := int16((int32(s1) + int32(s2)) / 2)

		outIdx := i * 4
		binary.LittleEndian.PutUint16(outData[outIdx:outIdx+2], uint16(s1))
		binary.LittleEndian.PutUint16(outData[outIdx+2:outIdx+4], uint16(mid))
	}
	return outData
}

// Resample16kTo8k downsamples 16kHz L16 PCM to 8kHz L16 PCM.
func Resample16kTo8k(pcm16k []byte) []byte {
	numInSamples := len(pcm16k) / 2
	numOutSamples := numInSamples / 2
	if numOutSamples == 0 {
		return nil
	}

	outData := make([]byte, numOutSamples*2)
	for i := 0; i < numOutSamples; i++ {
		s1 := int16(binary.LittleEndian.Uint16(pcm16k[i*4 : i*4+2]))
		s2 := int16(binary.LittleEndian.Uint16(pcm16k[i*4+2 : i*4+4]))
		avg := int16((int32(s1) + int32(s2)) / 2)
		binary.LittleEndian.PutUint16(outData[i*2:i*2+2], uint16(avg))
	}
	return outData
}

// Resample24kTo8k downsamples 24kHz L16 PCM to 8kHz L16 PCM (factor 3).
func Resample24kTo8k(pcm24k []byte) []byte {
	numInSamples := len(pcm24k) / 2
	numOutSamples := numInSamples / 3
	if numOutSamples == 0 {
		return nil
	}

	outData := make([]byte, numOutSamples*2)
	for i := 0; i < numOutSamples; i++ {
		s1 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6 : i*6+2])))
		s2 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6+2 : i*6+4])))
		s3 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6+4 : i*6+6])))
		avg := int16((s1 + s2 + s3) / 3)
		binary.LittleEndian.PutUint16(outData[i*2:i*2+2], uint16(avg))
	}
	return outData
}

// Resample24kTo16k downsamples 24kHz L16 PCM to 16kHz L16 PCM (ratio 3:2).
func Resample24kTo16k(pcm24k []byte) []byte {
	numInSamples := len(pcm24k) / 2
	triplets := numInSamples / 3
	if triplets == 0 {
		return nil
	}

	outData := make([]byte, triplets*4)
	for i := 0; i < triplets; i++ {
		s0 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6 : i*6+2])))
		s1 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6+2 : i*6+4])))
		s2 := int32(int16(binary.LittleEndian.Uint16(pcm24k[i*6+4 : i*6+6])))

		// Out 0 is closer to s0 and s1: (2*s0 + s1)/3
		out0 := int16((2*s0 + s1) / 3)
		// Out 1 is closer to s1 and s2: (s1 + 2*s2)/3
		out1 := int16((s1 + 2*s2) / 3)

		binary.LittleEndian.PutUint16(outData[i*4:i*4+2], uint16(out0))
		binary.LittleEndian.PutUint16(outData[i*4+2:i*4+4], uint16(out1))
	}
	return outData
}

// Resample16kTo24k upsamples 16kHz L16 PCM to 24kHz L16 PCM (ratio 2:3).
func Resample16kTo24k(pcm16k []byte) []byte {
	numInSamples := len(pcm16k) / 2
	pairs := numInSamples / 2
	if pairs == 0 {
		return nil
	}

	outData := make([]byte, pairs*6)
	for i := 0; i < pairs; i++ {
		s0 := int32(int16(binary.LittleEndian.Uint16(pcm16k[i*4 : i*4+2])))
		s1 := int32(int16(binary.LittleEndian.Uint16(pcm16k[i*4+2 : i*4+4])))

		out0 := int16(s0)
		out1 := int16((s0 + 2*s1) / 3)
		out2 := int16(s1)

		binary.LittleEndian.PutUint16(outData[i*6:i*6+2], uint16(out0))
		binary.LittleEndian.PutUint16(outData[i*6+2:i*6+4], uint16(out1))
		binary.LittleEndian.PutUint16(outData[i*6+4:i*6+6], uint16(out2))
	}
	return outData
}

// AudioFramer buffers incoming audio and slices it into fixed-size frames (e.g. 20ms chunks).
type AudioFramer struct {
	frameSize int
	buf       []byte
	mu        sync.Mutex
}

// NewAudioFramer creates a framer that outputs chunks of frameBytes.
// For example, 20ms @ 8kHz 16-bit mono = 320 bytes.
// 20ms @ 16kHz 16-bit mono = 640 bytes.
func NewAudioFramer(frameBytes int) *AudioFramer {
	if frameBytes <= 0 {
		frameBytes = 320
	}
	return &AudioFramer{
		frameSize: frameBytes,
		buf:       make([]byte, 0, frameBytes*4),
	}
}

// Push appends new audio bytes and returns all complete frames available.
func (f *AudioFramer) Push(data []byte) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buf = append(f.buf, data...)
	var frames [][]byte

	for len(f.buf) >= f.frameSize {
		frame := make([]byte, f.frameSize)
		copy(frame, f.buf[:f.frameSize])
		f.buf = f.buf[f.frameSize:]
		frames = append(frames, frame)
	}

	return frames
}

// Flush returns any remaining partial frame if present, padding with zero silence to frameSize.
func (f *AudioFramer) Flush() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.buf) == 0 {
		return nil
	}

	frame := make([]byte, f.frameSize)
	copy(frame, f.buf)
	f.buf = f.buf[:0]
	return frame
}

// Reset clears the internal framer buffer.
func (f *AudioFramer) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buf = f.buf[:0]
}
