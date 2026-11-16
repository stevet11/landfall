package main

import "testing"

// urgentSetup: a bench with a carving order, wood for roofs and a free tile to roof.
func urgentSetup() (*Game, pt) {
	g := quiet(7)
	place(g, OBench, -3, 2)
	g.addToStock(IStone, 40)
	g.addToStock(IWood, 60)
	i := recipeIndex("Carving")
	g.prod[i].On, g.prod[i].Target = true, 200
	tile := place(g, ONone, 3, 3)
	g.W.At(tile.x, tile.y).Desig = DRoof
	g.recompute()
	return g, tile
}

func idle(g *Game) *Colonist {
	for _, c := range g.cols {
		c.Job = Job{}
		c.Path = nil
	}
	return g.cols[0]
}

func TestOrdinaryRoofingWaitsForCrafting(t *testing.T) {
	g, _ := urgentSetup()
	c := idle(g)
	if !g.assignJob(c) || c.Job.Kind != JCraft {
		t.Fatalf("with recipes to make, colonists craft before roofing: job %v", c.Job.Kind)
	}
}

func TestUrgentRoofIsDoneBeforeCrafting(t *testing.T) {
	g, tile := urgentSetup()
	g.applyTool(TPriority, tile.x, tile.y, tile.x, tile.y)
	if !g.W.At(tile.x, tile.y).Urgent {
		t.Fatal("the Priority tool should mark the roof order")
	}
	c := idle(g)
	if !g.assignJob(c) || c.Job.Kind != JRoof {
		t.Fatalf("an urgent roof comes first: job %v", c.Job.Kind)
	}
}

func TestUrgentRecipeIsMadeBeforeBuilding(t *testing.T) {
	g := quiet(7)
	place(g, OBench, -3, 2)
	g.addToStock(IStone, 40)
	g.addToStock(IWood, 60)
	i := recipeIndex("Carving")
	g.prod[i].On, g.prod[i].Target = true, 200
	bp := place(g, ONone, 3, 3)
	g.W.At(bp.x, bp.y).Blue = BBed
	g.recompute()
	c := idle(g)
	if !g.assignJob(c) || c.Job.Kind != JBuild {
		t.Fatalf("by default building comes before crafting: %v", c.Job.Kind)
	}
	g.prod[i].Urgent = true
	g.abort(c)
	c = idle(g)
	if !g.assignJob(c) || c.Job.Kind != JCraft {
		t.Fatalf("an urgent recipe beats building: %v", c.Job.Kind)
	}
}

func TestUrgentOrdersDoNotOverrideSurvival(t *testing.T) {
	g, tile := urgentSetup()
	g.applyTool(TPriority, tile.x, tile.y, tile.x, tile.y)
	c := idle(g)
	c.Hunger = 10
	if !g.assignJob(c) || c.Job.Kind != JEat {
		t.Fatalf("hungry colonists still eat first: %v", c.Job.Kind)
	}
}

func TestPriorityToolTogglesAndTidiesUp(t *testing.T) {
	g := quiet(7)
	a, b := place(g, ONone, 3, 3), place(g, ONone, 4, 3)
	g.W.At(a.x, a.y).Blue = BBed
	g.W.At(b.x, b.y).Desig = DRoof
	g.applyTool(TPriority, a.x, a.y, b.x, b.y)
	if !g.W.At(a.x, a.y).Urgent || !g.W.At(b.x, b.y).Urgent {
		t.Fatal("both orders become urgent")
	}
	g.applyTool(TPriority, a.x, a.y, b.x, b.y)
	if g.W.At(a.x, a.y).Urgent || g.W.At(b.x, b.y).Urgent {
		t.Fatal("dragging again over urgent orders clears the mark")
	}
	g.applyTool(TPriority, a.x, a.y, b.x, b.y)
	g.applyTool(TErase, a.x, a.y, a.x, a.y)
	if g.W.At(a.x, a.y).Urgent || g.W.At(a.x, a.y).Blue != BNone {
		t.Fatal("erasing removes the mark with the order")
	}
	g.W.At(b.x, b.y).Desig = DNone // the roof gets finished
	g.slow(1)
	if g.W.At(b.x, b.y).Urgent {
		t.Fatal("a finished order leaves no stale mark")
	}
	n := place(g, ONone, 8, 8)
	g.applyTool(TPriority, n.x, n.y, n.x, n.y) // nothing there
	if g.W.At(n.x, n.y).Urgent {
		t.Fatal("empty ground cannot be urgent")
	}
}

func TestUrgentSettingsAreSaved(t *testing.T) {
	g := quiet(7)
	i := recipeIndex("Parka")
	g.prod[i].Urgent = true
	p := place(g, ONone, 3, 3)
	g.W.At(p.x, p.y).Desig = DChop
	g.W.At(p.x, p.y).Obj = OTree
	g.W.At(p.x, p.y).Urgent = true
	path := t.TempDir() + "/u.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	if !h.prod[i].Urgent || !h.W.At(p.x, p.y).Urgent {
		t.Fatal("urgent marks should survive a save")
	}
}

func TestProductionPanelHasUrgentButtons(t *testing.T) {
	g := quiet(7)
	g.sw, g.sh, g.scale = 1280, 760, 1
	g.panel = panelProd
	n := 0
	for _, b := range g.buttons() {
		if b.label == "!" {
			n++
		}
	}
	if n != len(recipes) {
		t.Fatalf("expected %d urgent buttons, got %d", len(recipes), n)
	}
	if int(numTools) != len(toolInfo) || toolInfo[TPriority].Name != "Priority" {
		t.Fatal("tool table out of step with the Tool enum")
	}
}
