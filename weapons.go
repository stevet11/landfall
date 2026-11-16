package main

import "math"

// Weapons: melee blades and axes, and firearms that need ammo. Colonists keep one weapon
// equipped, picked up from the stockpile automatically, best first.

type WeaponDef struct {
	Name  string
	Melee bool
	Dmg   float64
	Cool  float64 // seconds between attacks
	Range float64 // tiles (ranged weapons)
	Chop  float64 // chopping speed multiplier (axes)
	Acc   float64 // accuracy bonus for firearms
}

var weaponDefs = map[Item]WeaponDef{
	IKnife:  {Name: "Knife", Melee: true, Dmg: 14, Cool: 0.5},
	IAxe:    {Name: "Axe", Melee: true, Dmg: 22, Cool: 1.0, Chop: 1.5},
	ISword:  {Name: "Sword", Melee: true, Dmg: 28, Cool: 0.8},
	IPistol: {Name: "Pistol", Dmg: 16, Cool: 0.8, Range: 8, Acc: 0},
	IRifle:  {Name: "Rifle", Dmg: 30, Cool: 1.2, Range: 13, Acc: 0.1},
}

// Gear is non-weapon equipment: a tool in hand and clothing worn. Colonists pick it up from the
// stockpile on their own.
type GearDef struct {
	Name string
	Slot int
	Mine float64 // mining speed multiplier (pickaxe)
	Cold float64 // multiplier on cold damage (clothing)
}

const (
	slotTool = iota
	slotWear
)

var gearDefs = map[Item]GearDef{
	IPickaxe: {Name: "Pickaxe", Slot: slotTool, Mine: 1.6},
	IParka:   {Name: "Parka", Slot: slotWear, Cold: 0.55},
}

// gearSlot returns a pointer to the colonist's slot for an item.
func (c *Colonist) gearSlot(it Item) *Item {
	if gd, ok := gearDefs[it]; ok && gd.Slot == slotWear {
		return &c.Wear
	}
	return &c.Tool
}

// wantsGear says whether this colonist should go and fetch a piece of gear.
func (g *Game) wantsGear(c *Colonist, it Item) bool {
	gd := gearDefs[it]
	if g.stock[it] == 0 || *c.gearSlot(it) != INone {
		return false
	}
	if gd.Mine > 0 { // pickaxes go to people who will actually mine
		return g.anyMineOrders() || c.lvl(SMining) >= 3
	}
	return true
}

func (g *Game) anyMineOrders() bool {
	for i := range g.W.T {
		if g.W.T[i].Desig == DMine {
			return true
		}
	}
	return false
}

// Shot is a bullet tracer drawn briefly.
type Shot struct {
	X1, Y1, X2, Y2 float64
	T              float64
}

const (
	bareDmg  = 9.0
	bareCool = 0.7
	meleeRng = 1.15
)

// meleeDmg scales a base damage by combat skill and the Brawler trait.
func (c *Colonist) meleeDmg(base float64) float64 {
	d := base * (0.8 + 0.04*float64(c.lvl(SCombat)))
	if c.has(TrBrawler) {
		d *= 1.3
	}
	return d
}

// attackProfile is how this colonist fights right now: damage, seconds between attacks, reach,
// and whether they shoot. A gun without ammo is just a club.
func (g *Game) attackProfile(c *Colonist) (dmg, cool, rng float64, ranged bool) {
	wd, armed := weaponDefs[c.Weapon]
	switch {
	case !armed:
		return c.meleeDmg(bareDmg), bareCool, meleeRng, false
	case wd.Melee:
		return c.meleeDmg(wd.Dmg), wd.Cool, meleeRng, false
	case g.stock[IAmmo] > 0:
		return wd.Dmg * (0.85 + 0.03*float64(c.lvl(SCombat))), wd.Cool, wd.Range, true
	}
	return c.meleeDmg(6), 1.0, meleeRng, false // pistol-whipping
}

// rangedEdge is how much a gun's score is boosted for fighting at a distance: the target is hit before
// it arrives and cannot hit back. It makes rifles beat swords and pistols beat knives and axes.
const rangedEdge = 2.2

// weaponScore rates a weapon for a colonist (roughly damage per second; guns get a bonus for
// keeping the fight at a distance, and are nearly useless with no ammo).
func (g *Game) weaponScore(c *Colonist, it Item) float64 {
	wd, ok := weaponDefs[it]
	if !ok {
		return c.meleeDmg(bareDmg) / bareCool
	}
	if wd.Melee {
		return c.meleeDmg(wd.Dmg) / wd.Cool
	}
	if g.stock[IAmmo] < 5 {
		return c.meleeDmg(6)
	}
	return wd.Dmg * (0.85 + 0.03*float64(c.lvl(SCombat))) * 0.72 / wd.Cool * rangedEdge
}

