package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	panelNone = iota
	panelProd
	panelTrade
	panelBuild
	panelMat
	panelCrop
	panelFloor
	panelSaves
)

type btn struct {
	x, y, w, h float64
	label      string
	fn         func()
	active     bool
	disabled   bool
}

func (b btn) hit(mx, my float64) bool {
	return mx >= b.x && mx <= b.x+b.w && my >= b.y && my <= b.y+b.h
}

// buildMenu lists what the Build palette offers, in order.
var buildMenu = []Blue{BHorse, BChecker, BShrine, BBench, BFurnace, BStove, BDesk, BLamp, BGen, BSolar, BFab, BEHeater}

// materials offered for walls and doors
var matMenu = []Material{MatWood, MatStone, MatIron}

func costText(c Cost) string {
	s := fmt.Sprintf("%d %s", c.N, itemNames[c.Item])
	if c.N2 > 0 {
		s += fmt.Sprintf(" + %d %s", c.N2, itemNames[c.Item2])
	}
	return s
}

func (g *Game) shift() bool {
	return ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
}

// ---------- geometry ----------

func (g *Game) modalRect() (x, y, w, h float64) {
	s := g.scale
	w = math.Min(700*s, float64(g.sw)-30*s)
	switch g.panel {
	case panelProd:
		w = math.Min(900*s, float64(g.sw)-30*s) // wide enough for recipe text, numbers and buttons
		h = (74 + float64(len(recipes))*30) * s
	case panelTrade:
		h = math.Min((96+float64(g.tradeRows())*24)*s, float64(g.sh)-g.barH()-70*s)
	case panelSaves:
		w = math.Min(640*s, float64(g.sw)-30*s)
		h = (76 + float64(numSlots+2)*34) * s
	default:
		return 0, 0, 0, 0
	}
	return (float64(g.sw) - w) / 2, 60 * s, w, h
}

func (g *Game) paletteRect() (x, y, w, h float64) {
	s := g.scale
	bx, _, _, _ := g.buttonRect(int(TBuild))
	w, h = 330*s, (28+float64(len(buildMenu))*34)*s
	x = math.Min(bx, float64(g.sw)-w-8*s)
	return x, float64(g.sh) - g.barH() - h - 4*s, w, h
}

// matPaletteRect is the wall/door material picker, above the toolbar button of the current tool.
func (g *Game) matPaletteRect() (x, y, w, h float64) {
	s := g.scale
	bx, _, _, _ := g.buttonRect(int(g.tool))
	w, h = 360*s, (28+float64(len(matMenu))*34)*s
	x = math.Min(bx, float64(g.sw)-w-8*s)
	return x, float64(g.sh) - g.barH() - h - 4*s, w, h
}

// floorPaletteRect is the floor picker shown with the Floor tool.
func (g *Game) floorPaletteRect() (x, y, w, h float64) {
	s := g.scale
	bx, _, _, _ := g.buttonRect(int(TFloor))
	w, h = 380*s, (28+float64(numFloors)*34)*s
	x = math.Min(bx, float64(g.sw)-w-8*s)
	return x, float64(g.sh) - g.barH() - h - 4*s, w, h
}

// cropPaletteRect is the crop picker shown with the Farm tool.
func (g *Game) cropPaletteRect() (x, y, w, h float64) {
	s := g.scale
	bx, _, _, _ := g.buttonRect(int(TFarm))
	w, h = 400*s, (28+float64(numCrops)*34)*s
	x = math.Min(bx, float64(g.sw)-w-8*s)
	return x, float64(g.sh) - g.barH() - h - 4*s, w, h
}

func (g *Game) matCost(m Material) (Cost, Blue) {
	if g.tool == TDoor {
		return costFor(BDoor, m), BDoor
	}
	return costFor(BWall, m), BWall
}

func (g *Game) curMat() Material {
	if g.tool == TDoor {
		return g.doorMat
	}
	return g.wallMat
}

// tradeSellItems and tradeBuyItems are the rows of the two trade lists. While the trade window is
// open a row never moves: an item that sells out stays in its place (greyed out, buttons off) and
// anything new is added at the bottom, so the list can't shift under the cursor between a glance
// and a click. The lists start afresh each time the window is opened.
func (g *Game) tradeSellItems() []Item {
	g.sellRows = mergeRows(g.sellRows, func(it Item) bool { return g.stock[it] > 0 })
	return g.sellRows
}

func (g *Game) tradeBuyItems() []Item {
	if g.trader == nil {
		return nil
	}
	g.buyRows = mergeRows(g.buyRows, func(it Item) bool { return g.trader.Stock[it] > 0 })
	return g.buyRows
}

// mergeRows keeps the rows already listed and appends items that have just become available.
func mergeRows(rows []Item, avail func(Item) bool) []Item {
	have := map[Item]bool{}
	for _, it := range rows {
		have[it] = true
	}
	for it := IWood; it < numItems; it++ {
		if it != ISilver && !have[it] && avail(it) {
			rows = append(rows, it)
		}
	}
	return rows
}

