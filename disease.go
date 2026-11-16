package main

import (
	"fmt"
	"math"
)

// Disease: outbreaks of flu and marsh fever that spread between colonists who stay close, and
// food poisoning from eating rotten food. Each illness is a race between how bad it gets (severity)
// and how fast the body fights it off (immunity). Rest in a bed, medicine from a skilled medic, warmth
// and food all tilt the race; cold, hunger and working while sick tilt it the other way. At high
// severity the illness damages health, and at 100% it kills. Recovery brings immunity for a while.

type Disease int

const (
	DisFlu Disease = iota
	DisFever
	DisPoison
	numDiseases
)

type DiseaseDef struct {
	Name       string
	Contagious bool
	Spread     float64 // chance per second of passing it on to someone in close contact
	Grow       float64 // severity gained per second
	Imm        float64 // immunity gained per second
	Immunity   float64 // seconds of immunity after recovering
}

var diseaseDefs = [numDiseases]DiseaseDef{
	{"Flu", true, 0.007, 0.0015, 0.0038, 600},
	{"Marsh fever", true, 0.010, 0.0050, 0.0030, 2400},
	{"Food poisoning", false, 0, 0.0030, 0.0060, 0},
}

// Illness is one sickness a colonist has. Tended: 0 none, 1 basic care, 2 medicine.
type Illness struct {
	Kind    Disease
	Sev     float64 // 0..1: at 1 it is fatal
	Imm     float64 // 0..1: at 1 the body has won
	Tended  int
	Quality float64
}

const (
	critSev       = 0.8  // above this the illness starts damaging health
	contactRadius = 2.2  // tiles within which a sick colonist can infect someone
	roomContact   = 1.8  // being in the same enclosed room makes it spread faster
	contagionMin  = 0.05 // very early (and nearly cured) cases barely spread
)

func (c *Colonist) hasIll(d Disease) bool {
	for _, i := range c.Ill {
		if i.Kind == d {
			return true
		}
	}
	return false
}

func (c *Colonist) sick() bool { return len(c.Ill) > 0 }

// sickSeverity is the worst illness severity (0 when healthy).
func (c *Colonist) sickSeverity() float64 {
	s := 0.0
	for _, i := range c.Ill {
		s = math.Max(s, i.Sev)
	}
	return s
}

// illWork is the work-speed multiplier of someone feverish (1 = healthy).
func (c *Colonist) illWork() float64 { return 1 - 0.4*c.sickSeverity() }

// illSpeed is the walking-speed multiplier.
func (c *Colonist) illSpeed() float64 { return 1 - 0.25*c.sickSeverity() }

func (c *Colonist) worstIllness() *Illness {
	var w *Illness
	for i := range c.Ill {
		if w == nil || c.Ill[i].Sev > w.Sev {
			w = &c.Ill[i]
		}
	}
	return w
}

// infect gives a colonist a disease unless they are immune or already have it.
func (g *Game) infect(c *Colonist, d Disease) bool {
	if c.Dead || c.hasIll(d) || (int(d) < len(c.Immune) && c.Immune[d] > 0) || len(c.Ill) >= 2 {
		return false
	}
	c.Ill = append(c.Ill, Illness{Kind: d})
	g.stats.Illnesses++
	g.emit("cough", c.X, c.Y)
	return true
}

// resting reports how well the colonist is resting: 2 in a bed, 1 on the ground, 0 not resting.
func (c *Colonist) resting() int {
	if (c.Job.Kind == JSleep || c.Job.Kind == JRecover) && len(c.Path) == 0 {
		if c.Job.Stage == 0 {
			return 2
		}
		return 1
	}
	return 0
}

// careMults says how care changes the race: multipliers on severity growth and immunity gain.
func careMults(ill *Illness) (grow, imm float64) {
	q := math.Max(1, ill.Quality)
	switch ill.Tended {
	case 2:
		return 0.4 / math.Sqrt(q), 1.9 * math.Pow(q, 0.3)
	case 1:
		return 0.85, 1.15
	}
	return 1, 1
}

