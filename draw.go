package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ---------- sprite helpers ----------

// blit draws a 32x32 sprite scaled to size px with its top-left at (x,y).
func (g *Game) blit(dst, img *ebiten.Image, x, y, size float64, flip bool, alpha float32) {
	op := &g.op
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.Filter = ebiten.FilterNearest
	s := size / 32
	if flip {
		op.GeoM.Scale(-s, s)
		op.GeoM.Translate(x+size, y)
	} else {
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(x, y)
	}
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(alpha)
	}
	dst.DrawImage(img, op)
}

// blitTile draws a tile sprite snapped to whole pixels so neighbours never leave seams.
func (g *Game) blitTile(dst, img *ebiten.Image, sx, sy, sz float64, alpha float32) {
	x0, y0 := math.Round(sx), math.Round(sy)
	x1, y1 := math.Round(sx+sz), math.Round(sy+sz)
	op := &g.op
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale((x1-x0)/32, (y1-y0)/32)
	op.GeoM.Translate(x0, y0)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(alpha)
	}
	dst.DrawImage(img, op)
}

func (g *Game) blueSprite(b Blue) *ebiten.Image { return g.blueSpriteMat(b, MatNone) }

func (g *Game) blueSpriteMat(b Blue, m Material) *ebiten.Image {
	switch b {
	case BWall:
		return art.wall[resolveMat(false, m)]
	case BDoor:
		return art.door[resolveMat(true, m)]
	case BFurnace:
		return art.furnace[0]
	case BStove:
		return art.stove[0]
	case BHorse:
		return art.horse
	case BChecker:
		return art.checker
	case BShrine:
		return art.shrine
	case BBed:
		return art.bed
	case BHeater:
		return art.heater[0]
	case BBench:
		return art.bench
	case BFab:
		return art.fab
	case BGen:
		return art.gen[0]
	case BSolar:
		return art.solar
	case BLamp:
		return art.lamp[1]
	case BEHeater:
		return art.eheater[1]
	case BDesk:
		return art.desk
	}
	return art.floor[FloorWood][0]
}

// ---------- particles ----------

type particle struct {
	x, y, vx, vy, life, max, size float64
	c                             color.NRGBA
	grav                          float64
}

func (g *Game) burst(x, y float64, n int, c color.NRGBA, speed, size, life, grav float64) {
	for i := 0; i < n && len(g.parts) < 400; i++ {
		a := rand.Float64() * 2 * math.Pi
		sp := speed * (0.3 + rand.Float64()*0.7)
		g.parts = append(g.parts, particle{x: x, y: y, vx: math.Cos(a) * sp, vy: math.Sin(a)*sp - speed*0.4, life: life * (0.6 + rand.Float64()*0.4), max: life, size: size, c: c, grav: grav})
	}
}

func (g *Game) updateParticles(dt float64) {
	live := g.parts[:0]
	for _, p := range g.parts {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vy += p.grav * dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		live = append(live, p)
	}
	g.parts = live
}

var eventVol = map[string]float64{
	"chop": 0.7, "mine": 0.6, "hammer": 0.6, "rustle": 0.5, "hit": 0.8, "bite": 0.9,
	"yelp": 0.8, "thud": 0.8, "death": 1, "heal": 0.6, "howl": 0.9, "levelup": 0.5, "coin": 0.7, "save": 0.6, "trader": 0.9, "fire": 0.8, "bray": 0.8, "groan": 0.8, "splash": 0.6, "hiss": 0.7, "shot": 0.8, "rifle": 1.0, "slash": 0.6, "horn": 1.0, "cough": 0.5, "clang": 0.7, "bell": 0.6,
}

// handleEvent turns simulation events into sound (positional) and particles.
func (g *Game) handleEvent(name string, x, y float64) {
	vol := eventVol[name]
	if x >= 0 {
		cx, cy := g.camX/TileSize, g.camY/TileSize
		span := float64(g.sw) / (2 * TileSize * g.zoom)
		d := math.Hypot(x-cx, (y-cy)*1.3)
		vol *= clampf(1-d/(span*1.5), 0, 1)
	}
	g.audio.play(name, vol)
	if x < 0 {
		return
	}
	switch name {
	case "chop":
		g.burst(x, y, 5, color.NRGBA{170, 120, 70, 255}, 3, 3, 0.5, 9)
	case "mine":
		g.burst(x, y, 6, color.NRGBA{210, 210, 220, 255}, 4, 2, 0.35, 8)
	case "hammer":
		g.burst(x, y, 4, color.NRGBA{220, 190, 120, 255}, 2.5, 2, 0.4, 6)
	case "rustle":
		g.burst(x, y, 6, color.NRGBA{90, 190, 80, 255}, 2, 3, 0.7, 3)
	case "hit", "bite":
		g.burst(x, y, 7, color.NRGBA{200, 30, 40, 255}, 3.5, 3, 0.5, 9)
	case "thud":
		g.burst(x, y, 8, color.NRGBA{150, 130, 100, 255}, 2.5, 4, 0.6, 2)
	case "heal":
		g.burst(x, y, 6, color.NRGBA{120, 240, 170, 255}, 1.5, 3, 1.0, -3)
	case "shot", "rifle":
		g.burst(x, y, 4, color.NRGBA{255, 220, 120, 255}, 2.2, 3, 0.18, 0) // muzzle flash
	case "slash":
		g.burst(x, y, 4, color.NRGBA{235, 240, 250, 255}, 3, 2, 0.2, 2)
	case "splash":
		g.burst(x, y, 6, color.NRGBA{120, 180, 255, 255}, 2.8, 3, 0.5, 8)
	case "hiss":
		g.burst(x, y, 12, color.NRGBA{225, 228, 235, 220}, 1.2, 5, 1.2, -3)
	case "fire":
		g.burst(x, y, 10, color.NRGBA{255, 150, 40, 255}, 2.5, 3, 0.9, -4)
	}
}

