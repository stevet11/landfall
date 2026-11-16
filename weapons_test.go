package main

import (
	"math"
	"testing"
)

func armedWith(g *Game, c *Colonist, it Item) {
	c.Weapon = it
}

func TestWeaponProfiles(t *testing.T) {
	g := quiet(7)
	c := &Colonist{}
	fist, fcool, _, franged := g.attackProfile(c)
	if franged || fist < 7 || fcool != bareCool {
		t.Fatalf("bare hands: %v %v %v", fist, fcool, franged)
	}
	for _, it := range []Item{IKnife, IAxe, ISword} {
		c.Weapon = it
		d, cool, rng, ranged := g.attackProfile(c)
		if ranged || rng != meleeRng || d <= fist {
			t.Fatalf("%s should hit harder than fists: %v", itemNames[it], d)
		}
		if d/cool <= fist/fcool*1.3 && it != IAxe {
			t.Fatalf("%s should have a better damage rate than fists", itemNames[it])
		}
	}
	// guns need ammo
	c.Weapon = IPistol
	g.stock[IAmmo] = 0
	d, _, rng, ranged := g.attackProfile(c)
	if ranged || rng != meleeRng || d >= fist {
		t.Fatalf("an empty gun is a poor club: %v ranged=%v", d, ranged)
	}
	g.stock[IAmmo] = 10
	_, _, rng, ranged = g.attackProfile(c)
	if !ranged || rng < 7 {
		t.Fatal("a loaded pistol shoots from range")
	}
	c.Weapon = IRifle
	_, _, rng2, _ := g.attackProfile(c)
	if rng2 <= rng {
		t.Fatal("rifles outrange pistols")
	}
	// skill and trait
	lo, hi := &Colonist{Weapon: ISword}, &Colonist{Weapon: ISword, Traits: []Trait{TrBrawler}}
	hi.Skills[SCombat].Level = 10
	if a, _, _, _ := g.attackProfile(hi); a <= func() float64 { d, _, _, _ := g.attackProfile(lo); return d }()*1.5 {
		t.Fatal("combat skill and brawler should add up")
	}
}

func TestHitChance(t *testing.T) {
	g := quiet(7)
	rifle := weaponDefs[IRifle]
	novice, ace := &Colonist{}, &Colonist{}
	ace.Skills[SCombat].Level = 12
	if g.hitChance(ace, rifle, 8) <= g.hitChance(novice, rifle, 8) {
		t.Fatal("skill should improve aim")
	}
	if g.hitChance(novice, rifle, 3) <= g.hitChance(novice, rifle, 12) {
		t.Fatal("distance should hurt accuracy")
	}
	if g.hitChance(novice, rifle, 8) <= g.hitChance(novice, weaponDefs[IPistol], 8) {
		t.Fatal("rifles are more accurate")
	}
	for d := 0.0; d < 40; d += 5 {
		if p := g.hitChance(ace, rifle, d); p < 0.15 || p > 0.95 {
			t.Fatalf("chance out of bounds: %v", p)
		}
	}
}

func TestLineOfSight(t *testing.T) {
	g := quiet(7)
	for dx := 0; dx < 12; dx++ {
		for dy := -2; dy <= 2; dy++ {
			p := place(g, ONone, 5+dx, 10+dy)
			g.W.At(p.x, p.y).Obj = ONone
		}
	}
	x0, y0 := float64(g.start.x+5)+0.5, float64(g.start.y+10)+0.5
	x1, y1 := float64(g.start.x+14)+0.5, y0
	if !g.clearShot(x0, y0, x1, y1) {
		t.Fatal("open ground is a clear shot")
	}
	wall := place(g, OWall, 9, 10)
	_ = wall
	if g.clearShot(x0, y0, x1, y1) {
		t.Fatal("walls block shots")
	}
	g.W.At(g.start.x+9, g.start.y+10).Obj = OTree
	if g.clearShot(x0, y0, x1, y1) {
		t.Fatal("trees block shots")
	}
	g.W.At(g.start.x+9, g.start.y+10).Obj = OBench
	if !g.clearShot(x0, y0, x1, y1) {
		t.Fatal("a workbench is low cover, not a wall")
	}
	// the shooter's own tile and the target's tile never block
	g.W.At(g.start.x+5, g.start.y+10).Obj = OTree
	g.W.At(g.start.x+14, g.start.y+10).Obj = OTree
	if !g.clearShot(x0, y0, x1, y1) {
		t.Fatal("only tiles in between matter")
	}
}

