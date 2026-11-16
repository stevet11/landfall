package main

import (
	"fmt"
	"math"
)

// ---------- item values ----------

var itemValue = [numItems]float64{0, 2, 3, 4, 25, 1, 25, 30, 40, 220, 90, 4, 14, 40, 70, 120, 260, 520, 3, 60, 90, 3, 4, 5, 14, 14, 7, 12, 4}

// ---------- paying for things ----------

func (g *Game) canAfford(c Cost) bool {
	if g.stock[c.Item] < c.N {
		return false
	}
	return c.N2 == 0 || g.stock[c.Item2] >= c.N2
}

// pay takes the materials for a building out of the stockpile.
func (g *Game) pay(c Cost) bool {
	if !g.canAfford(c) {
		return false
	}
	g.takeStock(c.Item, c.N)
	if c.N2 > 0 {
		g.takeStock(c.Item2, c.N2)
	}
	return true
}

// addToStock puts items into stockpile slots (or on the ground nearby if it is full).
func (g *Game) addToStock(it Item, n int) {
	for n > 0 {
		k := min(n, maxStack(it))
		dst, ok := g.findDst(it, k, g.start.x, g.start.y)
		if !ok {
			g.placeItem(g.start.x, g.start.y, it, k)
		} else {
			t := g.W.At(dst.x, dst.y)
			t.Item = it
			t.ItemN += k
		}
		n -= k
	}
	g.recompute()
}

// ---------- recipes and production ----------

type Recipe struct {
	Name      string
	Bench     Obj
	In1       Item
	N1        int
	In2       Item
	N2        int
	Out       Item
	OutN      int
	Skill     Skill
	MinLvl    int
	Time      float64
	NeedPower bool
	Fuel      int // wood burned as fuel per batch (stoves)
}

var recipes = []Recipe{
	{"Carving", OBench, IWood, 3, INone, 0, ICarving, 1, SCrafting, 0, 6, false, 0},
	{"Sculpture", OBench, IStone, 3, INone, 0, ISculpture, 1, SCrafting, 2, 8, false, 0},
	{"Component", OBench, IStone, 2, IWood, 2, IComponent, 1, SCrafting, 3, 10, false, 0},
	{"Gadget", OFab, IComponent, 2, INone, 0, IGadget, 1, SEngineering, 4, 14, true, 0},
	{"Iron", OFurnace, IOre, 2, IWood, 1, IIron, 1, SCrafting, 2, 8, false, 0},
	{"Knife", OBench, IIron, 1, IWood, 1, IKnife, 1, SCrafting, 2, 8, false, 0},
	{"Axe", OBench, IIron, 2, IWood, 2, IAxe, 1, SCrafting, 3, 10, false, 0},
	{"Sword", OBench, IIron, 3, IWood, 1, ISword, 1, SCrafting, 5, 14, false, 0},
	{"Ammo", OFab, IIron, 1, INone, 0, IAmmo, 8, SEngineering, 3, 10, true, 0},
	{"Pistol", OFab, IIron, 2, IComponent, 2, IPistol, 1, SEngineering, 5, 16, true, 0},
	{"Rifle", OFab, IIron, 3, IComponent, 3, IRifle, 1, SEngineering, 8, 24, true, 0},
	{"Pickaxe", OBench, IIron, 2, IWood, 1, IPickaxe, 1, SCrafting, 2, 9, false, 0},
	{"Cloth", OBench, ICotton, 2, INone, 0, ICloth, 1, SCrafting, 1, 6, false, 0},
	{"Parka", OBench, ICloth, 3, INone, 0, IParka, 1, SCrafting, 3, 12, false, 0},
	{"Bread", OStove, IWheat, 2, INone, 0, IBread, 3, SCooking, 0, 8, false, 1},
	{"Meal", OStove, ICorn, 1, IFood, 1, IMeal, 2, SCooking, 3, 12, false, 1},
	{"Tile", OFurnace, IStone, 2, INone, 0, ITile, 4, SCrafting, 1, 6, false, 1},
}

type ProdSetting struct {
	On     bool
	Target int
	Urgent bool // worked ahead of building, roofing and other ordinary jobs
}

func defaultProd() []ProdSetting {
	return []ProdSetting{{Target: 10}, {Target: 10}, {Target: 6}, {Target: 2}, {Target: 12}, {Target: 2}, {Target: 1}, {Target: 1}, {Target: 40}, {Target: 1}, {Target: 1}, {Target: 2}, {Target: 6}, {Target: 3}, {Target: 20}, {Target: 10}, {Target: 16}}
}

