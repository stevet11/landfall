package main

import "testing"

func TestMovableThings(t *testing.T) {
	for _, o := range []Obj{OFurnace, OStove, OBench, OFab, ODesk, OBed, OHeater, OGen, OSolar, OLamp, OEHeater} {
		if !movable(o) {
			t.Errorf("%v should be movable", o)
		}
	}
	for _, o := range []Obj{ONone, OWall, ODoor, OTree, ORock, OBush} {
		if movable(o) {
			t.Errorf("%v must not be movable", o)
		}
	}
}

// movedFurnace sets up a furnace at (6,6) and returns it with a free destination.
func movedFurnace(g *Game) (src, dst pt) {
	openField(g, 3, 20, 3, 18) // nothing in the way of the carrying
	src = place(g, OFurnace, 6, 6)
	g.W.At(src.x, src.y).Mat = MatNone
	g.W.At(src.x, src.y).Fuel = 12
	dst = place(g, ONone, 12, 9)
	return
}

func TestColonistsCarryABuildingWithoutLosingAnything(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.recompute()
	before := totalItems(g)
	if why := g.orderMove(src.x, src.y, dst.x, dst.y); why != "" {
		t.Fatal(why)
	}
	if g.W.At(dst.x, dst.y).Blue != BFurnace || g.W.At(dst.x, dst.y).MoveFrom == 0 || g.W.At(src.x, src.y).MoveTo == 0 {
		t.Fatal("the order should mark both ends")
	}
	if g.W.At(src.x, src.y).Obj != OFurnace {
		t.Fatal("the furnace stays where it is until someone picks it up")
	}
	carried := false
	for i := 0; i < 60*80; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Cargo != nil {
				carried = true
				if g.W.At(src.x, src.y).Obj != ONone {
					t.Fatal("while it is in someone's arms it is not on its old tile")
				}
			}
		}
		if g.W.At(dst.x, dst.y).Obj == OFurnace {
			break
		}
	}
	if !carried {
		t.Fatal("a colonist should have carried it")
	}
	d, s := g.W.At(dst.x, dst.y), g.W.At(src.x, src.y)
	if d.Obj != OFurnace || d.Fuel != 12 {
		t.Fatalf("the furnace should arrive with its fuel: %+v", d)
	}
	if s.Obj != ONone || s.MoveTo != 0 || d.MoveFrom != 0 || d.Blue != BNone {
		t.Fatalf("both ends should be tidy: src %+v dst %+v", s, d)
	}
	g.recompute()
	if after := totalItems(g); after != before {
		t.Fatalf("nothing is spent or lost moving things: %v vs %v", after, before)
	}
	for _, c := range g.cols {
		if c.Cargo != nil || c.Job.Kind == JMove {
			t.Fatal("the job should be over")
		}
	}
}

func TestMovedBenchesStillWork(t *testing.T) {
	g := quiet(7)
	src := place(g, OBench, -3, 2)
	dst := place(g, ONone, -8, 6)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	run(g, 60)
	if g.W.At(dst.x, dst.y).Obj != OBench {
		t.Fatal("the bench should have moved")
	}
	g.prod[0].On, g.prod[0].Target = true, 2
	run(g, 100)
	g.recompute()
	if g.stock[ICarving] < 1 {
		t.Fatalf("carvings should be made at the new bench (status %q)", g.recipeStatus(0))
	}
}

func TestMovingAGeneratorKeepsThePowerOn(t *testing.T) {
	g := quiet(7)
	src := place(g, OGen, 4, 4)
	g.W.At(src.x, src.y).Fuel = 25
	lamp := place(g, OLamp, 6, 4)
	_ = lamp
	dst := place(g, ONone, 10, 4)
	g.slow(1)
	if g.powerSupply == 0 {
		t.Fatal("setup")
	}
	g.orderMove(src.x, src.y, dst.x, dst.y)
	run(g, 60)
	if g.W.At(dst.x, dst.y).Obj != OGen {
		t.Fatal("generator not moved")
	}
	g.slow(1)
	if g.powerSupply == 0 || len(g.lamps) != 1 {
		t.Fatalf("power should work after the move: supply %v lamps %v", g.powerSupply, g.lamps)
	}
}

