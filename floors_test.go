package main

import "testing"

func TestFloorKindsAndCosts(t *testing.T) {
	if floorCost(FloorWood).Item != IWood || floorCost(FloorWood).N != 2 {
		t.Fatal("wood floors still cost 2 wood")
	}
	if floorCost(FloorStone).Item != IStone || floorCost(FloorTile).Item != ITile || floorCost(FloorChecker).Item != ITile {
		t.Fatal("stone slabs from stone, tiles from Tile")
	}
	for k := FloorKind(0); k < numFloors; k++ {
		if floorNames[k] == "" || floorNotes[k] == "" || floorSpeed[k] <= 1 {
			t.Errorf("floor %d is incomplete", k)
		}
	}
	if !(floorSpeed[FloorTile] > floorSpeed[FloorStone] && floorSpeed[FloorStone] > floorSpeed[FloorWood]) {
		t.Fatal("harder floors are faster")
	}
	// an old save's floors were all wood planks
	if (&Tile{Floor: true}).FloorKind != FloorWood {
		t.Fatal("legacy floors are wood")
	}
}

func TestEachFloorKindIsBuiltFromItsOwnMaterial(t *testing.T) {
	for k := FloorKind(0); k < numFloors; k++ {
		g := quiet(7)
		g.addToStock(ITile, 6)
		g.addToStock(IStone, 10)
		g.floorSel = k
		p := place(g, ONone, 6, 6)
		cost := floorCost(k)
		g.recompute()
		before := g.stock[cost.Item]
		g.applyTool(TFloor, p.x, p.y, p.x, p.y)
		tl := g.W.At(p.x, p.y)
		if tl.Blue != BFloor || tl.BlueKind != k {
			t.Fatalf("%s: blueprint %+v", floorNames[k], tl.BlueKind)
		}
		if tl.blueName() != floorNames[k] {
			t.Fatalf("name %q", tl.blueName())
		}
		run(g, 40)
		if !tl.Floor || tl.FloorKind != k || tl.Blue != BNone {
			t.Fatalf("%s not built: floor=%v kind=%d", floorNames[k], tl.Floor, tl.FloorKind)
		}
		g.recompute()
		if got := before - g.stock[cost.Item]; got != cost.N {
			t.Fatalf("%s spent %d %s, want %d", floorNames[k], got, itemNames[cost.Item], cost.N)
		}
	}
}

func TestTileFloorsWaitForTiles(t *testing.T) {
	g := quiet(7)
	g.floorSel = FloorTile
	p := place(g, ONone, 6, 6)
	g.applyTool(TFloor, p.x, p.y, p.x, p.y)
	run(g, 30)
	if g.W.At(p.x, p.y).Floor {
		t.Fatal("no Tile in stock, so no tile floor")
	}
	why := g.blueprintBlock(g.W.At(p.x, p.y))
	if !contains(why, "Tile") || !contains(why, "turn on Tile production") {
		t.Fatalf("should explain how to get tiles: %q", why)
	}
}

func TestLayingAnotherFloorReplacesTheOldOne(t *testing.T) {
	g := quiet(7)
	g.addToStock(IStone, 10)
	p := place(g, ONone, 6, 6)
	tl := g.W.At(p.x, p.y)
	tl.Floor, tl.FloorKind = true, FloorWood
	g.floorSel = FloorWood
	g.applyTool(TFloor, p.x, p.y, p.x, p.y)
	if tl.Blue != BNone {
		t.Fatal("the same floor again is a no-op")
	}
	g.floorSel = FloorStone
	g.recompute()
	wood0 := totalItems(g)[IWood]
	g.applyTool(TFloor, p.x, p.y, p.x, p.y)
	if tl.Blue != BFloor || tl.BlueKind != FloorStone || tl.FloorKind != FloorWood {
		t.Fatalf("an upgrade blueprint over the old floor: %+v", tl)
	}
	run(g, 40)
	if tl.FloorKind != FloorStone || !tl.Floor || tl.Blue != BNone {
		t.Fatal("the stone floor should replace the wood one")
	}
	if got := totalItems(g)[IWood] - wood0; got != 1 {
		t.Fatalf("half the old floor's wood comes back, got %d", got)
	}
}

