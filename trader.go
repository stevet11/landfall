package main

import (
	"fmt"
	"math"
)

// traderStay is how long (game seconds) a trader waits in the colony.
const traderStay = 100

// BeastKind is the pack animal a trader travels with.
type BeastKind int

const (
	BeastDonkey BeastKind = iota
	BeastCamel
)

var beastNames = [...]string{"donkey", "camel"}

// Trader is a travelling merchant who visits the colony, buys goods and sells supplies.
type Trader struct {
	X, Y    float64
	Path    []pt
	Spot    pt // where they set up
	Exit    pt // where they came from
	Think   float64
	Arrived bool
	Leaving bool
	Stay    float64
	Age     float64
	Cash    int
	Stock   [numItems]int
	Mod     float64 // this visit's price mood
	Face    int
	Moving  bool
	Warned  bool      // the leaving-soon alert has been shown
	Beast   BeastKind // the pack animal that carries the goods
	BX, BY  float64   // its position (tile units)
	BPath   []pt
	BThink  float64
	BDX     float64
	BMoving bool
}

// landTile picks a random walkable tile reachable from the colony.
func (g *Game) landTile(rmin, rmax float64) (int, int, bool) {
	for try := 0; try < 120; try++ {
		a := g.rng.Float64() * 2 * math.Pi
		r := rmin + g.rng.Float64()*(rmax-rmin)
		x, y := g.start.x+int(math.Cos(a)*r), g.start.y+int(math.Sin(a)*r*0.8)
		if g.W.Walkable(x, y) && g.onMainland(x, y) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func (g *Game) spawnTrader() bool {
	x, y, ok := g.landTile(45, 62)
	if !ok {
		return false
	}
	spot := g.start
	bd := 1 << 30
	for _, i := range g.zoneList {
		sx, sy := i%MapW, i/MapW
		if g.W.Walkable(sx, sy) {
			if d := iabs(sx-g.start.x) + iabs(sy-g.start.y); d < bd {
				bd, spot = d, pt{sx, sy}
			}
		}
	}
	t := &Trader{X: float64(x) + 0.5, Y: float64(y) + 0.5, Spot: spot, Exit: pt{x, y}, Mod: 0.9 + g.rng.Float64()*0.2,
		Cash: 250 + g.rng.Intn(350)}
	t.Stock[IMedicine] = 3 + g.rng.Intn(7)
	t.Stock[IComponent] = 2 + g.rng.Intn(5)
	t.Stock[IFood] = 15 + g.rng.Intn(26)
	if g.rng.Float64() < 0.7 {
		t.Stock[IBook] = 1 + g.rng.Intn(3)
	}
	if g.rng.Float64() < 0.3 {
		t.Stock[IGadget] = 1
	}
	if g.rng.Float64() < 0.6 {
		t.Stock[IIron] = 2 + g.rng.Intn(6)
	}
	if g.rng.Float64() < 0.4 { // a blade or two
		t.Stock[[]Item{IKnife, IAxe, ISword}[g.rng.Intn(3)]] = 1
	}
	if g.rng.Float64() < 0.15 {
		t.Stock[IPistol] = 1
	}
	if g.rng.Float64() < 0.05 {
		t.Stock[IRifle] = 1
	}
	if g.rng.Float64() < 0.4 {
		t.Stock[IAmmo] = 15 + g.rng.Intn(30)
	}
	if g.rng.Float64() < 0.3 {
		t.Stock[IPickaxe] = 1
	}
	if g.rng.Float64() < 0.2 {
		t.Stock[IParka] = 1
	}
	if g.rng.Float64() < 0.3 {
		t.Stock[ICloth] = 2 + g.rng.Intn(4)
	}
	if g.rng.Float64() < 0.25 {
		t.Stock[IBread] = 4 + g.rng.Intn(8)
	}
	if g.rng.Float64() < 0.5 {
		t.Stock[ICarving] = 1 + g.rng.Intn(4)
	}
	// a pack animal carries the goods; a camel carries a lot more than a donkey
	t.Beast = BeastKind(g.rng.Intn(2))
	t.BX, t.BY, t.BDX = t.X, t.Y, 1
	if t.Beast == BeastCamel {
		for it := range t.Stock {
			t.Stock[it] = (t.Stock[it]*3 + 1) / 2
		}
		t.Cash += 150
	}
	g.trader = t
	g.alert("A trader with a %s is approaching the colony! Click the banner or press T to find them.", beastNames[t.Beast])
	g.emit("trader", -1, -1)
	return true
}

func (g *Game) updateTrader(dt float64) {
	t := g.trader
	if t == nil {
		if g.time >= g.nextTrader && len(g.cols) > 0 {
			if !g.spawnTrader() {
				g.nextTrader = g.time + 60
			}
		}
		return
	}
	ox, oy := t.X, t.Y
	t.Age += dt
	t.Think -= dt
	switch {
	case t.Leaving:
		if len(t.Path) == 0 || t.Age > 600 {
			if t.Think <= 0 {
				t.Think = 2
				tx, ty := int(t.X), int(t.Y)
				if p, ok := g.W.bfs(tx, ty, 150, func(a, b int) bool { return a == t.Exit.x && b == t.Exit.y }); ok && len(p) > 0 {
					t.Path = p
				} else {
					g.trader = nil
					g.afterTrader()
					return
				}
			}
		}
	case !t.Arrived:
		if math.Hypot(t.X-float64(t.Spot.x)-0.5, t.Y-float64(t.Spot.y)-0.5) < 1.5 {
			t.Arrived, t.Stay, t.Path = true, traderStay, nil
			g.alert("The trader has arrived! Press T (or click the banner) to trade.")
			g.emit("trader", -1, -1)
			g.emit(map[BeastKind]string{BeastDonkey: "bray", BeastCamel: "groan"}[t.Beast], t.BX, t.BY)
		} else if t.Think <= 0 && len(t.Path) == 0 {
			t.Think = 1
			tx, ty := int(t.X), int(t.Y)
			if p, ok := g.W.bfs(tx, ty, 150, func(a, b int) bool { return a == t.Spot.x && b == t.Spot.y }); ok {
				t.Path = p
			} else if t.Age > 30 {
				g.msg("The trader could not find a way in and left.")
				g.trader = nil
				g.afterTrader()
				return
			}
		}
	case t.Arrived:
		t.Stay -= dt
		if t.Stay < 40 && !t.Warned {
			t.Warned = true
			g.alert("The trader is leaving in %d seconds!", int(t.Stay))
			g.emit("trader", -1, -1)
		}
		if t.Stay <= 0 {
			t.Leaving, t.Think = true, 0
			g.msg("The trader packs up and leaves.")
		}
	}
	if len(t.Path) > 0 {
		if !g.stepPath(&t.X, &t.Y, &t.Path, 2.6, dt) {
			t.Path = nil
		}
	}
	dx, dy := t.X-ox, t.Y-oy
	t.Moving = math.Hypot(dx, dy) > 1e-5
	if t.Moving {
		if math.Abs(dx) > math.Abs(dy) {
			t.Face = 3
			if dx < 0 {
				t.Face = 2
			}
		} else {
			t.Face = 0
			if dy < 0 {
				t.Face = 1
			}
		}
	}
	g.updateBeast(t, dt)
	if t.Leaving && len(t.Path) == 0 && math.Hypot(t.X-float64(t.Exit.x)-0.5, t.Y-float64(t.Exit.y)-0.5) < 2 {
		g.trader = nil
		g.afterTrader()
	}
}

func (g *Game) afterTrader() {
	g.nextTrader = g.time + DayLen*(2+g.rng.Float64()*1.5)
}

// traderHere is true while the trader is in the colony and willing to deal.
func (g *Game) traderHere() bool { return g.trader != nil && g.trader.Arrived && !g.trader.Leaving }

func (g *Game) priceMod() (buy, sell float64) {
	m := 1.0
	if g.trader != nil {
		m = g.trader.Mod
	}
	buy, sell = 1.3*m, 0.55*m
	for _, c := range g.cols {
		if c.has(TrCharming) { // the best talker in the colony haggles for everyone
			buy *= 0.9
			sell *= 1.1
			break
		}
	}
	return
}

func (g *Game) buyPrice(it Item) int {
	b, _ := g.priceMod()
	return max(1, int(math.Ceil(itemValue[it]*b)))
}

func (g *Game) sellPrice(it Item) int {
	_, s := g.priceMod()
	return max(1, int(math.Floor(itemValue[it]*s)))
}

// sellItem sells up to n of an item from the stockpile; returns how many were sold.
func (g *Game) sellItem(it Item, n int) int {
	if !g.traderHere() || it == ISilver || it == INone {
		return 0
	}
	t := g.trader
	n = min(n, g.stock[it])
	price := g.sellPrice(it)
	n = min(n, t.Cash/price)
	if n <= 0 || !g.takeStock(it, n) {
		return 0
	}
	t.Cash -= n * price
	t.Stock[it] += n
	g.addToStock(ISilver, n*price)
	g.stats.Sold += n
	g.emit("coin", -1, -1)
	return n
}

// buyItem buys up to n of an item from the trader; returns how many were bought.
func (g *Game) buyItem(it Item, n int) int {
	if !g.traderHere() || it == ISilver || it == INone {
		return 0
	}
	t := g.trader
	price := g.buyPrice(it)
	n = min(n, t.Stock[it], g.stock[ISilver]/price)
	if n <= 0 || !g.takeStock(ISilver, n*price) {
		return 0
	}
	t.Stock[it] -= n
	t.Cash += n * price
	g.addToStock(it, n)
	g.stats.Bought += n
	g.emit("coin", -1, -1)
	return n
}

// traderStatus describes the trader for the on-screen banner; ok is false when none is around.
func (g *Game) traderStatus() (msg string, urgent, ok bool) {
	t := g.trader
	if t == nil || t.Leaving {
		return "", false, false
	}
	if !t.Arrived {
		d := math.Hypot(t.X-float64(t.Spot.x), t.Y-float64(t.Spot.y))
		return fmt.Sprintf("Trader approaching - %d tiles away (click to look)", int(d)), false, true
	}
	left := int(math.Max(0, t.Stay))
	return fmt.Sprintf("TRADER HERE - leaves in %d:%02d - click or press T to trade", left/60, left%60), left < 40, true
}

// updateBeast makes the pack animal trot after its owner and wait beside them.
func (g *Game) updateBeast(t *Trader, dt float64) {
	ox, oy := t.BX, t.BY
	t.BThink -= dt
	if math.Hypot(t.X-t.BX, t.Y-t.BY) > 1.6 {
		if t.BThink <= 0 || len(t.BPath) == 0 {
			t.BThink = 0.6
			tx, ty := int(t.X), int(t.Y)
			if p, ok := g.W.bfs(int(t.BX), int(t.BY), 30, func(a, b int) bool { return iabs(a-tx) <= 1 && iabs(b-ty) <= 1 }); ok {
				t.BPath = p
			} else if math.Hypot(t.X-t.BX, t.Y-t.BY) > 14 { // lost: catch up
				t.BX, t.BY, t.BPath = t.X, t.Y, nil
			}
		}
		if len(t.BPath) > 0 && !g.stepPath(&t.BX, &t.BY, &t.BPath, 3.0, dt) {
			t.BPath = nil
		}
	} else {
		t.BPath = nil
	}
	if dx := t.BX - ox; math.Abs(dx) > 1e-4 {
		t.BDX = dx
	}
	t.BMoving = math.Hypot(t.BX-ox, t.BY-oy) > 1e-5
}
