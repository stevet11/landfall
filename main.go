package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type ui struct {
	camX, camY, zoom float64
	tool             Tool
	dragging         bool
	dragStart        pt
	selected         int
	paused           bool
	speed            int
	showHelp         bool
	sw, sh           int
	scale            float64
	fontS, fontM     *text.GoTextFace
	srcR, srcB       *text.GoTextFaceSource
	lastFrame        time.Time
	lastMX, lastMY   float64
	autoT            float64
	audio            *Audio
	audioCh          chan *Audio
	panel            int
	slots            []SaveSlot // what the Saves panel lists
	saveArmed        int        // slot (1-based) waiting for a second click to overwrite
	tradeScroll      int
	sellRows         []Item // trade rows as listed since the window opened (see tradeSellItems)
	buyRows          []Item
	nightLayer       *ebiten.Image
	rpX, rpY         float64 // where the right button went down
	rpActive         bool    // a right-button gesture is in progress
	rpPan            bool    // ...and it has become a camera drag
	rpCanPan         bool
	topH             float64
	parts            []particle
	op               ebiten.DrawImageOptions
}

var toolInfo = [numTools]struct {
	Name string
	Key  ebiten.Key
	Lbl  string
	Col  color.NRGBA
}{
	{"Select", ebiten.Key1, "1", color.NRGBA{200, 200, 210, 255}},
	{"Chop", ebiten.Key2, "2", color.NRGBA{220, 70, 60, 255}},
	{"Mine", ebiten.Key3, "3", color.NRGBA{235, 150, 50, 255}},
	{"Harvest", ebiten.Key4, "4", color.NRGBA{235, 215, 60, 255}},
	{"Hunt", ebiten.Key5, "5", color.NRGBA{200, 60, 100, 255}},
	{"Stock", ebiten.Key6, "6", color.NRGBA{70, 130, 235, 255}},
	{"Farm", ebiten.Key7, "7", color.NRGBA{120, 200, 80, 255}},
	{"Wall", ebiten.Key8, "8", color.NRGBA{150, 150, 160, 255}},
	{"Door", ebiten.Key9, "9", color.NRGBA{170, 120, 70, 255}},
	{"Floor", ebiten.Key0, "0", color.NRGBA{200, 160, 100, 255}},
	{"Bed", ebiten.KeyMinus, "-", color.NRGBA{160, 140, 230, 255}},
	{"Heater", ebiten.KeyEqual, "=", color.NRGBA{255, 130, 40, 255}},
	{"Build", ebiten.KeyB, "B", color.NRGBA{90, 200, 230, 255}},
	{"Move", ebiten.KeyV, "V", color.NRGBA{120, 220, 200, 255}},
	{"Roof", ebiten.KeyR, "R", color.NRGBA{200, 140, 90, 255}},
	{"Salvage", ebiten.KeyX, "X", color.NRGBA{230, 110, 90, 255}},
	{"Erase", ebiten.KeyBackspace, "", color.NRGBA{220, 80, 200, 255}},
	{"Priority", ebiten.KeyU, "U", color.NRGBA{255, 90, 60, 255}},
}

func (g *Game) initUI() {
	g.camX, g.camY = (float64(g.start.x)+0.5)*TileSize, (float64(g.start.y)+0.5)*TileSize
	g.zoom, g.speed, g.showHelp, g.scale = 1.5, 1, true, 1
	var err error
	if g.srcR, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		log.Fatal(err)
	}
	if g.srcB, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		log.Fatal(err)
	}
	g.makeFonts()
	g.onEvent = g.handleEvent
}

