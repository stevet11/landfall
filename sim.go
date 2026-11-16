package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
)

type JobKind int

const (
	JNone JobKind = iota
	JEat
	JBuild
	JChop
	JMine
	JHarvest
	JHaul
	JSow
	JReap
	JSleep
	JWarm
	JHunt
	JFight
	JTend
	JRecover
	JCraft
	JStudy
	JDecon
	JRoof
	JFirefight
	JEquip
	JRead
	JMove
	JPlay
)

type Job struct {
	Kind    JobKind
	Target  pt
	Dst     pt
	Stage   int
	Work    float64
	Need    float64
	Paid    bool
	Animal  *Animal
	Patient int
	Last    float64
	Recipe  int
	Item    Item // what a JEquip job is fetching
}

type Colonist struct {
	ID              int
	Name            string
	X, Y            float64 // tile units; tile (tx,ty) centre is (tx+.5, ty+.5)
	Hunger          float64
	Rest            float64
	Health          float64
	Temp            float64
	Col             color.NRGBA
	Path            []pt
	Job             Job
	Carry           Item
	CarryN          int
	Idle            float64
	Status          string
	Dead            bool
	AtkCool         float64
	Repath          float64
	Think           float64
	FightCD         float64
	FireCD          float64 // cool-down after finding no fire to fight
	Weapon          Item    // equipped weapon (INone = bare hands)
	Tool            Item    // pickaxe
	Wear            Item    // parka
	Cargo           *Cargo  // a building being carried
	Joy, Calm, Sulk float64 // spirits; seconds of calm left; seconds of sulking left
	Ill             []Illness
	Immune          [numDiseases]float64 // seconds of immunity left, per disease
	Doctor          int                  // colonist currently tending this one
	Face            int                  // 0 down, 1 up, 2 left, 3 right
	Moving          bool
	Chilled         bool
	Skills          [numSkills]SkillState
	Traits          []Trait
	Focus           Skill // skill trained at a study desk
	Injuries        []Injury
}

// Injury is a wound. Tended: 0 untreated, 1 bandaged, 2 treated with medicine.
type Injury struct {
	Kind    string
	Sev     float64
	Bleed   bool
	Tended  int
	Infect  float64
	Quality float64 // healing multiplier from the medic's skill (0 means 1)
}

// Stats are counters used by the balance harness.
type Stats struct {
	Raids, WolvesKilled, Bites, DeerKilled         int
	Crafted, Sold, Bought, Salvaged                int
	Fires, Burned, Spoiled, Extinguished           int
	Shots, Hits                                    int
	RaidersSeen, RaidersKilled, RaiderHits, Stolen int
	Illnesses, Outbreaks, Recovered, IllDeaths     int
	Played, Sulks                                  int
	Starved, Froze, Heat, Bled, Killed             int
}

type Msg struct {
	Text  string
	T     float64
	Alert bool // important: stays longer and is drawn in gold
}

type Room struct {
	Size   int
	Indoor bool
}

const DayLen = 300.0
const YearLen = DayLen * 10

type Game struct {
	W                                    *World
	cols                                 []*Colonist
	animals                              []*Animal
	nextID                               int
	stock                                [numItems]int
	zoneList                             []int
	heaters                              []int
	roomOf                               []int32
	rooms                                []Room
	roomDirty                            bool
	outT                                 float64
	wet                                  bool    // a spell of precipitation is on
	precip                               float64 // 0..1 how hard it is falling (eased)
	weatherLeft                          float64
	rng                                  *rand.Rand
	start                                pt
	time                                 float64
	tick                                 int
	slowAcc                              float64
	regrowT                              float64
	immT                                 float64
	deerT                                float64
	nextRaid                             float64
	msgs                                 []Msg
	onEvent                              func(name string, x, y float64)
	stats                                Stats
	over                                 bool // every colonist is dead: the game is finished
	overDay                              int
	fullWarn                             float64
	land                                 []bool
	buildSel                             Blue
	wallMat, doorMat                     Material
	cropSel                              CropKind
	floorSel                             FloorKind
	moveSrc                              int // building picked by the Move tool (-1 none)
	lastFireAlert                        float64
	lastFireSound                        int
	fireCount                            int    // tiles burning right now
	shots                                []Shot // bullet tracers, for drawing
	raid                                 *raidInfo
	nextRaiders                          float64
	nextOutbreak                         float64
	noFireFight                          bool // tests: colonists ignore fires
	spoilOpen                            int
	lastSpoilAlert                       float64
	noHaul                               map[int]float64 // loose items nobody could deliver: tile -> time to try again
	hintT                                float64
	prod                                 []ProdSetting
	lamps                                []int
	eheaters                             []int
	powerSupply, powerDemand, powerRatio float64
	trader                               *Trader
	nextTrader                           float64
	landTick                             int
	lastSeason                           string

	ui
}

var names = []string{"Ada", "Bram", "Cleo", "Dov", "Edda", "Finn", "Gwen", "Hale", "Iris", "Jory", "Kit", "Lena", "Moss", "Nell", "Orin", "Pia"}
var colors = []color.NRGBA{{230, 90, 80, 255}, {80, 150, 230, 255}, {240, 200, 70, 255}, {170, 110, 220, 255}, {90, 200, 130, 255}, {240, 140, 60, 255}, {230, 120, 180, 255}, {120, 220, 220, 255}}

func newGame(seed int64) *Game {
	g := &Game{rng: rand.New(rand.NewSource(seed + 1))}
	g.W, g.start = newWorld(seed)
	g.time = DayLen * 0.3
	g.nextRaid = DayLen * 2.3
	g.nextRaiders = DayLen * 5
	g.nextOutbreak = DayLen * (3.5 + g.rng.Float64()*1.5)
	g.nextTrader = DayLen * 2.5
	g.weatherLeft = DayLen * (0.3 + 0.5*g.rng.Float64())
	g.prod = defaultProd()
	g.wallMat, g.doorMat = MatStone, MatWood
	g.moveSrc = -1
	g.powerRatio = 1
	// starting stockpile (3x3)
	sx, sy := g.start.x+2, g.start.y-1
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			g.W.At(sx+dx, sy+dy).Zone = true
			g.W.At(sx+dx, sy+dy).Roof = true
		}
	}
	g.W.At(sx, sy).Item, g.W.At(sx, sy).ItemN = IFood, 25
	g.W.At(sx+1, sy).Item, g.W.At(sx+1, sy).ItemN = IWood, 40
	g.W.At(sx+2, sy).Item, g.W.At(sx+2, sy).ItemN = IStone, 12
	// the founding crew: a medic, a crafter and an engineer, so every system is reachable
	for i, sp := range []struct {
		s Skill
		l int
	}{{SMedicine, 4}, {SCrafting, 4}, {SEngineering, 3}} {
		g.addColonist(float64(g.start.x-2+i)+0.5, float64(g.start.y)+0.5, sp.s, sp.l)
	}
	for i := 0; i < 12; i++ {
		g.spawnAnimal(ADeer, 18, 60)
	}
	g.computeRooms()
	g.outT = g.outdoorTemp()
	g.recompute()
	g.msg("Your colonists have landed. Press H for help.")
	g.initUI()
	return g
}

// addColonist creates a colonist with random traits and skills; spec/specLevel give them
// a standout skill (specLevel 0 means no guaranteed specialty).
func (g *Game) addColonist(x, y float64, spec Skill, specLevel int) *Colonist {
	g.nextID++
	i := g.nextID - 1
	c := &Colonist{
		ID: g.nextID, Name: names[i%len(names)], X: x, Y: y,
		Hunger: 80 + g.rng.Float64()*20, Rest: 70 + g.rng.Float64()*30, Health: 100, Joy: 65 + g.rng.Float64()*35,
		Col: colors[i%len(colors)], Status: "Idle",
	}
	g.rollColonist(c, spec, specLevel)
	g.cols = append(g.cols, c)
	return c
}

func (g *Game) msg(format string, a ...any) {
	g.push(Msg{Text: fmt.Sprintf(format, a...), T: 9})
}

// alert posts an important message that lingers (about 25 s) and is highlighted.
func (g *Game) alert(format string, a ...any) {
	g.push(Msg{Text: fmt.Sprintf(format, a...), T: 25, Alert: true})
}

