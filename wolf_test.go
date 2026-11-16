package main

import "testing"

func TestAnimalsSpawnOnMainland(t *testing.T) {
	islands := 0
	for seed := int64(1); seed <= 12; seed++ {
		g := newGame(seed)
		land := g.landmass()
		// does this world even have separate land? (otherwise the test proves little)
		for i := range g.W.T {
			if g.W.T[i].Ground != GWater && g.W.Walkable(i%MapW, i/MapW) && !land[i] {
				islands++
				break
			}
		}
		g.animals = nil
		for i := 0; i < 60; i++ {
			g.spawnAnimal(AWolf, 40, 62)
			g.spawnAnimal(ADeer, 40, 62)
		}
		for _, a := range g.animals {
			if !g.onMainland(int(a.X), int(a.Y)) {
				t.Fatalf("seed %d: animal spawned off the mainland at %.0f,%.0f", seed, a.X, a.Y)
			}
			// and it can actually walk to the colony
			if _, ok := g.W.bfs(int(a.X), int(a.Y), 200, func(x, y int) bool { return x == g.start.x && y == g.start.y }); !ok {
				t.Errorf("seed %d: animal at %.0f,%.0f cannot reach the start", seed, a.X, a.Y)
			}
		}
	}
	if islands == 0 {
		t.Log("warning: no test seed had an island")
	}
	t.Logf("%d of 12 seeds had separate islands", islands)
}

func TestStrandedWolfLeaves(t *testing.T) {
	g := newGame(3)
	// put a wolf in the sea-locked corner of the map
	var x, y int
	found := false
	for i := range g.W.T {
		if g.W.Walkable(i%MapW, i/MapW) && !g.landmass()[i] {
			x, y, found = i%MapW, i/MapW, true
			break
		}
	}
	if !found {
		t.Skip("no island on this seed")
	}
	g.animals = []*Animal{{Kind: AWolf, X: float64(x) + 0.5, Y: float64(y) + 0.5, HP: 40, MaxHP: 40}}
	run(g, 5)
	for _, a := range g.animals {
		if a.Kind == AWolf {
			t.Fatal("stranded wolf should have left")
		}
	}
}

func TestUnreachableWolfGivesUp(t *testing.T) {
	g := newGame(7)
	g.animals = nil
	c := g.cols[0]
	// box the colonist in with walls so no path exists
	cx, cy := g.tileOf(c)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				g.W.At(cx+dx, cy+dy).Obj = OWall
			}
		}
	}
	for _, o := range g.cols[1:] {
		o.Dead = true
	}
	g.cols = g.cols[:1]
	g.computeRooms()
	g.animals = []*Animal{{Kind: AWolf, X: float64(cx) + 6.5, Y: float64(cy) + 0.5, HP: 40, MaxHP: 40}}
	run(g, 60)
	for _, a := range g.animals {
		if a.Kind == AWolf {
			t.Fatal("a wolf that can never reach anyone should give up")
		}
	}
}