// tradeVisible is how many rows fit in the window; the lists scroll with the mouse wheel.
func (g *Game) tradeVisible() int {
	_, _, _, h := g.modalRect()
	return max(3, int((h-96*g.scale)/(24*g.scale))+0)
}

// scrollTrade moves the trade lists by delta rows (clamped).
func (g *Game) scrollTrade(delta int) {
	maxScroll := max(0, g.tradeRows()-g.tradeVisible())
	g.tradeScroll = max(0, min(maxScroll, g.tradeScroll+delta))
}

// tradeRows is how many rows the trade window needs: the longer of the two lists.
func (g *Game) tradeRows() int {
	return max(4, len(g.tradeSellItems()), len(g.tradeBuyItems()))
}

func (g *Game) selectedColonist() *Colonist {
	for _, c := range g.cols {
		if c.ID == g.selected {
			return c
		}
	}
	return nil
}

func (g *Game) colPanelRect(c *Colonist) (x, y, w, h float64) {
	s := g.scale
	w = 280 * s
	h = (352 + 17*float64(int(numSkills)-7) + float64(len(c.Injuries)+len(c.Ill))*16) * s
	return float64(g.sw) - w - 10*s, 54 * s, w, h
}

// overPanels reports whether the point is over a modal, the build palette or the colonist panel.
func (g *Game) overPanels(mx, my float64) bool {
	if bx, by, bw, bh, _, ok := g.raidBannerRect(); ok && mx >= bx && mx <= bx+bw && my >= by && my <= by+bh {
		return true
	}
	if bx, by, bw, bh, _, _, ok := g.bannerRect(); ok && mx >= bx && mx <= bx+bw && my >= by && my <= by+bh {
		return true
	}
	if g.panel == panelProd || g.panel == panelTrade || g.panel == panelSaves {
		x, y, w, h := g.modalRect()
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	if g.panel == panelBuild {
		x, y, w, h := g.paletteRect()
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	if g.panel == panelFloor {
		x, y, w, h := g.floorPaletteRect()
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	if g.panel == panelCrop {
		x, y, w, h := g.cropPaletteRect()
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	if g.panel == panelMat {
		x, y, w, h := g.matPaletteRect()
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	if c := g.selectedColonist(); c != nil {
		x, y, w, h := g.colPanelRect(c)
		if mx >= x && mx <= x+w && my >= y && my <= y+h {
			return true
		}
	}
	return false
}

// ---------- buttons ----------

func (g *Game) togglePanel(p int) {
	if g.panel == p {
		g.panel = panelNone
		return
	}
	if p == panelTrade && !g.traderHere() {
		if g.trader != nil {
			g.centerOnTrader()
			g.msg("The trader hasn't arrived yet - here they come.")
		} else {
			g.msg("No trader is here. Traders visit every few days.")
		}
		return
	}
	g.panel = p
	if p == panelSaves {
		g.saveArmed = 0
		g.refreshSlots()
	}
}

func (g *Game) centerOnTrader() {
	if t := g.trader; t != nil {
		g.camX, g.camY = t.X*TileSize, t.Y*TileSize
	}
}

// bannerRect is the clickable trader banner at the top centre.
func (g *Game) bannerRect() (x, y, w, h float64, msg string, urgent, ok bool) {
	msg, urgent, ok = g.traderStatus()
	if !ok || g.fontM == nil {
		return
	}
	s := g.scale
	tw, _ := text.Measure(msg, g.fontM, 0)
	w, h = tw+36*s, 32*s
	return (float64(g.sw) - w) / 2, g.topH + 6*s, w, h, msg, urgent, true
}

func (g *Game) buttons() []btn {
	s := g.scale
	var bs []btn
	if bx, by, bw, bh, _, ok := g.raidBannerRect(); ok {
		bs = append(bs, btn{x: bx, y: by, w: bw, h: bh, fn: func() {
			if a := g.raiderFocus(); a != nil {
				g.camX, g.camY = a.X*TileSize, a.Y*TileSize
			}
		}})
	}
	if bx, by, bw, bh, _, _, ok := g.bannerRect(); ok {
		bs = append(bs, btn{x: bx, y: by, w: bw, h: bh, fn: func() {
			if g.traderHere() {
				g.panel = panelTrade
			} else {
				g.centerOnTrader()
			}
		}})
	}
	if c := g.selectedColonist(); c != nil {
		px, py, pw, _ := g.colPanelRect(c)
		for sk := Skill(0); sk < numSkills; sk++ {
			sk := sk
			bs = append(bs, btn{x: px + 4*s, y: py + (212+float64(sk)*17)*s, w: pw - 8*s, h: 17 * s, fn: func() { c.Focus = sk }, active: c.Focus == sk})
		}
	}
	switch g.panel {
	case panelSaves:
		x, y, w, _ := g.modalRect()
		bs = append(bs, btn{x: x + w - 30*s, y: y + 6*s, w: 24 * s, h: 22 * s, label: "X", fn: func() { g.panel = panelNone }})
		for i, sl := range g.slots {
			i, sl := i, sl
			ry := y + (62+float64(i)*34)*s
			if !sl.Auto {
				label := "Save"
				if g.saveArmed == i+1 {
					label = "Sure?"
				}
				bs = append(bs, btn{x: x + w - 150*s, y: ry, w: 66 * s, h: 28 * s, label: label, active: g.saveArmed == i+1, fn: func() { g.saveSlot(i) }})
			}
			bs = append(bs, btn{x: x + w - 76*s, y: ry, w: 66 * s, h: 28 * s, label: "Load", disabled: !sl.Exists || sl.Bad, fn: func() { g.loadSlot(i) }})
		}
	case panelProd:
		x, y, w, _ := g.modalRect()
		bs = append(bs, btn{x: x + w - 30*s, y: y + 6*s, w: 24 * s, h: 22 * s, label: "X", fn: func() { g.panel = panelNone }})
		for i := range recipes {
			i := i
			ry := y + (62+float64(i)*30)*s
			bs = append(bs,
				btn{x: x + w - 70*s, y: ry, w: 60 * s, h: 28 * s, label: map[bool]string{true: "ON", false: "OFF"}[g.prod[i].On], active: g.prod[i].On,
					fn: func() { g.prod[i].On = !g.prod[i].On }},
				btn{x: x + w - 190*s, y: ry, w: 28 * s, h: 28 * s, label: "-", fn: func() {
					st := 1
					if g.shift() {
						st = 5
					}
					g.prod[i].Target = max(0, g.prod[i].Target-st)
				}},
				btn{x: x + w - 128*s, y: ry, w: 28 * s, h: 28 * s, label: "+", fn: func() {
					st := 1
					if g.shift() {
						st = 5
					}
					g.prod[i].Target = min(500, g.prod[i].Target+st)
				}})
		}
	case panelTrade:
		x, y, w, _ := g.modalRect()
		bs = append(bs, btn{x: x + w - 30*s, y: y + 6*s, w: 24 * s, h: 22 * s, label: "X", fn: func() { g.panel = panelNone }})
		colW := w/2 - 16*s
		vis := g.tradeVisible()
		for row, it := range g.tradeSellItems() {
			it := it
			row -= g.tradeScroll
			if row < 0 || row >= vis {
				continue
			}
			ry := y + (84+float64(row)*24)*s
			bs = append(bs,
				btn{x: x + colW - 110*s, y: ry, w: 50 * s, h: 22 * s, label: "Sell 1", disabled: g.stock[it] == 0, fn: func() { g.sellItem(it, 1) }},
				btn{x: x + colW - 56*s, y: ry, w: 56 * s, h: 22 * s, label: "Sell 10", disabled: g.stock[it] == 0, fn: func() { g.sellItem(it, 10) }})
		}
		for row, it := range g.tradeBuyItems() {
			it := it
			row -= g.tradeScroll
			if row < 0 || row >= vis {
				continue
			}
			ry := y + (84+float64(row)*24)*s
			bx := x + w/2 + 8*s
			bs = append(bs,
				btn{x: bx + colW - 108*s, y: ry, w: 48 * s, h: 22 * s, label: "Buy 1", disabled: g.trader.Stock[it] == 0, fn: func() { g.buyItem(it, 1) }},
				btn{x: bx + colW - 54*s, y: ry, w: 54 * s, h: 22 * s, label: "Buy 5", disabled: g.trader.Stock[it] == 0, fn: func() { g.buyItem(it, 5) }})
		}
	case panelFloor:
		x, y, w, _ := g.floorPaletteRect()
		for k := FloorKind(0); k < numFloors; k++ {
			k := k
			bs = append(bs, btn{x: x + 4*s, y: y + (24+float64(k)*34)*s, w: w - 8*s, h: 32 * s, active: g.floorSel == k,
				fn: func() { g.floorSel, g.panel = k, panelNone }})
		}
	case panelCrop:
		x, y, w, _ := g.cropPaletteRect()
		for k := CropKind(0); k < numCrops; k++ {
			k := k
			bs = append(bs, btn{x: x + 4*s, y: y + (24+float64(k)*34)*s, w: w - 8*s, h: 32 * s, active: g.cropSel == k,
				fn: func() { g.cropSel, g.panel = k, panelNone }})
		}
	case panelMat:
		x, y, w, _ := g.matPaletteRect()
		for i, m := range matMenu {
			m := m
			bs = append(bs, btn{x: x + 4*s, y: y + (24+float64(i)*34)*s, w: w - 8*s, h: 32 * s, active: g.curMat() == m,
				fn: func() {
					if g.tool == TDoor {
						g.doorMat = m
					} else {
						g.wallMat = m
					}
					g.panel = panelNone
				}})
		}
	case panelBuild:
		x, y, w, _ := g.paletteRect()
		for i, b := range buildMenu {
			b := b
			bs = append(bs, btn{x: x + 4*s, y: y + (24+float64(i)*34)*s, w: w - 8*s, h: 32 * s, active: g.buildSel == b,
				fn: func() { g.buildSel, g.tool, g.panel = b, TBuild, panelNone }})
		}
	}
	return bs
}

// handleUIClick processes a left click; returns true if the UI consumed it.
func (g *Game) handleUIClick(mx, my float64) bool {
	for _, b := range g.buttons() {
		if b.hit(mx, my) && !b.disabled {
			b.fn()
			g.audio.play("click", 0.4)
			return true
		}
	}
	return g.overPanels(mx, my)
}

// ---------- drawing ----------

func (g *Game) drawBtn(dst *ebiten.Image, b btn, mx, my float64) {
	c := color.NRGBA{50, 56, 72, 255}
	if b.hit(mx, my) {
		c = color.NRGBA{70, 80, 104, 255}
	}
	if b.active {
		c = color.NRGBA{60, 120, 90, 255}
	}
	if b.disabled {
		c = color.NRGBA{34, 37, 46, 255}
	}
	rect(dst, b.x, b.y, b.w, b.h, c)
	strokeRect(dst, b.x, b.y, b.w, b.h, 1, color.NRGBA{120, 130, 150, 255})
	if b.label != "" {
		w, _ := text.Measure(b.label, g.fontS, 0)
		g.text(dst, b.label, b.x+(b.w-w)/2, b.y+(b.h-float64(g.fontS.Size))/2-1, g.fontS, color.White)
	}
}

func (g *Game) drawColonistPanel(dst *ebiten.Image, c *Colonist, mx, my float64) {
	s := g.scale
	panel := color.NRGBA{16, 18, 24, 215}
	px, py, pw, ph := g.colPanelRect(c)
	rect(dst, px, py, pw, ph, panel)
	g.text(dst, c.Name, px+10*s, py+6*s, g.fontM, c.Col)
	g.text(dst, c.traitList(), px+10*s, py+28*s, g.fontS, color.NRGBA{190, 200, 230, 255})
	g.text(dst, c.Status, px+10*s, py+46*s, g.fontS, color.White)
	bars := []struct {
		n string
		v float64
		c color.NRGBA
	}{{"Food", c.Hunger, nrgba(220, 170, 60)}, {"Health", c.Health, nrgba(200, 60, 60)}, {"Rest", c.Rest, nrgba(110, 140, 230)}, {"Joy", c.Joy, nrgba(240, 130, 200)}}
	for i, b := range bars {
		y := py + (68+float64(i)*20)*s
		g.text(dst, b.n, px+10*s, y-2*s, g.fontS, color.White)
		g.bar(dst, px+62*s, y+1*s, 200*s, 10*s, b.v/100, b.c)
	}
	tc := color.NRGBA{255, 255, 255, 255}
	if c.Temp < 3 {
		tc = nrgba(120, 190, 255)
	} else if c.Temp > 33 {
		tc = nrgba(255, 150, 90)
	}
	g.text(dst, fmt.Sprintf("Temp %.0f°C", c.Temp), px+10*s, py+148*s, g.fontS, tc)
	if c.Carry != INone {
		g.text(dst, fmt.Sprintf("Carrying %d %s", c.CarryN, itemNames[c.Carry]), px+110*s, py+148*s, g.fontS, color.White)
	}
	weapon := "bare hands"
	if wd, ok := weaponDefs[c.Weapon]; ok {
		weapon = wd.Name
	}
	g.text(dst, "Weapon: "+weapon, px+10*s, py+164*s, g.fontS, color.NRGBA{255, 220, 170, 255})
	gear := func(it Item) string {
		if gd, ok := gearDefs[it]; ok {
			return gd.Name
		}
		return "-"
	}
	g.text(dst, "Tool: "+gear(c.Tool)+"   Wearing: "+gear(c.Wear), px+10*s, py+180*s, g.fontS, color.NRGBA{200, 215, 235, 255})
	g.text(dst, "Skills  (click to train at a desk)", px+10*s, py+196*s, g.fontS, color.NRGBA{170, 180, 200, 255})
	for sk := Skill(0); sk < numSkills; sk++ {
		st := c.Skills[sk]
		y := py + (212+float64(sk)*17)*s
		if c.Focus == sk {
			rect(dst, px+4*s, y, pw-8*s, 17*s, color.NRGBA{60, 120, 90, 120})
		}
		name := skillNames[sk]
		if st.Passion {
			vector.FillCircle(dst, float32(px+14*s), float32(y+8.5*s), float32(4*s), nrgba(255, 150, 60), true)
		}
		g.text(dst, name, px+24*s, y+1*s, g.fontS, color.White)
		g.text(dst, fmt.Sprintf("%2d", st.Level), px+124*s, y+1*s, g.fontS, color.NRGBA{255, 235, 150, 255})
		frac := 1.0
		if st.Level < maxSkill {
			frac = st.XP / xpNeeded(st.Level)
		}
		g.bar(dst, px+152*s, y+4*s, 118*s, 8*s, frac, nrgba(120, 200, 150))
	}
	iy := py + (336+17*float64(int(numSkills)-7))*s
	for i, in := range c.Injuries {
		st := [...]string{"untended", "bandaged", "treated"}[in.Tended]
		col := color.NRGBA{255, 120, 110, 255}
		if in.Tended > 0 {
			col = color.NRGBA{170, 230, 170, 255}
		}
		line := fmt.Sprintf("%s (%s)", in.Kind, st)
		if in.Infect >= 0.5 {
			line += " INFECTED"
			col = color.NRGBA{230, 220, 70, 255}
		}
		g.text(dst, line, px+10*s, iy+float64(i)*16*s, g.fontS, col)
	}
	for i, ill := range c.Ill {
		col := color.NRGBA{150, 230, 130, 255}
		if ill.Sev >= critSev {
			col = color.NRGBA{255, 120, 110, 255}
		}
		g.text(dst, sickLine(ill), px+10*s, iy+float64(len(c.Injuries)+i)*16*s, g.fontS, col)
	}
}

func (g *Game) drawPanels(dst *ebiten.Image) {
	s := g.scale
	mxi, myi := ebiten.CursorPosition()
	mx, my := float64(mxi), float64(myi)
	bg := color.NRGBA{12, 14, 20, 238}
	white := color.Color(color.White)
	dim := color.NRGBA{170, 178, 196, 255}

	if c := g.selectedColonist(); c != nil {
		g.drawColonistPanel(dst, c, mx, my)
	}

	switch g.panel {
	case panelSaves:
		x, y, w, h := g.modalRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Save slots", x+12*s, y+8*s, g.fontM, white)
		g.text(dst, "Save to a slot, or load one. Saving over a used slot asks you to click twice. (F5 / F9 quick save and load.)", x+12*s, y+34*s, g.fontS, dim)
		for i, sl := range g.slots {
			ry := y + (62+float64(i)*34)*s
			g.text(dst, sl.Name, x+14*s, ry+4*s, g.fontS, white)
			c := dim
			if !sl.Exists || sl.Bad {
				c = color.NRGBA{130, 136, 150, 255}
			}
			g.text(dst, slotText(sl), x+150*s, ry+4*s, g.fontS, c)
		}
	case panelProd:
		x, y, w, h := g.modalRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Production", x+12*s, y+8*s, g.fontM, white)
		g.text(dst, "Idle colonists make these at benches until stock reaches the target. Shift = step of 5.", x+12*s, y+34*s, g.fontS, dim)
		for i, r := range recipes {
			ry := y + (62+float64(i)*30)*s
			in := fmt.Sprintf("%d %s", r.N1, itemNames[r.In1])
			if r.N2 > 0 {
				in += fmt.Sprintf(" + %d %s", r.N2, itemNames[r.In2])
			}
			g.blit(dst, art.item[r.Out], x+10*s, ry-2*s, 26*s, false, 1)
			g.text(dst, r.Name, x+46*s, ry-2*s, g.fontM, white)
			g.text(dst, g.recipeLine(r, in, prodTextWidth(w, s)), x+46*s, ry+16*s, g.fontS, dim)
			st := g.recipeStatus(i)
			sc := dim
			if st == "working" {
				sc = color.NRGBA{140, 230, 160, 255}
			} else if st != "off" && st != "target reached" {
				sc = color.NRGBA{255, 170, 120, 255}
			}
			g.text(dst, fmt.Sprintf("have %d / target %d", g.stock[r.Out], g.prod[i].Target), x+w-prodStatsX*s, ry-2*s, g.fontS, white)
			g.text(dst, g.fit(st, (prodStatsX-200)*s, g.fontS), x+w-prodStatsX*s, ry+16*s, g.fontS, sc)
		}
	case panelTrade:
		x, y, w, h := g.modalRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Trade", x+12*s, y+8*s, g.fontM, white)
		if t := g.trader; t != nil {
			g.text(dst, fmt.Sprintf("Your silver: %d      Trader's silver: %d      Leaves in %ds", g.stock[ISilver], t.Cash, int(math.Max(0, t.Stay))), x+12*s, y+34*s, g.fontS, color.NRGBA{255, 235, 150, 255})
		}
		g.text(dst, "Sell (from your stockpile)", x+12*s, y+60*s, g.fontS, dim)
		g.text(dst, "Buy", x+w/2+8*s, y+60*s, g.fontS, dim)
		vis := g.tradeVisible()
		sells := g.tradeSellItems()
		for row, it := range sells {
			row -= g.tradeScroll
			if row < 0 || row >= vis {
				continue
			}
			ry := y + (84+float64(row)*24)*s
			g.blit(dst, art.item[it], x+8*s, ry-5*s, 24*s, false, 1)
			c := white
			if g.stock[it] == 0 {
				c = color.Color(color.NRGBA{120, 126, 140, 255})
			}
			g.text(dst, fmt.Sprintf("%s x%d  %d each", itemNames[it], g.stock[it], g.sellPrice(it)), x+38*s, ry+2*s, g.fontS, c)
		}
		if len(sells) == 0 {
			g.text(dst, "(nothing to sell)", x+12*s, y+90*s, g.fontS, dim)
		}
		for row, it := range g.tradeBuyItems() {
			row -= g.tradeScroll
			if row < 0 || row >= vis {
				continue
			}
			ry := y + (84+float64(row)*24)*s
			bx := x + w/2 + 8*s
			g.blit(dst, art.item[it], bx, ry-5*s, 24*s, false, 1)
			c := white
			if g.trader.Stock[it] == 0 {
				c = color.Color(color.NRGBA{120, 126, 140, 255})
			}
			g.text(dst, fmt.Sprintf("%s x%d  %d each", itemNames[it], g.trader.Stock[it], g.buyPrice(it)), bx+30*s, ry+2*s, g.fontS, c)
		}
		if g.tradeRows() > vis {
			g.text(dst, fmt.Sprintf("Scroll the mouse wheel for more (%d-%d of %d)", g.tradeScroll+1, min(g.tradeScroll+vis, g.tradeRows()), g.tradeRows()), x+12*s, y+h-20*s, g.fontS, color.NRGBA{255, 235, 150, 255})
		}
	case panelFloor:
		x, y, w, h := g.floorPaletteRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Floor for new tiles (laying over an old floor replaces it)", x+8*s, y+4*s, g.fontS, dim)
		for k := FloorKind(0); k < numFloors; k++ {
			ry := y + (24+float64(k)*34)*s
			cost := floorCost(k)
			if mx >= x+4*s && mx <= x+w-4*s && my >= ry && my <= ry+32*s {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{70, 80, 104, 255})
			}
			if g.floorSel == k {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{60, 120, 90, 160})
			}
			g.blit(dst, art.floor[k][0], x+8*s, ry, 30*s, false, 1)
			g.text(dst, floorNames[k], x+46*s, ry+1*s, g.fontS, white)
			cc := dim
			note := fmt.Sprintf("%s  (+%.0f%% speed, %s)", costText(cost), (floorSpeed[k]-1)*100, floorNotes[k])
			if g.stock[cost.Item] < cost.N {
				cc = color.NRGBA{255, 150, 120, 255}
				if cost.Item == ITile {
					note = costText(cost) + "  (fire Tiles from stone at a Furnace)"
				}
			}
			g.text(dst, note, x+46*s, ry+16*s, g.fontS, cc)
		}
	case panelCrop:
		x, y, w, h := g.cropPaletteRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Crop for new farm tiles", x+8*s, y+4*s, g.fontS, dim)
		for k := CropKind(0); k < numCrops; k++ {
			ry := y + (24+float64(k)*34)*s
			def := cropDefs[k]
			if mx >= x+4*s && mx <= x+w-4*s && my >= ry && my <= ry+32*s {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{70, 80, 104, 255})
			}
			if g.cropSel == k {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{60, 120, 90, 160})
			}
			g.blit(dst, art.crop[k][3], x+8*s, ry, 30*s, false, 1)
			g.text(dst, def.Name, x+46*s, ry+1*s, g.fontS, white)
			cc := dim
			if g.outT < def.MinT {
				cc = color.NRGBA{255, 190, 140, 255} // too cold to sprout outdoors right now
			}
			g.text(dst, fmt.Sprintf("%d %s in %.0f s, needs %.0f°C+: %s", def.N, itemNames[def.Out], def.Time, def.MinT, def.Note), x+46*s, ry+16*s, g.fontS, cc)
		}
	case panelMat:
		x, y, w, h := g.matPaletteRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		what := "wall"
		if g.tool == TDoor {
			what = "door"
		}
		g.text(dst, "Material for new "+what+"s", x+8*s, y+4*s, g.fontS, dim)
		for i, m := range matMenu {
			ry := y + (24+float64(i)*34)*s
			cost, b := g.matCost(m)
			if mx >= x+4*s && mx <= x+w-4*s && my >= ry && my <= ry+32*s {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{70, 80, 104, 255})
			}
			if g.curMat() == m {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{60, 120, 90, 160})
			}
			g.blit(dst, g.blueSpriteMat(b, m), x+8*s, ry, 30*s, false, 1)
			g.text(dst, blueLabel(b, m), x+46*s, ry+1*s, g.fontS, white)
			line := costText(cost)
			cc := dim
			switch {
			case g.stock[cost.Item] < cost.N:
				cc = color.NRGBA{255, 150, 120, 255}
				if m == MatIron {
					line += "  (smelt Ore at a Furnace)"
				}
			case m == MatWood:
				line += "  (can burn!)"
				cc = color.NRGBA{255, 200, 130, 255}
			default:
				line += "  (fireproof)"
			}
			g.text(dst, line, x+46*s, ry+16*s, g.fontS, cc)
		}
	case panelBuild:
		x, y, w, h := g.paletteRect()
		rect(dst, x, y, w, h, bg)
		strokeRect(dst, x, y, w, h, 2, color.NRGBA{100, 110, 140, 255})
		g.text(dst, "Build", x+8*s, y+4*s, g.fontS, dim)
		for i, b := range buildMenu {
			ry := y + (24+float64(i)*34)*s
			cost := costs[b]
			if mx >= x+4*s && mx <= x+w-4*s && my >= ry && my <= ry+32*s {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{70, 80, 104, 255})
			}
			if g.buildSel == b {
				rect(dst, x+4*s, ry, w-8*s, 32*s, color.NRGBA{60, 120, 90, 160})
			}
			g.blit(dst, g.blueSprite(b), x+8*s, ry, 30*s, false, 1)
			g.text(dst, blueNames[b], x+46*s, ry+1*s, g.fontS, white)
			line := costText(cost)
			if cost.MinEng > 0 {
				line += fmt.Sprintf("  (Engineering %d)", cost.MinEng)
			}
			ok := g.canAfford(cost)
			cc := dim
			if !ok {
				cc = color.NRGBA{255, 150, 120, 255}
			}
			g.text(dst, line, x+46*s, ry+16*s, g.fontS, cc)
		}
	}
	for _, b := range g.buttons() {
		if b.label != "" {
			g.drawBtn(dst, b, mx, my)
		}
	}
}