func (g *Game) makeFonts() {
	g.fontS = &text.GoTextFace{Source: g.srcR, Size: 13 * g.scale}
	g.fontM = &text.GoTextFace{Source: g.srcB, Size: 16 * g.scale}
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

func (g *Game) LayoutF(w, h float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s < 1 {
		s = 1
	}
	if s != g.scale {
		g.scale = s
		g.makeFonts()
	}
	g.sw, g.sh = int(w*s), int(h*s)
	return float64(g.sw), float64(g.sh)
}

// toolRows is how many rows the toolbar needs at the current window width.
func (g *Game) toolRows() int {
	per := int((float64(g.sw) - 16*g.scale) / (86 * g.scale))
	if per < 1 {
		per = 1
	}
	return (int(numTools) + per - 1) / per
}

func (g *Game) barH() float64 { return (float64(g.toolRows())*50 + 8) * g.scale }

func (g *Game) toTile(mx, my float64) (int, int) {
	wx := g.camX + (mx-float64(g.sw)/2)/g.zoom
	wy := g.camY + (my-float64(g.sh)/2)/g.zoom
	return int(math.Floor(wx / TileSize)), int(math.Floor(wy / TileSize))
}

func (g *Game) toScreen(tx, ty float64) (float64, float64) {
	return (tx*TileSize-g.camX)*g.zoom + float64(g.sw)/2, (ty*TileSize-g.camY)*g.zoom + float64(g.sh)/2
}

func (g *Game) buttonRect(i int) (x, y, w, h float64) {
	s := g.scale
	rows := g.toolRows()
	perRow := (int(numTools) + rows - 1) / rows
	row, col := i/perRow, i%perRow
	pitch := math.Min(96*s, (float64(g.sw)-16*s)/float64(perRow))
	return 8*s + float64(col)*pitch, float64(g.sh) - g.barH() + (8+float64(row)*50)*s, pitch - 4*s, 42 * s
}

// selectTool switches tools; the Build tool opens its palette, others close open panels.
func (g *Game) selectTool(t Tool) {
	g.tool, g.dragging, g.moveSrc = t, false, -1
	switch {
	case t == TBuild:
		g.panel = panelBuild
	case t == TWall || t == TDoor:
		g.panel = panelMat
	case t == TFarm:
		g.panel = panelCrop
	case t == TFloor:
		g.panel = panelFloor
	case g.panel == panelBuild || g.panel == panelMat || g.panel == panelCrop || g.panel == panelFloor:
		g.panel = panelNone
	}
}

func clampf(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func (g *Game) Update() error {
	if art == nil {
		initArt()
		g.audioCh = make(chan *Audio, 1)
		go func() { g.audioCh <- initAudio() }() // synthesis takes ~0.5s; don't stall the first frame
	}
	if g.audioCh != nil {
		select {
		case a := <-g.audioCh:
			g.audio, g.audioCh = a, nil
		default:
		}
	}
	now := time.Now()
	frameDT := 1.0 / 60
	if !g.lastFrame.IsZero() {
		frameDT = clampf(now.Sub(g.lastFrame).Seconds(), 0.001, 0.1)
	}
	g.lastFrame = now

	// UI messages fade in real time
	for i := range g.msgs {
		g.msgs[i].T -= frameDT
	}
	for len(g.msgs) > 0 && g.msgs[0].T <= 0 {
		g.msgs = g.msgs[1:]
	}

	if g.over {
		g.updateGameOver()
		return nil
	}

	if g.panel != panelTrade {
		g.sellRows, g.buyRows = nil, nil // the next visit lists afresh
	}

	// keys
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.quickSave()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		g.quickLoad(quickSavePath(), "quicksave")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.quickLoad(autoSavePath(), "autosave")
	}
	if g.autoT += frameDT; g.autoT > 120 {
		g.autoT = 0
		if !g.paused {
			g.autoSave()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) && g.audio != nil {
		g.audio.muted = !g.audio.muted
		g.msg("Sound %s.", map[bool]string{true: "muted", false: "on"}[g.audio.muted])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.showHelp = !g.showHelp
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.speed = g.speed%3 + 1
	}
	for i := 0; i < int(numTools); i++ {
		if inpututil.IsKeyJustPressed(toolInfo[i].Key) {
			g.selectTool(Tool(i))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.togglePanel(panelProd)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.togglePanel(panelTrade)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		g.togglePanel(panelSaves)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.panel != panelNone {
			g.panel = panelNone
		} else {
			g.tool, g.dragging, g.selected = TSelect, false, 0
		}
	}

	// camera
	pan := 900 * frameDT / g.zoom
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) {
		pan *= 2.2
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.camY -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.camY += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.camX -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.camX += pan
	}
	mxi, myi := ebiten.CursorPosition()
	mx, my := float64(mxi), float64(myi)
	g.rightMouse(mx, my, ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight), inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight))
	g.lastMX, g.lastMY = mx, my
	if _, wy := ebiten.Wheel(); wy != 0 && g.panel == panelTrade && g.overPanels(mx, my) {
		g.scrollTrade(-int(math.Round(wy)))
	} else if wy != 0 && !g.overPanels(mx, my) {
		bx, by := g.camX+(mx-float64(g.sw)/2)/g.zoom, g.camY+(my-float64(g.sh)/2)/g.zoom
		g.zoom = clampf(g.zoom*math.Pow(1.12, wy), 0.4, 4)
		g.camX = bx - (mx-float64(g.sw)/2)/g.zoom
		g.camY = by - (my-float64(g.sh)/2)/g.zoom
	}
	g.camX = clampf(g.camX, 0, MapW*TileSize)
	g.camY = clampf(g.camY, 0, MapH*TileSize)

	// mouse
	overBar := my > float64(g.sh)-g.barH()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.handleUIClick(mx, my) {
			// a panel button or panel background took the click
		} else if overBar {
			for i := 0; i < int(numTools); i++ {
				bx, by, bw, bh := g.buttonRect(i)
				if mx >= bx && mx <= bx+bw && my >= by && my <= by+bh {
					g.selectTool(Tool(i))
					g.audio.play("click", 0.5)
				}
			}
		} else if g.tool == TMove {
			tx, ty := g.toTile(mx, my)
			g.moveClick(tx, ty)
		} else if g.tool == TSelect {
			g.selected = 0
			best := 0.8
			wx, wy := g.camX/TileSize+(mx-float64(g.sw)/2)/g.zoom/TileSize, g.camY/TileSize+(my-float64(g.sh)/2)/g.zoom/TileSize
			for _, c := range g.cols {
				if d := math.Hypot(c.X-wx, c.Y-wy); d < best {
					best, g.selected = d, c.ID
				}
			}
		} else {
			tx, ty := g.toTile(mx, my)
			g.dragging, g.dragStart = true, pt{tx, ty}
			if g.panel == panelBuild || g.panel == panelMat || g.panel == panelCrop || g.panel == panelFloor {
				g.panel = panelNone // get the palette out of the way while dragging
			}
		}
	}
	if g.dragging && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		tx, ty := g.toTile(mx, my)
		g.applyTool(g.tool, g.dragStart.x, g.dragStart.y, tx, ty)
		g.dragging = false
	}

	g.audio.update(g, frameDT)
	g.updateParticles(frameDT)

	// simulate
	if !g.paused {
		n := g.speed
		for i := 0; i < n; i++ {
			g.step(1.0 / 60)
		}
	}
	return nil
}

