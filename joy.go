package main

import "math"

// Joy: colonists need some fun. Joy slowly drains; low joy slows work, and a colonist with none left
// eventually has a sulk and stops working for a while. Games (horseshoes, checkers) restore it, more if
// two people play together; meditating at a shrine restores some and leaves a calm that speeds work.
// A good meal also lifts the spirits.

type StationDef struct {
	Name    string
	Verb    string
	Joy     float64 // joy per second
	Time    float64 // seconds per session
	Partner bool    // playing with someone else is better
	Calm    bool    // leaves a calm that speeds up work
	Sound   string
}

var stationDefs = map[Obj]StationDef{
	OHorse:   {"horseshoe pit", "Playing horseshoes", 1.8, 22, true, false, "clang"},
	OChecker: {"checkers table", "Playing checkers", 1.5, 26, true, false, "click"},
	OShrine:  {"meditation shrine", "Meditating", 1.1, 24, false, true, "bell"},
}

const (
	joyDecay      = 0.08 // per second awake (about 5 days from full to empty)
	joyUrgent     = 40.0 // below this a colonist drops work to go and play
	joyIdle       = 75.0 // below this they play when they have nothing else to do
	calmSeconds   = 120.0
	sulkSeconds   = 25.0
	partnerBonus  = 1.5
	calmWorkBonus = 1.1
)

// moodFactor scales work speed by spirits: full speed above 50 joy, down to 80% at none.
func (c *Colonist) moodFactor() float64 {
	m := 1.0
	if c.Joy < 50 {
		m = 0.8 + 0.2*math.Max(0, c.Joy)/50
	}
	if c.Calm > 0 {
		m *= calmWorkBonus
	}
	return m
}

func (g *Game) addJoy(c *Colonist, n float64) { c.Joy = math.Min(100, math.Max(0, c.Joy+n)) }

// foodJoy is the cheer a bite of food brings.
func foodJoy(it Item) float64 {
	switch it {
	case IBread:
		return 4
	case IMeal:
		return 12
	}
	return 0
}

// updateJoy drains joy, runs out calm, and starts or ends sulks.
func (g *Game) updateJoy(c *Colonist, dt float64) {
	rate := joyDecay
	if (c.Job.Kind == JSleep || c.Job.Kind == JRecover) && len(c.Path) == 0 {
		rate *= 0.3
	}
	if c.Job.Kind != JPlay {
		g.addJoy(c, -rate*dt)
	}
	if c.Calm > 0 {
		c.Calm = math.Max(0, c.Calm-dt)
	}
	if c.Sulk > 0 {
		c.Sulk -= dt
		if c.Sulk <= 0 {
			c.Sulk = 0
			c.Joy = math.Max(c.Joy, 25)
			g.msg("%s has stopped sulking.", c.Name)
		}
		return
	}
	if c.Joy <= 0.5 && g.chance(dt*0.02) {
		c.Sulk = sulkSeconds
		g.stats.Sulks++
		g.abort(c)
		g.msg("%s is in a foul mood and has stopped working. A game or some quiet time would help.", c.Name)
	}
}

// stationFree reports whether a tile is a station with a free place.
func stationFree(t *Tile, self bool) bool {
	if _, ok := stationDefs[t.Obj]; !ok || t.Fire > 0 || (t.ResT != 0 && t.ResD != 0) {
		return false
	}
	return !self || !solidBlue(objBlue[t.Obj])
}

// tryPlay sends a colonist to a game or the shrine. Urgent means drop everything: spirits are low.
func (g *Game) tryPlay(c *Colonist, urgent bool) bool {
	limit := joyIdle
	if urgent {
		limit = joyUrgent
	}
	if c.Sulk > 0 || c.Joy >= limit || c.Rest < 20 || c.Hunger < 40 || g.fireCount > 0 || g.threatNear(c, 30) {
		return false // (no games while there is danger about)
	}
	x, y := g.tileOf(c)
	p, ok := g.W.bfs(x, y, 60, func(a, b int) bool {
		_, f := g.findTarget(a, b, func(t *Tile, _, _ int, self bool) bool { return stationFree(t, self) })
		return f
	})
	if !ok {
		return false
	}
	e := endOf(p, x, y)
	tg, _ := g.findTarget(e.x, e.y, func(t *Tile, _, _ int, self bool) bool { return stationFree(t, self) })
	t := g.W.At(tg.x, tg.y)
	slot := 0
	if t.ResT != 0 {
		slot, t.ResD = 1, c.ID
	} else {
		t.ResT = c.ID
	}
	def := stationDefs[t.Obj]
	c.Job = Job{Kind: JPlay, Target: tg, Stage: slot, Need: def.Time}
	c.Path, c.Status = p, "Going to the "+def.Name
	return true
}

// partnerOf finds the colonist playing at the same station, if any.
func (g *Game) partnerOf(c *Colonist) *Colonist {
	for _, o := range g.cols {
		if o != c && o.Job.Kind == JPlay && o.Job.Target == c.Job.Target && len(o.Path) == 0 {
			return o
		}
	}
	return nil
}

func (g *Game) doPlay(c *Colonist, dt float64) {
	j := &c.Job
	t := g.W.At(j.Target.x, j.Target.y)
	def, ok := stationDefs[t.Obj]
	if !ok {
		g.abort(c)
		return
	}
	c.Status = def.Verb
	gain := def.Joy
	if def.Partner && g.partnerOf(c) != nil {
		gain *= partnerBonus
	}
	g.addJoy(c, gain*dt)
	j.Work += dt
	if j.Work-j.Last > 2.5 {
		j.Last = j.Work
		g.emit(def.Sound, float64(j.Target.x)+0.5, float64(j.Target.y)+0.5)
	}
	if j.Work >= j.Need {
		if def.Calm {
			c.Calm = calmSeconds
			g.msg("%s feels calm and focused.", c.Name)
		}
		g.stats.Played++
		g.finish(c)
	}
}

// threatNear reports whether a wolf or raider is within r tiles of the colonist.
func (g *Game) threatNear(c *Colonist, r float64) bool {
	for _, a := range g.animals {
		if a.hostile() && math.Hypot(a.X-c.X, a.Y-c.Y) < r {
			return true
		}
	}
	return false
}
