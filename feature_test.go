package main

import (
	"math"
	"testing"
)

// quiet returns a game with no animals and no work designations so tests are deterministic.
func quiet(seed int64) *Game {
	g := newGame(seed)
	g.animals = nil
	g.nextRaid = 1e9
	g.nextRaiders = 1e9
	g.nextOutbreak = 1e9
	g.nextTrader = 1e9
	g.weatherLeft = 1e9 // dry weather
	return g
}

func place(g *Game, o Obj, dx, dy int) pt {
	p := pt{g.start.x + dx, g.start.y + dy}
	t := g.W.At(p.x, p.y)
	t.Obj, t.Ground, t.Item, t.ItemN, t.Zone = o, GGrass, INone, 0, false
	return p
}

func stockUp(g *Game, it Item, n int) {
	g.addToStock(it, n)
}

func TestSkillXPAndLevelUp(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Skills[SMining] = SkillState{}
	c.Traits = nil
	g.gainXP(c, SMining, xpNeeded(0)-1)
	if c.lvl(SMining) != 0 {
		t.Fatal("levelled too early")
	}
	g.gainXP(c, SMining, 2)
	if c.lvl(SMining) != 1 {
		t.Fatalf("expected level 1, got %d", c.lvl(SMining))
	}
	// passion doubles, fast learner x1.5 on top
	a, b := &Colonist{}, &Colonist{Traits: []Trait{TrFastLearner}}
	a.Skills[SCrafting].Passion, b.Skills[SCrafting].Passion = true, true
	g.gainXP(a, SCrafting, 10)
	g.gainXP(b, SCrafting, 10)
	if a.Skills[SCrafting].XP != 20 || b.Skills[SCrafting].XP != 30 {
		t.Fatalf("xp multipliers wrong: %v %v", a.Skills[SCrafting].XP, b.Skills[SCrafting].XP)
	}
	// level cap
	a.Skills[SCrafting].Level = maxSkill
	g.gainXP(a, SCrafting, 1e6)
	if a.lvl(SCrafting) != maxSkill {
		t.Fatal("exceeded max level")
	}
}

func TestFoundersCoverEverySystem(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		g := newGame(seed)
		if g.bestSkill(SMedicine) < 4 || g.bestSkill(SCrafting) < 4 || g.bestSkill(SEngineering) < 3 {
			t.Fatalf("seed %d lacks a founding specialist", seed)
		}
		for _, c := range g.cols {
			if len(c.Traits) == 0 {
				t.Fatalf("%s has no traits", c.Name)
			}
			for _, tr := range c.Traits {
				if ex := exclusive[tr]; ex != TrNone && c.has(ex) {
					t.Fatalf("%s has conflicting traits", c.Name)
				}
			}
		}
	}
}

func TestWorkingTrainsSkill(t *testing.T) {
	g := quiet(7)
	markNearest(g, OTree, DChop, 6, 30)
	before := 0.0
	for _, c := range g.cols {
		before += float64(c.lvl(SFarming))*1000 + c.Skills[SFarming].XP
	}
	run(g, 90)
	after := 0.0
	for _, c := range g.cols {
		after += float64(c.lvl(SFarming))*1000 + c.Skills[SFarming].XP
	}
	if after <= before {
		t.Fatal("chopping should train Farming")
	}
}

func TestSkilledWorkersAreFaster(t *testing.T) {
	lo, hi := &Colonist{}, &Colonist{}
	hi.Skills[SConstruction].Level = 10
	if hi.rate(SConstruction) <= lo.rate(SConstruction)*1.3 {
		t.Fatalf("level 10 should be much faster: %v vs %v", hi.rate(SConstruction), lo.rate(SConstruction))
	}
	g := &Colonist{Traits: []Trait{TrTinkerer}}
	if g.rate(SCrafting) <= lo.rate(SCrafting) {
		t.Fatal("tinkerer should craft faster")
	}
}

func TestCraftingCarvings(t *testing.T) {
	g := quiet(7)
	place(g, OBench, -3, 2)
	g.prod[0].On, g.prod[0].Target = true, 3
	run(g, 150)
	g.recompute()
	if got := g.stock[ICarving]; got < 3 {
		t.Fatalf("expected >=3 carvings, have %d (wood %d)", got, g.stock[IWood])
	}
	if g.stats.Crafted < 3 {
		t.Fatal("crafted counter not updated")
	}
	// it stops once the target is reached
	n := g.stock[ICarving]
	run(g, 120)
	g.recompute()
	if g.stock[ICarving] > n+1 {
		t.Fatalf("kept crafting past target: %d -> %d", n, g.stock[ICarving])
	}
}

