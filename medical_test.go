package main

import "testing"

func run(g *Game, secs float64) {
	for i := 0; i < int(secs*60); i++ {
		g.step(1.0 / 60)
	}
}

func clearJobsWork(g *Game) {
	// no work to distract colonists: remove designations
	for i := range g.W.T {
		g.W.T[i].Desig = DNone
	}
}

func TestWoundIsTendedAndHeals(t *testing.T) {
	g := newGame(7)
	clearJobsWork(g)
	g.animals = nil
	c := g.cols[0]
	g.addWound(c, "Bite", 8, true)
	if !c.untended() {
		t.Fatal("new wound should be untended")
	}
	run(g, 20)
	if c.untended() {
		t.Fatalf("wound still untended after 20s: %+v", c.Injuries)
	}
	run(g, 400)
	if len(c.Injuries) != 0 {
		t.Fatalf("wound never healed: %+v", c.Injuries)
	}
	if c.Dead {
		t.Fatal("colonist died")
	}
}

func TestMedicineUsedOnSeriousWound(t *testing.T) {
	g := newGame(7)
	clearJobsWork(g)
	g.animals = nil
	sx, sy := g.start.x+2, g.start.y-1
	g.W.At(sx+2, sy+1).Item, g.W.At(sx+2, sy+1).ItemN = IMedicine, 3
	g.recompute()
	c := g.cols[1]
	g.addWound(c, "Bite", 10, true)
	run(g, 20)
	g.recompute()
	if g.stock[IMedicine] != 2 {
		t.Fatalf("expected 1 medicine used, stock=%d", g.stock[IMedicine])
	}
	if c.Injuries[0].Tended != 2 {
		t.Fatalf("wound should be treated with medicine: %+v", c.Injuries)
	}
}

func TestUntendedBleedsAndInfects(t *testing.T) {
	g := newGame(7)
	g.animals = nil
	c := &Colonist{Health: 100, Hunger: 100}
	g.addWound(c, "Bite", 12, true)
	before := c.Health
	g.updateInjuries(c, 60)
	if c.Health >= before {
		t.Fatal("untended bite should cost health")
	}
	for i := 0; i < 400; i++ {
		g.updateInjuries(c, 1)
	}
	if len(c.Injuries) > 0 && c.Injuries[0].Infect < 1 {
		t.Fatalf("untended wound should infect: %+v", c.Injuries)
	}
}

func TestSaveKeepsInjuries(t *testing.T) {
	g := newGame(7)
	g.addWound(g.cols[0], "Bite", 9, true)
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := newGame(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if len(h.cols[0].Injuries) != 1 || h.cols[0].Injuries[0].Sev != 9 {
		t.Fatalf("injuries lost: %+v", h.cols[0].Injuries)
	}
}