// ---------- main draw ----------

func (g *Game) Draw(dst *ebiten.Image) {
	if art == nil {
		return
	}
	dst.Fill(nrgba(20, 30, 50))
	sz := TileSize * g.zoom
	W := g.W
	x0 := max(0, int(math.Floor((g.camX-float64(g.sw)/2/g.zoom)/TileSize)))
	y0 := max(0, int(math.Floor((g.camY-float64(g.sh)/2/g.zoom)/TileSize)))
	x1 := min(MapW-1, int(math.Ceil((g.camX+float64(g.sw)/2/g.zoom)/TileSize)))
	y1 := min(MapH-1, int(math.Ceil((g.camY+float64(g.sh)/2/g.zoom)/TileSize)))
	snow := clampf((2-g.outT)/8, 0, 1)
	waterFrame := int(g.time * 1.5)

	// ground, floors, farmland, zones
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := &W.T[ty*MapW+tx]
			sx, sy := g.toScreen(float64(tx), float64(ty))
			v := int(t.Shade % 4)
			if t.Ground == GWater {
				g.blitTile(dst, art.water[v][(waterFrame+int(t.Shade>>2))%4], sx, sy, sz, 1)
			} else {
				g.blitTile(dst, art.ground[t.Ground][v], sx, sy, sz, 1)
			}
			if t.Floor {
				g.blitTile(dst, art.floor[t.FloorKind][(tx+ty)%2], sx, sy, sz, 1)
			}
			if t.Farm {
				g.blitTile(dst, art.farm, sx, sy, sz, 1)
				if t.Crop {
					st := 3
					switch gr := float64(t.Grow); {
					case gr < 0.34:
						st = 0
					case gr < 0.67:
						st = 1
					case gr < 1:
						st = 2
					}
					g.blitTile(dst, art.crop[t.CropKind][st], sx, sy, sz, 1)
				}
			}
			if snow > 0 && t.Ground != GWater && !t.Floor && !t.Farm && !t.Roof && !g.indoors(tx, ty) {
				rect(dst, sx, sy, sz+1, sz+1, color.NRGBA{236, 241, 250, uint8(snow * 175)})
			}
			if t.Roof {
				a := uint8(34)
				if g.tool == TRoof {
					a = 90 // the Roof tool shows roofs clearly
				}
				rect(dst, sx, sy, sz+1, sz+1, color.NRGBA{120, 78, 50, a})
				if g.zoom >= 0.8 {
					rect(dst, sx, sy, sz+1, 1, color.NRGBA{70, 44, 28, a + 40})
					if g.tool == TRoof {
						rect(dst, sx, sy+sz*0.5, sz+1, 1, color.NRGBA{70, 44, 28, a})
					}
				}
			}
			if t.Zone {
				rect(dst, sx, sy, sz, sz, color.NRGBA{70, 130, 235, 55})
				strokeRect(dst, sx, sy, sz, sz, 1, color.NRGBA{90, 150, 255, 110})
			}
		}
	}

	// objects, items, overlays
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := &W.T[ty*MapW+tx]
			sx, sy := g.toScreen(float64(tx), float64(ty))
			cx, cy := sx+sz/2, sy+sz/2
			if t.Blue != BNone {
				if t.Blue == BFloor {
					g.blitTile(dst, art.floor[t.BlueKind][(tx+ty)%2], sx, sy, sz, 0.55)
				} else {
					g.blitTile(dst, g.blueSpriteMat(t.Blue, t.Mat), sx, sy, sz, 0.45)
				}
				if t.MoveFrom != 0 {
					rect(dst, sx, sy, sz, sz, color.NRGBA{255, 220, 120, 50})
					strokeRect(dst, sx+1, sy+1, sz-2, sz-2, 2, color.NRGBA{255, 225, 130, 230})
				} else if g.blueprintBlock(t) != "" {
					rect(dst, sx, sy, sz, sz, color.NRGBA{255, 90, 70, 60})
					strokeRect(dst, sx+1, sy+1, sz-2, sz-2, 2, color.NRGBA{255, 110, 90, 230})
				} else {
					rect(dst, sx, sy, sz, sz, color.NRGBA{120, 180, 255, 60})
					strokeRect(dst, sx+1, sy+1, sz-2, sz-2, 2, color.NRGBA{150, 200, 255, 220})
				}
			}
			switch t.Obj {
			case OTree:
				g.blitTile(dst, art.tree[int(t.Shade)%3], sx, sy-sz*0.12, sz*1.12, 1)
			case ORock:
				g.blitTile(dst, art.rock[int(t.Shade)%2], sx, sy, sz, 1)
			case OBush:
				g.blitTile(dst, art.bush, sx, sy, sz, 1)
			case OHerb:
				g.blitTile(dst, art.herb, sx, sy, sz, 1)
			case OWall:
				g.blitTile(dst, art.wall[t.builtMat()], sx, sy, sz, 1)
			case OOre:
				g.blitTile(dst, art.ore, sx, sy, sz, 1)
			case OHorse:
				g.blitTile(dst, art.horse, sx, sy, sz, 1)
			case OChecker:
				g.blitTile(dst, art.checker, sx, sy, sz, 1)
			case OShrine:
				g.blitTile(dst, art.shrine, sx, sy, sz, 1)
				vector.FillCircle(dst, float32(sx+sz*0.28), float32(sy+sz*0.45), float32(sz*(0.2+0.03*math.Sin(g.time*7))), color.NRGBA{255, 200, 90, 28}, true)
			case OStove:
				f := 0
				if g.burnerActive(ty*MapW + tx) {
					f = 1 + int(g.time*8+float64(tx))%2
				}
				g.blitTile(dst, art.stove[f], sx, sy, sz, 1)
			case OFurnace:
				f := 0
				if g.burnerActive(ty*MapW + tx) {
					f = 1 + int(g.time*8+float64(tx))%2
				}
				g.blitTile(dst, art.furnace[f], sx, sy, sz, 1)
			case OBed:
				g.blitTile(dst, art.bed, sx, sy, sz, 1)
			case ODoor:
				l, r := W.At(tx-1, ty), W.At(tx+1, ty)
				if (l != nil && l.Obj == OWall) || (r != nil && r.Obj == OWall) {
					g.blitTile(dst, art.door[t.builtMat()], sx, sy, sz, 1)
				} else { // door in a vertical wall: rotate the sprite
					op := &g.op
					op.GeoM.Reset()
					op.ColorScale.Reset()
					op.Filter = ebiten.FilterNearest
					op.GeoM.Translate(-16, -16)
					op.GeoM.Rotate(math.Pi / 2)
					op.GeoM.Scale(sz/32, sz/32)
					op.GeoM.Translate(cx, cy)
					dst.DrawImage(art.door[t.builtMat()], op)
				}
			case OBench:
				g.blitTile(dst, art.bench, sx, sy, sz, 1)
			case OFab:
				g.blitTile(dst, art.fab, sx, sy, sz, 1)
				if g.powerProblem(OFab) != "" {
					g.drawNoPower(dst, sx, sy, sz)
				}
			case ODesk:
				g.blitTile(dst, art.desk, sx, sy, sz, 1)
			case OSolar:
				g.blitTile(dst, art.solar, sx, sy, sz, 1)
			case OGen:
				f := 0
				if t.Fuel > 0 {
					f = 1 + int(g.time*8+float64(tx))%2
				}
				g.blitTile(dst, art.gen[f], sx, sy, sz, 1)
			case OLamp:
				on := 0
				if g.lampOn(ty*MapW + tx) {
					on = 1
				}
				g.blitTile(dst, art.lamp[on], sx, sy, sz, 1)
				if on == 0 {
					g.drawNoPower(dst, sx, sy, sz)
				}
			case OEHeater:
				on := 0
				if g.eheaterOn(ty*MapW + tx) {
					on = 1
					vector.FillCircle(dst, float32(cx), float32(cy), float32(sz*0.9), color.NRGBA{255, 120, 50, 30}, true)
				}
				g.blitTile(dst, art.eheater[on], sx, sy, sz, 1)
				if on == 0 {
					g.drawNoPower(dst, sx, sy, sz)
				}
			case OHeater:
				f := 0
				if t.Fuel > 0 {
					f = 1 + int(g.time*7+float64(tx))%2
					vector.FillCircle(dst, float32(cx), float32(cy), float32(sz*(0.9+0.1*math.Sin(g.time*9))), color.NRGBA{255, 140, 40, 38}, true)
				}
				g.blitTile(dst, art.heater[f], sx, sy, sz, 1)
			}
			if t.Item != INone && t.ItemN > 0 {
				g.blitTile(dst, art.item[t.Item], sx+sz*0.12, sy+sz*0.12, sz*0.76, 1)
				if t.ItemN >= 15 {
					g.blitTile(dst, art.item[t.Item], sx+sz*0.2, sy-sz*0.02, sz*0.76, 1)
				}
				if g.zoom >= 1.1 {
					g.text(dst, fmt.Sprint(t.ItemN), sx+2, sy+sz-float64(g.fontS.Size)-1, g.fontS, color.White)
				}
			}
			if t.Fire > 0 {
				rect(dst, sx, sy, sz, sz, color.NRGBA{20, 8, 4, 110})
				vector.FillCircle(dst, float32(cx), float32(cy), float32(sz*0.8), color.NRGBA{255, 120, 30, 40}, true)
				g.blitTile(dst, art.flame[(int(g.time*9)+tx+ty)%3], sx, sy-sz*0.1, sz, 0.95)
				if rand.Float64() < 0.05 {
					g.burst(float64(tx)+0.5, float64(ty)+0.3, 1, color.NRGBA{90, 90, 96, 200}, 0.8, 4, 1.3, -2.5)
				}
			}
			if t.Desig != DNone && desigValid(t) {
				c := desigColor(t.Desig)
				strokeRect(dst, sx+1, sy+1, sz-2, sz-2, 2, c)
				c.A = 45
				rect(dst, sx, sy, sz, sz, c)
			}
			if t.Urgent && (t.Blue != BNone || (t.Desig != DNone && desigValid(t))) {
				g.drawUrgentMark(dst, sx+sz*0.62, sy+sz*0.04, sz*0.34)
			}
		}
	}

	// pending moves: an arrow from where the building is to where it is going
	for i := range W.T {
		if mf := W.T[i].MoveFrom; mf != 0 {
			s := int(mf) - 1
			x1, y1 := g.toScreen(float64(s%MapW)+0.5, float64(s/MapW)+0.5)
			x2, y2 := g.toScreen(float64(i%MapW)+0.5, float64(i/MapW)+0.5)
			vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 2, color.NRGBA{255, 225, 130, 150}, true)
			vector.FillCircle(dst, float32(x2), float32(y2), float32(sz*0.09), color.NRGBA{255, 225, 130, 220}, true)
		}
	}
	if g.moveSrc >= 0 { // the building picked with the Move tool
		sx, sy := g.toScreen(float64(g.moveSrc%MapW), float64(g.moveSrc/MapW))
		pulse := 0.5 + 0.5*math.Sin(g.time*8)
		strokeRect(dst, sx-2, sy-2, sz+4, sz+4, 3, color.NRGBA{120, 255, 220, uint8(150 + 100*pulse)})
		mxi, myi := ebiten.CursorPosition()
		tx, ty := g.toTile(float64(mxi), float64(myi))
		if t := g.W.At(tx, ty); t != nil {
			px, py := g.toScreen(float64(tx), float64(ty))
			c := color.NRGBA{120, 255, 160, 255}
			if !g.canMoveTo(tx, ty) {
				c = color.NRGBA{255, 110, 100, 255}
			}
			strokeRect(dst, px, py, sz, sz, 2, c)
			if o := g.W.At(g.moveSrc%MapW, g.moveSrc/MapW).Obj; movable(o) {
				g.blitTile(dst, g.blueSpriteMat(objBlue[o], g.W.At(g.moveSrc%MapW, g.moveSrc/MapW).Mat), px, py, sz, 0.5)
			}
		}
	}

	// animals
	for _, a := range g.animals {
		sx, sy := g.toScreen(a.X, a.Y)
		fr := 0
		if a.Moving {
			fr = int(g.time*8) % 2
		}
		if a.Kind == ARaider {
			rf := 0
			if a.Moving {
				rf = int(g.time*9) % 4
			}
			g.blit(dst, art.raiderSprite(a.Face, rf), sx-sz/2, sy-sz*17/32, sz, false, 1)
			if _, armed := weaponDefs[a.Weapon]; armed {
				ox, flip := 0.3, false
				switch a.Face {
				case 2:
					ox, flip = -0.3, true
				case 1:
					ox = -0.28
				}
				g.blit(dst, art.item[a.Weapon], sx+ox*sz-sz*0.26, sy-sz*0.42, sz*0.52, flip, 1)
			}
			if a.LootN > 0 {
				g.blit(dst, art.item[a.Loot], sx-sz*0.2, sy-sz*0.95, sz*0.4, false, 1)
			}
			if mx := a.MaxHP; mx > 0 && a.HP < mx {
				g.bar(dst, sx-sz*0.25, sy-sz*0.55, sz*0.5, 3, a.HP/mx, nrgba(220, 60, 60))
			}
			continue
		}
		img := art.deer[fr]
		if a.Kind == AWolf {
			img = art.wolf[fr]
		}
		g.blit(dst, img, sx-sz/2, sy-sz*17/32, sz, a.DX < 0, 1)
		if a.Hunt {
			vector.StrokeCircle(dst, float32(sx), float32(sy+sz*0.1), float32(sz*0.4), 2, toolInfo[THunt].Col, true)
		}
		mx := a.MaxHP
		if mx > 0 && a.HP < mx {
			g.bar(dst, sx-sz*0.25, sy-sz*0.5, sz*0.5, 3, a.HP/mx, nrgba(220, 60, 60))
		}
	}

	// trader and pack animal
	if t := g.trader; t != nil {
		bsx, bsy := g.toScreen(t.BX, t.BY)
		bimg := art.donkey
		if t.Beast == BeastCamel {
			bimg = art.camel
		}
		bf := 0
		if t.BMoving {
			bf = int(g.time*7) % 2
		}
		g.blit(dst, bimg[bf], bsx-sz/2, bsy-sz*17/32, sz, t.BDX < 0, 1)
		sx, sy := g.toScreen(t.X, t.Y)
		fr := 0
		if t.Moving {
			fr = int(g.time*8) % 4
		}
		if t.Arrived && !t.Leaving { // a pulsing gold ring so the trader is easy to spot
			vector.StrokeCircle(dst, float32(sx), float32(sy+sz*0.15), float32(sz*(0.62+0.1*math.Sin(g.time*5))), 3, color.NRGBA{255, 214, 90, 200}, true)
		}
		g.blit(dst, art.traderSprite(t.Face, fr), sx-sz/2, sy-sz*17/32, sz, false, 1)
		if g.zoom >= 0.8 {
			label := "Trader"
			if g.traderHere() {
				label = "Trader (T)"
			}
			w, _ := text.Measure(label, g.fontS, 0)
			g.text(dst, label, sx-w/2, sy+sz*0.4, g.fontS, color.NRGBA{255, 220, 120, 255})
		}
	}

	// colonists
	for _, c := range g.cols {
		sx, sy := g.toScreen(c.X, c.Y)
		resting := (c.Job.Kind == JSleep || c.Job.Kind == JRecover) && len(c.Path) == 0
		if c.ID == g.selected {
			vector.StrokeCircle(dst, float32(sx), float32(sy+sz*0.2), float32(sz*0.42), 2, color.White, true)
		}
		if resting {
			g.blit(dst, art.lyingSprite(c), sx-sz/2, sy-sz*17/32, sz, false, 1)
		} else {
			fr := 0
			if c.Moving {
				fr = int(g.time*9) % 4
			}
			g.blit(dst, art.colonist(c, c.Face, fr), sx-sz/2, sy-sz*17/32, sz, false, 1)
		}
		if c.Carry != INone {
			g.blit(dst, art.item[c.Carry], sx-sz*0.2, sy-sz*0.95, sz*0.4, false, 1)
		}
		if cg := c.Cargo; cg != nil {
			g.blit(dst, g.blueSpriteMat(objBlue[cg.Obj], cg.Mat), sx-sz*0.3, sy-sz*1.05, sz*0.6, false, 1)
		}
		if _, armed := weaponDefs[c.Weapon]; armed && (c.Job.Kind == JFight || c.Job.Kind == JHunt) {
			ox, flip := 0.3, false // the weapon in hand, on the side they face
			switch c.Face {
			case 2:
				ox, flip = -0.3, true
			case 1:
				ox = -0.28
			}
			g.blit(dst, art.item[c.Weapon], sx+ox*sz-sz*0.26, sy-sz*0.42, sz*0.52, flip, 1)
		}
		if c.sick() { // a green cross: ill (injuries get a red one)
			gc := nrgba(120, 220, 110)
			if c.sickSeverity() >= critSev {
				gc = nrgba(255, uint8(120+100*math.Sin(g.time*8)*math.Sin(g.time*8)), 90)
			}
			mx, my := sx-sz*0.3, sy-sz*0.42
			rect(dst, mx-sz*0.04, my-sz*0.12, sz*0.08, sz*0.24, color.NRGBA{0, 0, 0, 200})
			rect(dst, mx-sz*0.12, my-sz*0.04, sz*0.24, sz*0.08, color.NRGBA{0, 0, 0, 200})
			rect(dst, mx-sz*0.03, my-sz*0.1, sz*0.06, sz*0.2, gc)
			rect(dst, mx-sz*0.1, my-sz*0.03, sz*0.2, sz*0.06, gc)
		}
		if len(c.Injuries) > 0 {
			mc := nrgba(235, 60, 60)
			if !c.untended() {
				mc = nrgba(240, 240, 240)
			}
			mx, my := sx+sz*0.28, sy-sz*0.42
			rect(dst, mx-sz*0.04, my-sz*0.12, sz*0.08, sz*0.24, color.NRGBA{0, 0, 0, 200})
			rect(dst, mx-sz*0.12, my-sz*0.04, sz*0.24, sz*0.08, color.NRGBA{0, 0, 0, 200})
			rect(dst, mx-sz*0.03, my-sz*0.1, sz*0.06, sz*0.2, mc)
			rect(dst, mx-sz*0.1, my-sz*0.03, sz*0.2, sz*0.06, mc)
		}
		if resting {
			g.text(dst, "z", sx+sz*0.1, sy-sz*0.7-math.Sin(g.time*2+float64(c.ID))*3, g.fontM, color.NRGBA{200, 220, 255, 230})
		}
		if g.zoom >= 1 {
			w, _ := text.Measure(c.Name, g.fontS, 0)
			g.text(dst, c.Name, sx-w/2, sy+sz*0.4, g.fontS, color.White)
		}
	}

	// bullet tracers
	for _, sh := range g.shots {
		x1, y1 := g.toScreen(sh.X1, sh.Y1)
		x2, y2 := g.toScreen(sh.X2, sh.Y2)
		a := uint8(255 * clampf(sh.T/0.12, 0, 1))
		vector.StrokeLine(dst, float32(x1), float32(y1-sz*0.1), float32(x2), float32(y2-sz*0.1), 2, color.NRGBA{255, 235, 150, a}, true)
		vector.FillCircle(dst, float32(x1), float32(y1-sz*0.1), float32(sz*0.1), color.NRGBA{255, 220, 120, a}, true)
	}

	// particles
	for _, p := range g.parts {
		sx, sy := g.toScreen(p.x, p.y)
		c := p.c
		c.A = uint8(255 * clampf(p.life/p.max*1.5, 0, 1))
		s := math.Max(2, p.size*g.zoom*g.scale/1.5)
		rect(dst, sx-s/2, sy-s/2, s, s, c)
	}

	// drag preview
	if g.dragging {
		mxi, myi := ebiten.CursorPosition()
		tx, ty := g.toTile(float64(mxi), float64(myi))
		ax, bx := min(g.dragStart.x, tx), max(g.dragStart.x, tx)
		ay, by := min(g.dragStart.y, ty), max(g.dragStart.y, ty)
		sx, sy := g.toScreen(float64(ax), float64(ay))
		c := toolInfo[g.tool].Col
		w, h := float64(bx-ax+1)*sz, float64(by-ay+1)*sz
		c.A = 60
		rect(dst, sx, sy, w, h, c)
		c.A = 255
		strokeRect(dst, sx, sy, w, h, 2, c)
		lbl := fmt.Sprintf("%dx%d", bx-ax+1, by-ay+1)
		g.text(dst, lbl, sx+4, sy-float64(g.fontS.Size)-4, g.fontS, color.White)
	} else if g.tool != TSelect {
		mxi, myi := ebiten.CursorPosition()
		if float64(myi) < float64(g.sh)-g.barH() {
			tx, ty := g.toTile(float64(mxi), float64(myi))
			sx, sy := g.toScreen(float64(tx), float64(ty))
			strokeRect(dst, sx, sy, sz, sz, 2, toolInfo[g.tool].Col)
		}
	}

	g.drawWeather(dst)

	// day / night: a dark overlay with holes cut out where lamps shine, so lit areas look like
	// daytime instead of being washed out with yellow
	light := 0.5 + 0.5*math.Cos((g.hour()/24-0.5)*2*math.Pi)
	dark := math.Pow(1-light, 1.5)
	if a := dark * 0.6; a > 0.01 {
		overlay := color.NRGBA{8, 14, 48, uint8(a * 255)}
		if art.glow == nil || len(g.lamps) == 0 {
			rect(dst, 0, 0, float64(g.sw), float64(g.sh), overlay)
		} else {
			if g.nightLayer == nil || g.nightLayer.Bounds().Dx() != g.sw || g.nightLayer.Bounds().Dy() != g.sh {
				g.nightLayer = ebiten.NewImage(g.sw, g.sh)
			}
			g.nightLayer.Fill(overlay)
			for _, l := range g.lamps {
				lx, ly := g.toScreen(float64(l%MapW)+0.5, float64(l/MapW)+0.5)
				r := lampRadius * sz
				if lx < -r || ly < -r || lx > float64(g.sw)+r || ly > float64(g.sh)+r {
					continue
				}
				op := &g.op
				op.GeoM.Reset()
				op.ColorScale.Reset()
				op.Filter = ebiten.FilterLinear
				op.Blend = ebiten.BlendDestinationOut
				op.GeoM.Scale(2*r/128, 2*r/128)
				op.GeoM.Translate(lx-r, ly-r)
				g.nightLayer.DrawImage(art.glow, op)
				op.Blend = ebiten.Blend{}
			}
			dst.DrawImage(g.nightLayer, &ebiten.DrawImageOptions{})
			// a faint warm tint so the light still reads as lamplight
			for _, l := range g.lamps {
				lx, ly := g.toScreen(float64(l%MapW)+0.5, float64(l/MapW)+0.5)
				r := lampRadius * sz
				if lx < -r || ly < -r || lx > float64(g.sw)+r || ly > float64(g.sh)+r {
					continue
				}
				op := &g.op
				op.GeoM.Reset()
				op.ColorScale.Reset()
				op.Filter = ebiten.FilterLinear
				op.Blend = ebiten.BlendLighter
				op.GeoM.Scale(2*r/128, 2*r/128)
				op.GeoM.Translate(lx-r, ly-r)
				op.ColorScale.ScaleAlpha(float32(0.10 * dark))
				dst.DrawImage(art.glow, op)
				op.Blend = ebiten.Blend{}
			}
		}
	}

	g.drawUI(dst)
	g.drawTooltip(dst)
}

