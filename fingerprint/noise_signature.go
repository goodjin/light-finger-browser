package fingerprint

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
)

const (
	canvasSignatureWidth  = 200
	canvasSignatureHeight = 50
	audioSignatureSamples = 128
)

func canvasNoiseHash(seed string) string {
	seedValue := canvasFingerprintSeed(seed)
	hasher := fnv.New32a()
	for y := 0; y < canvasSignatureHeight; y++ {
		for x := 0; x < canvasSignatureWidth; x++ {
			mix := mixCanvasFingerprintSeed(seedValue, uint32(x), uint32(y))
			noise := byte(mix & 0x07)
			_, _ = hasher.Write([]byte{noise})
		}
	}
	return fmt.Sprintf("%x", hasher.Sum32())
}

func audioNoiseHash(seed string) string {
	override := audioFingerprintOverrideFromSeed(seed)
	hasher := fnv.New32a()
	var buf [4]byte
	for i := 0; i < audioSignatureSamples; i++ {
		sample := audioSignatureBaseSample(i)
		adjusted := applyAudioFingerprintSample(sample, override)
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(adjusted))
		_, _ = hasher.Write(buf[:])
	}
	return fmt.Sprintf("%x", hasher.Sum32())
}

func audioSignatureBaseSample(i int) float32 {
	return float32((i%10)-5) / 10.0
}

type audioFingerprintOverride struct {
	enabled bool
	offset  float32
	scale   float32
}

func audioFingerprintOverrideFromSeed(seed string) audioFingerprintOverride {
	if seed == "" {
		return audioFingerprintOverride{enabled: false, offset: 0, scale: 1}
	}
	seedInt := fingerprintSeedInt(seed)
	offset := float32((seedInt%1000)-500) / 100000.0
	return audioFingerprintOverride{enabled: true, offset: offset, scale: 1}
}

func applyAudioFingerprintSample(value float32, override audioFingerprintOverride) float32 {
	if !override.enabled {
		return value
	}
	adjusted := value*override.scale + override.offset
	if adjusted > 1.0 {
		return 1.0
	}
	if adjusted < -1.0 {
		return -1.0
	}
	return adjusted
}

func fingerprintSeedInt(seed string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))
	return int(hasher.Sum32() & 0x7fffffff)
}

func canvasFingerprintSeed(seedValue string) uint64 {
	if seedValue == "" {
		return 0
	}
	if seed, err := strconv.ParseUint(seedValue, 10, 64); err == nil {
		return seed
	}
	var seed uint64 = 1469598103934665603
	for i := 0; i < len(seedValue); i++ {
		seed ^= uint64(seedValue[i])
		seed *= 1099511628211
	}
	return seed
}

func mixCanvasFingerprintSeed(seed uint64, x uint32, y uint32) uint64 {
	value := seed ^ (uint64(x) << 32) ^ uint64(y) ^ 0x9e3779b97f4a7c15
	value ^= value >> 33
	value *= 0xff51afd7ed558ccd
	value ^= value >> 33
	value *= 0xc4ceb9fe1a85ec53
	value ^= value >> 33
	return value
}