// drawTraderBanner shows a pulsing banner while a trader is around, plus an arrow at the
// screen edge pointing to them when they are out of view.
func (g *Game) drawTraderBanner(dst *ebiten.Image) {
	t := g.trader
	bx, by, bw, bh, msg, urgent, ok := g.bannerRect()
	if !ok {
		return
	}
	pulse := 0.5 + 0.5*math.Sin(g.time*4)
	bg := color.NRGBA{40, 64, 110, 235}
	edge := color.NRGBA{120, 170, 255, 255}
	if t.Arrived {
		bg = color.NRGBA{uint8(110 + 50*pulse), uint8(86 + 40*pulse), 20, 240}
		edge = color.NRGBA{255, 220, 110, 255}
		if urgent {
			bg = color.NRGBA{uint8(130 + 90*pulse), 40, 30, 240}
			edge = color.NRGBA{255, 150, 130, 255}
		}
	}
	rect(dst, bx, by, bw, bh, bg)
	strokeRect(dst, bx, by, bw, bh, 2, edge)
	tw, _ := text.Measure(msg, g.fontM, 0)
	g.text(dst, msg, bx+(bw-tw)/2, by+(bh-float64(g.fontM.Size))/2-1, g.fontM, color.White)

	g.drawEdgeArrow(dst, t.X, t.Y, "Trader", edge, pulse)
}