// toolIcon draws a small icon for a tool button.
func (g *Game) toolIcon(dst *ebiten.Image, t Tool, x, y, size float64) {
	switch t {
	case TSelect:
		var p vector.Path
		p.MoveTo(float32(x+size*0.2), float32(y+size*0.1))
		p.LineTo(float32(x+size*0.2), float32(y+size*0.85))
		p.LineTo(float32(x+size*0.42), float32(y+size*0.66))
		p.LineTo(float32(x+size*0.62), float32(y+size*0.95))
		p.LineTo(float32(x+size*0.74), float32(y+size*0.88))
		p.LineTo(float32(x+size*0.55), float32(y+size*0.6))
		p.LineTo(float32(x+size*0.82), float32(y+size*0.58))
		p.Close()
		vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = 1, 1, 1, 1
		}
		dst.DrawTriangles(vs, is, emptyImage(), &ebiten.DrawTrianglesOptions{})
		return
	case TPriority:
		g.drawUrgentMark(dst, x+size*0.1, y+size*0.1, size*0.8)
		return
	case TErase:
		vector.StrokeLine(dst, float32(x+size*0.15), float32(y+size*0.15), float32(x+size*0.85), float32(y+size*0.85), 3, color.NRGBA{230, 90, 210, 255}, true)
		vector.StrokeLine(dst, float32(x+size*0.85), float32(y+size*0.15), float32(x+size*0.15), float32(y+size*0.85), 3, color.NRGBA{230, 90, 210, 255}, true)
		return
	case TMove:
		c := color.NRGBA{120, 220, 200, 255}
		for _, a := range [][4]float64{{0.5, 0.12, 0.5, 0.88}, {0.12, 0.5, 0.88, 0.5}} {
			vector.StrokeLine(dst, float32(x+size*a[0]), float32(y+size*a[1]), float32(x+size*a[2]), float32(y+size*a[3]), 3, c, true)
		}
		g.fillPoly(dst, [][2]float64{{x + size*0.5, y + size*0.02}, {x + size*0.64, y + size*0.22}, {x + size*0.36, y + size*0.22}}, c)
		g.fillPoly(dst, [][2]float64{{x + size*0.5, y + size*0.98}, {x + size*0.64, y + size*0.78}, {x + size*0.36, y + size*0.78}}, c)
		g.fillPoly(dst, [][2]float64{{x + size*0.02, y + size*0.5}, {x + size*0.22, y + size*0.36}, {x + size*0.22, y + size*0.64}}, c)
		g.fillPoly(dst, [][2]float64{{x + size*0.98, y + size*0.5}, {x + size*0.78, y + size*0.36}, {x + size*0.78, y + size*0.64}}, c)
		return
	case TRoof:
		g.fillPoly(dst, [][2]float64{{x + size*0.5, y + size*0.08}, {x + size*0.96, y + size*0.52}, {x + size*0.04, y + size*0.52}}, color.NRGBA{190, 90, 60, 255})
		g.fillPoly(dst, [][2]float64{{x + size*0.5, y + size*0.2}, {x + size*0.8, y + size*0.48}, {x + size*0.2, y + size*0.48}}, color.NRGBA{220, 130, 90, 255})
		rect(dst, x+size*0.16, y+size*0.52, size*0.68, size*0.36, color.NRGBA{190, 170, 130, 255})
		rect(dst, x+size*0.42, y+size*0.62, size*0.16, size*0.26, color.NRGBA{110, 80, 50, 255})
		return
	case TDeconstruct:
		g.blit(dst, art.wall[MatStone], x, y, size, false, 0.7)
		vector.StrokeLine(dst, float32(x+size*0.1), float32(y+size*0.1), float32(x+size*0.9), float32(y+size*0.9), 3, color.NRGBA{255, 110, 90, 255}, true)
		vector.StrokeLine(dst, float32(x+size*0.9), float32(y+size*0.1), float32(x+size*0.1), float32(y+size*0.9), 3, color.NRGBA{255, 110, 90, 255}, true)
		return
	case TFarm:
		g.blit(dst, art.farm, x, y, size, false, 1)
		g.blit(dst, art.crop[g.cropSel][3], x, y, size, false, 1)
		return
	}
	var img *ebiten.Image
	switch t {
	case TChop:
		img = art.tree[0]
	case TMine:
		img = art.rock[0]
	case THarvest:
		img = art.bush
	case THunt:
		img = art.deer[0]
	case TZone:
		img = art.item[IWood]
	case TWall:
		img = art.wall[g.wallMat]
	case TDoor:
		img = art.door[g.doorMat]
	case TFloor:
		img = art.floor[g.floorSel][0]
	case TBed:
		img = art.bed
	case THeater:
		img = art.heater[1]
	case TBuild:
		img = art.bench
	}
	if img != nil {
		g.blit(dst, img, x, y, size, false, 1)
	}
}

