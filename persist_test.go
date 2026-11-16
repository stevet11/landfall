package main

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	g := newGame(7)
	selfSetup(g)
	for i := 0; i < 60*400; i++ {
		g.step(1.0 / 60)
	}
	path := filepath.Join(t.TempDir(), "x.sav")
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := newGame(99) // different world; load must replace it entirely
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if len(h.cols) != len(g.cols) || len(h.animals) != len(g.animals) || h.time != g.time {
		t.Fatalf("mismatch: cols %d/%d animals %d/%d", len(h.cols), len(g.cols), len(h.animals), len(g.animals))
	}
	for i := range g.W.T {
		a, b := g.W.T[i], h.W.T[i]
		a.ResT, a.ResD, a.ResI = 0, 0, 0
		a.Item, a.ItemN, b.Item, b.ItemN = 0, 0, 0, 0 // carried items get dropped on load
		if a != b {
			t.Fatalf("tile %d differs: %+v vs %+v", i, a, b)
		}
	}
	total := func(g *Game) (n [numItems]int) {
		for i := range g.W.T {
			if t := &g.W.T[i]; t.Item != INone {
				n[t.Item] += t.ItemN
			}
		}
		for _, c := range g.cols {
			if c.Carry != INone {
				n[c.Carry] += c.CarryN
			}
		}
		return
	}
	if total(h) != total(g) {
		t.Fatalf("items not conserved: %v vs %v", total(h), total(g))
	}
	for i := 0; i < 60*120; i++ { // loaded game keeps running
		h.step(1.0 / 60)
	}
	if h.stock[IWood] == 0 && h.stock[IFood] == 0 {
		t.Fatal("colony did nothing after load")
	}
}