func TestMoveOrdersAreChecked(t *testing.T) {
	g := quiet(7)
	src, _ := movedFurnace(g)
	bad := map[string]pt{}
	w := place(g, ONone, 9, 9)
	g.W.At(w.x, w.y).Ground = GWater
	bad["water"] = w
	bad["occupied"] = place(g, OBench, 10, 10)
	z := place(g, ONone, 11, 11)
	g.W.At(z.x, z.y).Zone = true
	bad["stockpile"] = z
	f := place(g, ONone, 12, 12)
	g.W.At(f.x, f.y).Farm = true
	bad["farm"] = f
	bad["itself"] = src
	for name, p := range bad {
		if why := g.orderMove(src.x, src.y, p.x, p.y); why == "" {
			t.Errorf("moving onto %s should be refused", name)
		}
	}
	if g.W.At(src.x, src.y).MoveTo != 0 {
		t.Fatal("refused orders leave no trace")
	}
	// bad sources
	wall := place(g, OWall, 14, 14)
	ok := place(g, ONone, 15, 15)
	if why := g.orderMove(wall.x, wall.y, ok.x, ok.y); why == "" {
		t.Fatal("walls can't be moved")
	}
	empty := place(g, ONone, 16, 16)
	if why := g.orderMove(empty.x, empty.y, ok.x, ok.y); why == "" {
		t.Fatal("nothing to move")
	}
	// a second order for the same building, and a second building to the same spot
	if why := g.orderMove(src.x, src.y, ok.x, ok.y); why != "" {
		t.Fatal(why)
	}
	if why := g.orderMove(src.x, src.y, empty.x, empty.y); why == "" {
		t.Fatal("already being moved")
	}
	other := place(g, ODesk, 18, 18)
	if why := g.orderMove(other.x, other.y, ok.x, ok.y); why == "" {
		t.Fatal("that spot is taken by the other move")
	}
}

func TestInterruptedMoveSetsTheBuildingBack(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	var carrier *Colonist
	for i := 0; i < 60*80 && carrier == nil; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Cargo != nil {
				carrier = c
			}
		}
	}
	if carrier == nil {
		t.Fatal("nobody picked it up")
	}
	g.abort(carrier) // a wolf appears...
	if carrier.Cargo != nil {
		t.Fatal("the colonist let go")
	}
	s, d := g.W.At(src.x, src.y), g.W.At(dst.x, dst.y)
	if s.Obj != OFurnace || s.Fuel != 12 {
		t.Fatalf("it goes back where it came from, fuel and all: %+v", s)
	}
	if d.Blue != BNone || d.MoveFrom != 0 || s.MoveTo != 0 {
		t.Fatal("the order is cancelled")
	}
}

func TestDroppedWhereTheOldSpotIsTaken(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	var carrier *Colonist
	for i := 0; i < 60*80 && carrier == nil; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Cargo != nil {
				carrier = c
			}
		}
	}
	if carrier == nil {
		t.Skip("never picked up")
	}
	g.W.At(src.x, src.y).Obj = OBench // someone built something on the old spot meanwhile
	g.abort(carrier)
	found := false
	for i := range g.W.T {
		if g.W.T[i].Obj == OFurnace {
			found = true
		}
	}
	if !found {
		t.Fatal("the furnace must not vanish: it is put down nearby")
	}
}

func TestDyingWhileCarryingDropsIt(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	var carrier *Colonist
	for i := 0; i < 60*80 && carrier == nil; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Cargo != nil {
				carrier = c
			}
		}
	}
	if carrier == nil {
		t.Skip("never picked up")
	}
	carrier.Health = -1
	run(g, 1)
	n := 0
	for i := range g.W.T {
		if g.W.T[i].Obj == OFurnace {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("exactly one furnace should exist after a death, found %d", n)
	}
}

func TestVanishedBuildingsCancelTheirMove(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	g.W.At(src.x, src.y).Obj = ONone // burnt down, say
	g.tidyMoves()
	if d := g.W.At(dst.x, dst.y); d.Blue != BNone || d.MoveFrom != 0 {
		t.Fatalf("the ghost should go: %+v", d)
	}
	// erasing the ghost cancels the move and frees the building
	src2 := place(g, OLamp, 5, 8)
	dst2 := place(g, ONone, 9, 12)
	g.orderMove(src2.x, src2.y, dst2.x, dst2.y)
	g.applyTool(TErase, dst2.x, dst2.y, dst2.x, dst2.y)
	g.tidyMoves()
	if g.W.At(src2.x, src2.y).MoveTo != 0 || g.W.At(src2.x, src2.y).Obj != OLamp {
		t.Fatal("erasing the destination cancels the move and leaves the building alone")
	}
	// clicking an already-moving building with the Move tool cancels too
	src3 := place(g, ODesk, 5, 14)
	dst3 := place(g, ONone, 9, 16)
	g.orderMove(src3.x, src3.y, dst3.x, dst3.y)
	g.tool = TMove
	g.moveSrc = -1
	g.moveClick(src3.x, src3.y)
	if g.W.At(dst3.x, dst3.y).MoveFrom != 0 || g.W.At(src3.x, src3.y).MoveTo != 0 {
		t.Fatal("click to cancel")
	}
}