var emptyImg *ebiten.Image

func emptyImage() *ebiten.Image {
	if emptyImg == nil {
		emptyImg = ebiten.NewImage(3, 3)
		emptyImg.Fill(color.White)
	}
	return emptyImg
}

// fillPoly fills a polygon given in screen coordinates.
func (g *Game) fillPoly(dst *ebiten.Image, pts [][2]float64, c color.NRGBA) {
	var p vector.Path
	for i, pt := range pts {
		if i == 0 {
			p.MoveTo(float32(pt[0]), float32(pt[1]))
		} else {
			p.LineTo(float32(pt[0]), float32(pt[1]))
		}
	}
	p.Close()
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	r, gg, b, a := c.RGBA()
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(r)/65535, float32(gg)/65535, float32(b)/65535, float32(a)/65535
	}
	dst.DrawTriangles(vs, is, emptyImage(), &ebiten.DrawTrianglesOptions{})
}

// drawNoPower puts a red lightning bolt on a building that isn't getting power.
func (g *Game) drawNoPower(dst *ebiten.Image, sx, sy, sz float64) {
	pts := [][2]float64{{0.62, 0.0}, {0.28, 0.5}, {0.5, 0.5}, {0.4, 1.0}, {0.78, 0.4}, {0.54, 0.4}}
	var big, small [][2]float64
	for _, p := range pts {
		x, y := sx+sz*0.55+p[0]*sz*0.4, sy+sz*0.02+p[1]*sz*0.45
		small = append(small, [2]float64{x, y})
		big = append(big, [2]float64{sx + sz*0.55 + (p[0]-0.5)*sz*0.4*1.35 + sz*0.2, sy + sz*0.02 + (p[1]-0.5)*sz*0.45*1.35 + sz*0.225})
	}
	g.fillPoly(dst, big, color.NRGBA{20, 10, 10, 230})
	g.fillPoly(dst, small, color.NRGBA{255, 70, 50, 255})
}

