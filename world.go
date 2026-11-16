package main

import (
	"math"
	"math/rand"
)

const (
	MapW, MapH = 160, 120
	TileSize   = 32
	MaxStack   = 50
)

type Ground uint8

const (
	GGrass Ground = iota
	GDirt
	GSand
	GWater
)

type Obj uint8

const (
	ONone Obj = iota
	OTree
	ORock
	OBush
	OWall
	OBed
	ODoor
	OHeater
	OHerb
	OBench
	OFab
	OGen
	OSolar
	OLamp
	OEHeater
	ODesk
	OOre
	OFurnace
	OStove
	OHorse
	OChecker
	OShrine
)

type Item uint8

const (
	INone Item = iota
	IWood
	IStone
	IFood
	IMedicine
	ISilver
	ICarving
	ISculpture
	IComponent
	IGadget
	IBook
	IOre
	IIron
	IKnife
	IAxe
	ISword
	IPistol
	IRifle
	IAmmo
	IPickaxe
	IParka
	IWheat
	ICorn
	ICotton
	ITobacco
	ICloth
	IBread
	IMeal
	ITile
	numItems
)

var itemNames = [numItems]string{"", "Wood", "Stone", "Food", "Medicine", "Silver", "Carving", "Sculpture", "Component", "Gadget", "Book", "Ore", "Iron", "Knife", "Axe", "Sword", "Pistol", "Rifle", "Ammo", "Pickaxe", "Parka", "Wheat", "Corn", "Cotton", "Tobacco", "Cloth", "Bread", "Meal", "Tile"}

type Desig uint8

const (
	DNone Desig = iota
	DChop
	DMine
	DHarvest
	DDecon // salvage a building, floor or roof
	DRoof  // build a roof
)

type Blue uint8

const (
	BNone Blue = iota
	BWall
	BFloor
	BDoor
	BBed
	BHeater
	BBench
	BFab
	BGen
	BSolar
	BLamp
	BEHeater
	BDesk
	BFurnace
	BStove
	BHorse
	BChecker
	BShrine
)

type Cost struct {
	Item   Item
	N      int
	Item2  Item
	N2     int
	Time   float64
	MinEng int // Engineering level a builder needs
}

var costs = map[Blue]Cost{
	BWall:    {Item: IStone, N: 4, Time: 3.0},
	BFloor:   {Item: IWood, N: 2, Time: 1.5},
	BDoor:    {Item: IWood, N: 2, Time: 1.5},
	BBed:     {Item: IWood, N: 3, Time: 2.0},
	BHeater:  {Item: IWood, N: 5, Time: 2.5},
	BBench:   {Item: IWood, N: 8, Time: 3.0},
	BDesk:    {Item: IWood, N: 5, Time: 2.0},
	BFurnace: {Item: IStone, N: 10, Item2: IWood, N2: 4, Time: 4.0},
	BStove:   {Item: IStone, N: 8, Item2: IWood, N2: 3, Time: 3.0},
	BHorse:   {Item: IWood, N: 3, Item2: IStone, N2: 2, Time: 2.0},
	BChecker: {Item: IWood, N: 6, Time: 3.0},
	BShrine:  {Item: ISculpture, N: 1, Item2: IStone, N2: 3, Time: 3.0},
	BLamp:    {Item: IComponent, N: 1, Item2: IWood, N2: 1, Time: 2.0, MinEng: 1},
	BFab:     {Item: IStone, N: 6, Item2: IComponent, N2: 3, Time: 5.0, MinEng: 3},
	BGen:     {Item: IStone, N: 10, Item2: IComponent, N2: 2, Time: 5.0, MinEng: 2},
	BSolar:   {Item: IComponent, N: 3, Item2: IStone, N2: 2, Time: 4.0, MinEng: 3},
	BEHeater: {Item: IComponent, N: 2, Item2: IStone, N2: 3, Time: 3.0, MinEng: 2},
}

// solidBlue reports whether the finished building blocks movement (so nobody may stand on it).
// floorable reports whether a floor can be laid under what stands on the tile: bare ground, or furniture and
// machines (beds, heaters, benches...) but not walls, trees, rocks, ore, bushes or herbs.
func floorable(o Obj) bool {
	if o == ONone {
		return true
	}
	_, built := objBlue[o]
	return built && o != OWall
}

func solidBlue(b Blue) bool {
	switch b {
	case BWall, BHeater, BBench, BFab, BGen, BSolar, BLamp, BEHeater, BDesk, BFurnace, BStove, BChecker, BShrine:
		return true
	}
	return false
}

