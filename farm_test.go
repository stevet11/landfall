package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// A panel that has buttons but no drawing code is invisible: make sure every panel kind has both.
func TestEveryPanelHasButtonsAndDrawing(t *testing.T) {
	src, err := os.ReadFile("ui2.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"panelProd", "panelTrade", "panelBuild", "panelMat", "panelCrop", "panelFloor", "panelSaves"} {
		if n := strings.Count(string(src), "case "+p+":"); n < 2 {
			t.Errorf("%s needs a case in both buttons() and drawPanels(), found %d", p, n)
		}
	}
}

// farmTile makes one warm, plantable farm tile of a crop and returns it.
func farmTile(g *Game, kind CropKind, dx, dy int) pt {
	p := place(g, ONone, dx, dy)
	t := g.W.At(p.x, p.y)
	t.Farm, t.CropKind, t.Ground = true, kind, GGrass
	return p
}

func TestEveryCropYieldsItsOwnProduce(t *testing.T) {
	for kind := CropKind(0); kind < numCrops; kind++ {
		g := quiet(7)
		p := farmTile(g, kind, 6, 6)
		for _, c := range g.cols {
			c.Skills[SFarming].Level = 0
			c.Traits = nil
		}
		tl := g.W.At(p.x, p.y)
		tl.Crop, tl.Grow = true, 1 // fully grown, waiting to be reaped
		def := cropDefs[kind]
		before := totalItems(g)
		run(g, 80)
		got := totalItems(g)[def.Out] - before[def.Out]
		if got < def.N {
			t.Errorf("%s: expected at least %d %s, got %d", def.Name, def.N, itemNames[def.Out], got)
		}
		if tl.Crop && tl.Grow >= 1 {
			t.Errorf("%s: harvested but the ripe crop is still standing", def.Name)
		}
	}
}

func TestCropsNeedTheirOwnWarmth(t *testing.T) {
	grown := func(kind CropKind, temp float64) float32 {
		g := quiet(7)
		p := farmTile(g, kind, 6, 6)
		tl := g.W.At(p.x, p.y)
		tl.Crop, tl.Grow = true, 0
		for i := 0; i < 400; i++ { // 100 s
			g.outT = temp
			g.slow(0.25)
		}
		return tl.Grow
	}
	// wheat sprouts at 7C, corn does not; at 14C both grow
	if grown(CropWheat, 7) <= 0 {
		t.Fatal("wheat should grow at 7C")
	}
	if grown(CropCorn, 7) != 0 {
		t.Fatal("corn should not grow at 7C")
	}
	if grown(CropCorn, 14) <= 0 || grown(CropCotton, 14) <= 0 {
		t.Fatal("corn and cotton grow at 14C")
	}
	if grown(CropCotton, 11) != 0 || grown(CropTobacco, 11) != 0 {
		t.Fatal("cotton and tobacco need 12C")
	}
	// slower crops take longer
	if grown(CropAloe, 20) >= grown(CropWheat, 20) {
		t.Fatal("aloe takes longer than wheat")
	}
	// frost kills everything
	g := quiet(7)
	p := farmTile(g, CropCorn, 6, 6)
	g.W.At(p.x, p.y).Crop = true
	g.outT = -5
	g.slow(0.25)
	if g.W.At(p.x, p.y).Crop {
		t.Fatal("frost should kill crops")
	}
}

func TestSowingWaitsForTheRightTemperature(t *testing.T) {
	g := quiet(7)
	p := farmTile(g, CropCorn, 6, 6)
	g.outT = 9 // too cold for corn (needs 10), fine for vegetables
	g.slow(0.25)
	for i := 0; i < 60*20; i++ {
		g.outT = 9
		g.step(1.0 / 60)
		g.outT = 9
	}
	if g.W.At(p.x, p.y).Crop {
		t.Fatal("nobody should sow corn at 9C")
	}
	g2 := quiet(7)
	q := farmTile(g2, CropVeg, 6, 6)
	run(g2, 20)
	if !g2.W.At(q.x, q.y).Crop {
		t.Fatal("vegetables are sown in mild weather")
	}
}

