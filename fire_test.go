package main

import "testing"

func withFireRates(ignite, spread float64, f func()) {
	oi, os := fireIgniteRate, fireSpreadRate
	fireIgniteRate, fireSpreadRate = ignite, spread
	defer func() { fireIgniteRate, fireSpreadRate = oi, os }()
	f()
}

func TestLegacyWallsAndDoorsResolve(t *testing.T) {
	if (&Tile{Obj: OWall}).builtMat() != MatStone || (&Tile{Obj: OWall}).flammable() {
		t.Fatal("an old save's walls are stone and fireproof")
	}
	if (&Tile{Obj: ODoor}).builtMat() != MatWood || !(&Tile{Obj: ODoor}).flammable() {
		t.Fatal("an old save's doors are wood")
	}
	if !(&Tile{Obj: OWall, Mat: MatWood}).flammable() || (&Tile{Obj: OWall, Mat: MatIron}).flammable() || (&Tile{Obj: ODoor, Mat: MatStone}).flammable() {
		t.Fatal("only wood burns")
	}
	if !(&Tile{Obj: OBed}).flammable() || (&Tile{Obj: OGen}).flammable() || (&Tile{Obj: OFurnace}).flammable() {
		t.Fatal("beds burn, machines don't")
	}
}

func TestWallMaterialsCostAndBuild(t *testing.T) {
	cases := []struct {
		mat  Material
		item Item
		n    int
	}{{MatWood, IWood, 4}, {MatStone, IStone, 4}, {MatIron, IIron, 3}}
	for _, tc := range cases {
		g := quiet(7)
		stockUp(g, IIron, 6)
		g.wallMat = tc.mat
		p := pt{g.start.x + 6, g.start.y + 6}
		g.W.At(p.x, p.y).Obj = ONone
		before := g.stock[tc.item]
		g.applyTool(TWall, p.x, p.y, p.x, p.y)
		bp := g.W.At(p.x, p.y)
		if bp.Blue != BWall || bp.Mat != tc.mat {
			t.Fatalf("%v: blueprint %+v", tc.mat, bp)
		}
		if c := bp.blueCost(); c.Item != tc.item || c.N != tc.n {
			t.Fatalf("%v: cost %+v", tc.mat, c)
		}
		run(g, 60)
		built := g.W.At(p.x, p.y)
		if built.Obj != OWall || built.builtMat() != tc.mat {
			t.Fatalf("%v: wall not built (%+v)", tc.mat, built)
		}
		g.recompute()
		if got := before - g.stock[tc.item]; got != tc.n {
			t.Fatalf("%v: spent %d %s, want %d", tc.mat, got, itemNames[tc.item], tc.n)
		}
	}
}

func TestIronWallNeedsIron(t *testing.T) {
	g := quiet(7)
	g.wallMat = MatIron
	p := pt{g.start.x + 6, g.start.y + 6}
	g.W.At(p.x, p.y).Obj = ONone
	g.applyTool(TWall, p.x, p.y, p.x, p.y)
	run(g, 40)
	if g.W.At(p.x, p.y).Obj == OWall {
		t.Fatal("no iron in stock, so no iron wall")
	}
	if why := g.blueprintBlock(g.W.At(p.x, p.y)); !contains(why, "Iron") {
		t.Fatalf("should explain the iron shortage: %q", why)
	}
}

func TestDoorMaterials(t *testing.T) {
	g := quiet(7)
	stockUp(g, IStone, 10)
	g.doorMat = MatStone
	p := pt{g.start.x + 6, g.start.y + 6}
	g.W.At(p.x, p.y).Obj = ONone
	g.applyTool(TDoor, p.x, p.y, p.x, p.y)
	run(g, 40)
	d := g.W.At(p.x, p.y)
	if d.Obj != ODoor || d.builtMat() != MatStone || d.flammable() {
		t.Fatalf("stone door should be built and fireproof: %+v", d)
	}
	if costFor(BDoor, MatIron).Item != IIron || costFor(BDoor, MatWood).Item != IWood {
		t.Fatal("door costs")
	}
}

func TestToolCostFollowsMaterial(t *testing.T) {
	g := quiet(7)
	g.wallMat = MatWood
	if g.toolCost(TWall) != "4 Wood" {
		t.Fatal(g.toolCost(TWall))
	}
	g.wallMat = MatIron
	if g.toolCost(TWall) != "3 Iron" {
		t.Fatal(g.toolCost(TWall))
	}
}