// blueObj maps a blueprint to the object it becomes.
var blueObj = map[Blue]Obj{
	BWall: OWall, BDoor: ODoor, BBed: OBed, BHeater: OHeater, BBench: OBench, BFab: OFab,
	BGen: OGen, BSolar: OSolar, BLamp: OLamp, BEHeater: OEHeater, BDesk: ODesk, BFurnace: OFurnace, BStove: OStove, BHorse: OHorse, BChecker: OChecker, BShrine: OShrine,
}

var blueNames = map[Blue]string{
	BWall: "Wall", BDoor: "Door", BBed: "Bed", BHeater: "Heater", BBench: "Workbench", BFab: "Fabricator",
	BGen: "Generator", BSolar: "Solar panel", BLamp: "Lamp", BEHeater: "Electric heater", BDesk: "Study desk", BFloor: "Floor", BFurnace: "Furnace", BStove: "Stove", BHorse: "Horseshoe pit", BChecker: "Checkers table", BShrine: "Meditation shrine",
}

// maxStack is how many of an item fit on one tile.
func maxStack(it Item) int {
	if it == ISilver {
		return 500
	}
	return MaxStack
}

type pt struct{ x, y int }

type Tile struct {
	Ground    Ground
	Obj       Obj
	Shade     uint8
	Floor     bool
	Zone      bool
	Mat       Material  // what a wall or door is made of (0 = legacy: stone wall, wooden door)
	Fire      float32   // seconds this tile has been burning (0 = not on fire)
	CropKind  CropKind  // what a farm tile grows
	Roof      bool      // sheltered: roofs make enclosed rooms indoors and protect goods from spoiling
	MoveFrom  int32     // on the destination of a pending move: source tile index+1
	MoveTo    int32     // on the source of a pending move: destination tile index+1
	FloorKind FloorKind // what the built floor is made of (0 = wood planks, as old floors were)
	BlueKind  FloorKind // what a floor blueprint will be made of
	Rot       float32   // spoilage accumulated on the item lying here
	Farm      bool
	Crop      bool
	Grow      float32
	Fuel      float32
	Desig     Desig
	Urgent    bool // the order or blueprint here is worked before ordinary jobs (Priority tool)
	Blue      Blue
	Item      Item
	ItemN     int
	ResT      int // colonist reserving this tile as a work target
	ResD      int // colonist reserving this tile as a haul destination
	ResI      int // colonist reserving the item lying here
}

type World struct {
	T        []Tile
	stamp    []int32
	parent   []int32
	queue    []int32
	curStamp int32
}

func (w *World) At(x, y int) *Tile {
	if x < 0 || y < 0 || x >= MapW || y >= MapH {
		return nil
	}
	return &w.T[y*MapW+x]
}

func (w *World) Walkable(x, y int) bool {
	t := w.At(x, y)
	if t == nil || t.Ground == GWater {
		return false
	}
	if t.Fire > 0 {
		return false
	}
	return t.Obj == ONone || t.Obj == OBush || t.Obj == OBed || t.Obj == ODoor || t.Obj == OHerb || t.Obj == OHorse
}

// ---- noise ----

func hash2(x, y int, seed int64) float64 {
	h := uint64(int64(x))*374761393 + uint64(int64(y))*668265263 + uint64(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffffff) / float64(0x1000000)
}

func vnoise(x, y float64, seed int64) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	fx, fy := x-xi, y-yi
	fx = fx * fx * (3 - 2*fx)
	fy = fy * fy * (3 - 2*fy)
	ix, iy := int(xi), int(yi)
	a := hash2(ix, iy, seed)
	b := hash2(ix+1, iy, seed)
	c := hash2(ix, iy+1, seed)
	d := hash2(ix+1, iy+1, seed)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

func fbm(x, y float64, seed int64, oct int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for i := 0; i < oct; i++ {
		sum += vnoise(x, y, seed+int64(i)*101) * amp
		norm += amp
		amp *= 0.5
		x *= 2
		y *= 2
	}
	return sum / norm
}