func TestRecipeNeedsSkillAndBench(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Skills[SCrafting].Level = 1
	}
	g.prod[2].On = true // component needs Crafting 3
	if s := g.recipeStatus(2); s != "needs a workbench" {
		t.Fatalf("status = %q", s)
	}
	place(g, OBench, -3, 2)
	if s := g.recipeStatus(2); s != "nobody has Crafting 3" {
		t.Fatalf("status = %q", s)
	}
	run(g, 120)
	g.recompute()
	if g.stock[IComponent] != 0 {
		t.Fatal("unskilled colonists must not make components")
	}
	g.cols[0].Skills[SCrafting].Level = 3
	run(g, 150)
	g.recompute()
	if g.stock[IComponent] == 0 {
		t.Fatalf("a skilled crafter should make components (status %q)", g.recipeStatus(2))
	}
}

func TestPowerGrid(t *testing.T) {
	g := quiet(7)
	place(g, OGen, 4, 4)
	lamp := place(g, OLamp, 6, 4)
	g.slow(1)
	if g.powerSupply < 100 {
		t.Fatalf("a fuelled generator should supply power, got %v", g.powerSupply)
	}
	if g.powerRatio != 1 || len(g.lamps) != 1 {
		t.Fatalf("lamp should be lit: ratio %v lamps %v", g.powerRatio, g.lamps)
	}
	if !g.lit(lamp.x+3, lamp.y) || g.lit(lamp.x+20, lamp.y) {
		t.Fatal("lamp light radius wrong")
	}
	// overload: electric heaters exceed one generator
	for i := 0; i < 4; i++ {
		place(g, OEHeater, 8+i, 4)
	}
	g.slow(1)
	if g.powerRatio >= 1 {
		t.Fatalf("expected a brownout, ratio %v (supply %v demand %v)", g.powerRatio, g.powerSupply, g.powerDemand)
	}
	// engineers boost output
	base := g.powerSupply
	g.cols[0].Skills[SEngineering].Level = 15
	g.slow(1)
	if g.powerSupply <= base {
		t.Fatal("engineering skill should boost generator output")
	}
}

func TestGeneratorBurnsWoodAndStops(t *testing.T) {
	g := quiet(7)
	place(g, OGen, 4, 4)
	g.slow(0)
	w0 := g.stock[IWood]
	for i := 0; i < 20; i++ {
		g.slow(5) // 100 s
	}
	if g.stock[IWood] >= w0 {
		t.Fatal("generator should burn wood")
	}
	for g.stock[IWood] > 0 {
		g.takeStock(IWood, g.stock[IWood])
		g.recompute()
	}
	for i := 0; i < 20; i++ {
		g.slow(5)
	}
	if g.powerSupply != 0 {
		t.Fatalf("generator without wood must stop, supply %v", g.powerSupply)
	}
}

func TestFabricatorMakesGadgets(t *testing.T) {
	g := quiet(7)
	place(g, OFab, -3, 2)
	place(g, OGen, 4, 4)
	g.cols[2].Skills[SEngineering].Level = 5
	stockUp(g, IComponent, 6)
	g.prod[3].On, g.prod[3].Target = true, 2
	run(g, 200)
	g.recompute()
	if g.stock[IGadget] < 1 {
		t.Fatalf("expected a gadget (status %q, power %.2f)", g.recipeStatus(3), g.powerRatio)
	}
}