// hitChance is the chance a shot lands, from skill, range and the gun.
func (g *Game) hitChance(c *Colonist, wd WeaponDef, dist float64) float64 {
	p := 0.72 + 0.03*float64(c.lvl(SCombat)) + wd.Acc - 0.02*dist
	return math.Max(0.15, math.Min(0.95, p))
}

// blocksShot says whether a tile stops bullets.
func blocksShot(t *Tile) bool {
	switch t.Obj {
	case OWall, ODoor, OTree, ORock, OOre:
		return true
	}
	return false
}

// clearShot checks the line between two points for walls, trees and rocks.
func (g *Game) clearShot(x0, y0, x1, y1 float64) bool {
	dist := math.Hypot(x1-x0, y1-y0)
	n := int(dist*2) + 1
	sx, sy := int(x0), int(y0)
	ex, ey := int(x1), int(y1)
	for i := 1; i < n; i++ {
		f := float64(i) / float64(n)
		tx, ty := int(x0+(x1-x0)*f), int(y0+(y1-y0)*f)
		if (tx == sx && ty == sy) || (tx == ex && ty == ey) {
			continue
		}
		if t := g.W.At(tx, ty); t == nil || blocksShot(t) {
			return false
		}
	}
	return true
}

// shoot fires one round (taking ammo from the stockpile) at an animal.
func (g *Game) shoot(c *Colonist, a *Animal, dmg float64) {
	wd := weaponDefs[c.Weapon]
	if !g.takeStock(IAmmo, 1) {
		return
	}
	g.stats.Shots++
	dist := math.Hypot(a.X-c.X, a.Y-c.Y)
	g.shots = append(g.shots, Shot{c.X, c.Y, a.X, a.Y, 0.12})
	g.emit(map[Item]string{IPistol: "shot", IRifle: "rifle"}[c.Weapon], c.X, c.Y)
	g.gainXP(c, SCombat, 2.5)
	if g.chance(g.hitChance(c, wd, dist)) {
		a.HP -= dmg
		a.Flee = 4
		g.stats.Hits++
		g.emit("hit", a.X, a.Y)
	}
}

// ---------- equipping ----------

// bestWeaponFor finds the stockpiled weapon that most improves on what the colonist holds.
func (g *Game) bestWeaponFor(c *Colonist) (Item, bool) {
	cur := g.weaponScore(c, c.Weapon)
	best, bs := INone, cur*1.1 // must be a real improvement
	for it := range weaponDefs {
		if g.stock[it] == 0 {
			continue
		}
		if s := g.weaponScore(c, it); s > bs {
			best, bs = it, s
		}
	}
	return best, best != INone
}

// tryEquip sends the colonist to the stockpile to pick up a better weapon.
func (g *Game) tryEquip(c *Colonist) bool {
	it, ok := g.bestWeaponFor(c)
	if !ok {
		for gi := range gearDefs {
			if g.wantsGear(c, gi) {
				it, ok = gi, true
				break
			}
		}
	}
	if !ok {
		return false
	}
	x, y := g.tileOf(c)
	p, found := g.W.bfs(x, y, 90, func(a, b int) bool {
		t := g.W.At(a, b)
		return t.Zone && t.Item == it && t.ItemN > 0
	})
	if !found {
		return false
	}
	c.Job = Job{Kind: JEquip, Target: endOf(p, x, y), Item: it}
	name := weaponDefs[it].Name
	if gd, isGear := gearDefs[it]; isGear {
		name = gd.Name
	}
	c.Path, c.Status = p, "Fetching a "+name
	return true
}

func (g *Game) doEquip(c *Colonist) {
	j := &c.Job
	it := j.Item
	if !g.takeStock(it, 1) {
		g.abort(c) // someone else got it first
		return
	}
	if gd, isGear := gearDefs[it]; isGear {
		*c.gearSlot(it) = it
		g.msg("%s picks up a %s.", c.Name, gd.Name)
		g.emit("click", -1, -1)
		g.finish(c)
		return
	}
	old := c.Weapon
	c.Weapon = it
	if old != INone {
		g.addToStock(old, 1) // hand the old one back
	}
	g.msg("%s picks up a %s.", c.Name, weaponDefs[it].Name)
	g.emit("click", -1, -1)
	g.finish(c)
}
