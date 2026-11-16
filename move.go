package main

import "math"

// Moving buildings: pick a furnace, desk, bed... and choose a free spot. A colonist walks to it,
// lifts it, carries it across and sets it down, with everything about it (material, fuel) intact
// and nothing lost. Walls and doors are not movable (they define rooms: rebuild or salvage them).

// Cargo is a building a colonist is carrying.
type Cargo struct {
	Obj  Obj
	Mat  Material
	Fuel float32
	Src  int // tile index it came from
	Dst  int // tile index it is going to
}

// movable says whether a built object can be picked up and moved.
func movable(o Obj) bool {
	_, built := objBlue[o]
	return built && o != OWall && o != ODoor
}

// canMoveTo checks whether a tile is a legal new home for a building.
func (g *Game) canMoveTo(x, y int) bool {
	t := g.W.At(x, y)
	return t != nil && g.W.Walkable(x, y) && t.Obj == ONone && t.Blue == BNone && !t.Zone && !t.Farm &&
		t.Item == INone && t.MoveFrom == 0 && t.MoveTo == 0 && t.Fire == 0
}

// orderMove marks a building to be carried to a new tile. Returns "" or a reason it can't.
func (g *Game) orderMove(sx, sy, dx, dy int) string {
	src, dst := g.W.At(sx, sy), g.W.At(dx, dy)
	switch {
	case src == nil || !movable(src.Obj):
		return "That can't be moved."
	case src.MoveTo != 0:
		return "That is already being moved."
	case src.Fire > 0:
		return "It is on fire!"
	case dst == nil || !g.canMoveTo(dx, dy):
		return "It can't go there."
	case sx == dx && sy == dy:
		return "It is already there."
	}
	b := objBlue[src.Obj]
	dst.Blue, dst.Mat, dst.MoveFrom = b, src.Mat, int32(sy*MapW+sx+1)
	src.MoveTo = int32(dy*MapW + dx + 1)
	return ""
}

func (g *Game) moveName(o Obj) string {
	if b, ok := objBlue[o]; ok {
		return blueNames[b]
	}
	return "building"
}

// moveClick handles a click with the Move tool: the first click picks a building, the second
// chooses where it goes.
func (g *Game) moveClick(tx, ty int) {
	if g.moveSrc < 0 {
		t := g.W.At(tx, ty)
		if t == nil || !movable(t.Obj) {
			g.msg("Click a furnace, desk, bed, machine... to move it.")
			return
		}
		if t.MoveTo != 0 { // clicking an item that is already being moved cancels that move
			g.cancelMoveFrom(ty*MapW + tx)
			g.msg("Move cancelled.")
			return
		}
		g.moveSrc = ty*MapW + tx
		return
	}
	sx, sy := g.moveSrc%MapW, g.moveSrc/MapW
	if why := g.orderMove(sx, sy, tx, ty); why != "" {
		g.msg("%s", why)
		return
	}
	g.msg("Colonists will carry the %s over.", g.moveName(g.W.At(sx, sy).Obj))
	g.moveSrc = -1
}

// cancelMoveFrom removes a pending move whose source is tile i.
func (g *Game) cancelMoveFrom(i int) {
	src := &g.W.T[i]
	if src.MoveTo != 0 {
		d := int(src.MoveTo) - 1
		if d >= 0 && d < len(g.W.T) && int(g.W.T[d].MoveFrom) == i+1 && !g.beingCarried(d) {
			g.W.T[d].Blue, g.W.T[d].Mat, g.W.T[d].MoveFrom = BNone, MatNone, 0
		}
	}
	src.MoveTo = 0
}

// beingCarried reports whether a colonist is carrying a building toward tile d right now.
func (g *Game) beingCarried(d int) bool {
	for _, c := range g.cols {
		if c.Cargo != nil && c.Cargo.Dst == d {
			return true
		}
	}
	return false
}

// tidyMoves clears pending moves whose building has vanished (burnt, salvaged, erased).
func (g *Game) tidyMoves() {
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.MoveFrom != 0 {
			s := int(t.MoveFrom) - 1
			ok := s >= 0 && s < len(g.W.T) && movable(g.W.T[s].Obj) && int(g.W.T[s].MoveTo) == i+1
			if !ok && !g.beingCarried(i) {
				t.Blue, t.Mat, t.MoveFrom = BNone, MatNone, 0
			}
		}
		if t.MoveTo != 0 {
			d := int(t.MoveTo) - 1
			if d < 0 || d >= len(g.W.T) || int(g.W.T[d].MoveFrom) != i+1 {
				t.MoveTo = 0
			}
		}
	}
}

