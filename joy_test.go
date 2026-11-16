package main

import "testing"

func TestJoyDrainsAndMoodScalesWork(t *testing.T) {
	g, c := lone(7)
	c.Joy = 80
	for i := 0; i < 100; i++ {
		g.updateJoy(c, 1)
	}
	if want := 80 - 100*joyDecay; c.Joy > want+0.5 || c.Joy < want-0.5 {
		t.Fatalf("joy after 100 s awake: %.1f, want %.1f", c.Joy, want)
	}
	// sleeping drains much more slowly
	g2, d := lone(7)
	d.Joy = 80
	d.Job = Job{Kind: JSleep, Stage: 0}
	d.Path = nil
	for i := 0; i < 100; i++ {
		g2.updateJoy(d, 1)
	}
	if d.Joy <= c.Joy+3 {
		t.Fatalf("sleep preserves spirits: %.1f vs %.1f", d.Joy, c.Joy)
	}
	g.addJoy(c, 1000)
	if c.Joy != 100 {
		t.Fatal("joy is capped at 100")
	}
	g.addJoy(c, -1000)
	if c.Joy != 0 {
		t.Fatal("and floored at 0")
	}
	hi, mid, lo := &Colonist{Joy: 90}, &Colonist{Joy: 25}, &Colonist{Joy: 0}
	if hi.moodFactor() != 1 || mid.moodFactor() >= 1 || lo.moodFactor() != 0.8 || mid.moodFactor() <= lo.moodFactor() {
		t.Fatalf("mood factors: %v %v %v", hi.moodFactor(), mid.moodFactor(), lo.moodFactor())
	}
	calm := &Colonist{Joy: 90, Calm: 50}
	if calm.moodFactor() <= 1 {
		t.Fatal("calm speeds work")
	}
}

func TestLowSpiritsSlowWork(t *testing.T) {
	chop := func(joy float64) float64 {
		g, c := lone(7)
		tree := place(g, OTree, 3, 3)
		g.applyTool(TChop, tree.x, tree.y, tree.x, tree.y)
		for i := 0; i < 60*60; i++ {
			c.Joy = joy
			g.step(1.0 / 60)
			if g.W.At(tree.x, tree.y).Obj != OTree {
				return float64(i) / 60
			}
		}
		return 60
	}
	if chop(0) <= chop(100)*1.12 {
		t.Fatal("an unhappy colonist works noticeably slower")
	}
}

func TestEntertainmentBuildings(t *testing.T) {
	for _, tc := range []struct {
		b     Blue
		o     Obj
		solid bool
		item  Item
	}{{BHorse, OHorse, false, IWood}, {BChecker, OChecker, true, IWood}, {BShrine, OShrine, true, ISculpture}} {
		if blueObj[tc.b] != tc.o || solidBlue(tc.b) != tc.solid || costs[tc.b].Item != tc.item {
			t.Errorf("%s: wrong definition", blueNames[tc.b])
		}
		if !movable(tc.o) {
			t.Errorf("%s should be movable", blueNames[tc.b])
		}
		if _, ok := stationDefs[tc.o]; !ok {
			t.Errorf("%s needs a station definition", blueNames[tc.b])
		}
	}
	g := quiet(7)
	p := place(g, OHorse, 6, 6)
	if !g.W.Walkable(p.x, p.y) {
		t.Fatal("you can walk over a horseshoe pit")
	}
	q := place(g, OChecker, 8, 6)
	if g.W.Walkable(q.x, q.y) || !g.W.At(q.x, q.y).flammable() {
		t.Fatal("a wooden table blocks the way and burns")
	}
	if place(g, OShrine, 10, 6); (g.W.At(g.start.x+10, g.start.y+6)).flammable() {
		t.Fatal("a stone shrine doesn't burn")
	}
	// the shrine needs a crafted sculpture
	g.W.At(g.start.x+12, g.start.y+6).Blue = BShrine
	why := g.blueprintBlock(g.W.At(g.start.x+12, g.start.y+6))
	g.stock[ISculpture] = 0
	why = g.blueprintBlock(g.W.At(g.start.x+12, g.start.y+6))
	if !contains(why, "Sculpture") {
		t.Fatalf("explain the sculpture: %q", why)
	}
	inMenu := map[Blue]bool{}
	for _, b := range buildMenu {
		inMenu[b] = true
	}
	if !inMenu[BHorse] || !inMenu[BChecker] || !inMenu[BShrine] {
		t.Fatal("all three are in the build menu")
	}
}