func makeWeapons(g *Game, it Item, n int) {
	g.addToStock(it, n)
}

func TestColonistsEquipTheBestWeaponAndReturnTheOld(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Weapon = INone
	}
	makeWeapons(g, IKnife, 1)
	run(g, 30)
	armed := 0
	for _, c := range g.cols {
		if c.Weapon == IKnife {
			armed++
		}
	}
	if armed != 1 {
		t.Fatalf("exactly one colonist should have taken the knife, %d did", armed)
	}
	g.recompute()
	if g.stock[IKnife] != 0 {
		t.Fatal("the knife left the stockpile")
	}
	// a better weapon arrives: someone upgrades and the knife goes back
	makeWeapons(g, ISword, 1)
	run(g, 40)
	g.recompute()
	swords, knives := 0, 0
	for _, c := range g.cols {
		switch c.Weapon {
		case ISword:
			swords++
		case IKnife:
			knives++
		}
	}
	if swords != 1 {
		t.Fatalf("the sword should be equipped by exactly one colonist, %d have it", swords)
	}
	if swords+knives != 2 || g.stock[IKnife]+g.stock[ISword] != 0 {
		t.Fatalf("both weapons should be in someone's hands: swords %d knives %d stock k%d s%d", swords, knives, g.stock[IKnife], g.stock[ISword])
	}
	// nothing is lost or duplicated
	total := map[Item]int{}
	for _, c := range g.cols {
		if c.Weapon != INone {
			total[c.Weapon]++
		}
	}
	for it, n := range totalItems(g) {
		if _, w := weaponDefs[Item(it)]; w {
			total[Item(it)] += n
		}
	}
	if total[IKnife] != 1 || total[ISword] != 1 {
		t.Fatalf("weapon count changed: %v", total)
	}
}

func TestGunsAreNotEquippedWithoutAmmo(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Weapon = INone
	}
	makeWeapons(g, IRifle, 1)
	run(g, 30)
	for _, c := range g.cols {
		if c.Weapon == IRifle {
			t.Fatal("a rifle with no ammo is worse than fists: don't pick it up")
		}
	}
	makeWeapons(g, IAmmo, 20)
	run(g, 30)
	got := false
	for _, c := range g.cols {
		if c.Weapon == IRifle {
			got = true
		}
	}
	if !got {
		t.Fatal("once there is ammo, someone takes the rifle")
	}
}

func TestDeadColonistsDropTheirWeapon(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Weapon = ISword
	c.Health = -1
	run(g, 1)
	if !c.Dead {
		t.Fatal("should be dead")
	}
	if totalItems(g)[ISword] != 1 {
		t.Fatal("the sword should be left behind")
	}
}

// duel puts one colonist (with the given weapon) against one wolf and returns how long it took,
// and whether the colonist survived.
func duel(weapon Item, ammo int, startDist float64, skill int) (secs float64, survivor bool, hp float64, ammoLeft int) {
	g := quiet(7)
	g.animals = nil
	for _, o := range g.cols[1:] {
		o.Dead = true
	}
	g.cols = g.cols[:1]
	c := g.cols[0]
	c.Weapon = weapon
	c.Skills[SCombat].Level = skill
	c.Traits = nil
	c.Health = 100
	if ammo > 0 {
		g.addToStock(IAmmo, ammo)
	}
	// open ground around the duel
	for dx := -3; dx < 20; dx++ {
		for dy := -4; dy <= 4; dy++ {
			p := place(g, ONone, 10+dx, 10+dy)
			g.W.At(p.x, p.y).Obj = ONone
		}
	}
	c.X, c.Y = float64(g.start.x+10)+0.5, float64(g.start.y+10)+0.5
	g.land = nil // the terrain changed: recompute what is reachable
	g.abort(c)
	w := &Animal{Kind: AWolf, X: c.X + startDist, Y: c.Y, HP: 40, MaxHP: 40}
	g.animals = []*Animal{w}
	for i := 0; i < 60*60; i++ {
		g.step(1.0 / 60)
		if w.Dead {
			return float64(i) / 60, !c.Dead, c.Health, g.stock[IAmmo]
		}
		if c.Dead {
			break
		}
	}
	return 60, !c.Dead, c.Health, g.stock[IAmmo]
}