func TestSalvagingFloorsRefundsByKind(t *testing.T) {
	for _, tc := range []struct {
		k    FloorKind
		item Item
		want int
	}{{FloorWood, IWood, 1}, {FloorStone, IStone, 2}, {FloorTile, ITile, 1}, {FloorChecker, ITile, 1}} {
		g := quiet(7)
		p := place(g, ONone, 6, 6)
		tl := g.W.At(p.x, p.y)
		tl.Floor, tl.FloorKind = true, tc.k
		before := totalItems(g)[tc.item]
		g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
		run(g, 30)
		if tl.Floor || tl.FloorKind != FloorWood {
			t.Fatalf("%s should be gone", floorNames[tc.k])
		}
		if got := totalItems(g)[tc.item] - before; got != tc.want {
			t.Errorf("%s: refunded %d %s, want %d", floorNames[tc.k], got, itemNames[tc.item], tc.want)
		}
	}
}

func TestEraseRemovesFloorsAndKinds(t *testing.T) {
	g := quiet(7)
	p := place(g, ONone, 6, 6)
	tl := g.W.At(p.x, p.y)
	tl.Floor, tl.FloorKind = true, FloorChecker
	g.applyTool(TErase, p.x, p.y, p.x, p.y)
	if tl.Floor || tl.FloorKind != FloorWood {
		t.Fatalf("erased: %+v", tl)
	}
}

func TestHardFloorsAreFasterToWalkOn(t *testing.T) {
	walk := func(k FloorKind, floored bool) float64 {
		g := quiet(7)
		for _, o := range g.cols[1:] {
			o.Dead = true
		}
		g.cols = g.cols[:1]
		c := g.cols[0]
		openField(g, 3, 40, 10, 12)
		for dx := 5; dx <= 30; dx++ {
			tl := g.W.At(g.start.x+dx, g.start.y+11)
			tl.Floor, tl.FloorKind = floored, k
		}
		c.X, c.Y = float64(g.start.x+5)+0.5, float64(g.start.y+11)+0.5
		g.abort(c)
		var path []pt
		for dx := 6; dx <= 30; dx++ {
			path = append(path, pt{g.start.x + dx, g.start.y + 11})
		}
		c.Path = path
		c.Job = Job{}
		c.Idle = 1e9 // just walk
		for i := 0; i < 60*30; i++ {
			g.updateColonist(c, 1.0/60)
			if len(c.Path) == 0 {
				return float64(i) / 60
			}
		}
		return 30
	}
	grass := walk(FloorWood, false)
	wood, stone, tile := walk(FloorWood, true), walk(FloorStone, true), walk(FloorTile, true)
	if !(grass > wood && wood > stone && stone > tile) {
		t.Fatalf("expected grass > wood > stone > tile floors: %.2f %.2f %.2f %.2f", grass, wood, stone, tile)
	}
	if grass/tile < 1.1 {
		t.Fatalf("tiles should be noticeably quicker than grass: %.2f vs %.2f", tile, grass)
	}
}

func TestTilesAreFiredFromStoneAtAFurnace(t *testing.T) {
	g := quiet(7)
	place(g, OFurnace, -3, 2)
	i := recipeIndex("Tile")
	if i < 0 {
		t.Fatal("no Tile recipe")
	}
	if recipes[i].Bench != OFurnace || recipes[i].Out != ITile || recipes[i].OutN < 2 || recipes[i].Fuel == 0 {
		t.Fatalf("recipe %+v: a kiln burning wood turns stone into several tiles", recipes[i])
	}
	g.prod[i].On, g.prod[i].Target = true, 8
	g.addToStock(IStone, 8)
	run(g, 100)
	g.recompute()
	if g.stock[ITile] < 4 {
		t.Fatalf("expected tiles: %d (status %q)", g.stock[ITile], g.recipeStatus(i))
	}
	if itemValue[ITile] <= 0 {
		t.Fatal("tiles have a value")
	}
	if len(defaultProd()) != len(recipes) {
		t.Fatal("production defaults cover every recipe")
	}
}