func lowJoy(g *Game, v float64) {
	for _, c := range g.cols {
		c.Joy = v
	}
}

func TestLowSpiritsSendColonistsToPlay(t *testing.T) {
	g := quiet(7)
	openField(g, 3, 20, 3, 12)
	pit := place(g, OHorse, 8, 8)
	lowJoy(g, 20)
	played, sawStatus := false, false
	for i := 0; i < 60*80; i++ {
		g.step(1.0 / 60)
		for _, c := range g.cols {
			if c.Job.Kind == JPlay && len(c.Path) == 0 {
				played = true
				if c.Status == "Playing horseshoes" {
					sawStatus = true
				}
			}
		}
	}
	if !sawStatus {
		t.Fatal("the panel should say what they are doing")
	}
	if !played {
		t.Fatal("unhappy colonists should go and play")
	}
	if g.stats.Played == 0 {
		t.Fatal("finished sessions are counted")
	}
	total := 0.0
	for _, c := range g.cols {
		total += c.Joy
	}
	if total/float64(len(g.cols)) < 45 {
		t.Fatalf("play should lift spirits, mean joy %.0f", total/float64(len(g.cols)))
	}
	_ = pit
}

func TestWorkComesBeforeFunUnlessSpiritsAreLow(t *testing.T) {
	setup := func(joy float64) (*Game, *Colonist) {
		g, c := lone(7)
		openField(g, 3, 20, 3, 12)
		place(g, OHorse, 8, 8)
		for i := 0; i < 6; i++ {
			p := place(g, OTree, 12+i, 4)
			g.applyTool(TChop, p.x, p.y, p.x, p.y)
		}
		c.Joy = joy
		return g, c
	}
	g, c := setup(60) // middling: only when idle
	for i := 0; i < 60*10; i++ {
		c.Joy = 60
		g.step(1.0 / 60)
		if c.Job.Kind == JPlay {
			t.Fatal("with work to do, a content colonist keeps working")
		}
	}
	g2, d := setup(30) // low: drops work
	played := false
	for i := 0; i < 60*20 && !played; i++ {
		g2.step(1.0 / 60)
		d.Joy = min(d.Joy, 30)
		if d.Job.Kind == JPlay {
			played = true
		}
	}
	if !played {
		t.Fatal("a colonist with low spirits takes a break even with work waiting")
	}
}

func TestIdleColonistsPlayWhenNotFullyHappy(t *testing.T) {
	g, c := lone(7)
	openField(g, 3, 20, 3, 12)
	place(g, OChecker, 8, 8)
	c.Joy = 70
	played := false
	for i := 0; i < 60*30 && !played; i++ {
		g.step(1.0 / 60)
		played = c.Job.Kind == JPlay
	}
	if !played {
		t.Fatal("idle and a bit bored: go and play")
	}
	g2, d := lone(7)
	openField(g2, 3, 20, 3, 12)
	place(g2, OChecker, 8, 8)
	d.Joy = 95
	for i := 0; i < 60*20; i++ {
		d.Joy = 95
		g2.step(1.0 / 60)
		if d.Job.Kind == JPlay {
			t.Fatal("happy people don't need games")
		}
	}
}