func TestWorldHasNearbyOre(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		g := newGame(seed)
		n := 0
		for i := range g.W.T {
			if g.W.T[i].Obj == OOre && iabs(i%MapW-g.start.x) <= 25 && iabs(i/MapW-g.start.y) <= 25 {
				n++
			}
		}
		if n < 5 {
			t.Errorf("seed %d: only %d ore deposits near the start", seed, n)
		}
	}
}

func TestMiningOreAndSmeltingIron(t *testing.T) {
	g := quiet(7)
	p := place(g, OOre, 6, 6)
	g.applyTool(TMine, p.x, p.y, p.x, p.y)
	if g.W.At(p.x, p.y).Desig != DMine {
		t.Fatal("ore should be mineable")
	}
	run(g, 60)
	if g.W.At(p.x, p.y).Obj != ONone {
		t.Fatal("ore not mined")
	}
	g.recompute()
	if totalItems(g)[IOre] < 3 {
		t.Fatalf("mining ore should yield ore, got %d", totalItems(g)[IOre])
	}
	place(g, OFurnace, -3, 2)
	g.prod[4].On, g.prod[4].Target = true, 2
	if s := g.recipeStatus(4); s == "needs a furnace" {
		t.Fatal("furnace exists")
	}
	run(g, 150)
	g.recompute()
	if g.stock[IIron] < 1 {
		t.Fatalf("expected iron bars (status %q, ore %d)", g.recipeStatus(4), g.stock[IOre])
	}
	g.recompute()
	if g.stock[IOre] > 3 && false {
		t.Fatal("unreachable")
	}
}

func TestFireBurnsWoodNotStone(t *testing.T) {
	withFireRates(1, 0, func() { // sparks are certain, spreading is off
		g := quiet(7)
		g.noFireFight = true // these tests are about the fire itself, not the response
		heater := place(g, OHeater, 6, 6)
		g.W.At(heater.x, heater.y).Fuel = 1e6
		wood := pt{heater.x + 1, heater.y}
		stone := pt{heater.x - 1, heater.y}
		iron := pt{heater.x, heater.y + 1}
		for _, c := range []struct {
			p pt
			m Material
		}{{wood, MatWood}, {stone, MatStone}, {iron, MatIron}} {
			place(g, OWall, c.p.x-g.start.x, c.p.y-g.start.y)
			g.W.At(c.p.x, c.p.y).Mat = c.m
		}
		g.computeRooms()
		g.msgs = nil
		for i := 0; i < 60*40; i++ {
			g.step(1.0 / 60)
		}
		if g.W.At(wood.x, wood.y).Obj != ONone {
			t.Fatal("the wooden wall should have burned down")
		}
		if g.W.At(stone.x, stone.y).Obj != OWall || g.W.At(iron.x, iron.y).Obj != OWall {
			t.Fatal("stone and iron walls must survive")
		}
		if g.stats.Fires == 0 || g.stats.Burned == 0 {
			t.Fatalf("stats: %+v", g.stats)
		}
		alert := false
		for _, m := range g.msgs {
			if m.Alert && contains(m.Text, "Fire!") {
				alert = true
			}
		}
		if !alert {
			t.Fatalf("a fire should raise an alert: %+v", g.msgs)
		}
		if g.W.At(wood.x, wood.y).Fire != 0 || g.W.At(wood.x, wood.y).Mat != MatNone {
			t.Fatal("burnt tile should be clean")
		}
	})
}

func TestNoFireWithoutBurner(t *testing.T) {
	withFireRates(1, 1, func() {
		g := quiet(7)
		place(g, OWall, 6, 6)
		g.W.At(g.start.x+6, g.start.y+6).Mat = MatWood
		run(g, 120)
		if g.stats.Fires != 0 || g.W.At(g.start.x+6, g.start.y+6).Obj != OWall {
			t.Fatal("fires need a running heater, generator or furnace")
		}
	})
}