func TestFloorPaletteAndLabels(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.selectTool(TFloor)
	if g.panel != panelFloor {
		t.Fatal("selecting the Floor tool opens the picker")
	}
	rows := 0
	for _, b := range g.buttons() {
		if b.h == 32 && b.w > 100 {
			rows++
		}
	}
	if rows != int(numFloors) {
		t.Fatalf("a row per floor: %d", rows)
	}
	if g.toolCost(TFloor) != "2 Wood" {
		t.Fatalf("label %q", g.toolCost(TFloor))
	}
	x, y, _, _ := g.floorPaletteRect()
	if !g.handleUIClick(x+20, y+24+34*2+10) {
		t.Fatal("click not handled")
	}
	if g.floorSel != FloorTile || g.panel != panelNone {
		t.Fatalf("tile floor not chosen: %d", g.floorSel)
	}
	if g.toolCost(TFloor) != "2 Tile" {
		t.Fatalf("label follows the choice: %q", g.toolCost(TFloor))
	}
	g.selectTool(TWall)
	if g.panel == panelFloor {
		t.Fatal("other tools close the picker")
	}
}

func TestFloorChoicesSurviveSaves(t *testing.T) {
	g := quiet(7)
	p := place(g, ONone, 6, 6)
	g.W.At(p.x, p.y).Floor, g.W.At(p.x, p.y).FloorKind = true, FloorChecker
	q := place(g, ONone, 8, 6)
	g.W.At(q.x, q.y).Blue, g.W.At(q.x, q.y).BlueKind = BFloor, FloorStone
	g.floorSel = FloorTile
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h.W.At(p.x, p.y).FloorKind != FloorChecker || h.W.At(q.x, q.y).BlueKind != FloorStone || h.floorSel != FloorTile {
		t.Fatal("floor kinds lost")
	}
}

func TestProductionPanelStillFits(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.panel = panelProd
	_, y, _, h := g.modalRect()
	if y+h > float64(g.sh)-g.barH() {
		t.Fatalf("%d recipes no longer fit above the toolbar (%.0f px)", len(recipes), h)
	}
}

func TestFloorsCanBeLaidUnderBedsAndHeaters(t *testing.T) {
	g := quiet(7)
	bed := place(g, OBed, 4, 4)
	heater := place(g, OHeater, 6, 4)
	wall := place(g, OWall, 8, 4)
	tree := place(g, OTree, 10, 4)
	g.addToStock(ITile, 20)
	g.floorSel = FloorChecker
	g.tool = TFloor
	for _, p := range []pt{bed, heater, wall, tree} {
		g.applyTool(TFloor, p.x, p.y, p.x, p.y)
	}
	if g.W.At(bed.x, bed.y).Blue != BFloor || g.W.At(heater.x, heater.y).Blue != BFloor {
		t.Fatal("beds and heaters can stand on a floor")
	}
	if g.W.At(wall.x, wall.y).Blue != BNone || g.W.At(tree.x, tree.y).Blue != BNone {
		t.Fatal("not under walls or trees")
	}
	run(g, 120)
	for _, p := range []pt{bed, heater} {
		tl := g.W.At(p.x, p.y)
		if !tl.Floor || tl.FloorKind != FloorChecker {
			t.Fatalf("floor not laid at %v", p)
		}
	}
	if g.W.At(bed.x, bed.y).Obj != OBed || g.W.At(heater.x, heater.y).Obj != OHeater {
		t.Fatal("the furniture stays")
	}
}