func TestTwoCanPlayTogetherAndPartnersBenefit(t *testing.T) {
	g := quiet(7)
	openField(g, 3, 20, 3, 12)
	tbl := place(g, OChecker, 8, 8)
	for _, o := range g.cols[2:] {
		o.Dead = true
	}
	g.cols = g.cols[:2]
	a, b := g.cols[0], g.cols[1]
	for i := 0; i < 60*60; i++ {
		a.Joy, b.Joy = min(a.Joy, 20), min(b.Joy, 20)
		g.step(1.0 / 60)
		if a.Job.Kind == JPlay && b.Job.Kind == JPlay && len(a.Path) == 0 && len(b.Path) == 0 {
			break
		}
	}
	if a.Job.Kind != JPlay || b.Job.Kind != JPlay {
		t.Fatal("both should sit down to play")
	}
	tl := g.W.At(tbl.x, tbl.y)
	if tl.ResT == 0 || tl.ResD == 0 || tl.ResT == tl.ResD {
		t.Fatalf("two different players hold the two places: %d %d", tl.ResT, tl.ResD)
	}
	if g.partnerOf(a) != b || g.partnerOf(b) != a {
		t.Fatal("they are each other's partner")
	}
	// the gain with a partner beats playing alone
	a.Joy, b.Joy = 10, 10
	g.doPlay(a, 1)
	with := a.Joy - 10
	b.Job = Job{}
	a.Joy = 10
	g.doPlay(a, 1)
	alone := a.Joy - 10
	if with <= alone*1.3 {
		t.Fatalf("a partner makes it better: %.2f vs %.2f", with, alone)
	}
	// a full table turns a third colonist away
	c3 := &Colonist{ID: 99, Joy: 10, Rest: 100, Hunger: 100, X: a.X, Y: a.Y}
	g.cols = append(g.cols, c3)
	b.Job = Job{Kind: JPlay, Target: a.Job.Target}
	if g.tryPlay(c3, true) {
		t.Fatal("no free place at the only table")
	}
}

func TestLeavingFreesTheSeat(t *testing.T) {
	g, c := lone(7)
	openField(g, 3, 20, 3, 12)
	tbl := place(g, OHorse, 8, 8)
	c.Joy = 10
	if !g.tryPlay(c, true) {
		t.Fatal("should go play")
	}
	tl := g.W.At(tbl.x, tbl.y)
	if tl.ResT != c.ID {
		t.Fatal("the place is reserved")
	}
	g.abort(c)
	if tl.ResT != 0 || tl.ResD != 0 {
		t.Fatal("an interrupted player frees the place")
	}
}

func TestMeditationLeavesCalmThatSpeedsWork(t *testing.T) {
	g, c := lone(7)
	openField(g, 3, 20, 3, 12)
	place(g, OShrine, 8, 8)
	c.Joy = 20
	for i := 0; i < 60*60 && c.Calm == 0; i++ {
		c.Joy = min(c.Joy, 20)
		g.step(1.0 / 60)
	}
	if c.Calm <= 0 {
		t.Fatal("meditation should leave a calm")
	}
	if c.Calm < calmSeconds-5 {
		t.Fatalf("calm lasts %.0f s", c.Calm)
	}
	c.Joy = 100
	if c.moodFactor() <= 1 {
		t.Fatal("calm speeds up work")
	}
	for i := 0; i < int(calmSeconds)+5; i++ {
		g.updateJoy(c, 1)
	}
	if c.Calm != 0 {
		t.Fatal("calm fades")
	}
}

func TestSulkingStopsWorkAndEnds(t *testing.T) {
	g, c := lone(7)
	c.Joy = 0
	g.msgs = nil
	for i := 0; i < 600 && c.Sulk == 0; i++ {
		c.Joy = 0
		g.updateJoy(c, 1)
	}
	if c.Sulk <= 0 || g.stats.Sulks != 1 {
		t.Fatal("a colonist with no joy left eventually sulks")
	}
	found := false
	for _, m := range g.msgs {
		if contains(m.Text, "foul mood") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the player should be told: %v", g.msgs)
	}
	tree := place(g, OTree, 3, 3)
	g.applyTool(TChop, tree.x, tree.y, tree.x, tree.y)
	g.abort(c)
	c.Hunger = 90
	for i := 0; i < 60*10; i++ {
		c.Sulk = max(c.Sulk, 5)
		g.step(1.0 / 60)
		if c.Job.Kind != JNone && c.Job.Kind != JEat {
			t.Fatalf("a sulker takes no jobs (job %d)", c.Job.Kind)
		}
	}
	if c.Status != "Sulking" && c.Status != "Idle" {
		t.Fatal(c.Status)
	}
	c.Sulk = 0.5
	c.Joy = 0
	g.updateJoy(c, 1)
	if c.Sulk != 0 || c.Joy < 25 {
		t.Fatalf("the sulk ends with a little cheer restored: sulk %.1f joy %.0f", c.Sulk, c.Joy)
	}
}