// ---- drawing ----

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

func nrgba(r, g, b uint8) color.NRGBA { return color.NRGBA{r, g, b, 255} }

func shadeCol(c color.NRGBA, d int) color.NRGBA {
	f := func(v uint8) uint8 { return uint8(max(0, min(255, int(v)+d))) }
	return color.NRGBA{f(c.R), f(c.G), f(c.B), c.A}
}

var groundCols = [...]color.NRGBA{{86, 140, 70, 255}, {128, 104, 76, 255}, {206, 190, 130, 255}, {50, 98, 160, 255}}
var itemCols = [numItems]color.NRGBA{{}, {160, 110, 60, 255}, {150, 150, 160, 255}, {220, 70, 90, 255}, {120, 220, 170, 255}, {210, 215, 230, 255}, {176, 122, 70, 255}, {190, 192, 202, 255}, {40, 130, 84, 255}, {90, 200, 230, 255}, {170, 50, 60, 255}, {214, 130, 60, 255}, {150, 164, 186, 255}, {190, 198, 212, 255}, {180, 150, 120, 255}, {200, 205, 220, 255}, {90, 96, 110, 255}, {80, 86, 100, 255}, {220, 170, 70, 255}, {170, 170, 180, 255}, {120, 150, 190, 255}, {226, 190, 90, 255}, {230, 200, 80, 255}, {236, 236, 240, 255}, {150, 120, 60, 255}, {200, 215, 235, 255}, {210, 160, 90, 255}, {200, 120, 70, 255}, {220, 226, 236, 255}}