func (g *Game) push(m Msg) {
	g.msgs = append(g.msgs, m)
	for len(g.msgs) > 7 { // drop the oldest ordinary message first so alerts survive
		drop := 0
		for i, x := range g.msgs {
			if !x.Alert {
				drop = i
				break
			}
		}
		g.msgs = append(g.msgs[:drop], g.msgs[drop+1:]...)
	}
}

// emit announces something audible/visible at tile position (x,y); x<0 means global.
func (g *Game) emit(name string, x, y float64) {
	if g.onEvent != nil {
		g.onEvent(name, x, y)
	}
}

func (g *Game) isNight() bool { h := g.hour(); return h < 6 || h >= 21 }

// ---- climate ----

func (g *Game) yearT() float64 { return math.Mod(g.time/YearLen, 1) }

func (g *Game) seasonName() string {
	y := g.yearT()
	switch {
	case y >= 0.2 && y < 0.4:
		return "Summer"
	case y >= 0.4 && y < 0.55:
		return "Autumn"
	case y >= 0.55 && y < 0.95:
		return "Winter"
	}
	return "Spring"
}

func (g *Game) announceSeason() {
	s := g.seasonName()
	if s == g.lastSeason {
		return
	}
	if g.lastSeason != "" {
		g.msg("%s", map[string]string{
			"Spring": "Spring has arrived. Crops can be planted again.",
			"Summer": "Summer has arrived.",
			"Autumn": "Autumn has arrived. Build a heated room before winter!",
			"Winter": "Winter has arrived! Colonists will freeze outdoors.",
		}[s])
	}
	g.lastSeason = s
}

func (g *Game) outdoorTemp() float64 {
	base := 8 + 19*math.Sin(2*math.Pi*(g.yearT()-0.02))
	return base + 5*math.Sin(2*math.Pi*(g.hour()-9)/24)
}

func (g *Game) computeRooms() {
	W := g.W
	n := MapW * MapH
	if g.roomOf == nil {
		g.roomOf = make([]int32, n)
	}
	for i := range g.roomOf {
		g.roomOf[i] = -2
	}
	g.rooms = g.rooms[:0]
	barrier := func(t *Tile) bool { return t.Obj == OWall || t.Obj == ODoor }
	var stack []int32
	for i := 0; i < n; i++ {
		if g.roomOf[i] != -2 {
			continue
		}
		if barrier(&W.T[i]) {
			g.roomOf[i] = -1
			continue
		}
		id := int32(len(g.rooms))
		size, edge := 0, false
		stack = append(stack[:0], int32(i))
		g.roomOf[i] = id
		for len(stack) > 0 {
			cur := int(stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			size++
			x, y := cur%MapW, cur/MapW
			if x == 0 || y == 0 || x == MapW-1 || y == MapH-1 {
				edge = true
			}
			for _, d := range dirs8[:4] {
				nx, ny := x+d.x, y+d.y
				if nx < 0 || ny < 0 || nx >= MapW || ny >= MapH {
					continue
				}
				j := ny*MapW + nx
				if g.roomOf[j] != -2 {
					continue
				}
				if barrier(&W.T[j]) {
					g.roomOf[j] = -1
					continue
				}
				g.roomOf[j] = id
				stack = append(stack, int32(j))
			}
		}
		g.rooms = append(g.rooms, Room{size, !edge && size <= 600})
	}
	g.roomDirty = false
}

func (g *Game) indoors(x, y int) bool {
	if x < 0 || y < 0 || x >= MapW || y >= MapH {
		return false
	}
	r := g.roomOf[y*MapW+x]
	return r >= 0 && g.rooms[r].Indoor && g.W.T[y*MapW+x].Roof
}

// covered reports whether a tile has a roof over it (shelter from the elements).
func (g *Game) covered(x, y int) bool {
	t := g.W.At(x, y)
	return t != nil && t.Roof
}

func (g *Game) tempAt(x, y int) float64 {
	T := g.outT
	if x < 0 || y < 0 || x >= MapW || y >= MapH {
		return T
	}
	r := g.roomOf[y*MapW+x]
	if r < 0 {
		return T + (16-T)*0.3
	}
	if g.rooms[r].Indoor && g.W.T[y*MapW+x].Roof {
		t := T + (16-T)*0.65
		bonus := 0.0
		for _, h := range g.heaters {
			if g.roomOf[h] == r {
				bonus += 300 / float64(g.rooms[r].Size)
			}
		}
		return t + math.Min(bonus, 28)
	}
	bonus := 0.0
	for _, h := range g.heaters {
		d := math.Hypot(float64(h%MapW-x), float64(h/MapW-y))
		if d < 5 {
			bonus += 6 * (1 - d/5)
		}
	}
	return T + math.Min(bonus, 10)
}

// ---- bookkeeping ----

func (g *Game) recompute() {
	g.stock = [numItems]int{}
	g.zoneList = g.zoneList[:0]
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Zone {
			g.zoneList = append(g.zoneList, i)
			if t.Item != INone {
				g.stock[t.Item] += t.ItemN
			}
		}
	}
}

// slow runs every few ticks: fuel, crops, heater list.
func (g *Game) slow(dt float64) {
	g.recompute()
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Urgent && t.Desig == DNone && t.Blue == BNone {
			t.Urgent = false // the order is done
		}
		if t.Obj == OHeater {
			t.Fuel -= float32(dt)
			if t.Fuel <= 0 {
				t.Fuel = 0
				if g.takeStock(IWood, 1) {
					t.Fuel = 40
				}
			}
		}
	}
	g.spreadDisease(dt)
	g.tidyMoves()
	g.updatePower(dt)
	g.updateFire(dt)
	g.updateSpoilage(dt)
	g.advise(dt)
	g.heaters = g.heaters[:0]
	for i := range g.W.T {
		if t := &g.W.T[i]; t.Obj == OHeater && t.Fuel > 0 {
			g.heaters = append(g.heaters, i)
		}
	}
	// warn when items are piling up with nowhere to go
	g.fullWarn -= dt
	if g.fullWarn <= 0 {
		free, loose := 0, 0
		for _, i := range g.zoneList {
			if g.W.T[i].ItemN == 0 {
				free++
			}
		}
		for i := range g.W.T {
			if t := &g.W.T[i]; !t.Zone && t.Item != INone && t.ItemN > 0 {
				loose++
			}
		}
		if len(g.zoneList) > 0 && free == 0 && loose > 0 {
			g.msg("The stockpile is full! Add more Stockpile tiles.")
			g.fullWarn = 90
		} else {
			g.fullWarn = 10
		}
	}
	g.heaters = append(g.heaters, g.eheaters...)
	frost := 0
	for i := range g.W.T {
		t := &g.W.T[i]
		if !t.Crop {
			continue
		}
		T := g.tempAt(i%MapW, i/MapW)
		switch {
		case T < -2:
			t.Crop, t.Grow = false, 0
			frost++
		case T >= cropDefs[t.CropKind].MinT && T <= 38 && t.Grow < 1:
			rate := 1.0
			if g.raining() && g.exposed(i%MapW, i/MapW) {
				rate = rainGrowth
			}
			t.Grow += float32(rate * dt / cropDefs[t.CropKind].Time)
		}
	}
	if frost > 0 {
		g.msg("Frost killed %d crops! Grow them indoors.", frost)
	}
}

func (g *Game) takeStock(it Item, n int) bool {
	if g.stock[it] < n {
		return false
	}
	for _, i := range g.zoneList {
		t := &g.W.T[i]
		if t.Item != it || t.ItemN == 0 {
			continue
		}
		k := min(n, t.ItemN)
		t.ItemN -= k
		n -= k
		g.stock[it] -= k
		if t.ItemN == 0 {
			t.Item = INone
		}
		if n == 0 {
			return true
		}
	}
	return n == 0
}

// placeItem drops an item stack on the nearest free walkable tile.
func (g *Game) placeItem(x, y int, it Item, n int) {
	p, ok := g.W.bfs(x, y, 8, func(a, b int) bool {
		t := g.W.At(a, b)
		return t.Item == INone && g.W.Walkable(a, b) // never on a bench, machine or wall: nobody could reach it
	})
	if !ok {
		return
	}
	e := endOf(p, x, y)
	t := g.W.At(e.x, e.y)
	t.Item, t.ItemN = it, n
}