func TestGoodFoodCheersPeopleUp(t *testing.T) {
	gain := func(it Item) float64 {
		g, c := lone(7)
		for _, i := range g.zoneList {
			g.W.T[i].Item, g.W.T[i].ItemN = INone, 0
		}
		g.recompute()
		g.addToStock(it, 2)
		c.Hunger = 20
		c.Joy = 40
		start := c.Joy
		peak := start
		for i := 0; i < 60*20; i++ {
			g.step(1.0 / 60)
			peak = max(peak, c.Joy)
		}
		return peak - start
	}
	if gain(IMeal) < 8 || gain(IBread) < 2.5 || gain(IBread) >= gain(IMeal) {
		t.Fatalf("meals %.1f, bread %.1f", gain(IMeal), gain(IBread))
	}
	if foodJoy(IFood) != 0 || foodJoy(IWood) != 0 {
		t.Fatal("plain food is not a treat")
	}
}

func TestJoySurvivesSavesAndOldSavesGetSomeCheer(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Joy, c.Calm, c.Sulk = 37, 55, 9
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	d := h.cols[0]
	if d.Joy != 37 || d.Calm != 55 || d.Sulk != 9 {
		t.Fatalf("lost: %v %v %v", d.Joy, d.Calm, d.Sulk)
	}
	sd := g.snapshot()
	sd.Version = 5
	for i := range sd.Cols {
		sd.Cols[i].Joy = 0 // a version-5 save has no joy
	}
	k := quiet(5)
	k.apply(sd)
	for _, c := range k.cols {
		if c.Joy < 60 {
			t.Fatalf("%s starts an old save in good spirits, not %.0f", c.Name, c.Joy)
		}
	}
}

func TestStationsCanBeMovedAndErased(t *testing.T) {
	g := quiet(7)
	openField(g, 3, 20, 3, 14)
	src := place(g, OChecker, 5, 5)
	dst := place(g, ONone, 12, 9)
	if why := g.orderMove(src.x, src.y, dst.x, dst.y); why != "" {
		t.Fatal(why)
	}
	run(g, 60)
	if g.W.At(dst.x, dst.y).Obj != OChecker {
		t.Fatal("the table should have been carried over")
	}
	g.applyTool(TErase, dst.x, dst.y, dst.x, dst.y)
	if g.W.At(dst.x, dst.y).Obj != ONone {
		t.Fatal("erase removes it")
	}
}

func TestColonistPanelHasARoomyJoyBar(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	c := g.cols[0]
	_, py, _, h := g.colPanelRect(c)
	for _, b := range g.buttons() {
		if b.h == 17 && b.w > 100 && b.y+b.h > py+h { // skill rows must be inside the panel
			t.Fatalf("a skill row sticks out of the panel: %+v (panel bottom %.0f)", b, py+h)
		}
	}
	if h < 352 {
		t.Fatalf("panel too short: %.0f", h)
	}
}

func TestNoGamesDuringDanger(t *testing.T) {
	g, c := lone(7)
	openField(g, 3, 30, 3, 12)
	place(g, OHorse, 8, 8)
	c.Joy = 10
	w := &Animal{Kind: AWolf, X: c.X + 12, Y: c.Y, HP: 50, MaxHP: 50}
	g.animals = []*Animal{w}
	if g.tryPlay(c, true) {
		t.Fatal("don't play with a wolf nearby")
	}
	g.animals = nil
	g.fireCount = 1
	if g.tryPlay(c, true) {
		t.Fatal("or with a fire burning")
	}
	g.fireCount = 0
	if !g.tryPlay(c, true) {
		t.Fatal("all clear: go and play")
	}
}