func TestFarmToolPlantsTheSelectedCrop(t *testing.T) {
	g := quiet(7)
	g.cropSel = CropWheat
	a := place(g, ONone, 6, 6)
	g.applyTool(TFarm, a.x, a.y, a.x, a.y)
	if tl := g.W.At(a.x, a.y); !tl.Farm || tl.CropKind != CropWheat {
		t.Fatalf("expected wheat: %+v", tl.CropKind)
	}
	// choosing another crop re-plants an empty tile ...
	g.cropSel = CropCorn
	g.applyTool(TFarm, a.x, a.y, a.x, a.y)
	if g.W.At(a.x, a.y).CropKind != CropCorn {
		t.Fatal("an empty farm tile can change crop")
	}
	// ... but never rips out a growing crop
	g.W.At(a.x, a.y).Crop = true
	g.cropSel = CropAloe
	g.applyTool(TFarm, a.x, a.y, a.x, a.y)
	if g.W.At(a.x, a.y).CropKind != CropCorn {
		t.Fatal("a growing crop must stay")
	}
}

func TestAloeGrowsMedicineAndSkillHelps(t *testing.T) {
	yield := func(farming int) int {
		g := quiet(7)
		g.outT = 20
		p := farmTile(g, CropAloe, 6, 6)
		for _, c := range g.cols {
			c.Skills[SFarming].Level = farming
			c.Traits = nil
		}
		tl := g.W.At(p.x, p.y)
		tl.Crop, tl.Grow = true, 1
		before := totalItems(g)[IMedicine]
		run(g, 60)
		return totalItems(g)[IMedicine] - before
	}
	if lo := yield(0); lo < 1 {
		t.Fatalf("aloe should grow medicine: %d", lo)
	}
	if yield(6) <= yield(0) {
		t.Fatal("skilled farmers get more from aloe")
	}
}

func TestLegacyFarmsStillGrowFood(t *testing.T) {
	g := quiet(7)
	p := place(g, ONone, 6, 6)
	g.W.At(p.x, p.y).Farm = true // an old save: no crop kind recorded
	if g.W.At(p.x, p.y).CropKind != CropVeg || cropDefs[CropVeg].Out != IFood {
		t.Fatal("old farms are vegetable farms that make food")
	}
}

func TestHerbsRegrowNearTheColonyEvenWhenFarOnesRemain(t *testing.T) {
	g := quiet(7)
	g.outT = 15
	// plenty of herbs far away, none near
	for i := range g.W.T {
		if g.W.T[i].Obj == OHerb {
			g.W.T[i].Obj = ONone
		}
	}
	for dx := 0; dx < 15; dx++ {
		p := pt{5 + dx*2, 5}
		g.W.At(p.x, p.y).Obj, g.W.At(p.x, p.y).Ground = OHerb, GGrass
	}
	for i := 0; i < 60*600; i++ {
		g.step(1.0 / 60)
	}
	near := 0
	for i := range g.W.T {
		if g.W.T[i].Obj == OHerb && iabs(i%MapW-g.start.x) <= 40 && iabs(i/MapW-g.start.y) <= 30 {
			near++
		}
	}
	if near == 0 {
		t.Fatal("herbs should regrow near the colony even if distant ones exist")
	}
}

// ---------- cooking ----------

func recipeIndex(name string) int {
	for i, r := range recipes {
		if r.Name == name {
			return i
		}
	}
	return -1
}

func TestStoveBakesBreadWithFuel(t *testing.T) {
	g := quiet(7)
	place(g, OStove, -3, 2)
	g.addToStock(IWheat, 6)
	i := recipeIndex("Bread")
	g.prod[i].On, g.prod[i].Target = true, 6
	wood0 := g.stock[IWood]
	run(g, 100)
	g.recompute()
	if g.stock[IBread] < 3 {
		t.Fatalf("expected bread: %d (status %q)", g.stock[IBread], g.recipeStatus(i))
	}
	if g.stock[IWood] >= wood0 {
		t.Fatal("the stove burns wood")
	}
	if g.stock[IWheat] > 4 {
		t.Fatal("wheat should be consumed")
	}
	if recipes[i].Skill != SCooking {
		t.Fatal("bread uses the cooking skill")
	}
}