// step advances the simulation by dt seconds.
func (g *Game) step(dt float64) {
	g.time += dt
	g.tick++
	g.outT = g.outdoorTemp()
	g.weatherStep(dt)
	g.announceSeason()
	if g.roomDirty {
		g.computeRooms()
	}
	for _, c := range g.cols {
		g.updateColonist(c, dt)
	}
	for _, a := range g.animals {
		g.updateAnimal(a, dt)
	}
	alive := g.cols[:0]
	for _, c := range g.cols {
		if !c.Dead {
			alive = append(alive, c)
		}
	}
	g.cols = alive
	g.checkGameOver()
	aa := g.animals[:0]
	for _, a := range g.animals {
		if !a.Dead {
			aa = append(aa, a)
		}
	}
	g.animals = aa

	g.slowAcc += dt
	if g.tick%15 == 0 {
		g.slow(g.slowAcc)
		g.slowAcc = 0
	}
	live := g.shots[:0]
	for _, sh := range g.shots {
		if sh.T -= dt; sh.T > 0 {
			live = append(live, sh)
		}
	}
	g.shots = live
	g.updateTrader(dt)
	g.worldEvents(dt)
}

func (g *Game) worldEvents(dt float64) {
	g.regrowT += dt
	if g.regrowT > 10 {
		g.regrowT = 0
		bushes := 0
		for i := range g.W.T {
			if g.W.T[i].Obj == OBush {
				bushes++
			}
		}
		if bushes < 45 && g.outT > 5 {
			for k := 0; k < 30; k++ {
				x, y := g.start.x+g.rng.Intn(70)-35, g.start.y+g.rng.Intn(50)-25
				t := g.W.At(x, y)
				if t != nil && t.Ground == GGrass && t.Obj == ONone && t.Item == INone && !t.Zone && !t.Farm && !t.Floor && t.Blue == BNone {
					t.Obj = OBush
					break
				}
			}
		}
	}
	if g.tick%600 == 0 && g.outT > 5 {
		herbs := 0 // only the herbs near the colony count, so clearing your area makes new ones grow
		for i := range g.W.T {
			if g.W.T[i].Obj == OHerb && iabs(i%MapW-g.start.x) <= 40 && iabs(i/MapW-g.start.y) <= 30 {
				herbs++
			}
		}
		for k := 0; herbs < 10 && k < 40; k++ {
			x, y := g.start.x+g.rng.Intn(70)-35, g.start.y+g.rng.Intn(50)-25
			t := g.W.At(x, y)
			if t != nil && t.Ground == GGrass && t.Obj == ONone && t.Item == INone && !t.Zone && !t.Farm && !t.Floor && t.Blue == BNone {
				t.Obj = OHerb
				break
			}
		}
	}
	g.immT += dt
	if g.immT > 600 { // a newcomer roughly every two days
		g.immT = 0
		if len(g.cols) < 12 && g.foodUnits() >= 10 {
			spec := Skill(g.rng.Intn(int(numSkills)))
			c := g.addColonist(float64(g.start.x)+0.5, float64(g.start.y)+0.5, spec, 4+g.rng.Intn(5))
			g.msg("%s joins the colony: %s %d, %s.", c.Name, skillNames[spec], c.lvl(spec), c.traitList())
		}
	}
	g.deerT += dt
	if g.deerT > 45 {
		g.deerT = 0
		deer := 0
		for _, a := range g.animals {
			if a.Kind == ADeer {
				deer++
			}
		}
		if deer < 12 {
			g.spawnAnimal(ADeer, 40, 62)
		}
	}
	if g.time >= g.nextRaiders && g.raid == nil && len(g.cols) > 0 {
		if g.spawnRaid() {
			g.nextRaiders = g.time + DayLen*(3.2+g.rng.Float64()*2.2)
		} else {
			g.nextRaiders = g.time + 60
		}
	}
	g.updateRaidState()
	g.updateOutbreaks()
	if g.time >= g.nextRaid && g.isNight() {
		n := min(8, 1+len(g.cols)/3+int(g.time/DayLen)/4)
		spawned := 0
		for i := 0; i < n; i++ {
			if g.spawnAnimal(AWolf, 45, 62) {
				spawned++
			}
		}
		if spawned > 0 {
			g.stats.Raids++
			g.msg("Wolves are approaching the colony!")
			g.emit("howl", -1, -1)
		}
		g.nextRaid = g.time + DayLen*(1.0+g.rng.Float64()*0.8)
	}
}

func (g *Game) tileOf(c *Colonist) (int, int) {
	return int(math.Floor(c.X)), int(math.Floor(c.Y))
}

func (g *Game) colonistAt(p pt) bool {
	for _, c := range g.cols {
		x, y := g.tileOf(c)
		if x == p.x && y == p.y {
			return true
		}
	}
	return false
}

var dirs5 = []pt{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}

type predFn func(t *Tile, x, y int, self bool) bool

func (g *Game) findTarget(x, y int, pred predFn) (pt, bool) {
	for _, d := range dirs5 {
		t := g.W.At(x+d.x, y+d.y)
		if t != nil && pred(t, x+d.x, y+d.y, d.x == 0 && d.y == 0) {
			return pt{x + d.x, y + d.y}, true
		}
	}
	return pt{}, false
}

func desigValid(t *Tile) bool {
	switch t.Desig {
	case DChop:
		return t.Obj == OTree
	case DMine:
		return t.Obj == ORock || t.Obj == OOre
	case DHarvest:
		return t.Obj == OBush || t.Obj == OHerb
	case DDecon:
		_, built := objBlue[t.Obj]
		return built || t.Floor || t.Roof
	case DRoof:
		return !t.Roof && t.Ground != GWater && t.Obj != OTree && t.Obj != ORock && t.Obj != OOre
	}
	return false
}

func (g *Game) findDst(it Item, n, fx, fy int) (pt, bool) {
	best, bd := -1, 1<<30
	for _, i := range g.zoneList {
		t := &g.W.T[i]
		if !t.Zone || t.ResD != 0 || t.Blue != BNone {
			continue
		}
		if !(t.ItemN == 0 || (t.Item == it && t.ItemN+n <= maxStack(it))) {
			continue
		}
		x, y := i%MapW, i/MapW
		if !g.W.Walkable(x, y) {
			continue
		}
		if d := iabs(x-fx) + iabs(y-fy); d < bd {
			bd, best = d, i
		}
	}
	if best < 0 {
		return pt{}, false
	}
	return pt{best % MapW, best / MapW}, true
}

func endOf(p []pt, x, y int) pt {
	if len(p) > 0 {
		return p[len(p)-1]
	}
	return pt{x, y}
}

// tryTarget walks to the nearest tile next to (or on) a tile matching pred.
func (g *Game) tryTarget(c *Colonist, pred predFn, mk func(tg pt, t *Tile) (Job, string)) bool {
	x, y := g.tileOf(c)
	p, ok := g.W.bfs(x, y, 90, func(x, y int) bool { _, f := g.findTarget(x, y, pred); return f })
	if !ok {
		return false
	}
	e := endOf(p, x, y)
	tg, _ := g.findTarget(e.x, e.y, pred)
	t := g.W.At(tg.x, tg.y)
	t.ResT = c.ID
	c.Job, c.Status = mk(tg, t)
	c.Job.Target = tg
	c.Path = p
	return true
}