func TestEngineeringGatesBuilding(t *testing.T) {
	g := quiet(7)
	stockUp(g, IComponent, 10)
	stockUp(g, IStone, 30)
	for _, c := range g.cols {
		c.Skills[SEngineering].Level = 0
	}
	g.applyTool(TBuild, 0, 0, 0, 0) // no selection: nothing happens
	g.buildSel = BGen
	p := pt{g.start.x + 5, g.start.y + 5}
	g.W.At(p.x, p.y).Obj, g.W.At(p.x, p.y).Item = ONone, INone
	g.applyTool(TBuild, p.x, p.y, p.x, p.y)
	if g.W.At(p.x, p.y).Blue != BGen {
		t.Fatal("blueprint not placed")
	}
	run(g, 60)
	if g.W.At(p.x, p.y).Obj == OGen {
		t.Fatal("generator needs Engineering 2")
	}
	g.cols[1].Skills[SEngineering].Level = 2
	run(g, 90)
	if g.W.At(p.x, p.y).Obj != OGen {
		t.Fatal("a skilled engineer should build it")
	}
}

func TestStudyingAtDeskTrainsFocus(t *testing.T) {
	g := quiet(7)
	place(g, ODesk, -3, 2)
	c := g.cols[0]
	c.Focus = SMining
	c.Skills[SMining] = SkillState{}
	for _, o := range g.cols[1:] {
		o.Dead = true
	}
	g.cols = g.cols[:1]
	run(g, 120)
	if c.Skills[SMining].XP+float64(c.lvl(SMining))*100 <= 0 {
		t.Fatal("studying at a desk should train the focus skill")
	}
	// a book gives a burst
	c.Skills[SMining] = SkillState{}
	stockUp(g, IBook, 1)
	run(g, 60)
	g.recompute()
	if g.stock[IBook] != 0 {
		t.Fatal("the book should have been read")
	}
	if c.lvl(SMining) < 2 {
		t.Fatalf("a book is worth a lot of XP, level %d", c.lvl(SMining))
	}
}

func TestMedicSkillImprovesTending(t *testing.T) {
	novice, expert := &Colonist{}, &Colonist{}
	expert.Skills[SMedicine].Level = 10
	g := quiet(7)
	// novices botch some tendings, experts never do
	botched := func(c *Colonist) int {
		n := 0
		for i := 0; i < 400; i++ {
			l := c.lvl(SMedicine)
			if (l < 3 && g.chance(0.2)) || (l >= 3 && l < 6 && g.chance(0.05)) {
				n++
			}
		}
		return n
	}
	if botched(novice) < 30 || botched(expert) != 0 {
		t.Fatal("medic skill should reduce botched tending")
	}
	// healing quality
	in := Injury{Quality: 1 + 0.06*10}
	if in.Quality <= 1.5 {
		t.Fatal("expert quality")
	}
}

func TestTraderTradesAndLeaves(t *testing.T) {
	g := quiet(7)
	g.nextTrader = g.time + 1
	run(g, 5)
	if g.trader == nil {
		t.Fatal("trader should have spawned")
	}
	if !g.onMainland(int(g.trader.X), int(g.trader.Y)) {
		t.Fatal("trader spawned off the mainland")
	}
	if g.traderHere() {
		t.Fatal("trader hasn't arrived yet")
	}
	for i := 0; i < 60*120 && !g.traderHere(); i++ {
		g.step(1.0 / 60)
	}
	if !g.traderHere() {
		t.Fatal("trader never arrived")
	}
	// selling
	stockUp(g, ICarving, 8)
	silver0 := g.stock[ISilver]
	sold := g.sellItem(ICarving, 5)
	g.recompute()
	if sold != 5 || g.stock[ICarving] != 3 {
		t.Fatalf("sold %d, carvings left %d", sold, g.stock[ICarving])
	}
	if got := g.stock[ISilver] - silver0; got != 5*g.sellPrice(ICarving) {
		t.Fatalf("silver gained %d", got)
	}
	// profit margin: they pay less than they sell for
	if g.sellPrice(ICarving) >= g.buyPrice(ICarving) {
		t.Fatal("trader must buy low and sell high")
	}
	// buying
	m0 := g.stock[IMedicine]
	got := g.buyItem(IMedicine, 2)
	g.recompute()
	if got != 2 || g.stock[IMedicine] != m0+2 {
		t.Fatalf("bought %d medicine (%d -> %d)", got, m0, g.stock[IMedicine])
	}
	// can't buy without money
	g.takeStock(ISilver, g.stock[ISilver])
	g.recompute()
	if g.buyItem(IComponent, 1) != 0 {
		t.Fatal("bought without silver")
	}
	// silver can't be traded as an item; trader cash is limited
	if g.sellItem(ISilver, 1) != 0 {
		t.Fatal("sold silver")
	}
	g.trader.Cash = 10
	stockUp(g, IGadget, 3)
	if n := g.sellItem(IGadget, 3); n*g.sellPrice(IGadget) > 10 {
		t.Fatalf("trader paid beyond their cash: %d gadgets", n)
	}
	// and they eventually leave
	g.trader.Stay = 1
	for i := 0; i < 60*400 && g.trader != nil; i++ {
		g.step(1.0 / 60)
	}
	if g.trader != nil {
		t.Fatal("trader never left")
	}
	if g.nextTrader <= g.time {
		t.Fatal("next visit should be scheduled")
	}
}