func rect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}
func circle(dst *ebiten.Image, x, y, r float64, c color.Color) {
	vector.FillCircle(dst, float32(x), float32(y), float32(r), c, true)
}
func strokeRect(dst *ebiten.Image, x, y, w, h, sw float64, c color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(sw), c, false)
}

func (g *Game) text(dst *ebiten.Image, s string, x, y float64, f *text.GoTextFace, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x+1, y+1)
	op.ColorScale.ScaleWithColor(color.NRGBA{0, 0, 0, 200})
	text.Draw(dst, s, f, op)
	op = &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

func (g *Game) hour() float64 { return math.Mod(g.time/DayLen, 1) * 24 }

func (g *Game) bar(dst *ebiten.Image, x, y, w, h, frac float64, c color.NRGBA) {
	rect(dst, x, y, w, h, color.NRGBA{0, 0, 0, 160})
	rect(dst, x, y, w*clampf(frac, 0, 1), h, c)
}

func (g *Game) drawUI(dst *ebiten.Image) {
	s := g.scale
	// top-left resource panel
	panel := color.NRGBA{16, 18, 24, 205}
	// lay the entries out by measured width so big numbers never run into the next label
	type entry struct {
		label string
		item  Item
	}
	var entries []entry
	for it := IWood; it < numItems; it++ {
		if it >= IMedicine+1 && g.stock[it] == 0 { // crafted goods only show once you have some
			continue
		}
		entries = append(entries, entry{fmt.Sprintf("%s %d", itemNames[it], g.stock[it]), it})
	}
	entries = append(entries, entry{fmt.Sprintf("Colonists %d", len(g.cols)), INone})
	if g.hasPower() {
		entries = append(entries, entry{fmt.Sprintf("Power %.0f/%.0fW", g.powerSupply, g.powerDemand), INone})
	}
	// flow the entries into rows that stop short of the clock on the right
	gap := 26 * s
	clockW, _ := text.Measure(fmt.Sprintf("%s  Day %d  00:00  -00°C  x3", g.seasonName(), int(g.time/DayLen)+1), g.fontM, 0)
	maxX := float64(g.sw) - clockW - 50*s
	type placed struct {
		x, y, w float64
		e       entry
	}
	var rowsPlaced []placed
	rowEnd := []float64{}
	x, row := 18*s, 0
	rowEnd = append(rowEnd, 0)
	for _, e := range entries {
		w, _ := text.Measure(e.label, g.fontM, 0)
		need := w + gap
		if e.item != INone {
			need += 17 * s
		}
		if x+need > maxX && x > 18*s {
			x, row = 18*s, row+1
			rowEnd = append(rowEnd, 0)
		}
		rowsPlaced = append(rowsPlaced, placed{x, float64(row), w, e})
		x += need
		rowEnd[row] = x
	}
	for r, end := range rowEnd {
		rect(dst, 8*s, (8+float64(r)*30)*s, end-8*s-gap/2, 28*s, panel)
	}
	for _, p := range rowsPlaced {
		y := (15 + p.y*30) * s
		x := p.x
		if p.e.item != INone {
			rect(dst, x, y+3*s, 12*s, 12*s, itemCols[p.e.item])
			x += 17 * s
		}
		g.text(dst, p.e.label, x, y-2*s, g.fontM, color.White)
	}
	topRows := float64(len(rowEnd))
	g.topH = (8 + topRows*30) * s
	wolves := 0
	for _, a := range g.animals {
		if a.Kind == AWolf {
			wolves++
		}
	}
	sick := 0
	for _, c := range g.cols {
		if c.sick() {
			sick++
		}
	}
	ly := 12 + topRows*30
	for _, l := range []struct {
		n   int
		fmt string
		c   color.NRGBA
	}{{wolves, "WOLVES: %d", nrgba(255, 90, 80)}, {len(g.liveRaiders()), "RAIDERS: %d", nrgba(255, 140, 70)}, {sick, "SICK: %d", nrgba(150, 230, 130)}} {
		if l.n > 0 {
			g.text(dst, fmt.Sprintf(l.fmt, l.n), 8*s, ly*s, g.fontM, l.c)
			ly += 22
		}
	}

	// clock
	h := g.hour()
	day := int(g.time/DayLen) + 1
	spd := fmt.Sprintf("x%d", g.speed)
	if g.paused {
		spd = "PAUSED"
	}
	wx := ""
	if n := g.weatherName(); n != "" {
		wx = "  " + n
	}
	clock := fmt.Sprintf("%s  Day %d  %02d:%02d  %.0f°C%s  %s", g.seasonName(), day, int(h), int(math.Mod(h, 1)*60), g.outT, wx, spd)
	w, _ := text.Measure(clock, g.fontM, 0)
	rect(dst, float64(g.sw)-w-28*s, 8*s, w+20*s, 34*s, panel)
	g.text(dst, clock, float64(g.sw)-w-18*s, 15*s, g.fontM, color.White)

	// messages
	my := float64(g.sh) - g.barH() - 10*s
	for i := len(g.msgs) - 1; i >= 0; i-- {
		my -= 20 * s
		a := uint8(255 * clampf(g.msgs[i].T/2, 0, 1))
		mc := color.NRGBA{255, 240, 200, a}
		if g.msgs[i].Alert {
			mc = color.NRGBA{255, 214, 90, a}
		}
		g.text(dst, g.msgs[i].Text, 12*s, my, g.fontS, mc)
	}

	// tool bar
	bh := g.barH()
	rect(dst, 0, float64(g.sh)-bh, float64(g.sw), bh, color.NRGBA{14, 16, 22, 235})
	mxi, myi := ebiten.CursorPosition()
	for i := 0; i < int(numTools); i++ {
		bx, by, bw, bhh := g.buttonRect(i)
		c := color.NRGBA{40, 44, 56, 255}
		hover := float64(mxi) >= bx && float64(mxi) <= bx+bw && float64(myi) >= by && float64(myi) <= by+bhh
		if hover {
			c = color.NRGBA{60, 66, 84, 255}
		}
		if Tool(i) == g.tool {
			c = color.NRGBA{76, 84, 112, 255}
		}
		rect(dst, bx, by, bw, bhh, c)
		rect(dst, bx, by+bhh-4*s, bw, 4*s, toolInfo[i].Col)
		if Tool(i) == g.tool {
			strokeRect(dst, bx, by, bw, bhh, 2, color.White)
		}
		g.toolIcon(dst, Tool(i), bx+3*s, by+6*s, 28*s)
		g.text(dst, toolInfo[i].Name, bx+33*s, by+5*s, g.fontS, color.White)
		if kw, _ := text.Measure(toolInfo[i].Lbl, g.fontS, 0); toolInfo[i].Lbl != "" {
			_ = kw
			g.text(dst, toolInfo[i].Lbl, bx+3*s, by+bhh-float64(g.fontS.Size)-2*s, g.fontS, color.NRGBA{255, 235, 150, 255}) // hotkey badge, bottom-left
		}
		g.text(dst, g.toolCost(Tool(i)), bx+33*s, by+21*s, g.fontS, color.NRGBA{170, 175, 190, 255})
	}

	g.drawTraderBanner(dst)
	g.drawRaidBanner(dst)
	g.drawPanels(dst)
	g.drawGameOver(dst)

	if g.showHelp {
		lines := []string{
			"LANDFALL COLONY",
			"",
			"WASD/arrows pan, Shift faster, wheel zoom, right-drag pan. Pick a tool, drag a rectangle.",
			"Chop / Mine / Harvest: mark trees, rocks, bushes, and herbs (pale-green flowers = medicine).",
			"Hunt: mark deer for colonists to kill. Wolf bites bleed: colonists bandage, or use medicine on bad wounds.",
			"Stockpile: zone where loose items are hauled (needed to build and to eat from).",
			"Farm: tilled soil, colonists sow and reap. Crops grow only at 6-38C and die in frost.",
			"Wall / Door: enclose a room (needs a door!). Rooms hold warmth; floors are cosmetic.",
			"Bed: colonists sleep there at night (ground sleeping is poor). Heater: burns 1 wood / 40s",
			"  from the stockpile and warms its room. Winters are deadly: build a heated room.",
			"Wolves raid at night. Colonists fight back, then tend wounds (red + marker) and rest in beds to recover.",
			"Farm tool: pick a crop (wheat, corn, cotton, aloe = medicine, tobacco). Cook bread and meals at a Stove (Cooking skill).",
			"Weapons: forge blades at a workbench (iron), guns and ammo at a fabricator. Colonists equip the best one automatically.",
			"Move (V): click a furnace, desk, bed or machine, then click a free spot: colonists carry it there, nothing lost.",
			"Roof (R): costs wood. Roofs make walled rooms warm and keep food from spoiling. Salvage (X) removes buildings, floors, roofs.",
			"B: build menu (workbench, desk, lamp, generator, solar, fabricator). P: production targets (! = urgent). T: trade. U: mark orders urgent.",
			"Colonists learn by working; click a skill in their panel to train it at a Desk. Books from traders help.",
			"Right-click / Esc: cancel   Space: pause   Tab: speed   F11: fullscreen   H: help",
			"F5: quick save   F9: quick load   F10: load autosave (autosaves every 2 min and on exit)   L: save slots   M: mute",
		}
		pw, ph := 720*s, float64(len(lines))*20*s+20*s
		px, py := (float64(g.sw)-pw)/2, 70*s
		rect(dst, px, py, pw, ph, color.NRGBA{10, 12, 18, 225})
		for i, l := range lines {
			f := g.fontS
			if i == 0 {
				f = g.fontM
			}
			g.text(dst, l, px+14*s, py+10*s+float64(i)*20*s, f, color.White)
		}
	}
}