func (g *Game) assignJob(c *Colonist) bool {
	W := g.W
	x, y := g.tileOf(c)

	if c.Hunger < 45 {
		p, ok := W.bfs(x, y, 90, func(x, y int) bool {
			t := W.At(x, y)
			return nutrition(t.Item) > 0 && t.ItemN > 0
		})
		if ok {
			c.Job = Job{Kind: JEat, Target: endOf(p, x, y), Need: 1.5}
			c.Path, c.Status = p, "Going to eat"
			return true
		}
	}

	if g.tryFirefight(c) {
		return true
	}

	// medical: tend the wounded (or yourself), then rest if badly hurt
	var patient *Colonist
	bd := 1e9
	for _, p := range g.cols {
		if p.Doctor != 0 || !g.needsTend(p) {
			continue
		}
		// the most urgent patient first (bleeding, then serious illness), nearest among equals
		d := math.Hypot(p.X-c.X, p.Y-c.Y) - 1000*g.tendUrgency(p)
		if p == c {
			d += 20 // prefer being tended by someone else
		}
		if d < bd {
			bd, patient = d, p
		}
	}
	if patient != nil {
		patient.Doctor = c.ID
		c.Job = Job{Kind: JTend, Patient: patient.ID, Need: 4 / (0.8 + 0.05*float64(c.lvl(SMedicine)))}
		c.Status = "Tending " + patient.Name
		return true
	}
	if c.Health < 55 && len(c.Injuries) > 0 || c.Health < 35 || c.sickSeverity() > 0.25 {
		p, ok := W.bfs(x, y, 90, func(x, y int) bool {
			t := W.At(x, y)
			return t.Obj == OBed && t.ResT == 0
		})
		if ok {
			e := endOf(p, x, y)
			W.At(e.x, e.y).ResT = c.ID
			c.Job = Job{Kind: JRecover, Target: e}
			c.Path, c.Status = p, "Going to recover"
			return true
		}
		c.Job = Job{Kind: JRecover, Target: pt{x, y}, Stage: 1}
		c.Status = "Recovering"
		return true
	}

	if g.tryEquip(c) {
		return true
	}

	if g.tryRead(c) {
		return true
	}

	if c.Rest < 25 || (g.isNight() && c.Rest < 85) {
		p, ok := W.bfs(x, y, 90, func(x, y int) bool {
			t := W.At(x, y)
			return t.Obj == OBed && t.ResT == 0
		})
		if ok {
			e := endOf(p, x, y)
			W.At(e.x, e.y).ResT = c.ID
			c.Job = Job{Kind: JSleep, Target: e}
			c.Path, c.Status = p, "Going to bed"
			return true
		}
		if c.Rest < 25 {
			c.Job = Job{Kind: JSleep, Target: pt{x, y}, Stage: 1}
			c.Status = "Sleeping"
			return true
		}
	}

	if c.Temp < 3 {
		p, ok := W.bfs(x, y, 70, func(x, y int) bool { return g.tempAt(x, y) >= 10 })
		if ok {
			c.Job = Job{Kind: JWarm, Target: endOf(p, x, y), Need: 25}
			c.Path, c.Status = p, "Seeking warmth"
			return true
		}
	}

	if c.Sulk > 0 {
		return false // in a foul mood: no work, just wandering
	}
	if g.tryPlay(c, true) { // spirits are low: take a break now
		return true
	}

	// too cold to work outside: stay in the warm room until it eases (unless the fuel is running out:
	// then somebody has to fetch wood or the heaters and generators die and the room goes cold anyway)
	if g.outT < -7 && c.Temp >= 8 && g.stock[IWood] >= fuelReserve {
		c.Idle = 2
		return true
	}

	if g.tryUrgent(c) {
		return true
	}

	if g.tryTarget(c, g.buildPred(c), func(tg pt, t *Tile) (Job, string) {
		return Job{Kind: JBuild, Need: t.blueCost().Time}, "Going to build"
	}) {
		return true
	}

	if g.tryMove(c) {
		return true
	}

	if len(g.zoneList) > 0 {
		hp := func(x, y int) bool {
			t := W.At(x, y)
			if t.Item == INone || t.ItemN == 0 || t.Zone || t.ResI != 0 {
				return false
			}
			if until, bad := g.noHaul[y*MapW+x]; bad && g.time < until {
				return false
			}
			_, ok := g.findDst(t.Item, t.ItemN, x, y)
			return ok
		}
		if p, ok := W.bfs(x, y, 90, hp); ok {
			e := endOf(p, x, y)
			t := W.At(e.x, e.y)
			dst, _ := g.findDst(t.Item, t.ItemN, e.x, e.y)
			t.ResI = c.ID
			W.At(dst.x, dst.y).ResD = c.ID
			c.Job = Job{Kind: JHaul, Target: e, Dst: dst}
			c.Path, c.Status = p, "Going to haul"
			return true
		}
	}

	// crops: reap first, then sow
	if g.tryTarget(c, func(t *Tile, _, _ int, self bool) bool {
		return self && t.Farm && t.Crop && t.Grow >= 1 && t.ResT == 0
	}, func(tg pt, t *Tile) (Job, string) { return Job{Kind: JReap, Need: 1.5}, "Going to reap" }) {
		return true
	}
	if g.tryTarget(c, func(t *Tile, x, y int, self bool) bool {
		return self && t.Farm && !t.Crop && t.ResT == 0 && t.Obj == ONone && t.Item == INone && g.tempAt(x, y) >= cropDefs[t.CropKind].MinT
	}, func(tg pt, t *Tile) (Job, string) { return Job{Kind: JSow, Need: 1.5}, "Going to sow" }) {
		return true
	}

	// hunting
	var prey *Animal
	bd = 60.0
	for _, a := range g.animals {
		if a.Kind == ADeer && a.Hunt && a.Skip < g.time {
			if d := math.Hypot(a.X-c.X, a.Y-c.Y); d < bd {
				bd, prey = d, a
			}
		}
	}
	if prey != nil {
		c.Job = Job{Kind: JHunt, Animal: prey}
		c.Status = "Hunting"
		return true
	}

	if g.stock[IWood] < fuelReserve && g.tryTarget(c, func(t *Tile, _, _ int, self bool) bool {
		return t.Desig == DChop && t.ResT == 0 && desigValid(t) // wood is short: fetch it before anything else
	}, func(tg pt, t *Tile) (Job, string) {
		return Job{Kind: JChop, Need: 3.5}, "Going to chop (wood is short)"
	}) {
		return true
	}

	if g.tryCraft(c, false) {
		return true
	}

	if g.tryTarget(c, g.gatherPred(), g.gatherJob) {
		return true
	}
	if g.tryPlay(c, false) { // nothing to do: have some fun
		return true
	}
	return g.tryStudy(c)
}

// buildPred accepts a blueprint this colonist can start building now.
func (g *Game) buildPred(c *Colonist) predFn {
	return func(t *Tile, _, _ int, self bool) bool {
		if t.Blue == BNone || t.ResT != 0 || t.MoveFrom != 0 {
			return false
		}
		if self && solidBlue(t.Blue) {
			return false
		}
		cost := t.blueCost()
		return c.lvl(SEngineering) >= cost.MinEng && g.canAfford(cost)
	}
}

// gatherPred accepts a tile with a pending chop, mine, harvest, roof or dismantle order.
func (g *Game) gatherPred() predFn {
	return func(t *Tile, _, _ int, self bool) bool {
		return t.Desig != DNone && t.ResT == 0 && desigValid(t) && (t.Desig != DRoof || g.stock[IWood] >= roofWood)
	}
}

// gatherJob makes the job for a tile accepted by gatherPred.
func (g *Game) gatherJob(tg pt, t *Tile) (Job, string) {
	switch t.Desig {
	case DChop:
		return Job{Kind: JChop, Need: 3.5}, "Going to chop"
	case DMine:
		return Job{Kind: JMine, Need: 4.5}, "Going to mine"
	case DRoof:
		return Job{Kind: JRoof, Need: 1.2}, "Going to roof"
	case DDecon:
		need := 1.5
		if b, ok := objBlue[t.Obj]; ok {
			need = costFor(b, t.builtMat()).Time * 0.6
		}
		return Job{Kind: JDecon, Need: need}, "Going to dismantle"
	}
	return Job{Kind: JHarvest, Need: 1.5}, "Going to harvest"
}

// tryUrgent does the orders the player marked urgent (Priority tool) and the recipes marked urgent in
// the production panel before any ordinary work.
func (g *Game) tryUrgent(c *Colonist) bool {
	bp, gp := g.buildPred(c), g.gatherPred()
	if g.tryTarget(c, func(t *Tile, x, y int, self bool) bool {
		return t.Urgent && (bp(t, x, y, self) || gp(t, x, y, self))
	}, func(tg pt, t *Tile) (Job, string) {
		if t.Blue != BNone {
			return Job{Kind: JBuild, Need: t.blueCost().Time}, "Going to build (urgent)"
		}
		j, st := g.gatherJob(tg, t)
		return j, st + " (urgent)"
	}) {
		return true
	}
	return g.tryCraft(c, true)
}