// drawEdgeArrow points to something off screen with an arrow at the screen edge and a label
// giving the distance. Nothing is drawn when the target is visible.
func (g *Game) drawEdgeArrow(dst *ebiten.Image, wx, wy float64, name string, edge color.NRGBA, pulse float64) {
	s := g.scale
	sx, sy := g.toScreen(wx, wy)
	viewH := float64(g.sh) - g.barH()
	if sx >= 0 && sx <= float64(g.sw) && sy >= 0 && sy <= viewH {
		return
	}
	cx, cy := float64(g.sw)/2, viewH/2
	dx, dy := sx-cx, sy-cy
	m := 34 * s
	k := math.Min((cx-m)/math.Max(math.Abs(dx), 1e-6), (cy-m)/math.Max(math.Abs(dy), 1e-6))
	px, py := cx+dx*k, cy+dy*k
	ang := math.Atan2(dy, dx)
	r := (16 + 3*pulse) * s
	var tri [][2]float64
	for _, a := range []float64{0, 2.5, -2.5} {
		rr := r
		if a == 0 {
			rr = r * 1.5
		}
		tri = append(tri, [2]float64{px + math.Cos(ang+a)*rr, py + math.Sin(ang+a)*rr})
	}
	g.fillPoly(dst, tri, color.NRGBA{20, 16, 8, 230})
	var inner [][2]float64
	for _, p := range tri {
		inner = append(inner, [2]float64{px + (p[0]-px)*0.7, py + (p[1]-py)*0.7})
	}
	g.fillPoly(dst, inner, edge)
	dist := int(math.Hypot(wx-g.camX/TileSize, wy-g.camY/TileSize))
	lbl := fmt.Sprintf("%s %d", name, dist)
	lw, _ := text.Measure(lbl, g.fontS, 0)
	lx := math.Max(4*s, math.Min(px-lw/2, float64(g.sw)-lw-4*s))
	ly := math.Max(4*s, math.Min(py+r*1.6, viewH-18*s))
	g.text(dst, lbl, lx, ly, g.fontS, edge)
}