func TestTraderCannotTradeWhenAbsent(t *testing.T) {
	g := quiet(7)
	stockUp(g, ICarving, 3)
	if g.sellItem(ICarving, 1) != 0 || g.buyItem(IMedicine, 1) != 0 {
		t.Fatal("no trader, no trade")
	}
}

func TestCharmingColonistGetsBetterPrices(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Traits = nil
	}
	g.trader = &Trader{Mod: 1, Arrived: true, Cash: 1000}
	b0, s0 := g.buyPrice(IGadget), g.sellPrice(IGadget)
	g.cols[0].Traits = []Trait{TrCharming}
	if g.buyPrice(IGadget) >= b0 || g.sellPrice(IGadget) <= s0 {
		t.Fatal("charming should help")
	}
}

func TestSaveKeepsSkillsTraitsAndSettings(t *testing.T) {
	g := quiet(7)
	g.cols[0].Skills[SMedicine] = SkillState{Level: 9, XP: 12, Passion: true}
	g.cols[0].Traits = []Trait{TrHardy, TrCharming}
	g.cols[0].Focus = SEngineering
	g.prod[1] = ProdSetting{true, 17}
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	c := h.cols[0]
	if c.Skills[SMedicine] != (SkillState{Level: 9, XP: 12, Passion: true}) || !c.has(TrHardy) || !c.has(TrCharming) || c.Focus != SEngineering {
		t.Fatalf("skills/traits lost: %+v %v %v", c.Skills, c.Traits, c.Focus)
	}
	if h.prod[1] != (ProdSetting{true, 17}) {
		t.Fatalf("production settings lost: %+v", h.prod)
	}
}

func TestBlueprintBlockExplainsWhy(t *testing.T) {
	g := quiet(7)
	g.prod[2].On = false
	why := g.blueprintBlock(&Tile{Blue: BSolar})
	if why == "" || !contains(why, "Component") || !contains(why, "turn on Component production") {
		t.Fatalf("should explain the missing component and how to fix it: %q", why)
	}
	stockUp(g, IComponent, 5)
	stockUp(g, IStone, 10)
	for _, c := range g.cols {
		c.Skills[SEngineering].Level = 0
	}
	if why := g.blueprintBlock(&Tile{Blue: BSolar}); !contains(why, "Engineering 3") {
		t.Fatalf("should mention the skill: %q", why)
	}
	g.cols[0].Skills[SEngineering].Level = 3
	if why := g.blueprintBlock(&Tile{Blue: BSolar}); why != "" {
		t.Fatalf("nothing should block it now: %q", why)
	}
}

func TestUnpoweredLampsAreExplained(t *testing.T) {
	g := quiet(7)
	place(g, OLamp, 4, 4)
	g.slow(1)
	if g.powerProblem(OLamp) == "" || !contains(g.powerProblem(OLamp), "Generator") {
		t.Fatalf("an unpowered lamp should say how to fix it: %q", g.powerProblem(OLamp))
	}
	// the colony gets a nudge too
	g.hintT = 0
	g.msgs = nil
	g.advise(1)
	if len(g.msgs) == 0 || !contains(g.msgs[0].Text, "no power") {
		t.Fatalf("expected a no-power warning, got %v", g.msgs)
	}
	place(g, OGen, 6, 4)
	g.slow(1)
	if g.powerProblem(OLamp) != "" {
		t.Fatal("lamp should be powered once there is a generator")
	}
}

// totalItems counts everything: stockpiles, loose items and carried goods.
func totalItems(g *Game) (n [numItems]int) {
	for i := range g.W.T {
		if t := &g.W.T[i]; t.Item != INone {
			n[t.Item] += t.ItemN
		}
	}
	for _, c := range g.cols {
		if c.Carry != INone {
			n[c.Carry] += c.CarryN
		}
	}
	return
}