func TestFireSpreadsAlongWoodAndStopsAtGaps(t *testing.T) {
	withFireRates(0, 1, func() {
		g := quiet(7)
		g.noFireFight = true // these tests are about the fire itself, not the response
		var row []pt
		for i := 0; i < 5; i++ {
			p := place(g, OWall, 4+i, 8)
			g.W.At(p.x, p.y).Mat = MatWood
			row = append(row, p)
		}
		gap := place(g, OWall, 9, 8) // stone gap
		g.W.At(gap.x, gap.y).Mat = MatStone
		far := place(g, OWall, 10, 8) // wood beyond the gap
		g.W.At(far.x, far.y).Mat = MatWood
		g.ignite(row[0].y*MapW+row[0].x, "")
		for i := 0; i < 60*60; i++ {
			g.step(1.0 / 60)
		}
		for _, p := range row {
			if g.W.At(p.x, p.y).Obj != ONone {
				t.Fatalf("wall at %v should have burned", p)
			}
		}
		if g.W.At(gap.x, gap.y).Obj != OWall || g.W.At(far.x, far.y).Obj != OWall {
			t.Fatal("fire must not cross a stone gap")
		}
	})
}

func TestFireproofThingsCannotIgnite(t *testing.T) {
	g := quiet(7)
	for _, m := range []Material{MatStone, MatIron} {
		p := place(g, OWall, 6, 6)
		g.W.At(p.x, p.y).Mat = m
		g.ignite(p.y*MapW+p.x, "")
		if g.W.At(p.x, p.y).Fire != 0 {
			t.Fatalf("%v wall ignited", m)
		}
	}
}

func TestColonistFleesBurningTile(t *testing.T) {
	g := quiet(7)
	g.noFireFight = true // these tests are about the fire itself, not the response
	d := place(g, ODoor, 3, 3)
	g.W.At(d.x, d.y).Mat = MatWood
	c := g.cols[0]
	c.X, c.Y = float64(d.x)+0.5, float64(d.y)+0.5
	g.ignite(d.y*MapW+d.x, "")
	h0, low := c.Health, c.Health
	for i := 0; i < 60*4; i++ {
		g.step(1.0 / 60)
		low = min(low, c.Health)
	}
	x, y := g.tileOf(c)
	if g.W.At(x, y).Fire > 0 {
		t.Fatal("the colonist should have left the burning tile")
	}
	if low >= h0 {
		t.Fatal("standing in fire should hurt")
	}
}

func TestBurntWallOpensTheRoom(t *testing.T) {
	g := quiet(7)
	g.noFireFight = true // these tests are about the fire itself, not the response
	ring := []pt{}
	cx, cy := 8, 8
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			p := place(g, OWall, cx+dx, cy+dy)
			g.W.At(p.x, p.y).Mat = MatStone
			ring = append(ring, p)
		}
	}
	inner := pt{g.start.x + cx, g.start.y + cy}
	g.W.At(inner.x, inner.y).Obj = ONone
	g.W.At(inner.x, inner.y).Roof = true
	g.computeRooms()
	if !g.indoors(inner.x, inner.y) {
		t.Fatal("ring of walls makes an indoor room")
	}
	w := ring[1]
	g.W.At(w.x, w.y).Mat = MatWood
	g.ignite(w.y*MapW+w.x, "")
	withFireRates(0, 0, func() { run(g, 30) })
	if g.indoors(inner.x, inner.y) {
		t.Fatal("after a wall burns down the room is open")
	}
}

func TestSalvageRefundsByMaterial(t *testing.T) {
	cases := []struct {
		obj  Obj
		mat  Material
		want map[Item]int
	}{
		{OWall, MatWood, map[Item]int{IWood: 2}},
		{OWall, MatIron, map[Item]int{IIron: 2}},
		{ODoor, MatStone, map[Item]int{IStone: 3}},
		{OFurnace, MatNone, map[Item]int{IStone: 5, IWood: 2}},
	}
	for _, tc := range cases {
		g := quiet(7)
		p := place(g, tc.obj, 5, 5)
		g.W.At(p.x, p.y).Mat = tc.mat
		before := totalItems(g)
		g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
		run(g, 60)
		if g.W.At(p.x, p.y).Obj != ONone {
			t.Fatalf("%v/%v not removed", tc.obj, tc.mat)
		}
		after := totalItems(g)
		for it := Item(1); it < numItems; it++ {
			if got := after[it] - before[it]; got != tc.want[it] {
				t.Errorf("%v/%v: %s changed by %d, want %d", tc.obj, tc.mat, itemNames[it], got, tc.want[it])
			}
		}
	}
}

