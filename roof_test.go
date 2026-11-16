package main

import "testing"

func TestRoofToolDesignatesAndColonistsBuildIt(t *testing.T) {
	g := quiet(7)
	var tiles []pt
	for dx := 0; dx < 3; dx++ {
		p := place(g, ONone, 8+dx, 8)
		tiles = append(tiles, p)
	}
	g.applyTool(TRoof, tiles[0].x, tiles[0].y, tiles[2].x, tiles[2].y)
	for _, p := range tiles {
		if g.W.At(p.x, p.y).Desig != DRoof {
			t.Fatalf("%v should be marked for roofing", p)
		}
	}
	g.recompute()
	wood0 := totalItems(g)[IWood]
	run(g, 60)
	for _, p := range tiles {
		if tl := g.W.At(p.x, p.y); !tl.Roof || tl.Desig != DNone {
			t.Fatalf("%v not roofed: %+v", p, tl)
		}
	}
	if got := wood0 - totalItems(g)[IWood]; got != 3*roofWood {
		t.Fatalf("3 roof tiles should cost %d wood, spent %d", 3*roofWood, got)
	}
	// marking an already roofed tile does nothing
	g.applyTool(TRoof, tiles[0].x, tiles[0].y, tiles[0].x, tiles[0].y)
	if g.W.At(tiles[0].x, tiles[0].y).Desig != DNone {
		t.Fatal("already roofed")
	}
}

func TestRoofNeedsWoodAndIsExplained(t *testing.T) {
	g := quiet(7)
	for g.stock[IWood] > 0 {
		g.takeStock(IWood, g.stock[IWood])
		g.recompute()
	}
	p := place(g, ONone, 8, 8)
	g.applyTool(TRoof, p.x, p.y, p.x, p.y)
	run(g, 30)
	if g.W.At(p.x, p.y).Roof {
		t.Fatal("no wood, no roof")
	}
	g.hintT = 0
	g.msgs = nil
	g.advise(1)
	if len(g.msgs) == 0 || !contains(g.msgs[0].Text, "wood") {
		t.Fatalf("expected a hint about wood: %v", g.msgs)
	}
	stockUp(g, IWood, 10)
	run(g, 40)
	if !g.W.At(p.x, p.y).Roof {
		t.Fatal("once there is wood it gets built")
	}
}

func TestRoofCannotGoOnWaterOrTrees(t *testing.T) {
	g := quiet(7)
	w := place(g, ONone, 9, 9)
	g.W.At(w.x, w.y).Ground = GWater
	tr := place(g, OTree, 10, 9)
	g.applyTool(TRoof, w.x, w.y, tr.x, tr.y)
	if g.W.At(w.x, w.y).Desig != DNone || g.W.At(tr.x, tr.y).Desig != DNone {
		t.Fatal("no roofs on water or trees")
	}
}

func TestSalvageRemovesRoofLast(t *testing.T) {
	g := quiet(7)
	p := place(g, OBench, 8, 8)
	g.W.At(p.x, p.y).Roof = true
	before := totalItems(g)
	g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
	run(g, 40)
	if g.W.At(p.x, p.y).Obj != ONone || !g.W.At(p.x, p.y).Roof {
		t.Fatal("the first salvage removes the bench but leaves the roof")
	}
	g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
	run(g, 40)
	if g.W.At(p.x, p.y).Roof {
		t.Fatal("the second salvage removes the roof")
	}
	if got := totalItems(g)[IWood] - before[IWood]; got != 4+1 { // half the bench (8) + half the roof (2)
		t.Fatalf("refund %d wood", got)
	}
}

func TestEraseRemovesRoofs(t *testing.T) {
	g := quiet(7)
	p := place(g, ONone, 8, 8)
	g.W.At(p.x, p.y).Roof = true
	g.applyTool(TErase, p.x, p.y, p.x, p.y)
	if g.W.At(p.x, p.y).Roof {
		t.Fatal("erase clears roofs")
	}
}

// buildHouse makes a walled 5x5 room (3x3 inside) with a door, optionally roofed.
func buildHouse(g *Game, roof bool) pt {
	cx, cy := 10, 10
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			p := place(g, ONone, cx+dx, cy+dy)
			tl := g.W.At(p.x, p.y)
			if iabs(dx) == 2 || iabs(dy) == 2 {
				tl.Obj, tl.Mat = OWall, MatStone
			} else {
				tl.Roof = roof
			}
		}
	}
	d := g.W.At(g.start.x+cx, g.start.y+cy+2)
	d.Obj, d.Mat = ODoor, MatWood
	g.computeRooms()
	return pt{g.start.x + cx, g.start.y + cy}
}