// raidBannerRect is the raid warning banner, stacked under the trader banner if there is one.
func (g *Game) raidBannerRect() (x, y, w, h float64, msg string, ok bool) {
	msg, ok = g.raidStatus()
	if !ok || g.fontM == nil {
		return
	}
	s := g.scale
	tw, _ := text.Measure(msg, g.fontM, 0)
	w, h = tw+36*s, 32*s
	y = g.topH + 6*s
	if _, _, _, _, _, _, tok := g.bannerRect(); tok {
		y += 38 * s
	}
	return (float64(g.sw) - w) / 2, y, w, h, msg, true
}

// drawRaidBanner flashes a red warning while raiders are about, and points at them.
func (g *Game) drawRaidBanner(dst *ebiten.Image) {
	bx, by, bw, bh, msg, ok := g.raidBannerRect()
	if !ok {
		return
	}
	pulse := 0.5 + 0.5*math.Sin(g.time*6)
	rect(dst, bx, by, bw, bh, color.NRGBA{uint8(120 + 100*pulse), 24, 24, 245})
	strokeRect(dst, bx, by, bw, bh, 2, color.NRGBA{255, 160, 140, 255})
	tw, _ := text.Measure(msg, g.fontM, 0)
	g.text(dst, msg, bx+(bw-tw)/2, by+(bh-float64(g.fontM.Size))/2-1, g.fontM, color.White)
	if a := g.raiderFocus(); a != nil {
		g.drawEdgeArrow(dst, a.X, a.Y, "Raiders", color.NRGBA{255, 110, 90, 255}, pulse)
	}
}

