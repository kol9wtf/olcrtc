package vp8

import (
	"bytes"
	"testing"
)

// TestBoolRoundTrip verifies the bool coder is internally consistent: anything
// we encode we can decode back. This is the foundation for VP8 compatibility.
func TestBoolRoundTrip(t *testing.T) {
	t.Parallel()
	type op struct {
		prob uint8
		val  bool
	}
	ops := make([]op, 0, 10000)
	rng := uint32(0xc0ffee)
	for i := 0; i < 10000; i++ {
		rng = rng*1103515245 + 12345
		ops = append(ops, op{prob: uint8(1 + rng%254), val: rng&0x10000 != 0})
	}
	w := newBoolWriter()
	for _, o := range ops {
		w.writeBool(o.prob, o.val)
	}
	w.flush()
	r := newBoolReader(w.bytes())
	for i, o := range ops {
		got := r.readBool(o.prob)
		if got != o.val {
			t.Fatalf("op %d: prob=%d want=%v got=%v", i, o.prob, o.val, got)
		}
	}
}

func TestBoolLitRoundTrip(t *testing.T) {
	t.Parallel()
	w := newBoolWriter()
	w.writeLit(7, 99)
	w.writeLit(14, 12345)
	w.writeLit(1, 1)
	w.writeSignedLit(5, -17)
	w.writeSignedLit(5, 17)
	w.flush()
	r := newBoolReader(w.bytes())
	if v := r.readLit(7); v != 99 {
		t.Fatalf("readLit(7)=%d want 99", v)
	}
	if v := r.readLit(14); v != 12345 {
		t.Fatalf("readLit(14)=%d want 12345", v)
	}
	if v := r.readLit(1); v != 1 {
		t.Fatalf("readLit(1)=%d want 1", v)
	}
	if v := r.readSignedLit(5); v != -17 {
		t.Fatalf("readSignedLit=-17 got %d", v)
	}
	if v := r.readSignedLit(5); v != 17 {
		t.Fatalf("readSignedLit=17 got %d", v)
	}
}

// TestBoolEmpty checks edge case: zero ops then flush.
func TestBoolEmpty(t *testing.T) {
	t.Parallel()
	w := newBoolWriter()
	w.flush()
	if len(w.bytes()) == 0 {
		t.Fatalf("expected nonempty output after flush")
	}
}

// TestBoolDeterministic verifies the encoder is deterministic.
func TestBoolDeterministic(t *testing.T) {
	t.Parallel()
	var out [2][]byte
	for k := 0; k < 2; k++ {
		w := newBoolWriter()
		for i := 0; i < 256; i++ {
			w.writeBool(128, i&1 != 0)
		}
		w.flush()
		out[k] = w.bytes()
	}
	if !bytes.Equal(out[0], out[1]) {
		t.Fatalf("encoder is not deterministic")
	}
}
