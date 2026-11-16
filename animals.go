package main

import "math"

type AKind int

const (
	ADeer AKind = iota
	AWolf
	ARaider
)

type Animal struct {
	ID     int
	Kind   AKind
	X, Y   float64
	DX     float64
	HP     float64
	MaxHP  float64
	Path   []pt
	Idle   float64
	Cool   float64
	Flee   float64
	Think  float64
	Hunt   bool
	Moving bool
	Stuck  float64 // seconds a wolf has been unable to path to anyone
	Skip   float64 // game time until which hunters ignore this animal
	Dead   bool
	Weapon Item // raiders
	Goal   int
	Loot   Item
	LootN  int
	Exit   pt
	Face   int
}

// spawnAnimal places an animal on a random walkable tile between rmin and rmax of the start.
func (g *Game) spawnAnimal(k AKind, rmin, rmax float64) bool {
	for try := 0; try < 80; try++ {
		a := g.rng.Float64() * 2 * math.Pi
		r := rmin + g.rng.Float64()*(rmax-rmin)
		x, y := g.start.x+int(math.Cos(a)*r), g.start.y+int(math.Sin(a)*r*0.8)
		if !g.W.Walkable(x, y) || !g.onMainland(x, y) {
			continue
		}
		hp := 25.0
		if k == AWolf {
			hp = math.Min(60, 40+1.5*math.Floor(g.time/DayLen)) // packs toughen over time
		}
		g.nextID++
		g.animals = append(g.animals, &Animal{ID: g.nextID, Kind: k, X: float64(x) + 0.5, Y: float64(y) + 0.5, HP: hp, MaxHP: hp, DX: 1})
		return true
	}
	return false
}

func (g *Game) nearestColonist(a *Animal, maxD float64) (*Colonist, float64) {
	var best *Colonist
	bd := maxD
	for _, c := range g.cols {
		if d := math.Hypot(c.X-a.X, c.Y-a.Y); d < bd {
			bd, best = d, c
		}
	}
	return best, bd
}

func (g *Game) wander(a *Animal) {
	x, y := int(a.X), int(a.Y)
	rx, ry := x+g.rng.Intn(13)-6, y+g.rng.Intn(13)-6
	if p, ok := g.W.bfs(x, y, 9, func(p, q int) bool { return p == rx && q == ry }); ok {
		a.Path = p
	}
}

func (g *Game) updateAnimal(a *Animal, dt float64) {
	px, py := a.X, a.Y
	a.Think -= dt
	a.Cool -= dt
	a.Flee -= dt
	x, y := int(a.X), int(a.Y)
	speed := 1.2
	switch a.Kind {
	case ADeer:
		c, d := g.nearestColonist(a, 8)
		if c != nil && (a.Flee > 0 || d < 3 || (a.Hunt && d < 6)) {
			speed = 3.0
			if a.Think <= 0 {
				a.Think = 0.5
				cx, cy := c.X, c.Y
				if p, ok := g.W.bfs(x, y, 14, func(p, q int) bool {
					return math.Hypot(float64(p)+0.5-cx, float64(q)+0.5-cy) > d+7
				}); ok {
					a.Path = p
				}
			}
		} else if len(a.Path) == 0 {
			a.Idle -= dt
			if a.Idle <= 0 {
				a.Idle = 2 + g.rng.Float64()*5
				g.wander(a)
			}
		}
	case ARaider:
		speed = g.updateRaider(a, dt)
	case AWolf:
		if a.Think <= 0 && !g.onMainland(x, y) { // stranded (e.g. an old save): it wanders off
			a.Dead = true
			return
		}
		c, d := g.nearestColonist(a, 45)
		if c != nil {
			speed = 3.4
			if d < 1.0 {
				a.Path = nil
				if a.Cool <= 0 {
					a.Cool = 0.8
					c.Health -= 6 * c.damageTakenMult()
					g.stats.Bites++
					g.addWound(c, "Bite", 3, true)
					g.emit("bite", c.X, c.Y)
				}
			} else if a.Think <= 0 {
				a.Think = 0.5
				tx, ty := int(c.X), int(c.Y)
				if p, ok := g.W.bfs(x, y, 50, func(p, q int) bool { return p == tx && q == ty }); ok {
					a.Path = p
					a.Stuck = 0
				} else {
					a.Path = nil
					a.Stuck += 0.5
					if a.Stuck > 40 { // walled out or boxed in: loses interest and leaves
						a.Dead = true
					}
				}
			}
		} else if len(a.Path) == 0 {
			a.Idle -= dt
			if a.Idle <= 0 {
				a.Idle = 2 + g.rng.Float64()*4
				g.wander(a)
			}
		}
	}
	if len(a.Path) > 0 {
		if !g.stepPath(&a.X, &a.Y, &a.Path, speed, dt) {
			a.Path = nil
		}
	}
	if dx, dy := a.X-px, a.Y-py; math.Abs(dx) > 1e-4 || math.Abs(dy) > 1e-4 {
		if math.Abs(dx) > math.Abs(dy) {
			a.Face = 3
			if dx < 0 {
				a.Face = 2
			}
		} else {
			a.Face = 0
			if dy < 0 {
				a.Face = 1
			}
		}
	}
	if dx := a.X - px; math.Abs(dx) > 0.001 {
		a.DX = dx
	}
	a.Moving = math.Hypot(a.X-px, a.Y-py) > 1e-5
}

// landmass returns the tiles an animal could walk to from the colony: a flood fill over
// walkable tiles from the colonists (or the start), with the same movement rules as
// pathfinding. Islands, forest pockets and walled-off areas are therefore excluded.
// The result is cached for ten seconds of game time.
func (g *Game) landmass() []bool {
	if g.land != nil && g.tick-g.landTick < 600 {
		return g.land
	}
	W := g.W
	land := make([]bool, MapW*MapH)
	var q []int
	seed := func(x, y int) {
		if W.Walkable(x, y) && !land[y*MapW+x] {
			land[y*MapW+x] = true
			q = append(q, y*MapW+x)
		}
	}
	for _, c := range g.cols {
		x, y := g.tileOf(c)
		seed(x, y)
	}
	if len(q) == 0 {
		seed(g.start.x, g.start.y)
	}
	for len(q) > 0 {
		cur := q[len(q)-1]
		q = q[:len(q)-1]
		x, y := cur%MapW, cur/MapW
		for _, d := range dirs8 {
			nx, ny := x+d.x, y+d.y
			if !W.Walkable(nx, ny) || land[ny*MapW+nx] {
				continue
			}
			if d.x != 0 && d.y != 0 && (!W.Walkable(x+d.x, y) || !W.Walkable(x, y+d.y)) {
				continue
			}
			land[ny*MapW+nx] = true
			q = append(q, ny*MapW+nx)
		}
	}
	g.land, g.landTick = land, g.tick
	return land
}

func (g *Game) onMainland(x, y int) bool {
	return x >= 0 && y >= 0 && x < MapW && y < MapH && g.landmass()[y*MapW+x]
}