// drawTooltip explains the building or blueprint under the cursor.
func (g *Game) drawTooltip(dst *ebiten.Image) {
	mxi, myi := ebiten.CursorPosition()
	mx, my := float64(mxi), float64(myi)
	if g.overPanels(mx, my) || my > float64(g.sh)-g.barH() {
		return
	}
	tx, ty := g.toTile(mx, my)
	t := g.W.At(tx, ty)
	if t == nil {
		return
	}
	var lines []string
	if t.MoveFrom != 0 {
		lines = append(lines, blueLabel(t.Blue, t.Mat)+" (moving here)", "waiting for a colonist to carry it over")
	} else if (t.Obj == OWall || t.Obj == ODoor) && t.Blue == BNone {
		lines = append(lines, blueLabel(objBlue[t.Obj], t.builtMat()))
		switch {
		case t.Fire > 0:
			lines = append(lines, "ON FIRE!")
		case t.flammable():
			lines = append(lines, "wood: can catch fire from heaters and flames")
		default:
			lines = append(lines, "fireproof")
		}
	} else if t.Blue != BNone {
		lines = append(lines, t.blueName()+" (blueprint)", "cost: "+costText(t.blueCost()))
		if why := g.blueprintBlock(t); why != "" {
			lines = append(lines, "waiting: "+why)
		} else {
			lines = append(lines, "waiting for a colonist")
		}
	} else if (t.Obj >= OBench && t.Obj <= ODesk) || t.Obj == OFurnace || t.Obj == OStove || t.Obj == OHorse || t.Obj == OChecker || t.Obj == OShrine {
		name := map[Obj]string{OBench: "Workbench", OFab: "Fabricator", OGen: "Generator", OSolar: "Solar panel", OLamp: "Lamp", OEHeater: "Electric heater", ODesk: "Study desk", OFurnace: "Furnace", OStove: "Stove", OHorse: "Horseshoe pit", OChecker: "Checkers table", OShrine: "Meditation shrine"}[t.Obj]
		lines = append(lines, name)
		if t.Fire > 0 {
			lines = append(lines, "ON FIRE!")
		} else if t.flammable() {
			lines = append(lines, "wooden: can catch fire")
		}
		if p := g.powerProblem(t.Obj); p != "" {
			lines = append(lines, p)
		} else if t.Obj == OGen {
			if t.Fuel > 0 {
				lines = append(lines, fmt.Sprintf("running (%.0f s of wood left)", t.Fuel))
			} else {
				lines = append(lines, "out of wood: stock some in a stockpile")
			}
		} else if sd, isStation := stationDefs[t.Obj]; isStation {
			lines = append(lines, fmt.Sprintf("colonists come here when low on joy (%s)", sd.Verb))
		} else if t.Obj == OFab || t.Obj == OBench || t.Obj == OFurnace || t.Obj == OStove {
			lines = append(lines, "turn on recipes with P")
		} else if t.Obj == ODesk {
			lines = append(lines, "colonists train their focus skill here")
		}
	}
	if g.tool == TMove && movable(t.Obj) && g.moveSrc < 0 {
		lines = append(lines, g.moveName(t.Obj), "click to pick it up and move it")
	} else if t.MoveTo != 0 {
		lines = append(lines, g.moveName(t.Obj)+": going to be moved", "click it with the Move tool to cancel")
	}
	if t.Farm && t.Blue == BNone {
		lines = append(lines, g.cropTip(t, tx, ty)...)
	}
	lines = append(lines, g.roomTip(tx, ty)...)
	if t.Floor && t.Blue == BNone && len(lines) == 0 && t.Item == INone {
		lines = append(lines, floorNames[t.FloorKind], fmt.Sprintf("+%.0f%% walking speed", (floorSpeed[t.FloorKind]-1)*100))
	}
	if t.Item != INone && t.ItemN > 0 && t.Blue == BNone {
		lines = append(lines, fmt.Sprintf("%s x%d", itemNames[t.Item], t.ItemN))
		if note := g.spoilNote(tx, ty); note != "" {
			lines = append(lines, note)
		}
	}
	if len(lines) == 0 {
		return
	}
	s := g.scale
	w := 0.0
	for _, l := range lines {
		lw, _ := text.Measure(l, g.fontS, 0)
		w = math.Max(w, lw)
	}
	h := float64(len(lines))*17*s + 8*s
	x, y := math.Min(mx+16*s, float64(g.sw)-w-20*s), math.Min(my+16*s, float64(g.sh)-g.barH()-h-4*s)
	rect(dst, x, y, w+12*s, h, color.NRGBA{12, 14, 20, 235})
	strokeRect(dst, x, y, w+12*s, h, 1, color.NRGBA{120, 130, 160, 255})
	for i, l := range lines {
		c := color.Color(color.White)
		if i == 0 {
			c = color.NRGBA{255, 235, 150, 255}
		} else if contains(l, "quickly") || (len(l) > 7 && (l[:7] == "waiting" || l[:2] == "no" || l[:3] == "not" || l[:3] == "out")) {
			c = color.NRGBA{255, 150, 120, 255}
		}
		g.text(dst, l, x+6*s, y+4*s+float64(i)*17*s, g.fontS, c)
	}
}

