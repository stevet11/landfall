package main

import (
	"math"
	"testing"
)

// openField clears a lane of ground so fights are not hindered by trees and rocks.
func openField(g *Game, dx0, dx1, dy0, dy1 int) {
	for dx := dx0; dx <= dx1; dx++ {
		for dy := dy0; dy <= dy1; dy++ {
			tl := g.W.At(g.start.x+dx, g.start.y+dy)
			if tl != nil && !tl.Zone {
				tl.Obj, tl.Ground, tl.Item, tl.ItemN, tl.Blue = ONone, GGrass, INone, 0, BNone
			}
		}
	}
	g.land = nil
}

func addRaider(g *Game, weapon Item, dx, dy int) *Animal {
	g.nextID++
	a := &Animal{ID: g.nextID, Kind: ARaider, X: float64(g.start.x+dx) + 0.5, Y: float64(g.start.y+dy) + 0.5,
		HP: 48, MaxHP: 48, DX: 1, Weapon: weapon, Exit: pt{g.start.x + dx, g.start.y + dy}}
	g.animals = append(g.animals, a)
	return a
}

func raiders(g *Game) (n int) {
	for _, a := range g.animals {
		if a.Kind == ARaider && !a.Dead {
			n++
		}
	}
	return
}

func TestCompassWord(t *testing.T) {
	for _, c := range []struct {
		dx, dy float64
		want   string
	}{{10, 0, "east"}, {-10, 0, "west"}, {0, -10, "north"}, {0, 10, "south"}, {7, -7, "north-east"}, {-7, 7, "south-west"}, {-7, -7, "north-west"}, {7, 7, "south-east"}} {
		if got := compassWord(c.dx, c.dy); got != c.want {
			t.Errorf("(%v,%v): %q, want %q", c.dx, c.dy, got, c.want)
		}
	}
}

func TestRaidSpawnsArmedMenOnReachableLand(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		g := quiet(seed)
		g.animals = nil
		g.msgs = nil
		if !g.spawnRaid() {
			t.Fatalf("seed %d: no raid", seed)
		}
		n := raiders(g)
		if n != g.raid.Start || n < 2 {
			t.Fatalf("seed %d: %d raiders, raid says %d", seed, n, g.raid.Start)
		}
		for _, a := range g.animals {
			if a.Kind != ARaider {
				continue
			}
			if _, armed := weaponDefs[a.Weapon]; !armed {
				t.Fatal("raiders are armed")
			}
			if !g.onMainland(int(a.X), int(a.Y)) {
				t.Fatalf("seed %d: raider spawned where it can't reach the colony", seed)
			}
			if d := math.Hypot(a.X-float64(g.start.x), a.Y-float64(g.start.y)); d < 35 {
				t.Fatalf("raiders arrive from afar, not %.0f tiles away", d)
			}
		}
		alert := false
		for _, m := range g.msgs {
			if m.Alert && contains(m.Text, "Raiders!") && contains(m.Text, "from the") {
				alert = true
			}
		}
		if !alert {
			t.Fatalf("a raid should raise an alert with a direction: %v", g.msgs)
		}
	}
}

func TestRaidsGrowWithTheColonyAndTime(t *testing.T) {
	g := quiet(7)
	small := g.raidSize()
	for i := 0; i < 9; i++ {
		g.addColonist(float64(g.start.x)+0.5, float64(g.start.y)+0.5, SCombat, 0)
	}
	if g.raidSize() <= small {
		t.Fatal("bigger colonies attract bigger raids")
	}
	big := g.raidSize()
	g.time = DayLen * 14
	if g.raidSize() < big || g.raidSize() > 7 {
		t.Fatalf("size stays within bounds: %d", g.raidSize())
	}
	weapons := func(day float64) map[Item]int {
		g := quiet(7)
		m := map[Item]int{}
		for i := 0; i < 400; i++ {
			m[g.raiderWeapon(day)]++
		}
		return m
	}
	early, mid, late := weapons(2), weapons(7), weapons(12)
	if early[IPistol]+early[IRifle]+early[ISword] != 0 {
		t.Fatalf("early raiders only carry knives and axes: %v", early)
	}
	if mid[ISword] == 0 || mid[IPistol] == 0 || mid[IRifle] != 0 {
		t.Fatalf("mid-game raiders: swords and pistols, no rifles yet: %v", mid)
	}
	if late[IRifle] == 0 {
		t.Fatalf("late raiders carry rifles: %v", late)
	}
}

func TestRaidsAreScheduledAndDontOverlap(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	g.nextRaiders = g.time + 1
	run(g, 3)
	if g.raid == nil || raiders(g) == 0 {
		t.Fatal("the scheduled raid should arrive")
	}
	if g.nextRaiders <= g.time+DayLen*3 {
		t.Fatalf("the next raid should be days away: %.0f s", g.nextRaiders-g.time)
	}
	// even if the timer says so, a second raid never starts on top of the first
	n := raiders(g)
	g.nextRaiders = g.time
	run(g, 2)
	if raiders(g) > n {
		t.Fatal("no second raid while one is in progress")
	}
}