func TestRoomsNeedARoofToBeIndoors(t *testing.T) {
	g := quiet(7)
	in := buildHouse(g, false)
	g.outT = -10
	if g.indoors(in.x, in.y) {
		t.Fatal("a walled room without a roof is open to the sky")
	}
	open := g.tempAt(in.x, in.y)
	if open > -9 {
		t.Fatalf("an unroofed room should be as cold as outside, got %.1f", open)
	}
	g.W.At(in.x, in.y).Roof = true
	if !g.indoors(in.x, in.y) {
		t.Fatal("with a roof it is indoors")
	}
	if g.tempAt(in.x, in.y) <= open+5 {
		t.Fatalf("a roof should shelter from the cold: %.1f vs %.1f", g.tempAt(in.x, in.y), open)
	}
	// a roof over open ground is shelter but not a heatable room
	q := place(g, ONone, 20, 20)
	g.W.At(q.x, q.y).Roof = true
	if g.indoors(q.x, q.y) || !g.covered(q.x, q.y) {
		t.Fatal("a canopy covers but is not indoors")
	}
}

func TestHeaterNeedsARoofedRoom(t *testing.T) {
	g := quiet(7)
	in := buildHouse(g, false)
	h := place(g, OHeater, 10, 9)
	g.W.At(h.x, h.y).Fuel = 1e6
	g.outT = -12
	g.slow(1)
	cold := g.tempAt(in.x, in.y)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			g.W.At(in.x+dx, in.y+dy).Roof = true
		}
	}
	g.slow(1)
	if warm := g.tempAt(in.x, in.y); warm < cold+15 {
		t.Fatalf("a roofed, heated room should be much warmer: %.1f vs %.1f", warm, cold)
	}
}

func TestStartingStockpileIsRoofed(t *testing.T) {
	g := newGame(7)
	n := 0
	for _, i := range g.zoneList {
		if g.W.T[i].Roof {
			n++
		}
	}
	if n != len(g.zoneList) || n == 0 {
		t.Fatalf("the starting stockpile should come roofed: %d of %d", n, len(g.zoneList))
	}
}

func TestFoodSpoilsInTheOpenFasterThanUnderARoof(t *testing.T) {
	g := quiet(7)
	g.outT = 15
	open := place(g, ONone, 8, 8)
	roofed := place(g, ONone, 10, 8)
	g.W.At(open.x, open.y).Item, g.W.At(open.x, open.y).ItemN = IFood, 40
	g.W.At(roofed.x, roofed.y).Item, g.W.At(roofed.x, roofed.y).ItemN = IFood, 40
	g.W.At(roofed.x, roofed.y).Roof = true
	for i := 0; i < int(DayLen); i++ { // one day
		g.updateSpoilage(1)
	}
	lostOpen := 40 - g.W.At(open.x, open.y).ItemN
	lostRoof := 40 - g.W.At(roofed.x, roofed.y).ItemN
	if lostOpen < 9 || lostOpen > 15 {
		t.Fatalf("about 30%% of the stack should rot in a day in the open, lost %d of 40", lostOpen)
	}
	if lostRoof != 0 {
		t.Fatalf("goods under a roof never spoil, but %d rotted", lostRoof)
	}
	if g.stats.Spoiled < lostOpen { // (the starting stockpile is roofed, so nothing there rots either)
		t.Fatal("spoilage should be counted")
	}
}

func TestFrozenAndNonPerishableGoodsKeep(t *testing.T) {
	g := quiet(7)
	a := place(g, ONone, 8, 8)
	b := place(g, ONone, 10, 8)
	c := place(g, ONone, 12, 8)
	g.W.At(a.x, a.y).Item, g.W.At(a.x, a.y).ItemN = IFood, 40
	g.W.At(b.x, b.y).Item, g.W.At(b.x, b.y).ItemN = IStone, 40
	g.W.At(c.x, c.y).Item, g.W.At(c.x, c.y).ItemN = IMedicine, 40
	g.outT = -5 // freezing
	for i := 0; i < int(DayLen)*2; i++ {
		g.updateSpoilage(1)
	}
	if g.W.At(a.x, a.y).ItemN != 40 || g.W.At(c.x, c.y).ItemN != 40 {
		t.Fatal("frozen goods don't rot")
	}
	g.outT = 15
	for i := 0; i < int(DayLen)*2; i++ {
		g.updateSpoilage(1)
	}
	if g.W.At(b.x, b.y).ItemN != 40 {
		t.Fatal("stone never rots")
	}
	if g.W.At(a.x, a.y).ItemN >= 30 {
		t.Fatal("food should rot once it thaws")
	}
	if med := g.W.At(c.x, c.y).ItemN; med >= 40 || med < 30 {
		t.Fatalf("medicine rots slowly: %d", med)
	}
}

func TestStackDisappearsWhenFullyRotten(t *testing.T) {
	g := quiet(7)
	g.outT = 20
	p := place(g, ONone, 8, 8)
	g.W.At(p.x, p.y).Item, g.W.At(p.x, p.y).ItemN = IFood, 2
	for i := 0; i < int(DayLen)*15; i++ {
		g.updateSpoilage(1)
	}
	if tl := g.W.At(p.x, p.y); tl.Item != INone || tl.ItemN != 0 {
		t.Fatalf("an empty stack should be cleared: %+v", tl.Item)
	}
}