func TestMealsNeedCornAndFoodAndSkill(t *testing.T) {
	g := quiet(7)
	place(g, OStove, -3, 2)
	g.addToStock(ICorn, 4)
	i := recipeIndex("Meal")
	g.prod[i].On, g.prod[i].Target = true, 6
	for _, c := range g.cols {
		c.Skills[SCooking].Level = 0
	}
	if s := g.recipeStatus(i); !contains(s, "Cooking 3") {
		t.Fatalf("an unskilled crew can't make meals: %q", s)
	}
	g.cols[1].Skills[SCooking].Level = 3
	run(g, 120)
	g.recompute()
	if g.stock[IMeal] < 2 {
		t.Fatalf("expected meals: %d (status %q)", g.stock[IMeal], g.recipeStatus(i))
	}
	if g.stock[ICorn] >= 4 {
		t.Fatal("corn should be used up")
	}
}

func TestCookingTrainsTheSkill(t *testing.T) {
	g := quiet(7)
	place(g, OStove, -3, 2)
	g.addToStock(IWheat, 20)
	g.prod[recipeIndex("Bread")].On, g.prod[recipeIndex("Bread")].Target = true, 30
	xp := func() (x float64) {
		for _, c := range g.cols {
			x += float64(c.lvl(SCooking))*1000 + c.Skills[SCooking].XP
		}
		return
	}
	x0 := xp()
	run(g, 100)
	if xp() <= x0 {
		t.Fatal("cooking should train the Cooking skill")
	}
}

func TestStovesCanStartFires(t *testing.T) {
	withFireRates(1, 0, func() {
		g := quiet(7)
		g.noFireFight = true
		s := place(g, OStove, 6, 6)
		w := place(g, OWall, 7, 6)
		g.W.At(w.x, w.y).Mat = MatWood
		g.addToStock(IWheat, 6)
		i := recipeIndex("Bread")
		g.prod[i].On, g.prod[i].Target = true, 9
		// a colonist is cooking at this very stove
		c := g.cols[0]
		c.Job = Job{Kind: JCraft, Recipe: i, Target: s}
		if !g.burnerActive(s.y*MapW + s.x) {
			t.Fatal("a stove in use is a burner")
		}
		g.updateFire(1)
		if g.W.At(w.x, w.y).Fire == 0 {
			t.Fatal("a cooking stove next to a wooden wall should be able to start a fire")
		}
	})
}

func TestColonistsEatBreadMealsAndCorn(t *testing.T) {
	for _, tc := range []struct {
		it   Item
		gain float64
	}{{IFood, 45}, {IBread, 40}, {IMeal, 65}, {ICorn, 20}} {
		g := quiet(7)
		for _, o := range g.cols[1:] {
			o.Dead = true
		}
		g.cols = g.cols[:1]
		// only the item under test is edible
		for _, i := range g.zoneList {
			if g.W.T[i].Item == IFood {
				g.W.T[i].Item, g.W.T[i].ItemN = INone, 0
			}
		}
		g.recompute()
		g.addToStock(tc.it, 3)
		c := g.cols[0]
		c.Hunger = 30
		h0 := c.Hunger
		peak := h0
		for i := 0; i < 60*25; i++ {
			g.step(1.0 / 60)
			peak = max(peak, c.Hunger)
		}
		if peak < h0+tc.gain-15 {
			t.Errorf("%s: hunger rose to %.0f from %.0f, expected about +%.0f", itemNames[tc.it], peak, h0, tc.gain)
		}
		g.recompute()
		if g.stock[tc.it] >= 3 {
			t.Errorf("%s was not eaten", itemNames[tc.it])
		}
	}
	if nutrition(IMeal) <= nutrition(IFood) || nutrition(ICorn) >= nutrition(IFood) || nutrition(IWood) != 0 {
		t.Fatal("meals are best, raw corn is poor, wood is not food")
	}
}

func TestImmigrationCountsAllFood(t *testing.T) {
	g := quiet(7)
	for _, i := range g.zoneList {
		g.W.T[i].Item, g.W.T[i].ItemN = INone, 0
	}
	g.recompute()
	g.addToStock(IMeal, 12)
	if g.foodUnits() != 12 {
		t.Fatalf("meals are food: %d", g.foodUnits())
	}
}

func TestPerishableFoodSpoilsAtDifferentRates(t *testing.T) {
	if !(spoilPerDay[IMeal] > spoilPerDay[IFood] && spoilPerDay[IBread] < spoilPerDay[IFood] && spoilPerDay[IWheat] < spoilPerDay[IBread]) {
		t.Fatal("meals rot fastest, then raw food, then bread, then grain")
	}
	if spoilPerDay[ICloth] != 0 || spoilPerDay[ITobacco] != 0 || spoilPerDay[ICotton] != 0 {
		t.Fatal("cloth, cotton and tobacco keep")
	}
}