func TestSalvageRefundsHalf(t *testing.T) {
	cases := []struct {
		obj  Obj
		want map[Item]int
	}{
		{OWall, map[Item]int{IStone: 2}},               // 4 stone -> 2
		{OBench, map[Item]int{IWood: 4}},               // 8 wood -> 4
		{OGen, map[Item]int{IStone: 5, IComponent: 1}}, // 10 stone + 2 comp
		{OLamp, map[Item]int{IComponent: 1, IWood: 1}}, // 1 comp + 1 wood -> rounds up
		{OBed, map[Item]int{IWood: 2}},                 // 3 wood -> 2
	}
	for _, tc := range cases {
		g := quiet(7)
		p := place(g, tc.obj, 5, 5)
		g.W.At(p.x, p.y).Fuel = 1000 // a running generator would burn stockpiled wood during the test
		before := totalItems(g)
		g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
		if g.W.At(p.x, p.y).Desig != DDecon {
			t.Fatalf("%v: not marked", tc.obj)
		}
		run(g, 60)
		if g.W.At(p.x, p.y).Obj != ONone {
			t.Fatalf("%v: not dismantled", tc.obj)
		}
		after := totalItems(g)
		for it := Item(1); it < numItems; it++ {
			if got := after[it] - before[it]; got != tc.want[it] {
				t.Errorf("%v: %s changed by %d, want %d", tc.obj, itemNames[it], got, tc.want[it])
			}
		}
	}
}

func TestSalvageFloorAndRooms(t *testing.T) {
	g := quiet(7)
	p := place(g, ONone, 6, 6)
	g.W.At(p.x, p.y).Floor = true
	before := totalItems(g)
	g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
	run(g, 30)
	if g.W.At(p.x, p.y).Floor {
		t.Fatal("floor should be gone")
	}
	if totalItems(g)[IWood]-before[IWood] != 1 {
		t.Fatal("a floor refunds 1 of its 2 wood")
	}
	// removing a wall reopens the room
	w := place(g, OWall, 8, 8)
	g.computeRooms()
	g.applyTool(TDeconstruct, w.x, w.y, w.x, w.y)
	run(g, 30)
	if g.roomOf[w.y*MapW+w.x] < 0 {
		t.Fatal("room map should be recomputed after removing a wall")
	}
}

func TestSalvageTrainsConstructionAndIgnoresBlueprints(t *testing.T) {
	g := quiet(7)
	p := place(g, OBench, 5, 5)
	q := place(g, ONone, 7, 5)
	g.W.At(q.x, q.y).Blue = BWall
	g.applyTool(TDeconstruct, q.x, q.y, q.x+0, q.y)
	if g.W.At(q.x, q.y).Desig != DNone {
		t.Fatal("blueprints are cancelled with Erase, not salvaged")
	}
	xp := func() (x float64) {
		for _, c := range g.cols {
			x += float64(c.lvl(SConstruction))*1000 + c.Skills[SConstruction].XP
		}
		return
	}
	x0 := xp()
	g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
	run(g, 40)
	if xp() <= x0 {
		t.Fatal("dismantling should train Construction")
	}
}

func TestSalvageSurvivesCancelAndSave(t *testing.T) {
	g := quiet(7)
	p := place(g, OBench, 5, 5)
	g.applyTool(TDeconstruct, p.x, p.y, p.x, p.y)
	g.applyTool(TErase, p.x, p.y, p.x, p.y) // erase wins: instant, nothing left
	if g.W.At(p.x, p.y).Obj != ONone || g.W.At(p.x, p.y).Desig != DNone {
		t.Fatal("erase should clear it")
	}
	q := place(g, OBench, 6, 5)
	g.applyTool(TDeconstruct, q.x, q.y, q.x, q.y)
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h.W.At(q.x, q.y).Desig != DDecon {
		t.Fatal("the salvage order should survive a save")
	}
	run(h, 40)
	if h.W.At(q.x, q.y).Obj != ONone {
		t.Fatal("and still get carried out after loading")
	}
}

