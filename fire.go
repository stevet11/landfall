package main

import "math"

// Fire: wooden buildings (wood walls and doors, beds, workbenches, desks) can catch fire from
// running heaters, generators and furnaces next to them, and the flames spread to neighbouring
// wooden things. Stone and iron never burn. Colonists caught on a burning tile are hurt and run.
// Firefighting is not implemented yet; fires burn themselves out.

var (
	fireIgniteRate = 1.0 / 5400 // per second, per flammable neighbour of a running burner
	fireSpreadRate = 0.12       // per second, per flammable neighbour of a burning tile
	fireBurnTime   = 22.0       // seconds a thing burns before it is gone
)

var dirs4 = []pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// burning reports whether a fuel-burning building is running (so it can start fires).
func (g *Game) burnerActive(i int) bool {
	t := &g.W.T[i]
	switch t.Obj {
	case OHeater, OGen:
		return t.Fuel > 0
	case OFurnace, OStove:
		for _, c := range g.cols {
			if c.Job.Kind == JCraft && recipes[c.Job.Recipe].Bench == t.Obj && c.Job.Target.y*MapW+c.Job.Target.x == i {
				return true
			}
		}
	}
	return false
}

func (g *Game) ignite(i int, why string) {
	t := &g.W.T[i]
	if t.Fire > 0 || !t.flammable() {
		return
	}
	t.Fire = 0.01
	g.stats.Fires++
	x, y := i%MapW, i/MapW
	if g.time-g.lastFireAlert > 20 {
		g.lastFireAlert = g.time
		name := "wooden building"
		if b, ok := objBlue[t.Obj]; ok {
			name = blueLabel(b, t.Mat)
		}
		g.alert("Fire! A %s is burning%s", name, why)
	}
	if g.tick-g.lastFireSound > 40 {
		g.lastFireSound = g.tick
		g.emit("fire", float64(x)+0.5, float64(y)+0.5)
	}
}

func (g *Game) burnOut(i int) {
	t := &g.W.T[i]
	if t.Obj == OWall || t.Obj == ODoor {
		g.roomDirty = true
	}
	t.Obj, t.Mat, t.Fire = ONone, MatNone, 0
	t.Item, t.ItemN = INone, 0
	t.Desig = DNone
	g.stats.Burned++
}

func (g *Game) updateFire(dt float64) {
	W := g.W
	defer func() {
		n := 0
		for i := range W.T {
			if W.T[i].Fire > 0 {
				n++
			}
		}
		g.fireCount = n
	}()
	// sparks from running burners
	for i := range W.T {
		o := W.T[i].Obj
		if o != OHeater && o != OGen && o != OFurnace && o != OStove {
			continue
		}
		if !g.burnerActive(i) {
			continue
		}
		x, y := i%MapW, i/MapW
		for _, d := range dirs4 {
			n := W.At(x+d.x, y+d.y)
			if n != nil && n.Fire == 0 && n.flammable() && g.rng.Float64() < dt*fireIgniteRate {
				g.ignite((y+d.y)*MapW+x+d.x, " next to the "+map[Obj]string{OHeater: "heater", OGen: "generator", OFurnace: "furnace", OStove: "stove"}[o])
			}
		}
	}
	// burning things burn, spread and eventually burn out
	var burning []int
	for i := range W.T {
		if W.T[i].Fire > 0 {
			burning = append(burning, i)
		}
	}
	for _, i := range burning {
		t := &W.T[i]
		t.Fire += float32(dt)
		x, y := i%MapW, i/MapW
		if g.raining() && g.exposed(x, y) && g.rng.Float64() < dt*rainDouse {
			t.Fire = 0 // the rain puts it out
			continue
		}
		for _, d := range dirs4 {
			n := W.At(x+d.x, y+d.y)
			if n != nil && n.Fire == 0 && n.flammable() && g.rng.Float64() < dt*fireSpreadRate {
				g.ignite((y+d.y)*MapW+x+d.x, "")
			}
		}
		if float64(t.Fire) >= fireBurnTime {
			g.burnOut(i)
		}
	}
}

// fireDanger hurts a colonist standing on a burning tile and sends them to safety.
func (g *Game) fireDanger(c *Colonist, dt float64) {
	x, y := g.tileOf(c)
	t := g.W.At(x, y)
	if t == nil || t.Fire == 0 {
		return
	}
	c.Health -= 6 * dt * c.damageTakenMult()
	if g.chance(dt * 0.8) {
		if g.addWound(c, "Burn", 2.5, false) {
			g.msg("%s is burning!", c.Name)
		}
	}
	if len(c.Path) == 0 || c.Job.Kind != JNone {
		g.abort(c)
		if p, ok := g.W.bfs(x, y, 8, func(a, b int) bool {
			n := g.W.At(a, b)
			return n.Fire == 0 && !(a == x && b == y) && math.Abs(float64(a-x))+math.Abs(float64(b-y)) >= 1
		}); ok {
			c.Path = p
		}
	}
}

// ---------- fire-fighting ----------

// fireFighter says whether a colonist is in a state to take on a fire.
func (g *Game) canFightFire(c *Colonist) bool {
	return !g.noFireFight && g.fireCount > 0 && c.Health > 40 && c.FireCD <= 0
}

// fireTargetPred picks burning tiles worth fighting that nobody else has taken.
func fireTargetPred(t *Tile, _, _ int, self bool) bool {
	return !self && t.Fire > 0 && float64(t.Fire) < fireBurnTime-2.5 && t.ResT == 0
}

// fireReachable checks, without side effects, that there is a fire this colonist can walk to.
func (g *Game) fireReachable(c *Colonist) bool {
	x, y := g.tileOf(c)
	_, ok := g.W.bfs(x, y, 40, func(a, b int) bool { _, f := g.findTarget(a, b, fireTargetPred); return f })
	if !ok {
		c.FireCD = 3
	}
	return ok
}

// tryFirefight sends the colonist to the nearest burning tile nobody else is dealing with.
func (g *Game) tryFirefight(c *Colonist) bool {
	if !g.canFightFire(c) {
		return false
	}
	if g.tryTarget(c, fireTargetPred, func(tg pt, t *Tile) (Job, string) {
		return Job{Kind: JFirefight, Need: 3.5}, "Going to fight fire"
	}) {
		return true
	}
	c.FireCD = 3 // nothing reachable: don't keep trying every tick
	return false
}

// fireHeat is the danger of standing next to flames: a little damage and a chance of a burn.
func (g *Game) fireHeat(c *Colonist, dt float64) {
	c.Health -= 0.8 * dt * c.damageTakenMult()
	if g.chance(dt * 0.05) {
		if g.addWound(c, "Burn", 1.5, false) {
			g.msg("%s got burnt fighting the fire.", c.Name)
		}
	}
}

func (g *Game) doFirefight(c *Colonist, dt float64) {
	j := &c.Job
	t := g.W.At(j.Target.x, j.Target.y)
	if t == nil || t.Fire == 0 {
		g.finish(c) // someone got it, or it burned out
		return
	}
	c.Status = "Fighting fire"
	g.fireHeat(c, dt)
	j.Work += dt * c.rate(SConstruction)
	g.gainXP(c, SConstruction, dt*0.7)
	if j.Work-j.Last > 0.8 {
		j.Last = j.Work
		g.emit("splash", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
	}
	if j.Work >= j.Need {
		t.Fire = 0
		g.stats.Extinguished++
		g.emit("hiss", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
		if g.fireCount <= 1 {
			g.msg("%s put out the last of the fire.", c.Name)
		}
		g.finish(c)
	}
}