func TestMoveToolClicks(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.selectTool(TMove)
	if g.moveSrc != -1 {
		t.Fatal("starts with nothing picked")
	}
	g.moveClick(g.start.x+1, g.start.y+1) // not a building
	if g.moveSrc != -1 {
		t.Fatal("empty ground can't be picked")
	}
	g.moveClick(src.x, src.y)
	if g.moveSrc != src.y*MapW+src.x {
		t.Fatal("the furnace is picked")
	}
	g.msgs = nil
	g.moveClick(src.x, src.y) // onto itself
	if g.moveSrc < 0 || len(g.msgs) == 0 {
		t.Fatal("a bad spot keeps the pick and explains")
	}
	g.moveClick(dst.x, dst.y)
	if g.moveSrc != -1 || g.W.At(dst.x, dst.y).MoveFrom == 0 {
		t.Fatal("the second click orders the move")
	}
	// switching tools drops a pick
	g.moveClick(g.start.x-5, g.start.y-5)
	g.selectTool(TChop)
	if g.moveSrc != -1 {
		t.Fatal("switching tools lets go")
	}
}

func TestRightClickLetsGoOfAPickedBuilding(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	src, _ := movedFurnace(g)
	g.selectTool(TMove)
	g.moveClick(src.x, src.y)
	g.rightMouse(50, 50, true, true, false)
	g.rightMouse(50, 50, false, false, true)
	if g.moveSrc != -1 {
		t.Fatal("right-click should drop the pick")
	}
	if g.tool != TMove {
		t.Fatal("but stay in the Move tool")
	}
}

func TestSavingMidCarryDoesNotLoseTheBuilding(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	var carrier *Colonist
	for i := 0; i < 60*80 && carrier == nil; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Cargo != nil {
				carrier = c
			}
		}
	}
	if carrier == nil {
		t.Skip("never picked up")
	}
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if carrier.Cargo == nil {
		t.Fatal("saving must not interrupt the game in progress")
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	n := 0
	for i := range h.W.T {
		if h.W.T[i].Obj == OFurnace {
			n++
		}
		if h.W.T[i].MoveFrom != 0 || h.W.T[i].MoveTo != 0 {
			t.Fatal("no dangling move markers after loading")
		}
	}
	if n != 1 || h.W.At(src.x, src.y).Obj != OFurnace || h.W.At(src.x, src.y).Fuel != 12 {
		t.Fatalf("the furnace should be back where it started, found %d", n)
	}
}

func TestPendingMoveSurvivesSaveAndLoad(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h.W.At(dst.x, dst.y).MoveFrom == 0 || h.W.At(src.x, src.y).MoveTo == 0 {
		t.Fatal("an order that hasn't started should still be there")
	}
	run(h, 80)
	if h.W.At(dst.x, dst.y).Obj != OFurnace || h.W.At(src.x, src.y).Obj != ONone {
		t.Fatal("and it should still get carried out")
	}
}

func TestSeveralMovesAtOnce(t *testing.T) {
	g := quiet(7)
	var srcs, dsts []pt
	for i, o := range []Obj{OFurnace, ODesk, OBed} {
		s := place(g, o, 5+i*2, 7)
		d := place(g, ONone, 5+i*2, 14)
		g.orderMove(s.x, s.y, d.x, d.y)
		srcs, dsts = append(srcs, s), append(dsts, d)
	}
	run(g, 120)
	for i, o := range []Obj{OFurnace, ODesk, OBed} {
		if g.W.At(dsts[i].x, dsts[i].y).Obj != o || g.W.At(srcs[i].x, srcs[i].y).Obj != ONone {
			t.Fatalf("move %d not done", i)
		}
	}
}

func TestNormalBuildersIgnoreMoveGhosts(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	g.stock[IStone], g.stock[IWood] = 0, 0 // a furnace blueprint would be unaffordable
	// the move is a carry, not a construction: no materials are needed
	run(g, 80)
	if g.W.At(dst.x, dst.y).Obj != OFurnace {
		t.Fatal("moving costs nothing")
	}
}

func TestMovingTrainsConstruction(t *testing.T) {
	g := quiet(7)
	src, dst := movedFurnace(g)
	g.orderMove(src.x, src.y, dst.x, dst.y)
	xp := func() (x float64) {
		for _, c := range g.cols {
			x += float64(c.lvl(SConstruction))*1000 + c.Skills[SConstruction].XP
		}
		return
	}
	x0 := xp()
	run(g, 60)
	if xp() <= x0 {
		t.Fatal("carrying heavy things trains Construction")
	}
}