func (g *Game) hasInputs(r Recipe) bool {
	if g.stock[r.In1] < r.N1 {
		return false
	}
	if r.Fuel > 0 { // fuel is wood, on top of any wood used as an ingredient
		need := r.Fuel
		if r.In1 == IWood {
			need += r.N1
		}
		if r.In2 == IWood {
			need += r.N2
		}
		if g.stock[IWood] < need {
			return false
		}
	}
	return r.N2 == 0 || g.stock[r.In2] >= r.N2
}

func (g *Game) takeInputs(r Recipe) bool {
	if !g.hasInputs(r) {
		return false
	}
	g.takeStock(r.In1, r.N1)
	if r.N2 > 0 {
		g.takeStock(r.In2, r.N2)
	}
	if r.Fuel > 0 {
		g.takeStock(IWood, r.Fuel)
	}
	return true
}

func (g *Game) inProgress(i int) int {
	n := 0
	for _, c := range g.cols {
		if c.Job.Kind == JCraft && c.Job.Recipe == i {
			n++
		}
	}
	return n
}

// benchCount counts finished buildings of a kind.
func (g *Game) benchCount(o Obj) int {
	n := 0
	for i := range g.W.T {
		if g.W.T[i].Obj == o {
			n++
		}
	}
	return n
}