func TestMeleeRaidersWoundColonists(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	for _, c := range g.cols {
		c.Weapon = INone
		c.Health = 100
		c.X, c.Y = float64(g.start.x+20)+0.5, float64(g.start.y)+0.5
		g.abort(c)
	}
	a := addRaider(g, IKnife, 16, 0)
	g.raid = &raidInfo{Start: 1}
	run(g, 25)
	if g.stats.RaiderHits == 0 {
		t.Fatal("the raider should have hit someone")
	}
	hurt := false
	for _, c := range g.cols {
		for _, in := range c.Injuries {
			if in.Kind == "Cut" && in.Bleed {
				hurt = true
			}
		}
	}
	if !hurt && !a.Dead {
		t.Fatal("a blade wound should bleed")
	}
}

func TestColonistsDefendAndLootTheDead(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	for _, c := range g.cols {
		c.Weapon = ISword
		c.Skills[SCombat].Level = 6
		c.X, c.Y = float64(g.start.x+12)+0.5, float64(g.start.y)+0.5
		g.abort(c)
	}
	addRaider(g, IKnife, 22, 0)
	addRaider(g, IAxe, 22, 1)
	g.raid = &raidInfo{Start: 2}
	silver0 := totalItems(g)[ISilver]
	run(g, 60)
	if raiders(g) != 0 {
		t.Fatalf("three swordsmen should beat two knife men; %d raiders left", raiders(g))
	}
	for _, c := range g.cols {
		if c.Dead {
			t.Fatalf("%s died", c.Name)
		}
	}
	if g.stats.RaidersKilled != 2 {
		t.Fatalf("kills counted: %d", g.stats.RaidersKilled)
	}
	if totalItems(g)[ISilver] <= silver0 {
		t.Fatal("dead raiders drop silver")
	}
	if g.raid != nil {
		t.Fatal("the raid should be over")
	}
}

func TestRangedRaidersShootFromAFar(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	for _, c := range g.cols {
		c.Weapon = INone
		c.X, c.Y = float64(g.start.x+10)+0.5, float64(g.start.y)+0.5
		g.abort(c)
		c.FightCD = 1e9 // they hold still so we can watch the raider
	}
	a := addRaider(g, IRifle, 22, 0)
	g.raid = &raidInfo{Start: 1}
	shotFrom := 0.0
	for i := 0; i < 60*12; i++ {
		g.step(1.0 / 60)
		if len(g.shots) > 0 && shotFrom == 0 {
			c, d := g.nearestColonist(a, 100)
			if c != nil {
				shotFrom = d
			}
		}
	}
	if len(g.shots) == 0 && g.stats.RaiderHits == 0 && shotFrom == 0 {
		t.Fatal("a rifleman should open fire")
	}
	if shotFrom < 5 {
		t.Fatalf("riflemen shoot from range, not point blank: %.1f tiles", shotFrom)
	}
}

func TestRaiderBulletsStopAtWalls(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	c := g.cols[0]
	for _, o := range g.cols[1:] {
		o.Dead = true
	}
	g.cols = g.cols[:1]
	c.X, c.Y = float64(g.start.x+10)+0.5, float64(g.start.y)+0.5
	c.FightCD = 1e9
	for dy := -6; dy <= 6; dy++ { // a long wall between them
		w := g.W.At(g.start.x+16, g.start.y+dy)
		w.Obj, w.Mat = OWall, MatStone
	}
	g.computeRooms()
	a := addRaider(g, IRifle, 22, 0)
	g.raid = &raidInfo{Start: 1}
	for i := 0; i < 60*3; i++ {
		before := g.stats.RaiderHits
		blocked := !g.clearShot(a.X, a.Y, c.X, c.Y)
		g.step(1.0 / 60)
		if blocked && g.stats.RaiderHits != before {
			t.Fatal("bullets must not pass through a wall")
		}
	}
}

func TestRaidersFleeWhenTheRaidIsBroken(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 40, -6, 6)
	var rs []*Animal
	for i := 0; i < 5; i++ {
		rs = append(rs, addRaider(g, IKnife, 30, i-2))
	}
	g.raid = &raidInfo{Start: 5, Origin: pt{g.start.x + 30, g.start.y}}
	for _, c := range g.cols {
		c.FightCD = 1e9 // stay out of it
	}
	g.msgs = nil
	// the colonists' side kills three of five
	for _, a := range rs[:3] {
		a.Dead = true
	}
	g.updateRaidState()
	if !g.raid.Broken {
		t.Fatal("with 2 of 5 left the raid should break")
	}
	fled := false
	for _, m := range g.msgs {
		if m.Alert && contains(m.Text, "fleeing") {
			fled = true
		}
	}
	if !fled {
		t.Fatal("the player should be told the raiders flee")
	}
	for _, a := range rs[3:] {
		a.X, a.Y = float64(g.start.x+30)+0.5-8, float64(g.start.y)+0.5 // a few steps from the exit
	}
	run(g, 20)
	if raiders(g) != 0 {
		t.Fatalf("fleeing raiders should leave the map, %d remain", raiders(g))
	}
	if g.raid != nil {
		t.Fatal("raid over")
	}
}

