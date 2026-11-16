package main

import "testing"

func TestDeepWinterDoesNotLockColoniesIndoorsWithoutWood(t *testing.T) {
	setup := func(wood int) (*Game, *Colonist) {
		g, c := lone(7)
		openField(g, 3, 30, 3, 12)
		for g.stock[IWood] > 0 {
			g.takeStock(IWood, g.stock[IWood])
			g.recompute()
		}
		if wood > 0 {
			g.addToStock(IWood, wood)
		}
		p := place(g, OTree, 8, 8)
		g.applyTool(TChop, p.x, p.y, p.x, p.y)
		g.outT = -14
		c.Temp, c.Hunger, c.Rest, c.Joy = 20, 100, 100, 100
		return g, c
	}
	// plenty of wood: stay in the warm room
	g, c := setup(200)
	if !g.assignJob(c) || c.Job.Kind != JNone {
		t.Fatalf("with a full woodpile the warm colonist waits out the deep cold (job %d)", c.Job.Kind)
	}
	// no wood: somebody has to go and fetch it, or the generators and heaters die
	g2, d := setup(0)
	if !g2.assignJob(d) || d.Job.Kind != JChop {
		t.Fatalf("with the woodpile empty the colonist must go chopping (job %d)", d.Job.Kind)
	}
	// the threshold is a reserve, not just 'zero'
	g3, e := setup(fuelReserve - 1)
	if !g3.assignJob(e) || e.Job.Kind != JChop {
		t.Fatal("below the reserve they go out")
	}
	g4, f := setup(fuelReserve)
	if !g4.assignJob(f) || f.Job.Kind != JNone {
		t.Fatal("at the reserve they can stay in")
	}
}

func TestShortOfWoodChoppingComesFirst(t *testing.T) {
	g, c := lone(7)
	openField(g, 3, 40, 3, 20)
	for g.stock[IWood] > 0 {
		g.takeStock(IWood, g.stock[IWood])
		g.recompute()
	}
	rock := place(g, ORock, 6, 6) // close
	g.applyTool(TMine, rock.x, rock.y, rock.x, rock.y)
	tree := place(g, OTree, 22, 14) // far
	g.applyTool(TChop, tree.x, tree.y, tree.x, tree.y)
	c.Temp, c.Hunger, c.Rest, c.Joy = 18, 100, 100, 100
	g.outT = 5
	if !g.assignJob(c) || c.Job.Kind != JChop {
		t.Fatalf("with no wood the colonist chops even though a rock is closer (job %d)", c.Job.Kind)
	}
	// with a healthy woodpile the nearer job wins as before
	g2, d := lone(7)
	openField(g2, 3, 40, 3, 20)
	g2.addToStock(IWood, 100)
	rock2 := place(g2, ORock, 6, 6)
	g2.applyTool(TMine, rock2.x, rock2.y, rock2.x, rock2.y)
	tree2 := place(g2, OTree, 22, 14)
	g2.applyTool(TChop, tree2.x, tree2.y, tree2.x, tree2.y)
	d.Temp, d.Hunger, d.Rest, d.Joy = 18, 100, 100, 100
	g2.outT = 5
	if !g2.assignJob(d) || d.Job.Kind != JMine {
		t.Fatalf("normally the nearer job comes first (job %d)", d.Job.Kind)
	}
}