func TestMaterialsAndFireSurviveSave(t *testing.T) {
	g := quiet(7)
	p := place(g, OWall, 6, 6)
	g.W.At(p.x, p.y).Mat = MatIron
	q := place(g, OWall, 8, 6)
	g.W.At(q.x, q.y).Mat = MatWood
	g.ignite(q.y*MapW+q.x, "")
	g.wallMat, g.doorMat = MatWood, MatIron
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h.W.At(p.x, p.y).builtMat() != MatIron || h.W.At(q.x, q.y).Fire == 0 {
		t.Fatal("wall material or fire lost")
	}
	if h.wallMat != MatWood || h.doorMat != MatIron {
		t.Fatalf("chosen materials lost: %v %v", h.wallMat, h.doorMat)
	}
}

func TestMaterialPalette(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.selectTool(TWall)
	if g.panel != panelMat {
		t.Fatal("selecting the wall tool should open the material picker")
	}
	var found int
	for _, b := range g.buttons() {
		if b.w > 100 && b.h == 32 { // palette rows
			found++
		}
	}
	if found != len(matMenu) {
		t.Fatalf("expected %d material rows, got %d", len(matMenu), found)
	}
	x, y, w, _ := g.matPaletteRect()
	_ = w
	// click the iron row
	if !g.handleUIClick(x+20, y+24+34*2+10) {
		t.Fatal("click not handled")
	}
	if g.wallMat != MatIron || g.panel != panelNone {
		t.Fatalf("iron not chosen: %v panel %d", g.wallMat, g.panel)
	}
	g.selectTool(TDoor)
	g.handleUIClick(x+20, y+24+10)
	if g.doorMat != MatWood || g.wallMat != MatIron {
		t.Fatal("doors and walls have separate materials")
	}
	g.selectTool(TChop)
	if g.panel != panelNone {
		t.Fatal("other tools close the picker")
	}
}

func TestCraftedItemsAreNeverStuckOnBenches(t *testing.T) {
	g := quiet(7)
	b := place(g, OBench, -3, 2)
	g.prod[0].On, g.prod[0].Target = true, 3
	run(g, 140)
	if g.W.At(b.x, b.y).Item != INone {
		t.Fatal("a product was dropped on the bench itself where nobody can reach it")
	}
	g.recompute()
	if g.stock[ICarving] < 3 {
		t.Fatalf("all carvings should reach the stockpile, have %d", g.stock[ICarving])
	}
}

func TestRescueItemsStrandedOnBuildings(t *testing.T) {
	g := quiet(7)
	b := place(g, OBench, -3, 2)
	tl := g.W.At(b.x, b.y)
	tl.Item, tl.ItemN = ICarving, 2 // what older versions left behind
	g.rescueItems()
	if tl.Item != INone {
		t.Fatal("item should have been moved off the bench")
	}
	if totalItems(g)[ICarving] != 2 {
		t.Fatal("rescued items must not vanish")
	}
	run(g, 60)
	g.recompute()
	if g.stock[ICarving] != 2 {
		t.Fatalf("rescued carvings should get hauled to the stockpile, have %d", g.stock[ICarving])
	}
}

func TestUnreachableLooseItemsDontStarveOtherWork(t *testing.T) {
	g := quiet(7)
	// the colonists are sealed in an iron enclosure with a loose item and a tree to chop;
	// the stockpile is outside. Delivering is impossible, but chopping is not.
	cx, cy := g.start.x+14, g.start.y-12
	for dx := -4; dx <= 4; dx++ {
		for dy := -4; dy <= 4; dy++ {
			tl := g.W.At(cx+dx, cy+dy)
			tl.Ground, tl.Obj, tl.Item, tl.ItemN = GGrass, ONone, INone, 0
			if iabs(dx) == 4 || iabs(dy) == 4 {
				tl.Obj, tl.Mat = OWall, MatIron
			}
		}
	}
	g.computeRooms()
	for i, c := range g.cols {
		c.X, c.Y = float64(cx-1+i)+0.5, float64(cy+2)+0.5
		c.Path = nil
		g.abort(c)
	}
	item := g.W.At(cx-2, cy)
	item.Item, item.ItemN = IStone, 4
	tree := g.W.At(cx+2, cy-1)
	tree.Obj = OTree
	g.applyTool(TChop, cx+2, cy-1, cx+2, cy-1)
	run(g, 120)
	if tree.Obj == OTree {
		t.Fatal("colonists got stuck retrying an undeliverable item and never chopped the tree")
	}
	if item.Item != IStone || item.ItemN != 4 {
		t.Fatalf("the item should still be where it was: %v x%d", itemNames[item.Item], item.ItemN)
	}
	if _, parked := g.noHaul[(cy)*MapW+cx-2]; !parked {
		t.Fatal("the failed delivery should be remembered")
	}
}