func TestBadlyWoundedRaidersRunOff(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	a := addRaider(g, ISword, 20, 0)
	g.raid = &raidInfo{Start: 3}
	a.HP = 10 // under 30%
	for _, c := range g.cols {
		c.FightCD = 1e9
	}
	run(g, 1)
	if a.Goal != goalFlee {
		t.Fatalf("goal %d: a badly hurt raider flees", a.Goal)
	}
}

func TestRaidersStealWhenNobodyCanBeReached(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 40, -8, 8)
	// every colonist is sealed in an iron box
	for _, c := range g.cols {
		c.X, c.Y = float64(g.start.x-12)+0.5, float64(g.start.y-12)+0.5
		g.abort(c)
	}
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			tl := g.W.At(g.start.x-12+dx, g.start.y-12+dy)
			tl.Ground, tl.Obj = GGrass, ONone
			if iabs(dx) == 2 || iabs(dy) == 2 {
				tl.Obj, tl.Mat = OWall, MatIron
			}
		}
	}
	g.computeRooms()
	g.addToStock(IGadget, 4)
	total0 := totalItems(g)[IGadget]
	a := addRaider(g, IKnife, 26, 0)
	g.raid = &raidInfo{Start: 1}
	g.msgs = nil
	stole := false
	for i := 0; i < 60*80 && !a.Dead; i++ {
		g.step(1.0 / 60)
		if a.LootN > 0 {
			stole = true
		}
	}
	if !stole {
		t.Fatalf("with no colonist reachable the raider goes for the stockpile (goal %d)", a.Goal)
	}
	if totalItems(g)[IGadget] >= total0 {
		t.Fatalf("the gadgets (the most valuable thing there) should be what gets stolen: %d of %d left", totalItems(g)[IGadget], total0)
	}
	if !a.Dead {
		t.Fatal("it should have escaped with the goods")
	}
	if g.stats.Stolen == 0 {
		t.Fatal("stolen goods should be counted")
	}
}

func TestKillingAThiefReturnsTheLoot(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	a := addRaider(g, IKnife, 5, 5)
	a.Loot, a.LootN = IGadget, 3
	before := totalItems(g)[IGadget]
	g.raiderDown(g.cols[0], a)
	if totalItems(g)[IGadget] != before+3 {
		t.Fatal("the stolen goods drop where the thief dies")
	}
	if g.stats.RaidersKilled != 1 || !a.Dead {
		t.Fatal("kill counted")
	}
}

func TestRaidBannerTexts(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	if _, ok := g.raidStatus(); ok {
		t.Fatal("no banner without raiders")
	}
	a := addRaider(g, IKnife, 20, 0)
	b := addRaider(g, ISword, 20, 1)
	if m, ok := g.raidStatus(); !ok || !contains(m, "RAIDERS!") || !contains(m, "2") {
		t.Fatalf("attack banner: %q", m)
	}
	a.Goal = goalSteal
	if m, _ := g.raidStatus(); !contains(m, "stealing") {
		t.Fatalf("theft banner: %q", m)
	}
	a.Goal, b.Goal = goalFlee, goalFlee
	if m, _ := g.raidStatus(); !contains(m, "retreating") {
		t.Fatalf("retreat banner: %q", m)
	}
	if f := g.raiderFocus(); f == nil {
		t.Fatal("the banner points at a raider")
	}
}

func TestRaidersSurviveSaves(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	a := addRaider(g, IRifle, 30, 3)
	a.Goal, a.Loot, a.LootN = goalFlee, IGadget, 2
	a.HP = 33
	g.raid = &raidInfo{Start: 1}
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	var b *Animal
	for _, x := range h.animals {
		if x.Kind == ARaider {
			b = x
		}
	}
	if b == nil || b.Weapon != IRifle || b.Goal != goalFlee || b.Loot != IGadget || b.LootN != 2 || b.HP != 33 || b.Exit != a.Exit {
		t.Fatalf("raider state lost: %+v", b)
	}
	if h.raid == nil {
		t.Fatal("a raid in progress should still be tracked after loading")
	}
}

func TestWolvesAndRaidersLeaveEachOtherAlone(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	openField(g, -4, 30, -6, 6)
	for _, c := range g.cols {
		c.FightCD = 1e9
		c.X, c.Y = float64(g.start.x-3)+0.5, float64(g.start.y-5)+0.5
		g.abort(c)
	}
	r := addRaider(g, IKnife, 20, 0)
	w := &Animal{Kind: AWolf, X: r.X + 1.5, Y: r.Y, HP: 50, MaxHP: 50}
	g.animals = append(g.animals, w)
	g.raid = &raidInfo{Start: 1}
	run(g, 3)
	if r.HP < 48 || w.HP < 50 {
		t.Fatal("raiders and wolves are not enemies of each other")
	}
}