// updateIllness advances every illness one tick.
func (g *Game) updateIllness(c *Colonist, dt float64) {
	for i := range c.Immune {
		if c.Immune[i] > 0 {
			c.Immune[i] -= dt
		}
	}
	if len(c.Ill) == 0 {
		return
	}
	if g.chance(dt * 0.025) {
		g.emit("cough", c.X, c.Y)
	}
	for i := 0; i < len(c.Ill); {
		ill := &c.Ill[i]
		def := diseaseDefs[ill.Kind]
		gm, im := careMults(ill)
		switch c.resting() {
		case 2: // in a bed
			gm, im = gm*0.75, im*1.7
		case 1: // lying on the ground
			gm, im = gm*0.85, im*1.25
		}
		if c.Temp < 5 {
			gm *= 1.3 // the cold makes it worse
		}
		if c.Hunger < 30 {
			gm, im = gm*1.3, im*0.8 // so does going hungry
		}
		ill.Sev = math.Min(1, ill.Sev+def.Grow*gm*dt)
		ill.Imm += def.Imm * im * dt
		if crit := (ill.Sev - critSev) / (1 - critSev); crit > 0 {
			c.Health -= 0.5 * crit * dt
		}
		if ill.Imm >= 1 {
			c.Immune[ill.Kind] = def.Immunity
			g.msg("%s recovered from %s.", c.Name, lowerFirst(def.Name))
			g.stats.Recovered++
			c.Ill = append(c.Ill[:i], c.Ill[i+1:]...)
			continue
		}
		i++
	}
}

func lowerFirst(s string) string {
	if s == "Flu" {
		return "the flu"
	}
	if len(s) == 0 {
		return s
	}
	return string(s[0]+32) + s[1:]
}

// spreadDisease lets sick colonists pass contagious illnesses to those close to them.
func (g *Game) spreadDisease(dt float64) {
	for _, s := range g.cols {
		for _, ill := range s.Ill {
			def := diseaseDefs[ill.Kind]
			if !def.Contagious || ill.Imm > 0.9 || ill.Sev+ill.Imm < contagionMin {
				continue
			}
			sx, sy := g.tileOf(s)
			for _, h := range g.cols {
				if h == s || h.hasIll(ill.Kind) || h.Immune[ill.Kind] > 0 {
					continue
				}
				d := math.Hypot(h.X-s.X, h.Y-s.Y)
				factor := 1.0
				hx, hy := g.tileOf(h)
				if d > contactRadius {
					// not close: but people shut in the same small room still breathe the same air
					same := g.indoors(sx, sy) && g.indoors(hx, hy) && g.roomOf[sy*MapW+sx] == g.roomOf[hy*MapW+hx] && d < 6
					if !same {
						continue
					}
				} else if g.indoors(sx, sy) && g.indoors(hx, hy) {
					factor = roomContact
				}
				if g.chance(def.Spread * factor * dt) {
					if g.infect(h, ill.Kind) {
						g.msg("%s caught %s from %s.", h.Name, lowerFirst(def.Name), s.Name)
					}
				}
			}
		}
	}
}

// exposeDoctor is the risk a medic takes when tending someone contagious.
func (g *Game) exposeDoctor(doc, patient *Colonist) {
	for _, ill := range patient.Ill {
		def := diseaseDefs[ill.Kind]
		if def.Contagious && doc != patient && g.chance(0.25*(1-0.05*float64(doc.lvl(SMedicine)))) {
			if g.infect(doc, ill.Kind) {
				g.msg("%s caught %s while nursing %s.", doc.Name, lowerFirst(def.Name), patient.Name)
			}
		}
	}
}

// illNeedsCare: someone is sick enough that a medic should look at them.
func (g *Game) illNeedsCare(p *Colonist) bool {
	for _, ill := range p.Ill {
		if (ill.Tended == 0 && ill.Sev > 0.08) || (ill.Tended < 2 && ill.Sev > 0.3 && g.stock[IMedicine] > 0) {
			return true
		}
	}
	return false
}