func TestBetterWeaponsWinFights(t *testing.T) {
	tFist, _, hFist, _ := duel(INone, 0, 8, 3)
	tSword, _, hSword, _ := duel(ISword, 0, 8, 3)
	tKnife, _, _, _ := duel(IKnife, 0, 8, 3)
	if tSword >= tFist || tKnife >= tFist {
		t.Fatalf("blades should kill faster: fist %.1fs knife %.1fs sword %.1fs", tFist, tKnife, tSword)
	}
	if hSword <= hFist {
		t.Fatalf("a sword should cost less health: %.0f vs %.0f", hSword, hFist)
	}
}

func TestRiflesKillBeforeContact(t *testing.T) {
	secs, ok, hp, ammoLeft := duel(IRifle, 30, 12, 4)
	if !ok || secs >= 60 {
		t.Fatal("the rifleman should win")
	}
	if hp < 95 {
		t.Fatalf("a wolf shot while charging should barely touch a rifleman, health %.0f", hp)
	}
	if ammoLeft >= 30 {
		t.Fatal("shooting uses ammo")
	}
	// and with no ammo the rifle is a club: they take real damage
	_, _, hpEmpty, _ := duel(IRifle, 0, 12, 4)
	if hpEmpty >= hp {
		t.Fatalf("no ammo should hurt: %.0f vs %.0f", hpEmpty, hp)
	}
}

func TestPistolsShootFromRange(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	g.addToStock(IAmmo, 20)
	c := g.cols[0]
	c.Weapon = IPistol
	w := &Animal{Kind: AWolf, X: c.X + 6, Y: c.Y, HP: 400, MaxHP: 400}
	for dx := -1; dx < 9; dx++ {
		p := place(g, ONone, -2+dx, -6)
		g.W.At(p.x, p.y).Obj = ONone
	}
	c.X, c.Y = float64(g.start.x-2)+0.5, float64(g.start.y-6)+0.5
	w.X, w.Y = c.X+6, c.Y
	g.animals = []*Animal{w}
	g.abort(c)
	c.Job = Job{Kind: JFight, Animal: w}
	g.recompute()
	for i := 0; i < 60*8; i++ {
		g.chase(c, 1.0/60)
	}
	if g.stats.Shots < 5 {
		t.Fatalf("a pistol should be firing: %d shots", g.stats.Shots)
	}
	if math.Hypot(w.X-c.X, w.Y-c.Y) < 5 {
		t.Fatal("the shooter must not walk up to the target")
	}
	if len(g.shots) == 0 {
		t.Fatal("tracers should be recorded for drawing")
	}
}

func TestWallsStopBullets(t *testing.T) {
	g := quiet(7)
	g.animals = nil
	g.addToStock(IAmmo, 20)
	c := g.cols[0]
	c.Weapon = IRifle
	c.X, c.Y = float64(g.start.x-2)+0.5, float64(g.start.y-6)+0.5
	for dx := 0; dx < 8; dx++ {
		p := place(g, ONone, -2+dx, -6)
		g.W.At(p.x, p.y).Obj = ONone
	}
	wall := place(g, OWall, 2, -6)
	g.W.At(wall.x, wall.y).Mat = MatStone
	w := &Animal{Kind: AWolf, X: float64(g.start.x+4) + 0.5, Y: c.Y, HP: 400, MaxHP: 400}
	g.animals = []*Animal{w}
	g.abort(c)
	c.Job = Job{Kind: JFight, Animal: w}
	x0, y0 := c.X, c.Y
	for i := 0; i < 60*6; i++ {
		blocked := !g.clearShot(c.X, c.Y, w.X, w.Y)
		before := g.stats.Shots
		g.chase(c, 1.0/60)
		if blocked && g.stats.Shots != before {
			t.Fatal("no shooting through a wall")
		}
	}
	if g.stats.Shots == 0 {
		t.Fatal("once they have moved to a clear line they should shoot")
	}
	if c.X == x0 && c.Y == y0 {
		t.Fatal("a blocked shooter should reposition")
	}
}

