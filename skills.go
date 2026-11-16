package main

import "math"

// ---------- skills ----------

type Skill int

const (
	SCombat Skill = iota
	SConstruction
	SMining
	SFarming
	SMedicine
	SCrafting
	SEngineering
	SCooking
	numSkills
)

var skillNames = [numSkills]string{"Combat", "Construction", "Mining", "Farming", "Medicine", "Crafting", "Engineering", "Cooking"}

const maxSkill = 20

type SkillState struct {
	Level   int
	XP      float64
	Passion bool // learns twice as fast
}

// xpNeeded is the XP to go from level l to l+1.
func xpNeeded(l int) float64 { return 50 + 35*float64(l) }

// ---------- traits ----------

type Trait int

const (
	TrNone Trait = iota
	TrHardy
	TrFrail
	TrFastLearner
	TrBrawler
	TrGreenThumb
	TrTinkerer
	TrGlutton
	TrLightEater
	TrNimble
	TrCharming
	numTraits
)

var traitName = [numTraits]string{"", "Hardy", "Frail", "Fast learner", "Brawler", "Green thumb", "Tinkerer", "Glutton", "Light eater", "Nimble", "Charming"}
var traitDesc = [numTraits]string{
	"", "Takes 40% less cold damage", "Takes 25% more damage", "Learns 50% faster", "Hits 30% harder",
	"Plants and harvests faster, bigger yields", "Crafts and engineers faster", "Gets hungry 30% faster",
	"Gets hungry 25% slower", "Walks 15% faster", "Better trade prices",
}

func (c *Colonist) has(t Trait) bool {
	for _, x := range c.Traits {
		if x == t {
			return true
		}
	}
	return false
}

func (c *Colonist) lvl(s Skill) int { return c.Skills[s].Level }

// rate is the work-speed multiplier a skill gives (level 0 = 0.85, level 10 = 1.25).
func (c *Colonist) rate(s Skill) float64 {
	r := 0.85 + 0.04*float64(c.Skills[s].Level)
	if (s == SCrafting || s == SEngineering) && c.has(TrTinkerer) {
		r *= 1.25
	}
	if s == SFarming && c.has(TrGreenThumb) {
		r *= 1.3
	}
	return r
}

// gainXP trains a skill; passion and traits speed it up. Returns true on a level-up.
func (g *Game) gainXP(c *Colonist, s Skill, amt float64) bool {
	st := &c.Skills[s]
	if st.Level >= maxSkill {
		return false
	}
	if st.Passion {
		amt *= 2
	}
	if c.has(TrFastLearner) {
		amt *= 1.5
	}
	st.XP += amt
	up := false
	for st.Level < maxSkill && st.XP >= xpNeeded(st.Level) {
		st.XP -= xpNeeded(st.Level)
		st.Level++
		up = true
		g.msg("%s's %s is now level %d!", c.Name, skillNames[s], st.Level)
		g.emit("levelup", c.X, c.Y)
	}
	return up
}

// bestSkill returns the highest level any colonist has in a skill.
func (g *Game) bestSkill(s Skill) int {
	b := 0
	for _, c := range g.cols {
		b = max(b, c.Skills[s].Level)
	}
	return b
}

// ---------- creating colonists ----------

var exclusive = map[Trait]Trait{TrHardy: TrFrail, TrFrail: TrHardy, TrGlutton: TrLightEater, TrLightEater: TrGlutton}

// rollColonist gives a new colonist traits and skills. specialty (>=0) guarantees a
// standout skill at the given level.
func (g *Game) rollColonist(c *Colonist, specialty Skill, specLevel int) {
	c.Traits = nil
	n := 1 + g.rng.Intn(2)
	for tries := 0; len(c.Traits) < n && tries < 20; tries++ {
		t := Trait(1 + g.rng.Intn(int(numTraits)-1))
		if c.has(t) || (exclusive[t] != TrNone && c.has(exclusive[t])) {
			continue
		}
		c.Traits = append(c.Traits, t)
	}
	for s := Skill(0); s < numSkills; s++ {
		c.Skills[s] = SkillState{Level: g.rng.Intn(4)}
	}
	// two passions with a head start
	for k := 0; k < 2; k++ {
		s := Skill(g.rng.Intn(int(numSkills)))
		c.Skills[s].Passion = true
		c.Skills[s].Level += 1 + g.rng.Intn(3)
	}
	if specLevel > 0 {
		c.Skills[specialty].Level = max(c.Skills[specialty].Level, specLevel)
		c.Skills[specialty].Passion = true
	}
	c.Focus = specialty
	if specLevel == 0 {
		for s := Skill(0); s < numSkills; s++ {
			if c.Skills[s].Passion {
				c.Focus = s
				break
			}
		}
	}
}

// bestAt names a colonist's strongest skill for announcements.
func (c *Colonist) bestAt() (Skill, int) {
	bs, bl := Skill(0), -1
	for s := Skill(0); s < numSkills; s++ {
		if c.Skills[s].Level > bl {
			bs, bl = s, c.Skills[s].Level
		}
	}
	return bs, bl
}

func (c *Colonist) traitList() string {
	out := ""
	for i, t := range c.Traits {
		if i > 0 {
			out += ", "
		}
		out += traitName[t]
	}
	if out == "" {
		return "no notable traits"
	}
	return out
}

// bernoulli rolls a chance in [0,1].
func (g *Game) chance(p float64) bool { return g.rng.Float64() < math.Max(0, math.Min(1, p)) }

// ---------- trait effects ----------

func (c *Colonist) hungerMult() float64 {
	switch {
	case c.has(TrGlutton):
		return 1.3
	case c.has(TrLightEater):
		return 0.75
	}
	return 1
}

func (c *Colonist) coldMult() float64 {
	m := 1.0
	if c.has(TrHardy) {
		m *= 0.6
	}
	if gd, ok := gearDefs[c.Wear]; ok && gd.Cold > 0 {
		m *= gd.Cold
	}
	return m
}

func (c *Colonist) speedMult() float64 {
	if c.has(TrNimble) {
		return 1.15
	}
	return 1
}

func (c *Colonist) damageTakenMult() float64 {
	if c.has(TrFrail) {
		return 1.25
	}
	return 1
}

// bonus returns 1 if the colonist has the trait (for flat additions).
func (c *Colonist) bonus(t Trait) int {
	if c.has(t) {
		return 1
	}
	return 0
}