// cropTip says what grows on a farm tile and how far along it is.
func (g *Game) cropTip(t *Tile, x, y int) []string {
	def := cropDefs[t.CropKind]
	if !t.Crop {
		return []string{"Farm: " + def.Name + " (not sown yet)"}
	}
	if t.Grow >= 1 {
		return []string{def.Name + ": ripe, ready to harvest"}
	}
	lines := []string{fmt.Sprintf("%s: %.0f%% grown", def.Name, t.Grow*100)}
	if T := g.tempAt(x, y); T < def.MinT {
		lines = append(lines, fmt.Sprintf("not growing: %.0f° is below %.0f°", T, def.MinT))
	}
	return lines
}

// roomTip describes an enclosed room: whether it is roofed and heated.
func (g *Game) roomTip(x, y int) []string {
	if x < 0 || y < 0 || x >= MapW || y >= MapH {
		return nil
	}
	r := g.roomOf[y*MapW+x]
	if r < 0 || !g.rooms[r].Indoor {
		return nil
	}
	if !g.W.T[y*MapW+x].Roof {
		return []string{"Room: walled but no roof", "not indoors: no warmth, goods spoil (Roof tool, R)"}
	}
	heated := false
	for _, h := range g.heaters {
		if g.roomOf[h] == r {
			heated = true
			break
		}
	}
	line := "Room: roofed, not heated"
	if heated {
		line = "Room: roofed and heated"
	}
	return []string{line, fmt.Sprintf("%.0f° inside (%.0f° outside)", g.tempAt(x, y), g.outT)}
}