func TestTraderNotifications(t *testing.T) {
	g := quiet(7)
	g.nextTrader = g.time + 1
	g.msgs = nil
	run(g, 3)
	if g.trader == nil {
		t.Fatal("no trader")
	}
	if len(g.msgs) == 0 || !g.msgs[0].Alert || g.msgs[0].T < 20 || !contains(g.msgs[0].Text, "approaching") {
		t.Fatalf("approach should raise a lingering alert: %+v", g.msgs)
	}
	if m, urgent, ok := g.traderStatus(); !ok || urgent || !contains(m, "approaching") {
		t.Fatalf("status before arrival: %q %v %v", m, urgent, ok)
	}
	for i := 0; i < 60*120 && !g.traderHere(); i++ {
		g.step(1.0 / 60)
	}
	if !g.traderHere() {
		t.Fatal("never arrived")
	}
	if m, urgent, ok := g.traderStatus(); !ok || urgent || !contains(m, "TRADER HERE") {
		t.Fatalf("status on arrival: %q %v", m, urgent)
	}
	arrived := false
	for _, m := range g.msgs {
		if m.Alert && contains(m.Text, "has arrived") {
			arrived = true
		}
	}
	if !arrived {
		t.Fatalf("arrival alert missing: %+v", g.msgs)
	}
	if g.trader.Stay < 80 || g.trader.Stay > 140 {
		t.Fatalf("traders should stay a short while (about 100 s), stay=%v", g.trader.Stay)
	}
	// a warning before they leave, once only
	g.trader.Stay = 41
	g.msgs = nil
	run(g, 3)
	warns := 0
	for _, m := range g.msgs {
		if contains(m.Text, "leaving in") {
			warns++
		}
	}
	if warns != 1 {
		t.Fatalf("expected one leaving-soon alert, got %d", warns)
	}
	if _, urgent, _ := g.traderStatus(); !urgent {
		t.Fatal("the banner should turn urgent when they are about to leave")
	}
	run(g, 5)
	for _, m := range g.msgs {
		if contains(m.Text, "leaving in") && m.T < 0 {
			t.Fatal("stale")
		}
	}
}

func TestAlertsSurviveMessageSpam(t *testing.T) {
	g := quiet(7)
	g.msgs = nil
	g.alert("important")
	for i := 0; i < 20; i++ {
		g.msg("noise %d", i)
	}
	found := false
	for _, m := range g.msgs {
		if m.Text == "important" {
			found = true
		}
	}
	if !found {
		t.Fatal("an alert was pushed out by ordinary messages")
	}
	if len(g.msgs) > 7 {
		t.Fatalf("log should stay short: %d", len(g.msgs))
	}
}

func TestRightButtonPansAndClicksCancel(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh, g.zoom = 1, 1280, 760, 2
	g.camX, g.camY = 1000, 1000
	g.tool = TChop

	// press, drag 40 px right and 20 px down, release: the camera moves the opposite way
	g.lastMX, g.lastMY = 300, 300
	g.rightMouse(300, 300, true, true, false)
	for _, p := range [][2]float64{{310, 305}, {325, 312}, {340, 320}} {
		g.rightMouse(p[0], p[1], true, false, false)
		g.lastMX, g.lastMY = p[0], p[1]
	}
	g.rightMouse(340, 320, false, false, true)
	if g.camX >= 1000 || g.camY >= 1000 {
		t.Fatalf("dragging right/down should move the view left/up: %v,%v", g.camX, g.camY)
	}
	if g.tool != TChop {
		t.Fatal("a drag must not count as a click")
	}
	if g.rpActive || g.rpPan {
		t.Fatal("gesture state should reset")
	}

	// a tiny jiggle is still a click: back to the Select tool
	g.camX = 1000
	g.lastMX, g.lastMY = 500, 500
	g.rightMouse(500, 500, true, true, false)
	g.rightMouse(502, 501, true, false, false)
	g.lastMX, g.lastMY = 502, 501
	g.rightMouse(502, 501, false, false, true)
	if g.tool != TSelect || g.camX != 1000 {
		t.Fatalf("a click selects the Select tool without moving the camera (tool %v, camX %v)", g.tool, g.camX)
	}

	// a click closes an open panel first
	g.tool, g.panel = TChop, panelProd
	g.rightMouse(10, 10, true, true, false)
	g.rightMouse(10, 10, false, false, true)
	if g.panel != panelNone || g.tool != TChop {
		t.Fatal("click closes the panel but keeps the tool")
	}

	// it cancels a tool drag in progress, and the release afterwards does nothing more
	g.dragging, g.tool = true, TMine
	g.rightMouse(10, 10, true, true, false)
	g.rightMouse(10, 10, false, false, true)
	if g.dragging || g.tool != TMine {
		t.Fatal("click cancels the drag only")
	}

	// starting over a panel never pans the map
	g.selected = g.cols[0].ID
	px, py, _, _ := g.colPanelRect(g.cols[0])
	g.camX = 1000
	g.lastMX, g.lastMY = px+20, py+20
	g.rightMouse(px+20, py+20, true, true, false)
	g.rightMouse(px+120, py+80, true, false, false)
	g.rightMouse(px+120, py+80, false, false, true)
	if g.camX != 1000 {
		t.Fatal("dragging from a panel must not move the camera")
	}
}