func newWorld(seed int64) (*World, pt) {
	rng := rand.New(rand.NewSource(seed))
	w := &World{
		T:      make([]Tile, MapW*MapH),
		stamp:  make([]int32, MapW*MapH),
		parent: make([]int32, MapW*MapH),
		queue:  make([]int32, 0, MapW*MapH),
	}
	for y := 0; y < MapH; y++ {
		for x := 0; x < MapW; x++ {
			fx, fy := float64(x), float64(y)
			dx := (fx - MapW/2) / (MapW / 2)
			dy := (fy - MapH/2) / (MapH / 2)
			d := math.Sqrt(dx*dx + dy*dy)
			e := fbm(fx*0.04, fy*0.04, seed, 4) + 0.08 - 0.45*math.Pow(d, 3)
			m := fbm(fx*0.07, fy*0.07, seed+11, 3)
			f := fbm(fx*0.09, fy*0.09, seed+23, 3)
			r := fbm(fx*0.11, fy*0.11, seed+37, 3)
			t := w.At(x, y)
			t.Shade = uint8(rng.Intn(256))
			switch {
			case e < 0.33:
				t.Ground = GWater
			case e < 0.37:
				t.Ground = GSand
			case m < 0.38:
				t.Ground = GDirt
			default:
				t.Ground = GGrass
			}
			if t.Ground == GGrass || t.Ground == GDirt {
				switch {
				case f > 0.56 && rng.Float64() < 0.5, f > 0.47 && rng.Float64() < 0.04:
					t.Obj = OTree
				case r > 0.60 && rng.Float64() < 0.55:
					t.Obj = ORock
					if r > 0.66 && rng.Float64() < 0.22 {
						t.Obj = OOre
					}
				case t.Ground == GGrass && rng.Float64() < 0.012:
					t.Obj = OBush
				case t.Ground == GGrass && rng.Float64() < 0.004:
					t.Obj = OHerb
				}
			}
		}
	}
	start := pt{MapW / 2, MapH / 2}
	// clear a landing area
	for y := -6; y <= 6; y++ {
		for x := -6; x <= 6; x++ {
			if x*x+y*y <= 36 {
				t := w.At(start.x+x, start.y+y)
				t.Ground, t.Obj = GGrass, ONone
			}
		}
	}
	// guarantee nearby resources
	scatter := func(o Obj, n int) {
		for placed, tries := 0, 0; placed < n && tries < 500; tries++ {
			a := rng.Float64() * 2 * math.Pi
			r := 8 + rng.Float64()*8
			x, y := start.x+int(math.Cos(a)*r), start.y+int(math.Sin(a)*r)
			t := w.At(x, y)
			if t != nil && t.Obj == ONone && (t.Ground == GGrass || t.Ground == GDirt) {
				t.Obj = o
				placed++
			}
		}
	}
	scatter(OTree, 16)
	scatter(ORock, 22)
	scatter(OOre, 10)
	scatter(OBush, 8)
	scatter(OHerb, 5)
	return w, start
}

// bfs finds the nearest tile (8-way, no corner cutting) satisfying goal and
// returns the path to it (excluding the start). goal must not call bfs.
func (w *World) bfs(sx, sy, maxR int, goal func(x, y int) bool) ([]pt, bool) {
	if sx < 0 || sy < 0 || sx >= MapW || sy >= MapH {
		return nil, false
	}
	w.curStamp++
	st := w.curStamp
	q := w.queue[:0]
	s := sy*MapW + sx
	w.stamp[s] = st
	w.parent[s] = -1
	q = append(q, int32(s))
	for h := 0; h < len(q); h++ {
		cur := int(q[h])
		cx, cy := cur%MapW, cur/MapW
		if goal(cx, cy) {
			var p []pt
			for i := cur; w.parent[i] >= 0; i = int(w.parent[i]) {
				p = append(p, pt{i % MapW, i / MapW})
			}
			for a, b := 0, len(p)-1; a < b; a, b = a+1, b-1 {
				p[a], p[b] = p[b], p[a]
			}
			w.queue = q[:0]
			return p, true
		}
		for _, d := range dirs8 {
			nx, ny := cx+d.x, cy+d.y
			if iabs(nx-sx) > maxR || iabs(ny-sy) > maxR || !w.Walkable(nx, ny) {
				continue
			}
			if d.x != 0 && d.y != 0 && (!w.Walkable(cx+d.x, cy) || !w.Walkable(cx, cy+d.y)) {
				continue
			}
			ni := ny*MapW + nx
			if w.stamp[ni] == st {
				continue
			}
			w.stamp[ni] = st
			w.parent[ni] = int32(cur)
			q = append(q, int32(ni))
		}
	}
	w.queue = q[:0]
	return nil, false
}