// recipeStatus explains why a recipe is (not) being made, for the production panel.
func (g *Game) recipeStatus(i int) string {
	r := recipes[i]
	if g.benchCount(r.Bench) == 0 {
		return "needs a " + map[Obj]string{OBench: "workbench", OFab: "fabricator", OFurnace: "furnace", OStove: "stove"}[r.Bench]
	}
	if r.NeedPower && g.powerRatio < 0.5 {
		return "not enough power"
	}
	if g.bestSkill(r.Skill) < r.MinLvl {
		return "nobody has " + skillNames[r.Skill] + " " + itoa(r.MinLvl)
	}
	if !g.hasInputs(r) {
		return "missing materials"
	}
	if !g.prod[i].On {
		return "off"
	}
	if g.stock[r.Out] >= g.prod[i].Target {
		return "target reached"
	}
	return "working"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// tryCraft finds a recipe that needs making and a bench to make it at.
func (g *Game) tryCraft(c *Colonist, urgentOnly bool) bool {
	for i, r := range recipes {
		if urgentOnly && !g.prod[i].Urgent {
			continue
		}
		if !g.prod[i].On || g.stock[r.Out]+g.inProgress(i)*r.OutN >= g.prod[i].Target {
			continue
		}
		if c.lvl(r.Skill) < r.MinLvl || !g.hasInputs(r) || (r.NeedPower && g.powerRatio < 0.5) {
			continue
		}
		bench := r.Bench
		if g.tryTarget(c, func(t *Tile, _, _ int, self bool) bool {
			return t.Obj == bench && t.ResT == 0 && !self
		}, func(tg pt, t *Tile) (Job, string) {
			return Job{Kind: JCraft, Recipe: i, Need: r.Time}, "Going to craft"
		}) {
			return true
		}
	}
	return false
}

// tryStudy sends an idle colonist to a free study desk.
func (g *Game) tryStudy(c *Colonist) bool {
	if c.Rest < 35 || c.Hunger < 40 || c.lvl(c.Focus) >= 14 {
		return false
	}
	return g.tryTarget(c, func(t *Tile, _, _ int, self bool) bool {
		return t.Obj == ODesk && t.ResT == 0 && !self
	}, func(tg pt, t *Tile) (Job, string) {
		return Job{Kind: JStudy, Need: 25}, "Going to study"
	})
}

func (g *Game) doCraft(c *Colonist, dt float64) {
	j := &c.Job
	r := recipes[j.Recipe]
	t := g.W.At(j.Target.x, j.Target.y)
	if t.Obj != r.Bench {
		g.abort(c)
		return
	}
	if !j.Paid {
		if !g.takeInputs(r) {
			g.abort(c)
			c.Idle = 3
			return
		}
		j.Paid = true
	}
	c.Status = "Making " + r.Name
	if r.NeedPower && g.powerRatio < 0.5 {
		c.Status = "Waiting for power"
		return
	}
	rate := c.rate(r.Skill)
	if r.Skill != SCrafting {
		rate = (rate + c.rate(SCrafting)) / 2
	}
	x, y := g.tileOf(c)
	j.Work += dt * rate * g.nightFactor(x, y)
	g.gainXP(c, r.Skill, dt*0.9)
	if j.Work-j.Last > 1.2 {
		j.Last = j.Work
		g.emit("hammer", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
	}
	if j.Work >= j.Need {
		n := r.OutN
		if g.chance(0.03 * float64(c.lvl(r.Skill))) { // skilled hands waste less
			n++
		}
		g.placeItem(j.Target.x, j.Target.y, r.Out, n)
		g.stats.Crafted += n
		g.finish(c)
	}
}

func (g *Game) doStudy(c *Colonist, dt float64) {
	j := &c.Job
	if t := g.W.At(j.Target.x, j.Target.y); t.Obj != ODesk {
		g.abort(c)
		return
	}
	if !j.Paid {
		j.Paid = true
		if g.stock[IBook] > 0 && c.lvl(c.Focus) < 14 && g.takeStock(IBook, 1) {
			g.msg("%s reads a book on %s.", c.Name, skillNames[c.Focus])
			g.gainXP(c, c.Focus, 150)
		}
	}
	c.Status = "Studying " + skillNames[c.Focus]
	if c.lvl(c.Focus) < 10 {
		g.gainXP(c, c.Focus, dt*2.2)
	}
	j.Work += dt
	if j.Work >= j.Need {
		g.finish(c)
	}
}

// ---------- electricity ----------

// updatePower recomputes supply, demand and which lamps/heaters run. Generators burn
// wood from the stockpile; solar panels only work in daylight; the skill of the best
// engineer boosts output. All powered buildings share one grid.
func (g *Game) updatePower(dt float64) {
	g.lamps = g.lamps[:0]
	g.eheaters = g.eheaters[:0]
	eng := 1 + 0.03*float64(g.bestSkill(SEngineering))
	sun := math.Max(0, math.Sin(math.Pi*(g.hour()-6)/12))
	if g.precipitating() {
		sun *= rainSolar
	}
	if g.outT < -15 { // panels are less efficient in deep freeze? keep simple: no effect
		sun *= 1
	}
	supply, demand := 0.0, 0.0
	var lamps, heaters []int
	for i := range g.W.T {
		t := &g.W.T[i]
		switch t.Obj {
		case OGen:
			t.Fuel -= float32(dt)
			if t.Fuel <= 0 {
				t.Fuel = 0
				if g.takeStock(IWood, 1) {
					t.Fuel = 30
				}
			}
			if t.Fuel > 0 {
				supply += 150 * eng
			}
		case OSolar:
			supply += 90 * sun * eng
		case OLamp:
			demand += 25
			lamps = append(lamps, i)
		case OEHeater:
			demand += 150
			heaters = append(heaters, i)
		case OFab:
			demand += 100
		}
	}
	g.powerSupply, g.powerDemand = supply, demand
	g.powerRatio = 1
	if demand > 0 {
		g.powerRatio = math.Min(1, supply/demand)
	}
	if g.powerRatio >= 0.3 {
		g.lamps = append(g.lamps, lamps...)
	}
	if g.powerRatio >= 0.4 {
		g.eheaters = append(g.eheaters, heaters...)
	}
}

func (g *Game) hasPower() bool { return g.powerSupply > 0 || g.powerDemand > 0 }

// lit reports whether a tile is within reach of a working lamp.
func (g *Game) lit(x, y int) bool {
	for _, l := range g.lamps {
		if math.Hypot(float64(l%MapW-x), float64(l/MapW-y)) <= lampRadius {
			return true
		}
	}
	return false
}

// nightFactor slows outdoor work in the dark unless a lamp is nearby.
func (g *Game) nightFactor(x, y int) float64 {
	if g.isNight() && !g.indoors(x, y) && !g.lit(x, y) {
		return 0.8
	}
	return 1
}

func (g *Game) lampOn(i int) bool {
	for _, l := range g.lamps {
		if l == i {
			return true
		}
	}
	return false
}

func (g *Game) eheaterOn(i int) bool {
	for _, l := range g.eheaters {
		if l == i {
			return true
		}
	}
	return false
}

// blueprintBlock explains why a blueprint isn't being built ("" if nothing is wrong).
func (g *Game) blueprintBlock(t *Tile) string {
	c := t.blueCost()
	missing := func(it Item, n int) string {
		if g.stock[it] >= n {
			return ""
		}
		s := fmt.Sprintf("needs %d more %s", n-g.stock[it], itemNames[it])
		for i, r := range recipes {
			if r.Out == it && !g.prod[i].On {
				s += fmt.Sprintf(" (turn on %s production: P)", r.Name)
			}
		}
		return s
	}
	if s := missing(c.Item, c.N); s != "" {
		return s
	}
	if c.N2 > 0 {
		if s := missing(c.Item2, c.N2); s != "" {
			return s
		}
	}
	if c.MinEng > 0 && g.bestSkill(SEngineering) < c.MinEng {
		return fmt.Sprintf("nobody has Engineering %d", c.MinEng)
	}
	return ""
}

// powerProblem describes an unpowered building ("" if it works).
func (g *Game) powerProblem(o Obj) string {
	need := map[Obj]float64{OLamp: 0.3, OEHeater: 0.4, OFab: 0.5}[o]
	if need == 0 || g.powerRatio >= need {
		return ""
	}
	if g.powerSupply == 0 {
		return "no power: " + g.noPowerReason()
	}
	return fmt.Sprintf("not enough power (%.0f of %.0f W)", g.powerSupply, g.powerDemand)
}

// advise nudges the player about things that silently don't work.
func (g *Game) advise(dt float64) {
	g.hintT -= dt
	if g.hintT > 0 {
		return
	}
	g.hintT = 60
	if msg := g.powerAdvice(); msg != "" {
		g.msg("%s", msg)
		return
	}
	if g.burnerCount() > 0 && g.stock[IWood] < 15 {
		g.msg("Wood is running low (%d left): generators, heaters and stoves burn it. Mark more trees to chop (key 2).", g.stock[IWood])
		return
	}
	if g.stock[IWood] < roofWood {
		for i := range g.W.T {
			if g.W.T[i].Desig == DRoof {
				g.msg("Roofs need %d wood each; stock some wood in a stockpile.", roofWood)
				return
			}
		}
	}
	type key struct {
		b Blue
		m Material
	}
	seen := map[key]bool{}
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Blue == BNone || seen[key{t.Blue, t.Mat}] {
			continue
		}
		seen[key{t.Blue, t.Mat}] = true
		if why := g.blueprintBlock(t); why != "" {
			g.msg("The %s can't be built yet: %s.", t.blueName(), why)
			return
		}
	}
}

// ---------- roofs and spoilage ----------

// roofWood is the wood a roof tile costs (removing one returns half).
const roofWood = 2

// spoilPerDay is the fraction of a stack that rots per day when left in the open.
var spoilPerDay = [numItems]float64{IFood: 0.30, IMedicine: 0.06, IBook: 0.04, IBread: 0.15, IMeal: 0.35, ICorn: 0.20, IWheat: 0.04}

const shelteredSpoil = 0.0 // goods under a roof never spoil

// spoilFactor scales spoilage by temperature: frozen goods keep, warm ones rot faster.
func spoilFactor(temp float64) float64 {
	switch {
	case temp <= 0:
		return 0
	case temp < 10:
		return 0.5
	case temp < 26:
		return 1
	}
	return 1.6
}

// updateSpoilage rots perishable items lying on the ground or in stockpiles. Goods under a roof
// keep far longer, and frozen goods do not rot at all.
func (g *Game) updateSpoilage(dt float64) {
	lost, openLost := 0, 0
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Item == INone || t.ItemN == 0 || spoilPerDay[t.Item] == 0 {
			continue
		}
		x, y := i%MapW, i/MapW
		f := spoilFactor(g.tempAt(x, y))
		if t.Roof {
			f *= shelteredSpoil
		} else if g.raining() {
			f *= rainSpoil
		}
		t.Rot += float32(float64(t.ItemN) * spoilPerDay[t.Item] * f * dt / DayLen)
		for t.Rot >= 1 && t.ItemN > 0 {
			t.Rot--
			t.ItemN--
			lost++
			if !t.Roof {
				openLost++
			}
		}
		if t.ItemN == 0 {
			t.Item, t.Rot = INone, 0
		}
	}
	if lost > 0 {
		g.stats.Spoiled += lost
		g.spoilOpen += openLost
		g.recompute()
	}
	// tell the player, at most about twice a day
	if g.spoilOpen >= 5 && g.time-g.lastSpoilAlert > DayLen*0.5 {
		g.lastSpoilAlert = g.time
		g.alert("%d goods rotted in the open! Roof your stockpile with the Roof tool (R).", g.spoilOpen)
		g.spoilOpen = 0
	}
}