func (g *Game) release(c *Colonist) {
	j := &c.Job
	switch j.Kind {
	case JBuild, JChop, JMine, JHarvest, JSow, JReap, JSleep, JRecover, JCraft, JStudy, JDecon, JRoof, JFirefight, JMove:
		if t := g.W.At(j.Target.x, j.Target.y); t != nil && t.ResT == c.ID {
			t.ResT = 0
		}
	case JPlay:
		if t := g.W.At(j.Target.x, j.Target.y); t != nil {
			if t.ResT == c.ID {
				t.ResT = 0
			}
			if t.ResD == c.ID {
				t.ResD = 0
			}
		}
	case JTend:
		if p := g.colByID(j.Patient); p != nil && p.Doctor == c.ID {
			p.Doctor = 0
		}
	case JHaul:
		if t := g.W.At(j.Target.x, j.Target.y); t != nil && t.ResI == c.ID {
			t.ResI = 0
		}
		if t := g.W.At(j.Dst.x, j.Dst.y); t != nil && t.ResD == c.ID {
			t.ResD = 0
		}
	}
}

func (g *Game) dropCarry(c *Colonist) {
	if c.Carry == INone {
		return
	}
	x, y := g.tileOf(c)
	it, n := c.Carry, c.CarryN
	c.Carry, c.CarryN = INone, 0
	p, ok := g.W.bfs(x, y, 12, func(x, y int) bool {
		t := g.W.At(x, y)
		return t.Item == INone || (t.Item == it && t.ItemN+n <= maxStack(it))
	})
	if !ok {
		return
	}
	e := endOf(p, x, y)
	t := g.W.At(e.x, e.y)
	t.Item = it
	t.ItemN += n
}

func (g *Game) abort(c *Colonist) {
	g.release(c)
	g.dropCarry(c)
	g.dropCargo(c)
	c.Job = Job{}
	c.Path = nil
	c.Idle = 0.3
}

func (g *Game) finish(c *Colonist) {
	g.release(c)
	c.Job = Job{}
	c.Idle = 0
}

// stepPath moves along path; returns false when the next tile is blocked.
func (g *Game) stepPath(x, y *float64, path *[]pt, speed, dt float64) bool {
	remaining := speed * dt
	for remaining > 1e-9 && len(*path) > 0 {
		nx := (*path)[0]
		if !g.W.Walkable(nx.x, nx.y) {
			return false
		}
		tx, ty := float64(nx.x)+0.5, float64(nx.y)+0.5
		dx, dy := tx-*x, ty-*y
		dist := math.Hypot(dx, dy)
		if dist <= remaining { // reach the waypoint and carry on with what is left of this step
			*x, *y = tx, ty
			*path = (*path)[1:]
			remaining -= dist
			continue
		}
		*x += dx / dist * remaining
		*y += dy / dist * remaining
		break
	}
	return true
}

func (g *Game) updateColonist(c *Colonist, dt float64) {
	ox, oy := c.X, c.Y
	g.updateColonist0(c, dt)
	dx, dy := c.X-ox, c.Y-oy
	c.Moving = math.Hypot(dx, dy) > 1e-5
	if !c.Moving {
		tx, ty, ok := 0.0, 0.0, false
		switch c.Job.Kind {
		case JChop, JMine, JHarvest, JBuild, JSow, JReap:
			tx, ty, ok = float64(c.Job.Target.x)+0.5, float64(c.Job.Target.y)+0.5, true
		case JFight, JHunt:
			if a := c.Job.Animal; a != nil {
				tx, ty, ok = a.X, a.Y, true
			}
		}
		if ok {
			dx, dy = tx-c.X, ty-c.Y
			c.Moving = false
		} else {
			return
		}
	}
	if math.Abs(dx) > math.Abs(dy) {
		c.Face = 3
		if dx < 0 {
			c.Face = 2
		}
	} else if dy != 0 {
		c.Face = 0
		if dy < 0 {
			c.Face = 1
		}
	}
}

func (g *Game) updateColonist0(c *Colonist, dt float64) {
	x, y := g.tileOf(c)
	c.Temp = g.tempAt(x, y) - g.wetChill(x, y)
	g.fireDanger(c, dt)
	asleep := c.Job.Kind == JSleep && len(c.Path) == 0
	if asleep {
		c.Hunger -= dt * 0.15
	} else {
		c.Hunger -= dt * 0.30 * c.hungerMult()
		c.Rest = math.Max(0, c.Rest-dt*0.2)
	}
	if c.Hunger <= 0 {
		c.Hunger = 0
		c.Health -= dt * 0.5
	}
	if c.Temp < 0 {
		c.Health -= dt * math.Min(0.35, -c.Temp/30) * c.coldMult()
		if !c.Chilled {
			c.Chilled = true
			g.msg("%s is freezing!", c.Name)
		}
	} else if c.Temp > 38 {
		c.Health -= dt * math.Min(0.6, (c.Temp-38)/15)
	}
	g.updateInjuries(c, dt)
	g.updateIllness(c, dt)
	g.updateJoy(c, dt)
	if c.Temp < -3 && g.rng.Float64() < dt*0.03 {
		if g.addWound(c, "Frostbite", 3, false) {
			g.msg("%s has frostbite.", c.Name)
		}
	}
	if c.Temp > 5 {
		c.Chilled = false
	}
	if len(c.Injuries) == 0 && !c.sick() && c.Hunger > 40 && c.Temp >= 2 && c.Temp <= 38 && c.Health < 100 {
		c.Health = math.Min(100, c.Health+dt*0.8)
	}
	if c.Health <= 0 {
		c.Dead = true
		for _, it := range []*Item{&c.Weapon, &c.Tool, &c.Wear} {
			if *it != INone {
				g.placeItem(int(c.X), int(c.Y), *it, 1)
				*it = INone
			}
		}
		g.emit("death", c.X, c.Y)
		g.abort(c)
		switch {
		case c.Hunger <= 0:
			g.stats.Starved++
			g.msg("%s starved to death.", c.Name)
		case c.Temp < 0:
			g.stats.Froze++
			g.msg("%s froze to death.", c.Name)
		case c.Temp > 38:
			g.stats.Heat++
			g.msg("%s died of heat.", c.Name)
		case c.untended():
			g.stats.Bled++
			g.msg("%s bled out.", c.Name)
		case c.sick():
			g.stats.IllDeaths++
			g.alert("%s died of %s.", c.Name, lowerFirst(diseaseDefs[c.worstIllness().Kind].Name))
		default:
			g.stats.Killed++
			g.msg("%s was killed.", c.Name)
		}
		return
	}
	if (c.Job.Kind == JSleep || c.Job.Kind == JRecover) && len(c.Path) == 0 && c.Hunger < 25 {
		g.finish(c) // get up and eat
	}

	// defend against wolves
	c.Think -= dt
	c.FightCD -= dt
	c.FireCD -= dt
	thought := false
	if c.Think <= 0 {
		c.Think = 0.4
		thought = true
		if c.Job.Kind != JFight && c.Health > 30 && c.FightCD <= 0 {
			var w *Animal
			bd := 14.0
			for _, a := range g.animals {
				if a.hostile() {
					reach := bd
					if a.Kind == ARaider {
						reach = 20 // gunmen shoot from afar: meet them
					}
					if d := math.Hypot(a.X-c.X, a.Y-c.Y); d < reach && (w == nil || d < math.Hypot(w.X-c.X, w.Y-c.Y)) {
						bd, w = d, a
					}
				}
			}
			if w != nil {
				g.abort(c)
				c.Job = Job{Kind: JFight, Animal: w}
				c.Status = "Fighting"
			}
		}
	}
	// fire alarm: drop what you are doing (even sleeping) and go put it out
	if thought && g.canFightFire(c) && c.Job.Kind != JFight && c.Job.Kind != JFirefight && c.Job.Kind != JEat && g.fireReachable(c) {
		g.abort(c)
		g.tryFirefight(c)
	}
	if c.Job.Kind == JFight || c.Job.Kind == JHunt {
		g.chase(c, dt)
		return
	}
	if c.Job.Kind == JTend {
		g.tend(c, dt)
		return
	}

	if len(c.Path) > 0 {
		sp := 3.2 * c.speedMult() * c.illSpeed() * g.floorSpeedAt(c) * g.weatherWalk(g.tileOf(c))
		if c.Hunger < 20 {
			sp *= 0.7
		}
		if !g.stepPath(&c.X, &c.Y, &c.Path, sp, dt) {
			g.abort(c)
		}
		return
	}

	if c.Job.Kind == JNone {
		c.Status = "Idle"
		if c.Sulk > 0 {
			c.Status = "Sulking"
		}
		if c.Idle > 0 {
			c.Idle -= dt
			return
		}
		if g.assignJob(c) {
			return
		}
		c.Idle = 0.5 + g.rng.Float64()
		if g.outT > 0 && g.rng.Float64() < 0.25 { // wander a little
			rx, ry := x+g.rng.Intn(11)-5, y+g.rng.Intn(11)-5
			if p, ok := g.W.bfs(x, y, 8, func(a, b int) bool { return a == rx && b == ry }); ok {
				c.Path = p
			}
		}
		return
	}
	g.doJob(c, dt)
}

