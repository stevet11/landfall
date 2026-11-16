package main

import (
	"os"
	"testing"
)

func TestDumpHowl(t *testing.T) {
	p := os.Getenv("CRAFT_WAV")
	if p == "" {
		t.Skip()
	}
	b := synthAll()["howl"]
	h := make([]byte, 44)
	copy(h, "RIFF")
	put := func(o int, v uint32, n int) {
		for i := 0; i < n; i++ {
			h[o+i] = byte(v >> (8 * i))
		}
	}
	put(4, uint32(36+len(b)), 4)
	copy(h[8:], "WAVEfmt ")
	put(16, 16, 4)
	put(20, 1, 2)
	put(22, 2, 2)
	put(24, sampleRate, 4)
	put(28, sampleRate*4, 4)
	put(32, 4, 2)
	put(34, 16, 2)
	copy(h[36:], "data")
	put(40, uint32(len(b)), 4)
	os.WriteFile(p, append(h, b...), 0o644)
}