func (g *Game) toolCost(t Tool) string {
	var c Cost
	switch t {
	case TWall:
		c = costFor(BWall, g.wallMat)
	case TDoor:
		c = costFor(BDoor, g.doorMat)
	case TFarm:
		if g.cropSel == CropVeg {
			return "Veg"
		}
		return cropDefs[g.cropSel].Name
	case TFloor:
		c = floorCost(g.floorSel)
	case TBed:
		c = costs[BBed]
	case THeater:
		c = costs[BHeater]
	default:
		return ""
	}
	return fmt.Sprintf("%d %s", c.N, itemNames[c.Item])
}

func main() {
	seed := flag.Int64("seed", time.Now().UnixNano()%100000, "world seed")
	self := flag.Bool("selftest", false, "run a headless simulation and print results")
	demo := flag.String("demo", "", "set up a showcase colony (prod, trade, build, night) for screenshots")
	fresh := flag.Bool("new", false, "start a new game instead of continuing the autosave")
	flag.Parse()
	g := newGame(*seed)
	if *self {
		selfTest(g)
		return
	}
	if !*fresh {
		if err := g.LoadFile(autoSavePath()); err == nil {
			g.msg("Continued from autosave. (Run with -new for a fresh world.)")
		}
	}
	if *demo != "" {
		g.setupDemo(*demo)
	}
	ebiten.SetWindowSize(1280, 760)
	title := "Landfall Colony"
	if *demo != "" {
		title = "Landfall Colony (demo)"
	}
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetVsyncEnabled(true)
	err := ebiten.RunGame(g)
	g.autoSave()
	if err != nil {
		log.Fatal(err)
	}
}