// desigColor is the outline colour for a pending order.
func desigColor(d Desig) color.NRGBA {
	switch d {
	case DDecon:
		return toolInfo[TDeconstruct].Col
	case DRoof:
		return toolInfo[TRoof].Col
	}
	return toolInfo[Tool(d)].Col // DChop..DHarvest line up with TChop..THarvest
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// drawWeather paints falling rain or snow over the map, with a grey cast on the light.
func (g *Game) drawWeather(dst *ebiten.Image) {
	if g.precip < 0.02 {
		return
	}
	p := g.precip
	W, H := float64(g.sw), float64(g.sh)
	rect(dst, 0, 0, W, H, color.NRGBA{40, 52, 70, uint8(p * 55)})
	s := g.scale
	n := int(260 * p)
	frac := func(i int, k float64) float64 { // cheap per-drop pseudo random in [0,1)
		v := math.Sin(float64(i)*12.9898+k*78.233) * 43758.5453
		return v - math.Floor(v)
	}
	if g.outT < 1 { // snow: slow, drifting flakes
		for i := 0; i < n; i++ {
			x := frac(i, 1)*W + math.Sin(g.time*0.8+float64(i))*14*s
			y := math.Mod(frac(i, 2)*H+g.time*(40+30*frac(i, 3))*s, H)
			x = math.Mod(x+W, W)
			r := (1 + 1.5*frac(i, 4)) * s
			rect(dst, x, y, r, r, color.NRGBA{240, 244, 252, uint8(120 + 100*frac(i, 5))})
		}
		return
	}
	for i := 0; i < n; i++ { // rain: fast slanted streaks
		x := frac(i, 1) * (W + 60*s)
		y := math.Mod(frac(i, 2)*H+g.time*(520+200*frac(i, 3))*s, H)
		l := (9 + 8*frac(i, 4)) * s
		vector.StrokeLine(dst, float32(x), float32(y), float32(x-l*0.25), float32(y+l), 1, color.NRGBA{170, 195, 235, uint8(90 + 90*frac(i, 5))}, false)
	}
}

// drawUrgentMark paints a red badge with an exclamation mark in a size x size box at x, y.
func (g *Game) drawUrgentMark(dst *ebiten.Image, x, y, size float64) {
	vector.FillCircle(dst, float32(x+size/2), float32(y+size/2), float32(size/2), color.NRGBA{225, 50, 40, 255}, true)
	vector.StrokeCircle(dst, float32(x+size/2), float32(y+size/2), float32(size/2), 1, color.NRGBA{255, 230, 200, 255}, true)
	rect(dst, x+size*0.44, y+size*0.2, size*0.12, size*0.38, color.White)
	rect(dst, x+size*0.44, y+size*0.67, size*0.12, size*0.12, color.White)
}