// careForIllness is what a medic does at the bedside; returns true if an illness was treated.
func (g *Game) careForIllness(doc, p *Colonist) bool {
	var worst *Illness
	for i := range p.Ill {
		ill := &p.Ill[i]
		need := (ill.Tended == 0 && ill.Sev > 0.08) || (ill.Tended < 2 && ill.Sev > 0.3 && g.stock[IMedicine] > 0)
		if need && (worst == nil || ill.Sev > worst.Sev) {
			worst = ill
		}
	}
	if worst == nil {
		return false
	}
	g.exposeDoctor(doc, p)
	q := 1 + 0.06*float64(doc.lvl(SMedicine))
	name := diseaseDefs[worst.Kind].Name
	if (worst.Sev > 0.25 || worst.Kind == DisFever) && g.takeStock(IMedicine, 1) {
		worst.Tended, worst.Quality = 2, q
		g.msg("%s treated %s's %s with medicine.", doc.Name, p.Name, lowerFirst(name))
		g.emit("heal", p.X, p.Y)
	} else if worst.Tended == 0 {
		worst.Tended, worst.Quality = 1, q
		g.msg("%s is nursing %s.", doc.Name, p.Name)
	}
	return true
}

// ---------- outbreaks ----------

// pickOutbreak chooses a disease to strike, weighted by season and how far into the game it is.
func (g *Game) pickOutbreak() Disease { return g.pickOutbreakIn(g.seasonName()) }

func (g *Game) pickOutbreakIn(season string) Disease {
	flu, fever := 1.0, 0.0
	switch season {
	case "Winter":
		flu = 3
	case "Autumn":
		flu = 1.5
	case "Summer":
		flu = 0.3
	}
	if g.time/DayLen >= 7 {
		fever = 0.8
		if season == "Spring" || season == "Summer" {
			fever = 1.8
		}
	}
	if g.rng.Float64()*(flu+fever) < flu {
		return DisFlu
	}
	return DisFever
}

// updateOutbreaks starts an outbreak now and then.
func (g *Game) updateOutbreaks() {
	if g.time < g.nextOutbreak || len(g.cols) < 2 {
		return
	}
	g.nextOutbreak = g.time + DayLen*(3+g.rng.Float64()*3)
	d := g.pickOutbreak()
	var candidates []*Colonist
	for _, c := range g.cols {
		if !c.sick() && c.Immune[d] <= 0 {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return
	}
	c := candidates[g.rng.Intn(len(candidates))]
	if g.infect(c, d) {
		g.stats.Outbreaks++
		g.alert("Outbreak! %s has come down with %s. It spreads between people who stay close: put the sick to bed and tend them.", c.Name, lowerFirst(diseaseDefs[d].Name))
	}
}

// maybePoison: eating food that sat in the open on a warm day can make you ill.
func (g *Game) maybePoison(c *Colonist, it Item, onRoofedTile bool, temp float64) {
	if onRoofedTile || temp < 10 {
		return
	}
	p := 0.0
	switch it {
	case IFood, ICorn:
		p = 0.08
	case IBread, IMeal:
		p = 0.03
	}
	if g.chance(p) && g.infect(c, DisPoison) {
		g.msg("%s ate some bad food and feels sick.", c.Name)
	}
}

// sickLabel describes a colonist's illnesses for the panel.
func sickLine(ill Illness) string {
	care := []string{"untreated", "nursed", "on medicine"}[ill.Tended]
	line := fmt.Sprintf("%s %.0f%% (%s)", diseaseDefs[ill.Kind].Name, 100*ill.Sev, care)
	if ill.Sev >= critSev {
		line += " CRITICAL"
	}
	return line
}

// tendUrgency ranks how soon a patient needs a medic: bleeding wounds first, then dangerous
// illness, then everything else that needs care.
func (g *Game) tendUrgency(p *Colonist) float64 {
	u := 1.0
	for _, in := range p.Injuries {
		if in.Tended == 0 && in.Bleed {
			u = math.Max(u, 3)
		}
	}
	if p.sickSeverity() > 0.5 {
		u = math.Max(u, 2)
	}
	return u
}
