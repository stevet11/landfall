package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"testing"
)

// Balance harness: scripted "players" of different skill play several seeds.
// Run with LANDFALL_BALANCE=1 go test -run Balance -v -timeout 30m

type strategy int

const (
	stratCompetent strategy = iota
	stratLazy
	stratIdle
)

func (s strategy) String() string { return [...]string{"competent", "lazy", "idle"}[s] }

// markNearest designates up to n undesignated objects of kind o near the start.
func markNearest(g *Game, o Obj, d Desig, n, rmax int) {
	type cand struct {
		i int
		d float64
	}
	var cs []cand
	active := 0
	for i := range g.W.T {
		t := &g.W.T[i]
		if t.Desig == d {
			active++
		}
		if t.Obj == o && t.Desig == DNone {
			dx, dy := float64(i%MapW-g.start.x), float64(i/MapW-g.start.y)
			if r := math.Hypot(dx, dy); r <= float64(rmax) {
				cs = append(cs, cand{i, r})
			}
		}
	}
	sort.Slice(cs, func(a, b int) bool { return cs[a].d < cs[b].d })
	for k := 0; k < len(cs) && active+k < n; k++ {
		g.W.T[cs[k].i].Desig = d
	}
}

func setupBase(g *Game) {
	s := g.start
	rx, ry := s.x-12, s.y+3
	for y := ry - 1; y <= ry+5; y++ {
		for x := rx - 1; x <= rx+7; x++ {
			g.W.At(x, y).Obj, g.W.At(x, y).Ground = ONone, GGrass
		}
	}
	g.applyTool(TDoor, rx+3, ry+4, rx+3, ry+4)
	g.applyTool(TWall, rx, ry, rx+6, ry)
	g.applyTool(TWall, rx, ry+4, rx+6, ry+4)
	g.applyTool(TWall, rx, ry+1, rx, ry+3)
	g.applyTool(TWall, rx+6, ry+1, rx+6, ry+3)
	g.applyTool(THeater, rx+5, ry+3, rx+5, ry+3)
	g.applyTool(TBed, rx+1, ry+3, rx+3, ry+3)
	g.applyTool(TFloor, rx+1, ry+1, rx+5, ry+2)
	g.applyTool(TRoof, rx+1, ry+1, rx+5, ry+3) // a sensible player roofs the house
	g.applyTool(TFarm, s.x-9, s.y-9, s.x-4, s.y-6)
	// a sensible player gives people something to do with their free time
	for i, bl := range []Blue{BHorse, BChecker} {
		for dy := 0; dy < 3; dy++ {
			g.W.At(s.x+8+i*3, s.y+3+dy).Obj = ONone
		}
		g.buildSel = bl
		g.applyTool(TBuild, s.x+8+i*3, s.y+3, s.x+8+i*3, s.y+3)
	}
}

func botTick(g *Game, st strategy, sec int) {
	if st == stratIdle {
		return
	}
	if sec == 0 && st == stratCompetent {
		setupBase(g)
	}
	if st == stratCompetent && sec == 60 {
		g.applyTool(TZone, g.start.x-3, g.start.y-8, g.start.x+3, g.start.y-4) // a second stockpile
	}
	if sec%20 != 0 {
		return
	}
	stockWood, stockStone, stockFood := g.stock[IWood], g.stock[IStone], g.stock[IFood]
	// a sensible player marks what they need now: stone first while the room is unfinished
	if stockWood < 100 && !(stockStone < 40 && stockWood >= 40) {
		markNearest(g, OTree, DChop, 8, 30)
	}
	if stockStone < 60 {
		markNearest(g, ORock, DMine, 8, 45)
	}
	if stockFood < 70 {
		markNearest(g, OBush, DHarvest, 6, 30)
	}
	if g.stock[IMedicine] < 4 {
		markNearest(g, OHerb, DHarvest, 3, 40)
	}
	if st == stratCompetent && stockFood < 45 {
		for _, a := range g.animals {
			if a.Kind == ADeer && !a.Hunt && math.Hypot(a.X-float64(g.start.x), a.Y-float64(g.start.y)) < 40 {
				a.Hunt = true
				break
			}
		}
	}
	// a sensible player arms the colony before the raids get serious: ore, a furnace, then swords
	if st == stratCompetent && sec == int(DayLen*3) {
		s := g.start
		for dy := 0; dy < 2; dy++ {
			g.W.At(s.x+8, s.y+7+dy).Obj = ONone
		}
		g.buildSel = BFurnace
		g.applyTool(TBuild, s.x+8, s.y+7, s.x+8, s.y+7)
		for _, name := range []string{"Iron", "Knife", "Sword"} {
			g.prod[recipeIndex(name)].On = true
		}
		g.prod[recipeIndex("Iron")].Target = 12
		g.prod[recipeIndex("Sword")].Target = 3
	}
	if st == stratCompetent && sec >= int(DayLen*3) && sec%60 == 0 && g.stock[IOre] < 6 {
		markNearest(g, OOre, DMine, 6, 45)
	}
	// keep the farm tilled while it is warm enough
	if st == stratCompetent && sec%300 == 0 {
		g.applyTool(TFarm, g.start.x-9, g.start.y-9, g.start.x-4, g.start.y-6)
	}
}

