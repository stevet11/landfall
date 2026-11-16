package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// checkGameOver ends the game once every colonist has died. The sim stops (Update no longer
// steps it) and the game-over window offers a fresh start.
func (g *Game) checkGameOver() {
	if g.over || len(g.cols) > 0 {
		return
	}
	g.over = true
	g.overDay = int(g.time/DayLen) + 1
	g.emit("death", -1, -1)
}

// deathSummary says how the colony fell, from the death counters.
func (g *Game) deathSummary() string {
	s := g.stats
	var parts []string
	for _, p := range []struct {
		n int
		w string
	}{{s.Starved, "starved"}, {s.Froze, "froze"}, {s.Heat, "died of heat"}, {s.Bled, "bled out"},
		{s.IllDeaths, "died of illness"}, {s.Killed, "were killed"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.w))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i, p := range parts[1:] {
		if i == len(parts)-2 {
			out += " and " + p
		} else {
			out += ", " + p
		}
	}
	return out
}

// restart throws the dead colony away and starts a new world, keeping window, fonts and sound.
func (g *Game) restart(seed int64) {
	old := *g
	*g = *newGame(seed)
	g.sw, g.sh, g.scale = old.sw, old.sh, old.scale
	g.audio, g.audioCh, g.lastFrame = old.audio, old.audioCh, old.lastFrame
	g.srcR, g.srcB = old.srcR, old.srcB
	g.makeFonts()
	g.onEvent = g.handleEvent
	g.showHelp = false
	g.autoT = 0
}

func (g *Game) restartNow() { g.restart(time.Now().UnixNano() % 100000) }

func (g *Game) overRect() (x, y, w, h float64) {
	s := g.scale
	w, h = math.Min(520*s, float64(g.sw)-30*s), 190*s
	return (float64(g.sw) - w) / 2, (float64(g.sh) - h) / 2, w, h
}

func (g *Game) restartBtn() btn {
	x, y, w, h := g.overRect()
	s := g.scale
	return btn{x: x + (w-200*s)/2, y: y + h - 54*s, w: 200 * s, h: 36 * s, label: "New colony (R)", fn: g.restartNow}
}

// updateGameOver handles input while the game is over; the world stays frozen.
func (g *Game) updateGameOver() {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.restartNow()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		g.quickLoad(quickSavePath(), "quicksave")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		g.quickLoad(autoSavePath(), "autosave")
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if b := g.restartBtn(); b.hit(float64(mx), float64(my)) {
			g.audio.play("click", 0.4)
			b.fn()
		}
	}
}

func (g *Game) drawGameOver(dst *ebiten.Image) {
	if !g.over {
		return
	}
	s := g.scale
	rect(dst, 0, 0, float64(g.sw), float64(g.sh), color.NRGBA{0, 0, 0, 150})
	x, y, w, h := g.overRect()
	rect(dst, x, y, w, h, color.NRGBA{12, 14, 20, 245})
	strokeRect(dst, x, y, w, h, 2, color.NRGBA{200, 80, 70, 255})
	g.text(dst, "THE COLONY HAS FALLEN", x+16*s, y+14*s, g.fontM, color.NRGBA{255, 110, 100, 255})
	white := color.Color(color.White)
	dim := color.NRGBA{170, 178, 196, 255}
	g.text(dst, fmt.Sprintf("All your colonists are dead. They lasted until day %d (%s).", g.overDay, g.seasonName()), x+16*s, y+50*s, g.fontS, white)
	if d := g.deathSummary(); d != "" {
		g.text(dst, "Deaths: "+d+".", x+16*s, y+74*s, g.fontS, dim)
	}
	g.text(dst, "Start again with a new colony? (F9 / F10 load the quicksave / autosave.)", x+16*s, y+104*s, g.fontS, dim)
	mx, my := ebiten.CursorPosition()
	g.drawBtn(dst, g.restartBtn(), float64(mx), float64(my))
}