func (g *Game) chase(c *Colonist, dt float64) {
	a := c.Job.Animal
	if a == nil || a.Dead || (c.Job.Kind == JHunt && !a.Hunt) {
		g.finish(c)
		return
	}
	c.AtkCool -= dt
	dmg, cool, rng, ranged := g.attackProfile(c)
	dist := math.Hypot(a.X-c.X, a.Y-c.Y)
	if ranged && dist <= rng && g.clearShot(c.X, c.Y, a.X, a.Y) {
		// in range with a clear line: stand and shoot
		c.Path = nil
		c.Status = "Shooting"
		if c.AtkCool <= 0 {
			c.AtkCool = cool
			g.shoot(c, a, dmg)
			if a.HP <= 0 {
				g.animalDown(c, a)
				g.finish(c)
			}
		}
		return
	}
	if !ranged && dist < 1.15 {
		c.Path = nil
		if c.AtkCool <= 0 {
			c.AtkCool = cool
			a.HP -= dmg
			g.gainXP(c, SCombat, 2.5)
			a.Flee = 4
			if c.Weapon != INone {
				g.emit("slash", a.X, a.Y)
			}
			g.emit("hit", a.X, a.Y)
			if a.HP <= 0 {
				g.animalDown(c, a)
				g.finish(c)
			}
		}
		return
	}
	c.Repath -= dt
	if c.Repath <= 0 || len(c.Path) == 0 {
		c.Repath = 0.4
		x, y := g.tileOf(c)
		ax, ay := int(a.X), int(a.Y)
		p, ok := g.W.bfs(x, y, 40, func(a, b int) bool { return a == ax && b == ay })
		if !ok {
			c.FightCD = 4
			a.Skip = g.time + 30
			g.abort(c)
			return
		}
		c.Path = p
	}
	if len(c.Path) > 0 && !g.stepPath(&c.X, &c.Y, &c.Path, 3.3, dt) {
		c.Path = nil
	}
}

// animalDown handles the death of a wolf or deer at a colonist's hands.
func (g *Game) animalDown(c *Colonist, a *Animal) {
	if a.Kind == ARaider {
		g.raiderDown(c, a)
		return
	}
	a.Dead = true
	if a.Kind == AWolf {
		g.stats.WolvesKilled++
		g.emit("yelp", a.X, a.Y)
		g.msg("%s killed a wolf.", c.Name)
		return
	}
	g.emit("thud", a.X, a.Y)
	g.stats.DeerKilled++
	g.placeItem(int(a.X), int(a.Y), IFood, 8)
	g.msg("%s brought down a deer.", c.Name)
}