var dirs8 = []pt{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

func iabs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// objBlue is the blueprint kind that produced a built object (for salvage refunds).
var objBlue = func() map[Obj]Blue {
	m := map[Obj]Blue{}
	for b, o := range blueObj {
		m[o] = b
	}
	return m
}()

// refund is what salvaging returns: half of each material, rounded up.
func refund(c Cost) (Item, int, Item, int) {
	return c.Item, (c.N + 1) / 2, c.Item2, (c.N2 + 1) / 2
}

// ---------- materials ----------

type Material uint8

const (
	MatNone Material = iota // legacy tiles: resolved by kind (stone wall, wooden door)
	MatWood
	MatStone
	MatIron
)

var matNames = [...]string{"", "Wood", "Stone", "Iron"}

var wallCosts = [...]Cost{MatWood: {Item: IWood, N: 4, Time: 2.0}, MatStone: {Item: IStone, N: 4, Time: 3.0}, MatIron: {Item: IIron, N: 3, Time: 4.0}}
var doorCosts = [...]Cost{MatWood: {Item: IWood, N: 2, Time: 1.5}, MatStone: {Item: IStone, N: 5, Time: 3.0}, MatIron: {Item: IIron, N: 2, Time: 3.0}}

func resolveMat(door bool, m Material) Material {
	if m == MatNone {
		if door {
			return MatWood
		}
		return MatStone
	}
	return m
}

// costFor is the price of a blueprint; walls and doors depend on their material.
func costFor(b Blue, m Material) Cost {
	switch b {
	case BWall:
		return wallCosts[resolveMat(false, m)]
	case BDoor:
		return doorCosts[resolveMat(true, m)]
	}
	return costs[b]
}

// blueCost is what this tile's blueprint costs.
func (t *Tile) blueCost() Cost {
	if t.Blue == BFloor {
		return floorCost(t.BlueKind)
	}
	return costFor(t.Blue, t.Mat)
}

// blueName names this tile's blueprint, e.g. "Stone floor" or "Iron wall".
func (t *Tile) blueName() string {
	if t.Blue == BFloor {
		return floorNames[t.BlueKind]
	}
	return blueLabel(t.Blue, t.Mat)
}

// builtMat is the material of the finished wall or door on this tile.
func (t *Tile) builtMat() Material { return resolveMat(t.Obj == ODoor, t.Mat) }

// blueLabel names a blueprint kind, e.g. "Iron wall".
func blueLabel(b Blue, m Material) string {
	switch b {
	case BWall:
		return matNames[resolveMat(false, m)] + " wall"
	case BDoor:
		return matNames[resolveMat(true, m)] + " door"
	}
	return blueNames[b]
}

// flammable reports whether the tile holds something that can catch fire.
func (t *Tile) flammable() bool {
	switch t.Obj {
	case OWall, ODoor:
		return t.builtMat() == MatWood
	case OBed, OBench, ODesk, OChecker:
		return true
	}
	return false
}

// ---------- crops ----------

type CropKind uint8

const (
	CropVeg CropKind = iota // vegetables and berries (what farms grew before other crops existed)
	CropWheat
	CropCorn
	CropCotton
	CropAloe
	CropTobacco
	numCrops
)

type CropDef struct {
	Name string
	Time float64 // seconds to grow, when warm enough
	MinT float64 // coldest temperature it will sprout and grow in
	Out  Item
	N    int
	Note string
}

var cropDefs = [numCrops]CropDef{
	{"Vegetables", 110, 8, IFood, 5, "food, eaten as it is"},
	{"Wheat", 110, 6, IWheat, 6, "for bread"},
	{"Corn", 135, 10, ICorn, 5, "meals, or eaten raw"},
	{"Cotton", 125, 12, ICotton, 4, "cloth, then parkas"},
	{"Aloe", 150, 8, IMedicine, 1, "grows medicine"},
	{"Tobacco", 140, 12, ITobacco, 4, "a trade good"},
}

// ---------- floors ----------

type FloorKind uint8

const (
	FloorWood FloorKind = iota
	FloorStone
	FloorTile
	FloorChecker
	numFloors
)

var floorNames = [numFloors]string{"Wood floor", "Stone floor", "Tile floor", "Checker floor"}

// floorSpeed is the walking-speed bonus a floor gives.
var floorSpeed = [numFloors]float64{1.05, 1.10, 1.15, 1.15}

var floorNotes = [numFloors]string{"warm planks", "hard-wearing flagstones", "smooth ceramic tiles", "black and white tiles"}

func floorCost(k FloorKind) Cost {
	switch k {
	case FloorStone:
		return Cost{Item: IStone, N: 3, Time: 2.0}
	case FloorTile, FloorChecker:
		return Cost{Item: ITile, N: 2, Time: 1.5}
	}
	return Cost{Item: IWood, N: 2, Time: 1.5}
}