type result struct {
	strat      strategy
	seed       int64
	start, end int
	minCols    int
	stats      Stats
	injuries   int
	food, wood []int
	stone      []int
	deathDay   []int
}

func play(seed int64, st strategy, days int) result {
	g := newGame(seed)
	// the tests count raids/kills; don't depend on the UI
	g.onEvent = nil
	r := result{strat: st, seed: seed, start: len(g.cols), minCols: len(g.cols)}
	prev := len(g.cols)
	for sec := 0; sec < int(DayLen)*days; sec++ {
		botTick(g, st, sec)
		for k := 0; k < 60; k++ {
			g.step(1.0 / 60)
		}
		if n := len(g.cols); n != prev {
			if n < prev {
				r.deathDay = append(r.deathDay, sec/int(DayLen)+1)
			}
			prev = n
			r.minCols = min(r.minCols, n)
		}
		if sec%int(DayLen) == 0 {
			r.food = append(r.food, g.stock[IFood])
			r.wood = append(r.wood, g.stock[IWood])
			r.stone = append(r.stone, g.stock[IStone])
		}
		for _, c := range g.cols {
			r.injuries += len(c.Injuries) * 0
		}
	}
	r.end = len(g.cols)
	r.stats = g.stats
	return r
}

func TestBalance(t *testing.T) {
	if os.Getenv("LANDFALL_BALANCE") == "" {
		t.Skip("set LANDFALL_BALANCE=1 to run")
	}
	days := 12
	seeds := []int64{1, 2, 3, 4}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var results []result
	for _, st := range []strategy{stratCompetent, stratLazy} {
		for _, sd := range seeds {
			wg.Add(1)
			go func(st strategy, sd int64) {
				defer wg.Done()
				r := play(sd, st, days)
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}(st, sd)
		}
	}
	wg.Wait()
	sort.Slice(results, func(i, j int) bool {
		if results[i].strat != results[j].strat {
			return results[i].strat < results[j].strat
		}
		return results[i].seed < results[j].seed
	})
	for _, r := range results {
		fmt.Printf("%-9s seed=%d cols %d->%d (min %d) raids=%d wolvesKilled=%d bites=%d starved=%d froze=%d bled=%d killed=%d raiders=%d/%d stolen=%d ill=%d recovered=%d illDeaths=%d sulks=%d played=%d deathDays=%v\n  food/day=%v stone/day=%v\n",
			r.strat, r.seed, r.start, r.end, r.minCols, r.stats.Raids, r.stats.WolvesKilled, r.stats.Bites,
			r.stats.Starved, r.stats.Froze, r.stats.Bled, r.stats.Killed, r.stats.RaidersKilled, r.stats.RaidersSeen, r.stats.Stolen, r.stats.Illnesses, r.stats.Recovered, r.stats.IllDeaths, r.stats.Sulks, r.stats.Played, r.deathDay, r.food, r.stone)
	}
}