// rightMouse handles the right button. Dragging pans the camera; a plain click (no drag) cancels:
// a tool drag in progress, else an open panel, else it goes back to the Select tool.
// down/pressed/released are the button state this frame; g.lastMX/lastMY hold last frame's cursor.
func (g *Game) rightMouse(mx, my float64, down, pressed, released bool) {
	if pressed {
		g.rpX, g.rpY = mx, my
		g.rpActive, g.rpPan = true, false
		g.rpCanPan = !g.overPanels(mx, my)
		if g.dragging { // cancels the drag; nothing else should happen for this press
			g.dragging = false
			g.rpActive = false
		}
		if g.moveSrc >= 0 { // a picked-up building: right-click lets go of it
			g.moveSrc = -1
			g.rpActive = false
		}
	}
	if down && g.rpActive && g.rpCanPan {
		if !g.rpPan && math.Hypot(mx-g.rpX, my-g.rpY) > 6*g.scale {
			g.rpPan = true
		}
		if g.rpPan {
			g.camX -= (mx - g.lastMX) / g.zoom
			g.camY -= (my - g.lastMY) / g.zoom
		}
	}
	if released {
		if g.rpActive && !g.rpPan {
			if g.panel != panelNone {
				g.panel = panelNone
			} else {
				g.tool = TSelect
			}
		}
		g.rpActive, g.rpPan = false, false
	}
}

// prodStatsX is how far from the panel's right edge the have/target column starts.
const prodStatsX = 440

// prodTextWidth is the room for a recipe's description: from the icon to the have/target column.
func prodTextWidth(w, s float64) float64 { return w - prodStatsX*s - 56*s }

// fit shortens text with an ellipsis until it fits in maxW pixels.
func (g *Game) fit(s string, maxW float64, f *text.GoTextFace) string {
	for len(s) > 1 {
		if w, _ := text.Measure(s, f, 0); w <= maxW {
			return s
		}
		r := []rune(s)
		s = string(r[:len(r)-2]) + "…"
		if len(r) <= 3 {
			break
		}
	}
	return s
}

// recipeLine describes what a recipe consumes and makes, shortened to fit the room available.
func (g *Game) recipeLine(r Recipe, in string, maxW float64) string {
	full := fmt.Sprintf("%s -> %d   (%s %d)", in, r.OutN, skillNames[r.Skill], r.MinLvl)
	if w, _ := text.Measure(full, g.fontS, 0); w <= maxW {
		return full
	}
	short := fmt.Sprintf("%s (%s %d)", in, skillNames[r.Skill][:3], r.MinLvl)
	return g.fit(short, maxW, g.fontS)
}
