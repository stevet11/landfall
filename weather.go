package main

// Weather: spells of precipitation come and go. Warm precipitation is rain, cold is snow
// (the ground only turns white when it is cold enough, see draw.go). Weather is not saved:
// a loaded game simply starts with a fresh spell of whatever the season brings.

const (
	rainGrowth    = 1.5  // outdoor crops grow this much faster while it rains
	rainSolar     = 0.3  // solar panels keep this fraction of their output under cloud
	rainSpoil     = 1.5  // unroofed goods rot faster when wet
	rainWalk      = 0.9  // walking speed outdoors in rain or snow
	rainChill     = 4.0  // degrees colder for someone standing in the rain
	snowChill     = 2.0  // ... or in falling snow
	rainDouse     = 0.12 // chance per second that rain puts out an exposed burning tile
	weatherPrecip = 0.3  // intensity above which it counts as precipitating
)

// weatherStep advances the weather clock and eases the intensity towards its goal.
func (g *Game) weatherStep(dt float64) {
	g.weatherLeft -= dt
	if g.weatherLeft <= 0 {
		g.wet = !g.wet
		if g.wet {
			g.weatherLeft = DayLen * (0.12 + 0.3*g.rng.Float64())
		} else {
			g.weatherLeft = DayLen * (0.5 + 1.1*g.rng.Float64())
		}
		if g.wet && g.precip < weatherPrecip {
			if g.outT < 1 {
				g.msg("It starts to snow.")
			} else {
				g.msg("It starts to rain.")
			}
		}
	}
	goal := 0.0
	if g.wet {
		goal = 1
	}
	g.precip += (goal - g.precip) * minf(1, dt*0.25)
	if g.precip < 0.01 && goal == 0 {
		g.precip = 0
	}
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// precipitating reports whether rain or snow is actually falling.
func (g *Game) precipitating() bool { return g.precip >= weatherPrecip }

func (g *Game) raining() bool         { return g.precipitating() && g.outT >= 1 }
func (g *Game) snowing() bool         { return g.precipitating() && g.outT < 1 }
func (g *Game) exposed(x, y int) bool { return !g.covered(x, y) && !g.indoors(x, y) }

// weatherName is shown in the top bar ("" when it is dry).
func (g *Game) weatherName() string {
	switch {
	case g.raining():
		return "Rain"
	case g.snowing():
		return "Snow"
	}
	return ""
}

// wetChill is how many degrees colder the weather makes it for someone standing at x, y.
func (g *Game) wetChill(x, y int) float64 {
	if !g.precipitating() || !g.exposed(x, y) {
		return 0
	}
	if g.outT < 1 {
		return snowChill
	}
	return rainChill
}

// weatherWalk scales walking speed outdoors.
func (g *Game) weatherWalk(x, y int) float64 {
	if g.precipitating() && g.exposed(x, y) {
		return rainWalk
	}
	return 1
}
