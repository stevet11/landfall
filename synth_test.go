package main

import (
	"testing"
	"time"
)

func TestSynth(t *testing.T) {
	st := time.Now()
	b := synthAll()
	t.Logf("synth took %v, %d sounds", time.Since(st), len(b))
	for _, n := range []string{"chop", "mine", "hammer", "rustle", "hit", "bite", "yelp", "thud", "death", "howl", "click", "save", "heal", "bird1", "bird2", "cricket", "levelup", "coin", "trader", "fire", "bray", "groan", "splash", "hiss", "shot", "rifle", "slash", "horn", "cough", "clang", "bell", "wind", "drone", "note0", "note7"} {
		if len(b[n]) < 1000 || len(b[n])%4 != 0 {
			t.Errorf("sound %q bad length %d", n, len(b[n]))
		}
	}
	// howl must be audible, unclipped and long enough to read as a call
	if h := b["howl"]; len(h) < 3*sampleRate*4 {
		t.Errorf("howl too short: %d bytes", len(h))
	} else {
		peak := 0
		for i := 0; i+1 < len(h); i += 4 {
			v := int(int16(h[i]) | int16(h[i+1])<<8)
			if v < 0 {
				v = -v
			}
			if v > peak {
				peak = v
			}
		}
		if peak < 6000 || peak > 32000 {
			t.Errorf("howl peak %d out of range", peak)
		}
	}
	// chop must not be bassy: little energy below ~250 Hz
	pcm := b["chop"]
	var low, all, y float64
	for i := 0; i+3 < len(pcm); i += 4 {
		x := float64(int16(pcm[i])|int16(pcm[i+1])<<8) / 32768
		y += 0.035 * (x - y) // one-pole lowpass, ~250 Hz at 44.1 kHz
		low += y * y
		all += x * x
	}
	if low/all > 0.05 {
		t.Errorf("chop too bassy: %.1f%% of energy is low frequency", 100*low/all)
	}
	// wind must loop without a click: first/last samples close
	w := b["wind"]
	first := int16(w[0]) | int16(w[1])<<8
	last := int16(w[len(w)-4]) | int16(w[len(w)-3])<<8
	if d := int(first) - int(last); d > 6000 || d < -6000 {
		t.Errorf("wind loop seam too large: %d", d)
	}
}
