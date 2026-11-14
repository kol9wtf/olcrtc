package vp8_test

import (
	"testing"

	"codeberg.org/rape4me/kc/vp8"
)

func BenchmarkEncode64(b *testing.B) {
	gray := make([]byte, 64*64)
	for i := range gray {
		gray[i] = byte(i)
	}
	enc := vp8.NewEncoder(64, 64, 4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Encode(gray)
	}
}

func BenchmarkEncode1024(b *testing.B) {
	gray := make([]byte, 1024*1024)
	for i := range gray {
		gray[i] = byte(i)
	}
	enc := vp8.NewEncoder(1024, 1024, 4)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Encode(gray)
	}
}

func BenchmarkDecode64(b *testing.B) {
	gray := make([]byte, 64*64)
	for i := range gray {
		gray[i] = byte(i)
	}
	enc := vp8.NewEncoder(64, 64, 4)
	frame, _ := enc.Encode(gray)
	dec := vp8.NewDecoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec.Decode(frame)
	}
}

func BenchmarkDecode1024(b *testing.B) {
	gray := make([]byte, 1024*1024)
	for i := range gray {
		gray[i] = byte(i)
	}
	enc := vp8.NewEncoder(1024, 1024, 4)
	frame, _ := enc.Encode(gray)
	dec := vp8.NewDecoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec.Decode(frame)
	}
}

func BenchmarkDecodeInter1024(b *testing.B) {
	gray := make([]byte, 1024*1024)
	for i := range gray {
		gray[i] = byte(i)
	}
	enc := vp8.NewEncoder(1024, 1024, 4)
	enc.SetKeyInterval(999)
	// First frame = keyframe.
	enc.Encode(gray)
	// Second frame = inter (mostly skip since identical).
	interFrame, _ := enc.Encode(gray)
	dec := vp8.NewDecoder()
	// Prime decoder with keyframe.
	keyFrame, _ := vp8.NewEncoder(1024, 1024, 4).Encode(gray)
	dec.Decode(keyFrame)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec.Decode(interFrame)
	}
}