func TestAxesSpeedUpChopping(t *testing.T) {
	chopTime := func(weapon Item) float64 {
		g := quiet(7)
		for _, o := range g.cols[1:] {
			o.Dead = true
		}
		g.cols = g.cols[:1]
		c := g.cols[0]
		c.Weapon = weapon
		tree := place(g, OTree, 3, 3)
		g.applyTool(TChop, tree.x, tree.y, tree.x, tree.y)
		for i := 0; i < 60*60; i++ {
			g.step(1.0 / 60)
			if g.W.At(tree.x, tree.y).Obj != OTree {
				return float64(i) / 60
			}
		}
		return 60
	}
	if with, without := chopTime(IAxe), chopTime(INone); with >= without-1 {
		t.Fatalf("an axe should chop faster: %.1fs vs %.1fs", with, without)
	}
}

func TestHuntingWithARifle(t *testing.T) {
	g := quiet(7)
	g.addToStock(IAmmo, 30)
	for _, c := range g.cols {
		c.Weapon = IRifle
	}
	// a deer 9 tiles away in the open
	for dx := 0; dx < 14; dx++ {
		for dy := -2; dy <= 2; dy++ {
			p := place(g, ONone, -4+dx, -10+dy)
			g.W.At(p.x, p.y).Obj = ONone
		}
	}
	d := &Animal{Kind: ADeer, X: float64(g.start.x+5) + 0.5, Y: float64(g.start.y-10) + 0.5, HP: 25, MaxHP: 25, Hunt: true}
	g.animals = []*Animal{d}
	run(g, 40)
	if !d.Dead {
		for _, a := range g.animals {
			if a == d && a.Dead {
				return
			}
		}
		t.Fatal("rifles should bring a deer down")
	}
	if g.stats.Shots == 0 {
		t.Fatal("it should have been shot, not clubbed")
	}
}

func TestWeaponRecipesAndSkillGates(t *testing.T) {
	g := quiet(7)
	idx := map[string]int{}
	for i, r := range recipes {
		idx[r.Name] = i
	}
	for _, n := range []string{"Knife", "Axe", "Sword", "Pistol", "Rifle", "Ammo"} {
		if _, ok := idx[n]; !ok {
			t.Fatalf("missing recipe %s", n)
		}
	}
	if len(g.prod) != len(recipes) || len(defaultProd()) != len(recipes) {
		t.Fatalf("production settings (%d) must cover every recipe (%d)", len(g.prod), len(recipes))
	}
	// higher manufacturing skills for better weapons
	if recipes[idx["Sword"]].MinLvl <= recipes[idx["Knife"]].MinLvl || recipes[idx["Rifle"]].MinLvl <= recipes[idx["Pistol"]].MinLvl {
		t.Fatal("better weapons need more skill")
	}
	if recipes[idx["Pistol"]].Bench != OFab || recipes[idx["Rifle"]].Bench != OFab || recipes[idx["Knife"]].Bench != OBench {
		t.Fatal("guns come from the fabricator, blades from the workbench")
	}
	// a knife gets made at a workbench from iron and wood
	place(g, OBench, -3, 2)
	g.addToStock(IIron, 3)
	g.prod[idx["Knife"]].On = true
	run(g, 120)
	g.recompute()
	made := g.stock[IKnife]
	for _, c := range g.cols {
		if c.Weapon == IKnife {
			made++
		}
	}
	if made < 1 {
		t.Fatalf("a knife should have been forged (status %q)", g.recipeStatus(idx["Knife"]))
	}
	// guns: need engineering, power, components
	g2 := quiet(7)
	place(g2, OFab, -3, 2)
	place(g2, OGen, 4, 4)
	g2.addToStock(IIron, 4)
	g2.addToStock(IComponent, 4)
	g2.prod[idx["Pistol"]].On = true
	for _, c := range g2.cols {
		c.Skills[SEngineering].Level = 2
	}
	if s := g2.recipeStatus(idx["Pistol"]); !contains(s, "Engineering 5") {
		t.Fatalf("an unskilled crew can't make pistols: %q", s)
	}
	g2.cols[2].Skills[SEngineering].Level = 5
	run(g2, 150)
	g2.recompute()
	pistols := g2.stock[IPistol]
	for _, c := range g2.cols {
		if c.Weapon == IPistol {
			pistols++
		}
	}
	// (no ammo, so nobody picks it up: it stays in stock)
	if pistols < 1 {
		t.Fatalf("a pistol should be made (status %q)", g2.recipeStatus(idx["Pistol"]))
	}
	if itemValue[IRifle] <= itemValue[IPistol] || itemValue[ISword] <= itemValue[IKnife] {
		t.Fatal("better weapons are worth more")
	}
}