// spoilNote describes how an item on a tile is faring, for tooltips ("" if it doesn't spoil).
func (g *Game) spoilNote(x, y int) string {
	t := g.W.At(x, y)
	if t == nil || t.Item == INone || spoilPerDay[t.Item] == 0 {
		return ""
	}
	switch {
	case t.Roof: // under a roof nothing spoils, whatever the weather
		return "roofed: it will keep"
	case g.tempAt(x, y) <= 0:
		switch t.Item {
		case IFood, IBread, IMeal, ICorn:
			return "frozen: it won't spoil"
		}
		return "too cold to spoil right now" // (books and medicine aren't frozen, just kept cold)
	}
	return "in the open: spoils quickly (roof it with R)"
}

// ---------- food ----------

// nutrition is how much hunger one unit of an item satisfies (0 = not edible).
func nutrition(it Item) float64 {
	switch it {
	case IFood:
		return 45
	case IBread:
		return 40
	case IMeal:
		return 65
	case ICorn:
		return 20
	}
	return 0
}

// foodUnits counts everything colonists will eat in the stockpile.
func (g *Game) foodUnits() int {
	return g.stock[IFood] + g.stock[IBread] + g.stock[IMeal] + g.stock[ICorn]
}

// ---------- reading books ----------

// bookXP is what one book teaches.
const bookXP = 150