// placeCargo sets a carried building down at tile i.
func (g *Game) placeCargo(cg *Cargo, i int) {
	t := &g.W.T[i]
	t.Obj, t.Mat, t.Fuel = cg.Obj, cg.Mat, cg.Fuel
	t.Zone, t.Farm, t.Crop = false, false, false
	t.Item, t.ItemN = INone, 0
	t.Blue, t.MoveFrom, t.MoveTo = BNone, 0, 0
}

// dropCargo puts a carried building back where it came from (or the nearest free spot).
func (g *Game) dropCargo(c *Colonist) {
	cg := c.Cargo
	if cg == nil {
		return
	}
	c.Cargo = nil
	if d := &g.W.T[cg.Dst]; int(d.MoveFrom) == cg.Src+1 { // the destination marker goes away
		d.Blue, d.Mat, d.MoveFrom = BNone, MatNone, 0
	}
	if s := &g.W.T[cg.Src]; s.Obj == ONone && s.Blue == BNone {
		g.placeCargo(cg, cg.Src)
		return
	}
	x, y := g.tileOf(c)
	if p, ok := g.W.bfs(x, y, 14, func(a, b int) bool {
		t := g.W.At(a, b)
		return t.Obj == ONone && t.Blue == BNone && !t.Farm && t.Item == INone && t.MoveFrom == 0 && t.MoveTo == 0
	}); ok {
		e := endOf(p, x, y)
		g.placeCargo(cg, e.y*MapW+e.x)
	}
}

// tryMove sends a colonist to pick up a building that has a move order.
func (g *Game) tryMove(c *Colonist) bool {
	return g.tryTarget(c, func(t *Tile, _, _ int, self bool) bool {
		return t.MoveTo != 0 && t.ResT == 0 && !self && movable(t.Obj) && t.Fire == 0
	}, func(tg pt, t *Tile) (Job, string) {
		b := objBlue[t.Obj]
		need := math.Max(1, costFor(b, t.Mat).Time*0.5)
		return Job{Kind: JMove, Dst: pt{(int(t.MoveTo) - 1) % MapW, (int(t.MoveTo) - 1) / MapW}, Need: need}, "Going to move the " + g.moveName(t.Obj)
	})
}

func (g *Game) doMove(c *Colonist, dt float64) {
	j := &c.Job
	W := g.W
	si, di := j.Target.y*MapW+j.Target.x, j.Dst.y*MapW+j.Dst.x
	src, dst := &W.T[si], &W.T[di]
	x, y := g.tileOf(c)
	rate := c.rate(SConstruction) * g.nightFactor(x, y)

	if j.Stage == 0 { // lift it
		if !movable(src.Obj) || int(src.MoveTo) != di+1 || int(dst.MoveFrom) != si+1 {
			g.abort(c)
			return
		}
		c.Status = "Lifting the " + g.moveName(src.Obj)
		j.Work += dt * rate
		g.gainXP(c, SConstruction, dt*0.5)
		if j.Work < j.Need {
			return
		}
		c.Cargo = &Cargo{Obj: src.Obj, Mat: src.Mat, Fuel: src.Fuel, Src: si, Dst: di}
		src.Obj, src.Mat, src.Fuel, src.MoveTo = ONone, MatNone, 0, 0
		j.Stage, j.Work = 1, 0
		p, ok := W.bfs(x, y, 120, func(a, b int) bool { return iabs(a-j.Dst.x)+iabs(b-j.Dst.y) <= 1 })
		if !ok {
			g.abort(c) // can't get there: the building goes back
			return
		}
		c.Path = p
		return
	}

	// carry it across and set it down
	cg := c.Cargo
	if cg == nil || dst.Obj != ONone || int(dst.MoveFrom) != si+1 {
		g.abort(c)
		return
	}
	c.Status = "Carrying the " + g.moveName(cg.Obj)
	if solidBlue(objBlue[cg.Obj]) {
		for _, o := range g.cols {
			if o != c {
				ox, oy := g.tileOf(o)
				if ox == j.Dst.x && oy == j.Dst.y {
					return // wait until nobody stands there
				}
			}
		}
		if x == j.Dst.x && y == j.Dst.y { // don't build it on top of yourself: step aside first
			for _, d := range dirs4 {
				if g.W.Walkable(x+d.x, y+d.y) {
					c.Path = []pt{{x + d.x, y + d.y}}
					return
				}
			}
		}
	}
	j.Work += dt * rate
	g.gainXP(c, SConstruction, dt*0.5)
	if j.Work >= j.Need {
		g.placeCargo(cg, di)
		c.Cargo = nil
		if cg.Obj == OWall || cg.Obj == ODoor {
			g.roomDirty = true
		}
		g.emit("hammer", float64(j.Dst.x)+0.5, float64(j.Dst.y)+0.5)
		g.finish(c)
	}
}