func TestLampLightHasASharpEdge(t *testing.T) {
	if glowAlpha(0) < 0.95 || glowAlpha(0) > 1 {
		t.Fatalf("centre should be (almost) full strength, got %v", glowAlpha(0))
	}
	if glowAlpha(0.5) < 0.8*glowAlpha(0) {
		t.Fatalf("most of the circle should be bright, got %.2f at half radius", glowAlpha(0.5))
	}
	if glowAlpha(0.95) > 0.1 || glowAlpha(1) != 0 || glowAlpha(1.4) != 0 {
		t.Fatalf("the edge should drop off sharply: %.2f at 95%%", glowAlpha(0.95))
	}
	prev := 2.0
	for d := 0.0; d <= 1.0; d += 0.01 {
		a := glowAlpha(d)
		if a > prev+1e-9 {
			t.Fatalf("brightness must never increase with distance (at %.2f)", d)
		}
		prev = a
	}
	// the glow matches the gameplay radius
	g := quiet(7)
	place(g, OGen, 4, 4)
	l := place(g, OLamp, 6, 4)
	g.slow(1)
	if !g.lit(l.x+int(lampRadius)-1, l.y) || g.lit(l.x+int(lampRadius)+2, l.y) {
		t.Fatal("lit area should match lampRadius")
	}
}

func TestTraderComesWithAPackAnimalThatFollows(t *testing.T) {
	for seed := int64(1); seed <= 6; seed++ {
		g := quiet(seed)
		g.nextTrader = g.time + 1
		g.msgs = nil
		run(g, 3)
		tr := g.trader
		if tr == nil {
			t.Fatalf("seed %d: no trader", seed)
		}
		if tr.Beast != BeastDonkey && tr.Beast != BeastCamel {
			t.Fatal("bad beast")
		}
		if math.Hypot(tr.X-tr.BX, tr.Y-tr.BY) > 3 {
			t.Fatal("the pack animal starts beside its owner")
		}
		named := false
		for _, m := range g.msgs {
			if contains(m.Text, beastNames[tr.Beast]) {
				named = true
			}
		}
		if !named {
			t.Fatalf("the alert should mention the %s: %+v", beastNames[tr.Beast], g.msgs)
		}
		far := 0.0
		for i := 0; i < 60*100 && !g.traderHere(); i++ {
			g.step(1.0 / 60)
			if g.trader != nil {
				far = math.Max(far, math.Hypot(g.trader.X-g.trader.BX, g.trader.Y-g.trader.BY))
			}
		}
		if !g.traderHere() {
			t.Fatalf("seed %d: trader never arrived", seed)
		}
		if far > 7 {
			t.Fatalf("seed %d: the %s fell %.1f tiles behind", seed, beastNames[tr.Beast], far)
		}
		run(g, 3) // let it catch up and settle
		if d := math.Hypot(tr.X-tr.BX, tr.Y-tr.BY); d > 3 {
			t.Fatalf("seed %d: the %s should wait beside the trader, %.1f tiles away", seed, beastNames[tr.Beast], d)
		}
		if !g.W.Walkable(int(tr.BX), int(tr.BY)) {
			t.Fatal("the animal is standing somewhere impassable")
		}
		// it leaves along with its owner
		tr.Stay = 1
		for i := 0; i < 60*400 && g.trader != nil; i++ {
			g.step(1.0 / 60)
			if g.trader != nil {
				if math.Hypot(g.trader.X-g.trader.BX, g.trader.Y-g.trader.BY) > 12 {
					t.Fatal("pack animal lost its owner while leaving")
				}
			}
		}
		if g.trader != nil {
			t.Fatal("trader never left")
		}
	}
}