// readers counts colonists currently on their way to (or busy with) a book.
func (g *Game) readers() int {
	n := 0
	for _, c := range g.cols {
		if c.Job.Kind == JRead {
			n++
		}
	}
	return n
}

// tryRead sends a colonist to read a stockpiled book. Reading needs no desk: a book is worth a
// lot of experience, so it jumps the queue ahead of ordinary work (but not eating, healing or sleep
// when exhausted).
func (g *Game) tryRead(c *Colonist) bool {
	if g.stock[IBook] <= g.readers() || c.Rest < 35 || c.Hunger < 45 || c.Skills[c.Focus].Level >= 14 {
		return false
	}
	x, y := g.tileOf(c)
	p, ok := g.W.bfs(x, y, 90, func(a, b int) bool {
		t := g.W.At(a, b)
		return t.Zone && t.Item == IBook && t.ItemN > 0
	})
	if !ok {
		return false
	}
	c.Job = Job{Kind: JRead, Target: endOf(p, x, y), Need: 10}
	c.Path, c.Status = p, "Going to read"
	return true
}

func (g *Game) doRead(c *Colonist, dt float64) {
	j := &c.Job
	if !j.Paid {
		if !g.takeStock(IBook, 1) {
			g.abort(c) // another reader got there first
			return
		}
		j.Paid = true
	}
	c.Status = "Reading about " + skillNames[c.Focus]
	j.Work += dt
	if j.Work >= j.Need {
		g.msg("%s read a book on %s.", c.Name, skillNames[c.Focus])
		g.gainXP(c, c.Focus, bookXP)
		g.finish(c)
	}
}

// fuelReserve is the wood stock below which getting more wood comes first (and nobody stays
// indoors out of the cold): generators, heaters, stoves and furnaces all burn it.
const fuelReserve = 30

func (g *Game) burnerCount() int {
	n := 0
	for i := range g.W.T {
		switch g.W.T[i].Obj {
		case OGen, OHeater, OStove, OFurnace:
			n++
		}
	}
	return n
}

// powerCensus counts power sources.
func (g *Game) powerCensus() (gens, dry, solar int) {
	for i := range g.W.T {
		switch t := &g.W.T[i]; t.Obj {
		case OGen:
			gens++
			if t.Fuel <= 0 {
				dry++
			}
		case OSolar:
			solar++
		}
	}
	return
}

// noPowerReason says, in a few words, why nothing is running ("" if power is fine).
func (g *Game) noPowerReason() string {
	gens, dry, solar := g.powerCensus()
	switch {
	case g.powerSupply > 0:
		return ""
	case gens > 0 && dry == gens && g.stock[IWood] == 0:
		return "the generator is out of wood"
	case gens > 0 && dry == gens:
		return "the generator has no fuel loaded yet"
	case gens == 0 && solar > 0 && g.isNight():
		return "it is night and solar panels need daylight: a Generator burns wood to run after dark"
	case gens == 0 && solar > 0:
		return "the solar panels are making nothing right now"
	case gens == 0 && solar == 0:
		return "build a Generator or Solar panel (B)"
	}
	return "the power sources are not producing"
}

// powerAdvice is the warning shown when machines can't run, tailored to what is actually wrong.
func (g *Game) powerAdvice() string {
	if g.powerDemand <= 0 {
		return ""
	}
	if r := g.noPowerReason(); r != "" {
		gens, dry, _ := g.powerCensus()
		if gens > 0 && dry == gens && g.stock[IWood] == 0 {
			return "Your generator is out of wood! Get wood into a stockpile (chop trees, key 2) to bring the lights and machines back."
		}
		return "Your lamps and machines have no power: " + r + "."
	}
	if g.powerRatio < 0.7 {
		return fmt.Sprintf("Not enough power: %.0f W available but %.0f W wanted. Add a generator or solar panels, or switch some machines off.", g.powerSupply, g.powerDemand)
	}
	return ""
}