func (g *Game) doJob(c *Colonist, dt float64) {
	W := g.W
	j := &c.Job
	x, y := g.tileOf(c)
	rate := 1.0 * c.illWork() * c.moodFactor()
	if c.Hunger < 20 {
		rate = 0.6 * c.illWork() * c.moodFactor()
	}
	if c.Rest < 15 {
		rate *= 0.7
	}
	switch j.Kind {
	case JEat:
		c.Status = "Eating"
		t := W.At(j.Target.x, j.Target.y)
		if nutrition(t.Item) == 0 || t.ItemN == 0 {
			g.abort(c)
			return
		}
		j.Work += dt
		if j.Work >= j.Need {
			c.Hunger = math.Min(100, c.Hunger+nutrition(t.Item))
			g.maybePoison(c, t.Item, t.Roof, g.tempAt(j.Target.x, j.Target.y))
			g.addJoy(c, foodJoy(t.Item))
			t.ItemN--
			if t.ItemN == 0 {
				t.Item = INone
			}
			g.finish(c)
		}

	case JSleep:
		c.Status = "Sleeping"
		gain := 0.7
		if j.Stage == 0 {
			if W.At(j.Target.x, j.Target.y).Obj != OBed {
				g.abort(c)
				return
			}
			gain = 1.4
		}
		c.Rest = math.Min(100, c.Rest+dt*gain)
		if c.Rest >= 100 || (!g.isNight() && c.Rest >= 60) {
			g.finish(c)
		}

	case JRecover:
		c.Status = "Recovering"
		gain := 0.5
		if j.Stage == 0 {
			if W.At(j.Target.x, j.Target.y).Obj != OBed {
				g.abort(c)
				return
			}
			gain = 1.0
		}
		c.Rest = math.Min(100, c.Rest+dt*gain)
		if !c.sick() && (c.Health >= 80 || len(c.Injuries) == 0 && c.Health >= 60) {
			g.finish(c)
		}

	case JWarm:
		c.Status = "Warming up"
		j.Work += dt
		if j.Work >= j.Need {
			g.finish(c)
		}

	case JHaul:
		if j.Stage == 0 {
			t := W.At(j.Target.x, j.Target.y)
			if t.Item == INone || t.ItemN == 0 {
				g.abort(c)
				return
			}
			c.Carry, c.CarryN = t.Item, t.ItemN
			t.Item, t.ItemN, t.ResI = INone, 0, 0
			j.Stage = 1
			p, ok := W.bfs(x, y, 120, func(a, b int) bool { return a == j.Dst.x && b == j.Dst.y })
			if !ok {
				// cut off from the stockpile (walled in by trees, rocks, walls...): put it back and
				// stop trying for a while so everyone else gets on with other work
				if g.noHaul == nil {
					g.noHaul = map[int]float64{}
				}
				g.noHaul[j.Target.y*MapW+j.Target.x] = g.time + 90
				g.abort(c)
				return
			}
			c.Path, c.Status = p, "Hauling "+itemNames[c.Carry]
			return
		}
		t := W.At(j.Dst.x, j.Dst.y)
		switch {
		case t.Item == INone || t.ItemN == 0:
			t.Item, t.ItemN = c.Carry, c.CarryN
		case t.Item == c.Carry && t.ItemN+c.CarryN <= maxStack(c.Carry):
			t.ItemN += c.CarryN
		default:
			g.abort(c)
			return
		}
		c.Carry, c.CarryN = INone, 0
		g.finish(c)

	case JBuild:
		t := W.At(j.Target.x, j.Target.y)
		if t.Blue == BNone {
			g.abort(c)
			return
		}
		cost := t.blueCost()
		if !j.Paid {
			if !g.pay(cost) {
				g.abort(c)
				c.Idle = 3
				return
			}
			j.Paid = true
		}
		c.Status = "Building"
		j.Work += dt * rate * c.rate(SConstruction) * g.nightFactor(x, y)
		g.gainXP(c, SConstruction, dt*0.8)
		if cost.MinEng > 0 {
			g.gainXP(c, SEngineering, dt*0.5)
		}
		if j.Work-j.Last > 0.6 {
			j.Last = j.Work
			g.emit("hammer", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
		}
		if j.Work >= j.Need {
			if t.Blue == BFloor {
				if t.Floor && t.FloorKind != t.BlueKind { // replacing an old floor: half its materials come back
					it1, n1, _, _ := refund(floorCost(t.FloorKind))
					g.placeItem(j.Target.x, j.Target.y, it1, n1)
				}
				t.Floor, t.FloorKind = true, t.BlueKind
			} else {
				if solidBlue(t.Blue) && g.colonistAt(j.Target) {
					return // wait until the tile is clear
				}
				t.Obj = blueObj[t.Blue]
				t.Zone, t.Farm, t.Crop = false, false, false
				if solidBlue(t.Blue) {
					t.Item, t.ItemN = INone, 0
				}
				if t.Obj == OWall || t.Obj == ODoor {
					g.roomDirty = true
				}
			}
			t.Blue = BNone
			g.finish(c)
		}

	case JDecon:
		t := W.At(j.Target.x, j.Target.y)
		if t.Desig != DDecon || !desigValid(t) {
			g.abort(c)
			return
		}
		c.Status = "Dismantling"
		j.Work += dt * rate * c.rate(SConstruction) * g.nightFactor(x, y)
		g.gainXP(c, SConstruction, dt*0.6)
		if j.Work-j.Last > 0.7 {
			j.Last = j.Work
			g.emit("hammer", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
		}
		if j.Work >= j.Need {
			g.salvage(j.Target.x, j.Target.y)
			g.finish(c)
		}

	case JFirefight:
		g.doFirefight(c, dt)

	case JEquip:
		g.doEquip(c)

	case JRead:
		g.doRead(c, dt)

	case JMove:
		g.doMove(c, dt)

	case JPlay:
		g.doPlay(c, dt)

	case JRoof:
		t := W.At(j.Target.x, j.Target.y)
		if t.Desig != DRoof || !desigValid(t) {
			g.abort(c)
			return
		}
		if !j.Paid {
			if !g.takeStock(IWood, roofWood) {
				g.abort(c)
				c.Idle = 3
				return
			}
			j.Paid = true
		}
		c.Status = "Roofing"
		j.Work += dt * rate * c.rate(SConstruction) * g.nightFactor(x, y)
		g.gainXP(c, SConstruction, dt*0.6)
		if j.Work-j.Last > 0.7 {
			j.Last = j.Work
			g.emit("hammer", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
		}
		if j.Work >= j.Need {
			t.Roof, t.Desig = true, DNone
			g.finish(c)
		}

	case JCraft:
		g.doCraft(c, dt)

	case JStudy:
		g.doStudy(c, dt)

	case JSow, JReap:
		t := W.At(j.Target.x, j.Target.y)
		if !t.Farm || (j.Kind == JSow && t.Crop) || (j.Kind == JReap && !t.Crop) {
			g.abort(c)
			return
		}
		c.Status = map[JobKind]string{JSow: "Sowing", JReap: "Reaping"}[j.Kind]
		j.Work += dt * rate * c.rate(SFarming) * g.nightFactor(x, y)
		g.gainXP(c, SFarming, dt*0.8)
		if j.Work >= j.Need {
			g.emit("rustle", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
			if j.Kind == JSow {
				t.Crop, t.Grow = true, 0
			} else {
				t.Crop, t.Grow = false, 0
				def := cropDefs[t.CropKind]
				t.Item, t.ItemN = def.Out, def.N+c.lvl(SFarming)/5+c.bonus(TrGreenThumb)
				if t.CropKind == CropAloe && c.lvl(SFarming) >= 4 { // skilled hands get more out of each plant
					t.ItemN++
				}
			}
			g.finish(c)
		}

	case JChop, JMine, JHarvest:
		t := W.At(j.Target.x, j.Target.y)
		want := map[JobKind]Desig{JChop: DChop, JMine: DMine, JHarvest: DHarvest}[j.Kind]
		if t.Desig != want || !desigValid(t) {
			g.abort(c)
			return
		}
		c.Status = map[JobKind]string{JChop: "Chopping", JMine: "Mining", JHarvest: "Harvesting"}[j.Kind]
		sk := SFarming
		if j.Kind == JMine {
			sk = SMining
		}
		chopBonus := 1.0
		if j.Kind == JMine && c.Tool != INone {
			chopBonus = gearDefs[c.Tool].Mine
		}
		if j.Kind == JChop {
			if wd, ok := weaponDefs[c.Weapon]; ok && wd.Chop > 0 {
				chopBonus = wd.Chop
			}
		}
		j.Work += dt * rate * c.rate(sk) * g.nightFactor(x, y) * chopBonus
		g.gainXP(c, sk, dt*0.8)
		if j.Kind != JHarvest && j.Work-j.Last > 0.75 {
			j.Last = j.Work
			g.emit(map[JobKind]string{JChop: "chop", JMine: "mine"}[j.Kind], float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
		}
		if j.Work >= j.Need {
			switch j.Kind {
			case JChop:
				t.Item, t.ItemN = IWood, 4
			case JMine:
				extra := 0
				if c.Tool == IPickaxe {
					extra = 1 // a pickaxe gets more out of the rock
				}
				if t.Obj == OOre {
					t.Item, t.ItemN = IOre, 3+c.lvl(SMining)/6+extra
				} else {
					t.Item, t.ItemN = IStone, 6+2*(c.lvl(SMining)/8)+extra
				}
				t.Ground = GDirt
			default:
				if t.Obj == OHerb {
					t.Item, t.ItemN = IMedicine, 2
				} else {
					t.Item, t.ItemN = IFood, 3
				}
			}
			t.Desig, t.Obj = DNone, ONone
			if j.Kind == JHarvest {
				g.emit("rustle", float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
			}
			g.finish(c)
		}
	}
}

// ---- player designations ----

type Tool int

const (
	TSelect Tool = iota
	TChop
	TMine
	THarvest
	THunt
	TZone
	TFarm
	TWall
	TDoor
	TFloor
	TBed
	THeater
	TBuild
	TMove
	TRoof
	TDeconstruct
	TErase
	TPriority
	numTools
)

func (g *Game) applyTool(tool Tool, x0, y0, x1, y1 int) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	if tool == TPriority {
		g.setPriority(x0, y0, x1, y1)
		return
	}
	if tool == THunt || tool == TErase {
		for _, a := range g.animals {
			ax, ay := int(a.X), int(a.Y)
			if ax >= x0 && ax <= x1 && ay >= y0 && ay <= y1 && a.Kind == ADeer {
				a.Hunt = tool == THunt
			}
		}
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			t := g.W.At(x, y)
			if t == nil {
				continue
			}
			free := g.W.Walkable(x, y) && t.Obj == ONone && t.Blue == BNone && !t.Zone && !t.Farm && t.Item == INone
			switch tool {
			case TChop:
				if t.Obj == OTree {
					t.Desig = DChop
				}
			case TMine:
				if t.Obj == ORock || t.Obj == OOre {
					t.Desig = DMine
				}
			case THarvest:
				if t.Obj == OBush || t.Obj == OHerb {
					t.Desig = DHarvest
				}
			case TZone:
				if g.W.Walkable(x, y) && t.Blue == BNone && !t.Farm {
					t.Zone = true
				}
			case TFarm:
				if (t.Ground == GGrass || t.Ground == GDirt) && t.Obj == ONone && t.Blue == BNone && !t.Zone && !t.Floor {
					if !t.Farm || !t.Crop { // (re)plant the chosen crop, but never rip out a growing one
						t.Farm, t.CropKind = true, g.cropSel
					}
				}
			case TWall:
				if free {
					t.Blue, t.Mat = BWall, g.wallMat
				}
			case TDoor:
				if free {
					t.Blue, t.Mat = BDoor, g.doorMat
				}
			case TBed:
				if free {
					t.Blue = BBed
				}
			case THeater:
				if free {
					t.Blue = BHeater
				}
			case TRoof:
				if t.Desig == DNone && desigValid(&Tile{Obj: t.Obj, Ground: t.Ground, Roof: t.Roof, Desig: DRoof}) {
					t.Desig = DRoof
				}
			case TDeconstruct:
				if t.Blue == BNone && desigValid(&Tile{Obj: t.Obj, Floor: t.Floor, Roof: t.Roof, Desig: DDecon}) {
					t.Desig = DDecon
				}
			case TBuild:
				if free && g.buildSel != BNone {
					t.Blue = g.buildSel
				}
			case TFloor:
				if t.Ground != GWater && floorable(t.Obj) && t.Blue == BNone && !t.Farm && (!t.Floor || t.FloorKind != g.floorSel) {
					t.Blue, t.BlueKind = BFloor, g.floorSel // (laying a different floor over an old one replaces it)
				}
			case TErase:
				t.Desig, t.Blue, t.Zone, t.Farm, t.Crop, t.Urgent = DNone, BNone, false, false, false, false
				if t.MoveTo != 0 {
					g.cancelMoveFrom(y*MapW + x)
				}
				t.MoveFrom = 0
				t.Fire, t.Roof = 0, false
				t.BlueKind = FloorWood
				switch t.Obj {
				case OWall, ODoor, OBed, OHeater, OBench, OFab, OGen, OSolar, OLamp, OEHeater, ODesk, OFurnace, OStove, OHorse, OChecker, OShrine:
					t.Obj, t.Mat = ONone, MatNone
					g.roomDirty = true
				default:
					t.Floor, t.FloorKind = false, FloorWood
				}
			}
		}
	}
	g.recompute()
}

// ---- medicine ----

func (c *Colonist) untended() bool {
	for _, in := range c.Injuries {
		if in.Tended == 0 {
			return true
		}
	}
	return false
}

func (g *Game) colByID(id int) *Colonist {
	for _, c := range g.cols {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (g *Game) needsTend(p *Colonist) bool {
	if g.illNeedsCare(p) {
		return true
	}
	for _, in := range p.Injuries {
		if in.Tended == 0 || (in.Tended < 2 && in.Infect >= 0.5 && g.stock[IMedicine] > 0) {
			return true
		}
	}
	return false
}

// addWound adds (or worsens) an injury; reports whether a new one was created.
func (g *Game) addWound(c *Colonist, kind string, sev float64, bleed bool) bool {
	for i := range c.Injuries {
		in := &c.Injuries[i]
		if in.Kind == kind && in.Tended == 0 {
			in.Sev = math.Min(30, in.Sev+sev)
			return false
		}
	}
	if len(c.Injuries) >= 6 {
		for i := range c.Injuries {
			if c.Injuries[i].Kind == kind {
				c.Injuries[i].Sev = math.Min(30, c.Injuries[i].Sev+sev)
				c.Injuries[i].Tended = 0
				return false
			}
		}
		return false
	}
	c.Injuries = append(c.Injuries, Injury{Kind: kind, Sev: sev, Bleed: bleed})
	return true
}

func (g *Game) updateInjuries(c *Colonist, dt float64) {
	mult := 1.0
	if (c.Job.Kind == JSleep || c.Job.Kind == JRecover) && c.Job.Stage == 0 && len(c.Path) == 0 {
		mult = 2
	}
	for i := 0; i < len(c.Injuries); {
		in := &c.Injuries[i]
		rate := 0.03
		switch in.Tended {
		case 0:
			if in.Bleed {
				c.Health -= dt * 0.08 * (1 + in.Sev/10)
				in.Infect += dt * 0.004
			}
		case 1:
			rate = 0.06
			if in.Bleed {
				in.Infect += dt * 0.0008
			}
		case 2:
			rate = 0.12
			in.Infect = 0
		}
		if in.Infect >= 1 {
			in.Infect = 1
			c.Health -= dt * 0.3
			rate *= 0.3
		}
		if c.Hunger > 0 && c.Temp >= 0 { // too cold to heal
			q := in.Quality
			if q == 0 {
				q = 1
			}
			h := dt * rate * mult * q
			in.Sev -= h
			c.Health = math.Min(100, c.Health+h*3)
		}
		if in.Sev <= 0 {
			c.Injuries = append(c.Injuries[:i], c.Injuries[i+1:]...)
			continue
		}
		i++
	}
}

// tend walks to the patient and treats their worst wound.
func (g *Game) tend(c *Colonist, dt float64) {
	j := &c.Job
	p := g.colByID(j.Patient)
	if p == nil || p.Dead || !g.needsTend(p) {
		g.finish(c)
		return
	}
	if math.Hypot(p.X-c.X, p.Y-c.Y) > 1.3 {
		c.Repath -= dt
		if c.Repath <= 0 || len(c.Path) == 0 {
			c.Repath = 0.5
			x, y := g.tileOf(c)
			px, py := int(p.X), int(p.Y)
			path, ok := g.W.bfs(x, y, 60, func(a, b int) bool { return a == px && b == py })
			if !ok {
				p.Doctor = 0
				g.abort(c)
				c.Idle = 3
				return
			}
			c.Path = path
		}
		if len(c.Path) > 0 && !g.stepPath(&c.X, &c.Y, &c.Path, 3.2, dt) {
			c.Path = nil
		}
		return
	}
	c.Path = nil
	c.Status = "Tending " + p.Name
	j.Work += dt
	g.gainXP(c, SMedicine, dt*1.1)
	if j.Work < j.Need {
		return
	}
	// unskilled medics sometimes botch it
	if l := c.lvl(SMedicine); (l < 3 && g.chance(0.2)) || (l >= 3 && l < 6 && g.chance(0.05)) {
		g.msg("%s fumbled tending %s's wound.", c.Name, p.Name)
		p.Doctor = 0
		g.finish(c)
		c.Idle = 1
		return
	}
	if g.careForIllness(c, p) { // sickness first
		g.finish(c)
		return
	}
	// pick the worst wound that still needs care
	best := -1
	for i, in := range p.Injuries {
		need := in.Tended == 0 || (in.Tended < 2 && in.Infect >= 0.5)
		if need && (best < 0 || in.Sev > p.Injuries[best].Sev) {
			best = i
		}
	}
	if best >= 0 {
		in := &p.Injuries[best]
		if (in.Sev >= 4 || in.Infect >= 0.3) && g.takeStock(IMedicine, 1) {
			in.Tended, in.Infect = 2, 0
			in.Quality = 1 + 0.06*float64(c.lvl(SMedicine))
			g.msg("%s treated %s's %s with medicine.", c.Name, p.Name, in.Kind)
			g.emit("heal", p.X, p.Y)
		} else if in.Tended == 0 {
			in.Tended = 1
			in.Quality = 1 + 0.06*float64(c.lvl(SMedicine))
			if c.lvl(SMedicine) >= 7 {
				in.Infect *= 0.5
			}
		}
	}
	g.finish(c)
}

// floorSpeedAt is the walking-speed bonus from the floor under a colonist's feet.
func (g *Game) floorSpeedAt(c *Colonist) float64 {
	x, y := g.tileOf(c)
	if t := g.W.At(x, y); t != nil && t.Floor {
		return floorSpeed[t.FloorKind]
	}
	return 1
}

// salvage removes a built object (or, failing that, the floor) and drops half its materials.
func (g *Game) salvage(x, y int) {
	t := g.W.At(x, y)
	var c Cost
	if b, ok := objBlue[t.Obj]; ok {
		c = costFor(b, t.builtMat())
		if t.Obj == OWall || t.Obj == ODoor {
			g.roomDirty = true
		}
		t.Obj, t.Fuel, t.Mat = ONone, 0, MatNone
	} else if t.Floor {
		c = floorCost(t.FloorKind)
		t.Floor, t.FloorKind = false, FloorWood
	} else if t.Roof {
		c = Cost{Item: IWood, N: roofWood}
		t.Roof = false
	}
	t.Desig = DNone
	it1, n1, it2, n2 := refund(c)
	if n1 > 0 && it1 != INone {
		g.placeItem(x, y, it1, n1)
	}
	if n2 > 0 && it2 != INone {
		g.placeItem(x, y, it2, n2)
	}
	g.stats.Salvaged++
}

// rescueItems moves items stranded on solid buildings (a bug in earlier versions dropped a
// bench's first product on the bench itself) onto a reachable tile.
func (g *Game) rescueItems() {
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Item == INone || t.ItemN == 0 || g.W.Walkable(i%MapW, i/MapW) {
			continue
		}
		if _, built := objBlue[t.Obj]; !built {
			continue
		}
		it, n := t.Item, t.ItemN
		t.Item, t.ItemN = INone, 0
		g.placeItem(i%MapW, i/MapW, it, n)
	}
}

// setPriority marks every order and blueprint in the rectangle urgent, or, when they all already are,
// takes the mark off again.
func (g *Game) setPriority(x0, y0, x1, y1 int) {
	var orders []*Tile
	allUrgent := true
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			t := g.W.At(x, y)
			if t == nil || !(t.Blue != BNone || (t.Desig != DNone && desigValid(t))) {
				continue
			}
			orders = append(orders, t)
			allUrgent = allUrgent && t.Urgent
		}
	}
	switch {
	case len(orders) == 0:
		g.msg("Nothing to prioritise there: drag over orders or blueprints.")
	case allUrgent:
		for _, t := range orders {
			t.Urgent = false
		}
		g.msg("%d orders back to normal priority.", len(orders))
	default:
		for _, t := range orders {
			t.Urgent = true
		}
		g.msg("%d orders marked urgent: colonists do them first.", len(orders))
	}
}