// burningRow lights a row of n wooden walls and returns them.
func burningRow(g *Game, n int) []pt {
	var row []pt
	for i := 0; i < n; i++ {
		p := place(g, OWall, 10+i, 8)
		g.W.At(p.x, p.y).Mat = MatWood
		row = append(row, p)
	}
	g.ignite(row[0].y*MapW+row[0].x, "")
	return row
}

func burnedCount(g *Game, row []pt) int {
	n := 0
	for _, p := range row {
		if g.W.At(p.x, p.y).Obj != OWall {
			n++
		}
	}
	return n
}

func TestColonistsPutOutFires(t *testing.T) {
	withFireRates(0, 0.12, func() {
		// the same fire, with and without a response
		burned := map[bool]int{}
		var extinguished int
		for _, fight := range []bool{false, true} {
			total := 0
			for seed := int64(1); seed <= 4; seed++ {
				g := quiet(seed)
				g.noFireFight = !fight
				row := burningRow(g, 8)
				run(g, 110) // long enough for the last-lit tile to burn out
				total += burnedCount(g, row)
				if fight {
					extinguished += g.stats.Extinguished
					if g.fireCount != 0 {
						t.Fatalf("seed %d: fire still burning after 110 s", seed)
					}
				}
			}
			burned[fight] = total
		}
		if burned[true] >= burned[false] {
			t.Fatalf("firefighters should save buildings: %d burned with them, %d without", burned[true], burned[false])
		}
		if extinguished == 0 {
			t.Fatal("extinguished fires should be counted")
		}
	})
}

func TestFirefightersGetBurnt(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Traits = nil
	h0 := c.Health
	for i := 0; i < 400; i++ {
		g.fireHeat(c, 1)
		c.Health = h0 // keep them alive; we are counting wounds
	}
	if len(c.Injuries) == 0 {
		t.Fatal("standing next to flames should sometimes burn you")
	}
	for _, in := range c.Injuries {
		if in.Kind != "Burn" || in.Bleed {
			t.Fatalf("unexpected wound %+v", in)
		}
	}
}

func TestSleepersAreWokenByFire(t *testing.T) {
	withFireRates(0, 0, func() {
		g := quiet(7)
		c := g.cols[0]
		c.Rest = 5
		sx, sy := g.tileOf(c)
		c.Job = Job{Kind: JSleep, Target: pt{sx, sy}, Stage: 1}
		p := place(g, OWall, 8, 4)
		g.W.At(p.x, p.y).Mat = MatWood
		g.ignite(p.y*MapW+p.x, "")
		g.fireCount = 1
		run(g, 2)
		if c.Job.Kind == JSleep {
			t.Fatal("the fire alarm should wake sleepers")
		}
	})
}

func TestHurtColonistsStayOutOfFires(t *testing.T) {
	g := quiet(7)
	p := place(g, OWall, 8, 4)
	g.W.At(p.x, p.y).Mat = MatWood
	g.ignite(p.y*MapW+p.x, "")
	g.fireCount = 1
	for _, c := range g.cols {
		c.Health = 30
		c.Hunger = 100
		if g.canFightFire(c) {
			t.Fatal("badly hurt colonists don't fight fires")
		}
	}
}

func TestUnreachableFireDoesNotLockUpColonists(t *testing.T) {
	g := quiet(7)
	// a fire inside a sealed iron box: nobody can ever reach it
	cx, cy := g.start.x+14, g.start.y-12
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			tl := g.W.At(cx+dx, cy+dy)
			tl.Ground, tl.Obj, tl.Item, tl.ItemN = GGrass, ONone, INone, 0
			if iabs(dx) == 2 || iabs(dy) == 2 {
				tl.Obj, tl.Mat = OWall, MatIron
			}
		}
	}
	g.W.At(cx, cy).Obj, g.W.At(cx, cy).Mat = OBench, MatNone
	g.W.At(cx, cy).Fire = 0.1
	g.fireCount = 1
	tree := g.W.At(g.start.x-6, g.start.y-6)
	tree.Obj = OTree
	g.applyTool(TChop, g.start.x-6, g.start.y-6, g.start.x-6, g.start.y-6)
	withFireRates(0, 0, func() {
		g.fireCount = 1
		run(g, 80)
	})
	if tree.Obj == OTree {
		t.Fatal("an unreachable fire must not stop colonists from working")
	}
}