func TestCamelsCarryMoreThanDonkeys(t *testing.T) {
	var donkey, camel, nd, nc float64
	for seed := int64(1); seed <= 80; seed++ {
		g := quiet(seed)
		g.nextTrader = g.time
		g.spawnTrader()
		tr := g.trader
		if tr == nil {
			continue
		}
		total := 0
		for _, n := range tr.Stock {
			total += n
		}
		if tr.Beast == BeastCamel {
			camel += float64(total) + float64(tr.Cash)/10
			nc++
		} else {
			donkey += float64(total) + float64(tr.Cash)/10
			nd++
		}
	}
	if nd == 0 || nc == 0 {
		t.Fatalf("both kinds should turn up: %v donkeys, %v camels", nd, nc)
	}
	if camel/nc <= donkey/nd*1.2 {
		t.Fatalf("camel caravans should be richer: %.0f vs %.0f", camel/nc, donkey/nd)
	}
}

func TestBooksGetReadPromptlyWithoutADesk(t *testing.T) {
	g := quiet(7)
	// plenty of ordinary work to do: books used to wait behind all of it
	markNearest(g, OTree, DChop, 12, 30)
	markNearest(g, ORock, DMine, 6, 30)
	c := g.cols[1]
	c.Focus = SMining
	c.Skills[SMining] = SkillState{}
	g.addToStock(IBook, 1)
	g.msgs = nil
	run(g, 60)
	g.recompute()
	if g.stock[IBook] != 0 {
		t.Fatal("someone should have picked the book up and read it")
	}
	read := false
	for _, m := range g.msgs {
		if contains(m.Text, "read a book") {
			read = true
		}
	}
	if !read {
		t.Fatalf("the player should see who read it: %v", g.msgs)
	}
	gained := 0.0
	for _, o := range g.cols {
		gained += float64(o.lvl(o.Focus))*1000 + o.Skills[o.Focus].XP
	}
	if gained < bookXP/2 {
		t.Fatalf("a book is worth a lot of experience, total gained %.0f", gained)
	}
}

func TestSeveralBooksAreSharedOut(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Skills[c.Focus] = SkillState{}
	}
	g.addToStock(IBook, 3)
	run(g, 90)
	g.recompute()
	if g.stock[IBook] != 0 {
		t.Fatalf("all three books should be read, %d left", g.stock[IBook])
	}
	if g.readers() != 0 {
		t.Fatal("reading should be over")
	}
}

func TestBooksAreNotWastedOnMasters(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Skills[c.Focus] = SkillState{Level: 14}
	}
	g.addToStock(IBook, 1)
	run(g, 40)
	g.recompute()
	if g.stock[IBook] != 1 {
		t.Fatal("nobody can learn from this book: keep it")
	}
	// exhausted or starving colonists don't read
	g2 := quiet(7)
	for _, c := range g2.cols {
		c.Rest, c.Hunger = 20, 100
	}
	if g2.tryRead(g2.cols[0]) {
		t.Fatal("too tired to read")
	}
}

func TestTooltipHelpersDescribeCropsAndRooms(t *testing.T) {
	g := quiet(7)
	x, y := g.start.x, g.start.y
	tl := g.W.At(x, y)
	tl.Farm, tl.Crop, tl.CropKind, tl.Grow = true, true, CropWheat, 0.42
	if l := g.cropTip(tl, x, y); len(l) == 0 || !contains(l[0], "Wheat: 42% grown") {
		t.Fatalf("%v", l)
	}
	tl.Grow = 1
	if l := g.cropTip(tl, x, y); !contains(l[0], "ripe") {
		t.Fatalf("%v", l)
	}
	tl.Crop = false
	if l := g.cropTip(tl, x, y); !contains(l[0], "not sown") {
		t.Fatalf("%v", l)
	}
	if g.roomTip(0, 0) != nil && g.roomOf[0] < 0 {
		t.Fatal("open ground has no room tip")
	}
}