func TestWeaponSurvivesSave(t *testing.T) {
	g := quiet(7)
	g.cols[0].Weapon = ISword
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if h.cols[0].Weapon != ISword {
		t.Fatalf("weapon lost: %v", h.cols[0].Weapon)
	}
}

func TestTradersSometimesSellWeapons(t *testing.T) {
	blade, gun, ammo := 0, 0, 0
	for seed := int64(1); seed <= 200; seed++ {
		g := quiet(seed)
		g.nextTrader = g.time
		g.spawnTrader()
		tr := g.trader
		if tr == nil {
			continue
		}
		if tr.Stock[IKnife]+tr.Stock[IAxe]+tr.Stock[ISword] > 0 {
			blade++
		}
		if tr.Stock[IPistol]+tr.Stock[IRifle] > 0 {
			gun++
		}
		if tr.Stock[IAmmo] > 0 {
			ammo++
		}
	}
	if blade < 30 || gun < 5 || gun > blade || ammo < 30 {
		t.Fatalf("trader weapon stock off: blades %d guns %d ammo %d of 200", blade, gun, ammo)
	}
}

func TestTradeWindowFitsAndScrolls(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.nextTrader = g.time
	g.spawnTrader()
	g.trader.Arrived = true
	for it := IWood; it < numItems; it++ {
		if it != ISilver {
			g.trader.Stock[it] = 3
			g.stock[it] = 3 // (as if stockpiled)
		}
	}
	g.panel = panelTrade
	_, y, _, h := g.modalRect()
	if y+h > float64(g.sh)-g.barH() {
		t.Fatalf("the trade window (%.0f px) runs into the toolbar", h)
	}
	rows, vis := g.tradeRows(), g.tradeVisible()
	if rows <= vis {
		t.Skip("everything fits without scrolling")
	}
	if g.tradeScroll != 0 {
		t.Fatal("starts at the top")
	}
	count := func() (n int, lastY float64) {
		for _, b := range g.buttons() {
			if b.label == "Sell 1" || b.label == "Buy 1" {
				n++
				lastY = math.Max(lastY, b.y+b.h)
			}
		}
		return
	}
	n, lastY := count()
	if n != 2*vis || lastY > y+h {
		t.Fatalf("exactly the visible rows get buttons, inside the window: %d buttons (want %d), bottom %.0f of %.0f", n, 2*vis, lastY, y+h)
	}
	g.scrollTrade(5)
	if g.tradeScroll != 5 {
		t.Fatalf("scrolled %d", g.tradeScroll)
	}
	g.scrollTrade(1000)
	if g.tradeScroll != rows-vis {
		t.Fatalf("scroll should stop at the end: %d of max %d", g.tradeScroll, rows-vis)
	}
	g.scrollTrade(-1000)
	if g.tradeScroll != 0 {
		t.Fatal("and at the top")
	}
	// scrolled to the end, the last items are reachable
	g.scrollTrade(1000)
	found := false
	for _, b := range g.buttons() {
		if b.label == "Sell 1" {
			found = true
		}
	}
	if !found {
		t.Fatal("the last rows should have buttons")
	}
}

func TestColonistsPreferGunsToKnivesWhenThereIsAmmo(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	g.addToStock(IAmmo, 30)
	g.addToStock(IPistol, 1)
	g.addToStock(IRifle, 1)
	c.Weapon = IKnife
	if it, ok := g.bestWeaponFor(c); !ok || it != IRifle {
		t.Fatalf("a rifle with plenty of ammo beats a knife, got %v %v", it, ok)
	}
	c.Weapon = ISword
	if it, ok := g.bestWeaponFor(c); !ok || it != IRifle {
		t.Fatalf("a rifle beats a sword, got %v %v", it, ok)
	}
	g.takeStock(IRifle, 1)
	c.Weapon = IKnife
	if it, ok := g.bestWeaponFor(c); !ok || it != IPistol {
		t.Fatalf("a pistol beats a knife, got %v %v", it, ok)
	}
	c.Weapon = ISword
	if _, ok := g.bestWeaponFor(c); ok {
		t.Fatal("a sword is still better than a pistol")
	}
	// without ammo guns are not worth carrying
	g.takeStock(IAmmo, 30)
	c.Weapon = IKnife
	if _, ok := g.bestWeaponFor(c); ok {
		t.Fatal("no ammo: keep the knife")
	}
}