func TestPowerAdviceFitsTheProblem(t *testing.T) {
	// 1. the generator is out of wood
	g := quiet(7)
	for g.stock[IWood] > 0 {
		g.takeStock(IWood, g.stock[IWood])
		g.recompute()
	}
	gen := place(g, OGen, 4, 4)
	g.W.At(gen.x, gen.y).Fuel = 0
	place(g, OLamp, 6, 4)
	place(g, OSolar, 8, 4)
	g.time = DayLen*5 + DayLen*22/24 // night: the solar panel gives nothing
	g.slow(1)
	if g.powerSupply != 0 {
		t.Fatalf("setup: supply %v", g.powerSupply)
	}
	g.hintT, g.msgs = 0, nil
	g.advise(1)
	if len(g.msgs) == 0 || !contains(g.msgs[0].Text, "out of wood") {
		t.Fatalf("with a generator that is out of wood the warning must say so, not 'build a generator': %v", g.msgs)
	}
	if contains(g.msgs[0].Text, "Build a Generator") {
		t.Fatal("telling someone who owns a generator to build one is wrong")
	}
	if r := g.noPowerReason(); !contains(r, "out of wood") {
		t.Fatal(r)
	}
	if p := g.powerProblem(OLamp); !contains(p, "out of wood") {
		t.Fatalf("the lamp tooltip says the same: %q", p)
	}

	// 2. solar panels only, at night
	h := quiet(7)
	place(h, OSolar, 8, 4)
	place(h, OLamp, 6, 4)
	h.time = DayLen*5 + DayLen*22/24
	h.slow(1)
	h.hintT, h.msgs = 0, nil
	h.advise(1)
	if len(h.msgs) == 0 || !contains(h.msgs[0].Text, "night") || !contains(h.msgs[0].Text, "Generator") {
		t.Fatalf("solar-only at night: %v", h.msgs)
	}

	// 3. nothing at all
	k := quiet(7)
	place(k, OLamp, 6, 4)
	k.slow(1)
	k.hintT, k.msgs = 0, nil
	k.advise(1)
	if len(k.msgs) == 0 || !contains(k.msgs[0].Text, "Generator or Solar") {
		t.Fatalf("no sources: %v", k.msgs)
	}

	// 4. working, but not enough
	m := quiet(7)
	place(m, OGen, 4, 4)
	for i := 0; i < 5; i++ {
		place(m, OEHeater, 8+i, 4)
	}
	m.slow(1)
	if m.powerRatio >= 0.7 {
		t.Fatalf("setup ratio %v", m.powerRatio)
	}
	m.hintT, m.msgs = 0, nil
	m.advise(1)
	if len(m.msgs) == 0 || !contains(m.msgs[0].Text, "Not enough power") {
		t.Fatalf("brown-out: %v", m.msgs)
	}

	// 5. all fine: silence
	n := quiet(7)
	place(n, OGen, 4, 4)
	place(n, OLamp, 6, 4)
	n.slow(1)
	n.hintT, n.msgs = 0, nil
	n.advise(1)
	for _, msg := range n.msgs {
		if contains(msg.Text, "power") {
			t.Fatalf("no warning when it works: %v", n.msgs)
		}
	}
}

func TestLowWoodIsAnnouncedWhenThingsBurnIt(t *testing.T) {
	g := quiet(7)
	place(g, OHeater, 6, 6)
	for g.stock[IWood] > 5 {
		g.takeStock(IWood, g.stock[IWood]-5)
		g.recompute()
	}
	g.hintT, g.msgs = 0, nil
	g.advise(1)
	found := false
	for _, m := range g.msgs {
		if contains(m.Text, "Wood is running low") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a heater with 5 wood left should trigger a warning: %v", g.msgs)
	}
	h := quiet(7)
	h.addToStock(IWood, 200)
	place(h, OHeater, 6, 6)
	h.hintT, h.msgs = 0, nil
	h.advise(1)
	for _, m := range h.msgs {
		if contains(m.Text, "running low") {
			t.Fatal("no warning with a full woodpile")
		}
	}
}

func TestNoWoodSpiralFromTheSave(t *testing.T) {
	// a colony in the deep cold with an empty woodpile and a hundred trees marked: wood comes in
	g := quiet(7)
	openField(g, -4, 30, -6, 14)
	for g.stock[IWood] > 0 {
		g.takeStock(IWood, g.stock[IWood])
		g.recompute()
	}
	for i := 0; i < 8; i++ {
		p := place(g, OTree, 10+i*2, 6)
		g.applyTool(TChop, p.x, p.y, p.x, p.y)
	}
	g.time = DayLen*7 + DayLen*5/24 // deep winter, the coldest hour
	for i := 0; i < 60*240; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols { // keep everyone alive and warm enough to behave like the sheltered colonists of the bug report
			c.Health, c.Hunger, c.Rest, c.Joy = 100, 100, 100, 100
			c.Temp = max(c.Temp, 9)
		}
	}
	g.recompute()
	if g.stock[IWood] < 8 {
		t.Fatalf("colonists must fetch wood even in deep winter: %d in stock", g.stock[IWood])
	}
}