func TestSpoilageWarnsAndTooltipExplains(t *testing.T) {
	g := quiet(7)
	g.outT = 15
	for i := 0; i < 8; i++ {
		p := place(g, ONone, 8+i, 8)
		g.W.At(p.x, p.y).Item, g.W.At(p.x, p.y).ItemN = IFood, 50
	}
	g.msgs = nil
	g.time = DayLen * 3
	for i := 0; i < int(DayLen); i++ {
		g.updateSpoilage(1)
	}
	warned := false
	for _, m := range g.msgs {
		if m.Alert && contains(m.Text, "Roof") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("heavy spoilage in the open should alert the player: %v", g.msgs)
	}
	p := pt{g.start.x + 8, g.start.y + 8}
	if n := g.spoilNote(p.x, p.y); !contains(n, "roof it") {
		t.Fatalf("tooltip note: %q", n)
	}
	g.W.At(p.x, p.y).Roof = true
	if n := g.spoilNote(p.x, p.y); !contains(n, "roofed") || !contains(n, "keep") {
		t.Fatalf("tooltip note: %q", n)
	}
	// a roof wins over the weather: nothing under a roof spoils, so the cold isn't mentioned
	g.outT = -10
	if n := g.spoilNote(p.x, p.y); !contains(n, "roofed") {
		t.Fatalf("roofed goods say so even in the cold: %q", n)
	}
	// in the open and freezing: food is "frozen", but books and medicine are only "too cold to spoil"
	g.W.At(p.x, p.y).Roof = false
	if n := g.spoilNote(p.x, p.y); !contains(n, "frozen") {
		t.Fatalf("frozen food: %q", n)
	}
	for _, it := range []Item{IBook, IMedicine} {
		g.W.At(p.x, p.y).Item = it
		n := g.spoilNote(p.x, p.y)
		if contains(n, "frozen") || !contains(n, "too cold to spoil") {
			t.Fatalf("%s should not be called frozen: %q", itemNames[it], n)
		}
	}
	g.W.At(p.x, p.y).Item = IFood
	g.outT = 15
	if n := g.spoilNote(p.x, p.y); !contains(n, "roof it") {
		t.Fatalf("thawed food in the open: %q", n)
	}
}

func TestOldSavesGetRoofsOnEnclosedRoomsOnly(t *testing.T) {
	g := quiet(7)
	in := buildHouse(g, false)
	path := t.TempDir() + "/old.sav"
	sd := g.snapshot()
	sd.Version = 3 // written before roofs existed
	g.apply(sd)
	if !g.W.At(in.x, in.y).Roof || !g.indoors(in.x, in.y) {
		t.Fatal("an old save's enclosed rooms should keep their heat")
	}
	if g.W.At(g.start.x+3, g.start.y+3).Roof {
		t.Fatal("open ground must not be roofed")
	}
	found := false
	for _, m := range g.msgs {
		if contains(m.Text, "Roof tool") {
			found = true
		}
	}
	if !found {
		t.Fatal("old-save players should be told about spoilage")
	}
	_ = path
	// and the current version round-trips roofs and rot
	g.W.At(in.x, in.y).Rot = 0.5
	p := t.TempDir() + "/new.sav"
	if err := g.SaveFile(p); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if !h.W.At(in.x, in.y).Roof || h.W.At(in.x, in.y).Rot != 0.5 {
		t.Fatal("roof and rot should survive a save")
	}
}

func TestToolbarWrapsToRowsWithoutOverlap(t *testing.T) {
	for _, width := range []int{1280, 960, 700, 1600, 2200} {
		g := quiet(7)
		g.scale, g.sw, g.sh = 1, width, 760
		var rects [][4]float64
		for i := 0; i < int(numTools); i++ {
			x, y, w, h := g.buttonRect(i)
			if x < 0 || x+w > float64(width)+0.5 {
				t.Fatalf("width %d: button %d off screen (%.0f..%.0f)", width, i, x, x+w)
			}
			if y < float64(g.sh)-g.barH() || y+h > float64(g.sh) {
				t.Fatalf("width %d: button %d outside the bar", width, i)
			}
			for _, r := range rects {
				if x < r[0]+r[2] && x+w > r[0] && y < r[1]+r[3] && y+h > r[1] {
					t.Fatalf("width %d: button %d overlaps another", width, i)
				}
			}
			rects = append(rects, [4]float64{x, y, w, h})
		}
	}
}

func TestRoomTooltipSaysRoofedAndHeated(t *testing.T) {
	g := quiet(7)
	in := buildHouse(g, false)
	if l := g.roomTip(in.x, in.y); len(l) == 0 || !contains(l[0], "no roof") {
		t.Fatalf("%v", l)
	}
	buildHouse(g, true)
	if l := g.roomTip(in.x, in.y); !contains(l[0], "roofed, not heated") {
		t.Fatalf("%v", l)
	}
	h := place(g, OHeater, 10, 9)
	g.W.At(h.x, h.y).Fuel = 1e6
	g.slow(1)
	if l := g.roomTip(in.x, in.y); !contains(l[0], "roofed and heated") || len(l) != 2 {
		t.Fatalf("%v", l)
	}
}