// ---------- tools and clothing ----------

func TestPickaxeSpeedsMiningAndYieldsMore(t *testing.T) {
	mine := func(tool Item) (secs float64, stone int) {
		g := quiet(7)
		for _, o := range g.cols[1:] {
			o.Dead = true
		}
		g.cols = g.cols[:1]
		c := g.cols[0]
		c.Tool = tool
		c.Skills[SMining].Level = 0
		p := place(g, ORock, 4, 4)
		g.applyTool(TMine, p.x, p.y, p.x, p.y)
		before := totalItems(g)[IStone]
		for i := 0; i < 60*60; i++ {
			g.step(1.0 / 60)
			if g.W.At(p.x, p.y).Obj == ONone {
				run(g, 5)
				return float64(i) / 60, totalItems(g)[IStone] - before
			}
		}
		return 60, 0
	}
	slow, s0 := mine(INone)
	fast, s1 := mine(IPickaxe)
	if fast >= slow-1 {
		t.Fatalf("a pickaxe should mine faster: %.1fs vs %.1fs", fast, slow)
	}
	if s1 != s0+1 {
		t.Fatalf("a pickaxe yields one more stone: %d vs %d", s1, s0)
	}
}

func TestPickaxesGoToMiners(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols {
		c.Skills[SMining].Level = 0
	}
	g.addToStock(IPickaxe, 1)
	run(g, 40)
	for _, c := range g.cols {
		if c.Tool == IPickaxe {
			t.Fatal("nobody is mining: leave the pickaxe in the stockpile")
		}
	}
	p := place(g, ORock, 8, 8)
	g.applyTool(TMine, p.x, p.y, p.x, p.y)
	run(g, 40)
	n := 0
	for _, c := range g.cols {
		if c.Tool == IPickaxe {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("with mining orders exactly one colonist should take the single pickaxe, %d did", n)
	}
}

func TestParkasKeepColonistsWarm(t *testing.T) {
	plain, parka := &Colonist{}, &Colonist{Wear: IParka}
	hardyParka := &Colonist{Wear: IParka, Traits: []Trait{TrHardy}}
	if parka.coldMult() >= plain.coldMult() || hardyParka.coldMult() >= parka.coldMult() {
		t.Fatalf("parkas cut cold damage and stack with Hardy: %v %v %v", plain.coldMult(), parka.coldMult(), hardyParka.coldMult())
	}
	g := quiet(7)
	for _, c := range g.cols {
		c.Wear = INone
	}
	g.addToStock(IParka, 2)
	run(g, 60)
	worn := 0
	for _, c := range g.cols {
		if c.Wear == IParka {
			worn++
		}
	}
	if worn != 2 {
		t.Fatalf("both parkas should be worn, %d are", worn)
	}
}

func TestClothAndParkaRecipes(t *testing.T) {
	g := quiet(7)
	place(g, OBench, -3, 2)
	g.addToStock(ICotton, 8)
	g.prod[recipeIndex("Cloth")].On, g.prod[recipeIndex("Cloth")].Target = true, 4
	g.prod[recipeIndex("Parka")].On, g.prod[recipeIndex("Parka")].Target = true, 1
	for _, c := range g.cols {
		c.Skills[SCrafting].Level = 4
		c.Wear = INone // so a finished parka stays visible in the stockpile
	}
	for _, c := range g.cols {
		c.Skills[SCrafting].Level = 4
	}
	run(g, 200)
	total := totalItems(g)
	worn := 0
	for _, c := range g.cols {
		if c.Wear == IParka {
			worn++
		}
	}
	if total[ICloth] < 1 && worn == 0 {
		t.Fatalf("cotton should become cloth and parkas: cloth %d, parkas worn %d", total[ICloth], worn)
	}
	if total[IParka]+worn < 1 {
		t.Fatalf("a parka should have been sewn (status %q)", g.recipeStatus(recipeIndex("Parka")))
	}
}

func TestGearAndNewSkillSurviveSaves(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Tool, c.Wear = IPickaxe, IParka
	c.Skills[SCooking] = SkillState{Level: 6, XP: 10, Passion: true}
	g.cropSel = CropAloe
	p := farmTile(g, CropCotton, 6, 6)
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	d := h.cols[0]
	if d.Tool != IPickaxe || d.Wear != IParka || d.Skills[SCooking] != (SkillState{Level: 6, XP: 10, Passion: true}) {
		t.Fatalf("gear or cooking lost: %v %v %+v", d.Tool, d.Wear, d.Skills[SCooking])
	}
	if h.cropSel != CropAloe || h.W.At(p.x, p.y).CropKind != CropCotton {
		t.Fatal("crop choices lost")
	}
}

func TestVersion4SavesStillLoadWithACookingSkill(t *testing.T) {
	g := quiet(7)
	g.cols[0].Skills[SMedicine].Level = 9
	sd := g.snapshot()
	sd.Version = 4
	for i := range sd.Cols {
		sd.Cols[i].SkillList = nil // version 4 only knew seven skills
	}
	h := quiet(3)
	h.apply(sd)
	if h.cols[0].lvl(SMedicine) != 9 {
		t.Fatal("old skills must survive")
	}
	if len(h.cols) != len(g.cols) {
		t.Fatal("colonists lost")
	}
	// and a real gob round trip of the legacy layout still decodes
	path := t.TempDir() + "/v5.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if err := quiet(5).LoadFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestNewItemsHaveValuesIconsAndRecipes(t *testing.T) {
	for it := Item(1); it < numItems; it++ {
		if itemNames[it] == "" {
			t.Errorf("item %d has no name", it)
		}
		if it != ISilver && itemValue[it] <= 0 {
			t.Errorf("%s has no value", itemNames[it])
		}
	}
	for _, r := range recipes {
		if r.Bench == OStove && r.Skill != SCooking {
			t.Errorf("%s is cooking", r.Name)
		}
		if itemValue[r.Out]*float64(r.OutN) <= 0 {
			t.Errorf("%s makes something worthless", r.Name)
		}
	}
	if len(defaultProd()) != len(recipes) {
		t.Fatal("production defaults must cover every recipe")
	}
}

func TestProductionPanelFitsAllRecipes(t *testing.T) {
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.panel = panelProd
	_, y, _, h := g.modalRect()
	if y+h > float64(g.sh)-g.barH() {
		t.Fatalf("the production window (%.0f px for %d recipes) runs into the toolbar", h, len(recipes))
	}
}

func TestProductionTextNeverRunsIntoTheNumbers(t *testing.T) {
	for _, width := range []int{1280, 1024, 800} {
		g := quiet(7)
		g.scale, g.sw, g.sh = 1, width, 760
		g.panel = panelProd
		x, _, w, _ := g.modalRect()
		if x < 0 || x+w > float64(width) {
			t.Fatalf("width %d: the production window is off screen (%.0f..%.0f)", width, x, x+w)
		}
		room := prodTextWidth(w, g.scale)
		for _, r := range recipes {
			in := fmt.Sprintf("%d %s", r.N1, itemNames[r.In1])
			if r.N2 > 0 {
				in += fmt.Sprintf(" + %d %s", r.N2, itemNames[r.In2])
			}
			line := g.recipeLine(r, in, room)
			if lw, _ := text.Measure(line, g.fontS, 0); lw > room+0.5 {
				t.Errorf("width %d: %q is %.0f px wide but there are only %.0f", width, line, lw, room)
			}
		}
		for _, st := range []string{"nobody has Engineering 8", "needs a fabricator", "not enough power", "missing materials"} {
			fitted := g.fit(st, (prodStatsX-200)*g.scale, g.fontS)
			if lw, _ := text.Measure(fitted, g.fontS, 0); lw > (prodStatsX-200)*g.scale+0.5 {
				t.Errorf("status %q does not fit: %.0f px", st, lw)
			}
		}
	}
	// the stats column must not overlap the +/- buttons
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	g.panel = panelProd
	x, _, w, _ := g.modalRect()
	for _, b := range g.buttons() {
		if b.label == "-" && b.x < x+w-prodStatsX+200 {
			t.Fatalf("the status text (ends at %.0f) would run into the '-' button (%.0f)", x+w-prodStatsX+200, b.x)
		}
	}
}
