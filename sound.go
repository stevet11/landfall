package main

import (
	"bytes"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// All sound is synthesised at startup: no audio files are shipped.

const sampleRate = 44100

type synthBuf []float64

func gen(dur float64, f func(t float64) float64) synthBuf {
	n := int(dur * sampleRate)
	b := make(synthBuf, n)
	for i := range b {
		b[i] = f(float64(i) / sampleRate)
	}
	return b
}

func (b synthBuf) pcm(gain float64) []byte {
	out := make([]byte, len(b)*4)
	for i, v := range b {
		v *= gain
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		s := int16(v * 32767)
		lo, hi := byte(s), byte(s>>8)
		out[i*4], out[i*4+1], out[i*4+2], out[i*4+3] = lo, hi, lo, hi
	}
	return out
}

// add mixes src into b starting at offset seconds.
func (b synthBuf) add(src synthBuf, at float64, gain float64) {
	o := int(at * sampleRate)
	for i, v := range src {
		if o+i < len(b) {
			b[o+i] += v * gain
		}
	}
}

type lowpass struct{ y, a float64 }

func (l *lowpass) f(x float64) float64 { l.y += l.a * (x - l.y); return l.y }

func sine(f, t float64) float64 { return math.Sin(2 * math.Pi * f * t) }

func synthAll() map[string][]byte {
	r := rand.New(rand.NewSource(1))
	nz := func() float64 { return r.Float64()*2 - 1 }
	out := map[string][]byte{}

	{ // wood chop: a bright, dry "tock" - no low end
		hp := &lowpass{a: 0.12}
		out["chop"] = gen(0.16, func(t float64) float64 {
			n := nz()
			crack := (n - hp.f(n)) * math.Exp(-t*110) // high-passed noise transient
			wood := (sine(780, t)*0.6 + sine(1180, t)*0.35 + sine(1730, t)*0.2) * math.Exp(-t*48)
			return crack*0.8 + wood*0.45
		}).pcm(0.75)
	}
	out["mine"] = gen(0.3, func(t float64) float64 {
		m := sine(2100, t) + 0.6*sine(3330, t) + 0.5*sine(990, t)
		return m*math.Exp(-t*20)*0.3 + nz()*math.Exp(-t*130)*0.4
	}).pcm(0.8)
	out["hammer"] = gen(0.16, func(t float64) float64 {
		return sine(430, t)*math.Exp(-t*42)*0.7 + sine(1290, t)*math.Exp(-t*60)*0.25 + nz()*math.Exp(-t*160)*0.4
	}).pcm(0.8)
	{
		lp := &lowpass{a: 0.3}
		out["rustle"] = gen(0.4, func(t float64) float64 {
			w := math.Sin(math.Pi * t / 0.4)
			return lp.f(nz()) * w * w * 0.7 * (0.6 + 0.4*math.Sin(t*60))
		}).pcm(0.7)
	}
	{
		lp := &lowpass{a: 0.22}
		out["hit"] = gen(0.18, func(t float64) float64 {
			return lp.f(nz())*math.Exp(-t*26)*0.9 + sine(95, t)*math.Exp(-t*22)*0.7
		}).pcm(0.9)
	}
	out["bite"] = gen(0.28, func(t float64) float64 {
		ph := 320*t - 500*t*t
		return math.Sin(2*math.Pi*ph)*math.Exp(-t*14)*0.5 + nz()*math.Exp(-t*40)*0.5
	}).pcm(0.9)
	out["yelp"] = gen(0.4, func(t float64) float64 {
		ph := 1000*t - 1200*t*t + 8*math.Sin(2*math.Pi*14*t)/14
		return math.Sin(2*math.Pi*ph)*math.Exp(-t*7)*0.5 + 0.2*math.Sin(4*math.Pi*ph)*math.Exp(-t*9)
	}).pcm(0.7)
	{
		lp := &lowpass{a: 0.1}
		out["thud"] = gen(0.35, func(t float64) float64 {
			return sine(65, t)*math.Exp(-t*11)*0.9 + lp.f(nz())*math.Exp(-t*30)*0.5
		}).pcm(0.9)
	}
	out["death"] = gen(1.1, func(t float64) float64 {
		ph := 230*t - 90*t*t
		return (math.Sin(2*math.Pi*ph) + 0.4*math.Sin(2*math.Pi*ph*1.19)) * math.Exp(-t*3) * 0.35
	}).pcm(0.8)
	{ // wolf howl: fast rise, held plateau, long falling tail, breathy and harmonic-rich
		howl := func(dur, base, wobble float64, seed int64) synthBuf {
			hr := rand.New(rand.NewSource(seed))
			n := int(dur * sampleRate)
			b := make(synthBuf, n)
			ph, drift, tgt := 0.0, 0.0, 0.0
			bp1, bp2 := &lowpass{a: 0.22}, &lowpass{a: 0.06}
			for i := range b {
				t := float64(i) / sampleRate
				u := t / dur
				// pitch contour (relative to base)
				var c float64
				switch {
				case u < 0.17: // swoop up
					c = 0.62 + 0.38*math.Sin(u/0.17*math.Pi/2)
				case u < 0.52: // hold, creeping up
					c = 1 + 0.16*(u-0.17)/0.35
				default: // long glide down
					v := (u - 0.52) / 0.48
					c = 1.16 - 0.5*v - 0.12*v*v
				}
				if i%2205 == 0 { // slow random drift, like a living voice
					tgt = (hr.Float64()*2 - 1) * 9
				}
				drift += (tgt - drift) * 0.002
				vib := math.Min(1, math.Max(0, (u-0.2)/0.2)) * wobble * math.Sin(2*math.Pi*5.2*t)
				f := base*c + drift + vib
				ph += 2 * math.Pi * f / sampleRate
				// harmonics shaped by two vowel-like formants (~650 Hz and ~1150 Hz)
				v := 0.0
				for k := 1; k <= 12; k++ {
					fk := f * float64(k)
					amp := math.Exp(-math.Pow((fk-650)/380, 2)) + 0.6*math.Exp(-math.Pow((fk-1150)/420, 2)) + 0.08
					v += math.Sin(ph*float64(k)) * amp / float64(k)
				}
				env := math.Min(1, t/0.1) * math.Min(1, (dur-t)/0.7)
				breath := (bp1.f(nz()) - bp2.f(nz())) * 0.9 * (0.5 + 0.5*math.Sin(u*math.Pi))
				b[i] = (v*0.55 + breath*0.35) * env
			}
			return b
		}
		mix := make(synthBuf, int(5.4*sampleRate))
		mix.add(howl(3.4, 430, 9, 1), 0, 0.8)
		mix.add(howl(3.0, 520, 11, 2), 2.0, 0.5) // a second wolf answers once the first has finished
		out["howl"] = mix.pcm(0.7)
	}
	{
		b := make(synthBuf, int(0.9*sampleRate))
		for i, f := range []float64{523.25, 659.25, 783.99, 1046.5} {
			b.add(gen(0.5, func(t float64) float64 { return (sine(f, t) + 0.25*sine(f*2, t)) * math.Exp(-t*7) * 0.3 }), float64(i)*0.08, 1)
		}
		out["levelup"] = b.pcm(0.7)
	}
	{ // trader: a bright little jingle of bells, easy to tell apart from everything else
		b := make(synthBuf, int(1.6*sampleRate))
		for i, f := range []float64{1318.5, 1568, 2093, 1568, 2093, 2637} {
			bell := func(t float64) float64 {
				return (sine(f, t) + 0.5*sine(f*2.76, t) + 0.25*sine(f*5.4, t)) * math.Exp(-t*5.5)
			}
			b.add(gen(0.9, func(t float64) float64 { return bell(t) * 0.3 }), float64(i)*0.15, 1)
		}
		out["trader"] = b.pcm(0.7)
	}
	{ // fire: a soft roar with crackling pops
		n := int(1.6 * sampleRate)
		b := make(synthBuf, n)
		lp, lp2 := &lowpass{a: 0.05}, &lowpass{a: 0.4}
		var pop float64
		for i := range b {
			t := float64(i) / sampleRate
			w := math.Sin(math.Pi * t / 1.6)
			if r.Float64() < 0.0016 {
				pop = 0.5 + r.Float64()*0.5
			}
			pop *= 0.9985
			x := nz()
			b[i] = (lp.f(x)*3.2 + (x-lp2.f(x))*pop*0.9) * w * 0.35
		}
		out["fire"] = b.pcm(0.8)
	}
	{ // donkey: a rough "hee-haw" (rising then falling, with a rasp)
		n := int(1.5 * sampleRate)
		b := make(synthBuf, n)
		ph := 0.0
		rasp := &lowpass{a: 0.5}
		for i := range b {
			t := float64(i) / sampleRate
			var f, env float64
			if t < 0.55 { // hee
				u := t / 0.55
				f = 480 + 520*math.Sin(u*math.Pi/2)
				env = math.Min(1, t/0.05) * math.Min(1, (0.55-t)/0.08)
			} else if t > 0.65 && t < 1.45 { // haw
				u := (t - 0.65) / 0.8
				f = 420 - 190*u + 12*math.Sin(2*math.Pi*22*t)
				env = math.Min(1, (t-0.65)/0.05) * math.Min(1, (1.45-t)/0.2)
			} else {
				continue
			}
			ph += 2 * math.Pi * f / sampleRate
			v := math.Sin(ph) + 0.6*math.Sin(2*ph) + 0.4*math.Sin(3*ph) + 0.25*math.Sin(5*ph)
			b[i] = (v*0.22 + rasp.f(nz())*0.18) * env
		}
		out["bray"] = b.pcm(0.7)
	}
	out["groan"] = gen(1.1, func(t float64) float64 { // camel: a low, gurgling grumble
		f := 105 - 25*t + 6*math.Sin(2*math.Pi*9*t)
		env := math.Min(1, t/0.08) * math.Min(1, (1.1-t)/0.35)
		return (sine(f, t) + 0.6*sine(f*2, t) + 0.3*sine(f*3.1, t)) * (0.7 + 0.3*math.Sin(2*math.Pi*17*t)) * env * 0.4
	}).pcm(0.8)
	{ // water thrown on a fire, and the steam as it dies
		lp := &lowpass{a: 0.18}
		out["splash"] = gen(0.3, func(t float64) float64 {
			return lp.f(nz()) * math.Exp(-t*11) * (0.7 + 0.3*math.Sin(2*math.Pi*38*t)) * 0.8
		}).pcm(0.8)
		hp := &lowpass{a: 0.35}
		out["hiss"] = gen(0.8, func(t float64) float64 {
			x := nz()
			return (x - hp.f(x)) * math.Exp(-t*4.5) * 0.5
		}).pcm(0.7)
	}
	{ // gunfire: a sharp crack over a low thump; the rifle is bigger and echoes
		hp := &lowpass{a: 0.4}
		out["shot"] = gen(0.4, func(t float64) float64 {
			x := nz()
			return (x-hp.f(x))*math.Exp(-t*55)*0.9 + sine(110, t)*math.Exp(-t*28)*0.5
		}).pcm(0.9)
		hp2 := &lowpass{a: 0.3}
		rb := make(synthBuf, int(1.0*sampleRate))
		crack := gen(0.6, func(t float64) float64 {
			x := nz()
			return (x-hp2.f(x))*math.Exp(-t*38)*1.0 + sine(72, t)*math.Exp(-t*13)*0.7
		})
		rb.add(crack, 0, 1)
		rb.add(crack, 0.22, 0.28) // echo off the hills
		rb.add(crack, 0.47, 0.12)
		out["rifle"] = rb.pcm(0.9)
		lp := &lowpass{a: 0.02}
		out["slash"] = gen(0.22, func(t float64) float64 { // a blade swish
			lp.a = 0.05 + 0.6*t/0.22
			w := math.Sin(math.Pi * t / 0.22)
			return lp.f(nz()) * w * w * 1.4
		}).pcm(0.7)
	}
	{ // a war horn: two long brass blasts
		blast := func(f, dur float64) synthBuf {
			n := int(dur * sampleRate)
			b := make(synthBuf, n)
			ph := 0.0
			for i := range b {
				t := float64(i) / sampleRate
				fi := f * (0.94 + 0.06*math.Min(1, t/0.12)) // the note scoops up into pitch
				ph += 2 * math.Pi * fi / sampleRate
				env := math.Min(1, t/0.06) * math.Min(1, (dur-t)/0.25)
				v := math.Sin(ph) + 0.7*math.Sin(2*ph) + 0.5*math.Sin(3*ph) + 0.35*math.Sin(4*ph) + 0.2*math.Sin(5*ph)
				b[i] = v * 0.22 * env
			}
			return b
		}
		hb := make(synthBuf, int(2.6*sampleRate))
		hb.add(blast(146.8, 1.0), 0, 1)
		hb.add(blast(130.8, 1.3), 1.15, 1)
		out["horn"] = hb.pcm(0.85)
	}
	{ // a couple of coughs
		lp := &lowpass{a: 0.25}
		b := make(synthBuf, int(0.8*sampleRate))
		cough := gen(0.16, func(t float64) float64 {
			w := math.Sin(math.Pi * t / 0.16)
			return (lp.f(nz())*1.6 + sine(190-60*t/0.16, t)*0.5) * w * math.Exp(-t*8) * 0.8
		})
		b.add(cough, 0, 1)
		b.add(cough, 0.28, 0.8)
		out["cough"] = b.pcm(0.7)
	}
	{ // horseshoes: a ringing clang; checkers use the soft click; meditation a singing bowl
		hp := &lowpass{a: 0.5}
		out["clang"] = gen(0.6, func(t float64) float64 {
			x := nz()
			return ((x-hp.f(x))*math.Exp(-t*90)*0.4 + (sine(1180, t)+0.7*sine(1770, t)+0.5*sine(2650, t))*math.Exp(-t*9)*0.28)
		}).pcm(0.7)
		out["bell"] = gen(2.2, func(t float64) float64 {
			return (sine(392, t) + 0.5*sine(392*2.76, t) + 0.25*sine(392*5.4, t)) * math.Exp(-t*2.2) * math.Min(1, t/0.01) * 0.3
		}).pcm(0.7)
	}
	out["coin"] = gen(0.3, func(t float64) float64 {
		v := sine(1760, t)*math.Exp(-t*18) + 0.6*sine(2349, math.Max(0, t-0.07))*math.Exp(-math.Max(0, t-0.07)*16)
		return v * 0.35
	}).pcm(0.7)
	out["click"] = gen(0.05, func(t float64) float64 { return sine(900, t) * math.Exp(-t*90) * 0.6 }).pcm(0.7)
	chime := func(f, t float64) float64 {
		return (sine(f, t) + 0.3*sine(f*2.01, t)) * math.Exp(-t*6)
	}
	{
		b := make(synthBuf, int(0.8*sampleRate))
		b.add(gen(0.6, func(t float64) float64 { return chime(659.25, t) * 0.4 }), 0, 1)
		b.add(gen(0.6, func(t float64) float64 { return chime(880, t) * 0.4 }), 0.13, 1)
		out["save"] = b.pcm(0.7)
	}
	{
		b := make(synthBuf, int(0.8*sampleRate))
		for i, f := range []float64{523.25, 659.25, 783.99} {
			b.add(gen(0.5, func(t float64) float64 { return chime(f, t) * 0.3 }), float64(i)*0.11, 1)
		}
		out["heal"] = b.pcm(0.6)
	}
	chirp := func(f0, f1 float64) synthBuf {
		return gen(0.08, func(t float64) float64 {
			u := t / 0.08
			return sine(f0+(f1-f0)*u, t) * math.Sin(math.Pi*u) * 0.35
		})
	}
	{
		b := make(synthBuf, int(0.5*sampleRate))
		b.add(chirp(2600, 4200), 0, 1)
		b.add(chirp(3000, 4600), 0.12, 1)
		b.add(chirp(2400, 3800), 0.24, 1)
		out["bird1"] = b.pcm(0.6)
		c := make(synthBuf, int(0.5*sampleRate))
		c.add(chirp(4200, 2800), 0, 1)
		c.add(chirp(3800, 5000), 0.1, 1)
		out["bird2"] = c.pcm(0.6)
	}
	out["cricket"] = gen(0.7, func(t float64) float64 {
		gate := 0.0
		if math.Sin(2*math.Pi*26*t) > 0.2 {
			gate = 1
		}
		w := math.Sin(math.Pi * t / 0.7)
		return sine(4300, t) * gate * w * 0.25
	}).pcm(0.5)

	// looping beds -------------------------------------------------------
	{
		n := int(8 * sampleRate)
		fade := int(0.5 * sampleRate)
		raw := make(synthBuf, n+fade)
		l1, l2 := &lowpass{a: 0.02}, &lowpass{a: 0.12}
		for i := range raw {
			t := float64(i) / sampleRate
			x := nz()
			lfo := 0.6 + 0.4*math.Sin(2*math.Pi*t/8*1) + 0.15*math.Sin(2*math.Pi*t/8*3)
			raw[i] = (l1.f(x)*5 + (l2.f(x)-l1.f(x))*1.2) * lfo * 0.5
		}
		b := raw[:n]
		for i := 0; i < fade; i++ { // crossfade tail into head for a seamless loop
			w := float64(i) / float64(fade)
			b[i] = b[i]*w + raw[n+i]*(1-w)
		}
		out["wind"] = b.pcm(0.9)
	}
	// rain: a steady hiss with soft patter, looped
	{
		n := int(6 * sampleRate)
		fade := int(0.4 * sampleRate)
		raw := make(synthBuf, n+fade)
		l1, l2 := &lowpass{a: 0.5}, &lowpass{a: 0.06}
		for i := range raw {
			x := nz()
			hiss := l1.f(x) - l2.f(x) // band between the two cut-offs
			drop := 0.0
			if r.Float64() < 0.002 {
				drop = nz() * 0.5
			}
			raw[i] = hiss*0.9 + drop*0.3
		}
		b := raw[:n]
		for i := 0; i < fade; i++ {
			w := float64(i) / float64(fade)
			b[i] = b[i]*w + raw[n+i]*(1-w)
		}
		out["rain"] = b.pcm(0.6)
	}
	out["drone"] = gen(4, func(t float64) float64 {
		trem := 0.7 + 0.3*math.Sin(2*math.Pi*0.5*t)
		return (sine(55, t) + sine(58, t) + 0.5*sine(110, t) + 0.25*sine(165, t) + 0.15*sine(233, t)) * trem * 0.2
	}).pcm(0.9)

	// music notes (A minor pentatonic) ------------------------------------
	for i, f := range []float64{220, 261.63, 293.66, 329.63, 392, 440, 523.25, 587.33} {
		b := make(synthBuf, int(5.2*sampleRate))
		note := gen(3.5, func(t float64) float64 {
			a := math.Min(1, t/0.01)
			return (sine(f, t)*math.Exp(-t*1.3) + 0.35*sine(f*2, t)*math.Exp(-t*2.2) + 0.12*sine(f*3, t)*math.Exp(-t*3.5) + 0.2*sine(f*0.5, t)*math.Exp(-t*0.9)) * a * 0.5
		})
		b.add(note, 0, 1)
		b.add(note, 0.34, 0.32) // soft echoes
		b.add(note, 0.71, 0.18)
		out["note"+string(rune('0'+i))] = b.pcm(0.55)
	}
	return out
}

type Audio struct {
	ctx      *audio.Context
	bufs     map[string][]byte
	players  []*audio.Player
	wind     *audio.Player
	rain     *audio.Player
	drone    *audio.Player
	muted    bool
	nextNote float64
	nextBird float64
	rng      *rand.Rand
	droneVol float64
}

func initAudio() (a *Audio) {
	defer func() {
		if r := recover(); r != nil {
			a = nil // no audio device: run silently
		}
	}()
	a = &Audio{rng: rand.New(rand.NewSource(5)), nextNote: 3, nextBird: 5}
	a.ctx = audio.NewContext(sampleRate)
	a.bufs = synthAll()
	loop := func(name string) *audio.Player {
		b := a.bufs[name]
		p, err := a.ctx.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b))))
		if err != nil {
			return nil
		}
		p.SetVolume(0)
		p.Play()
		return p
	}
	a.wind, a.drone, a.rain = loop("wind"), loop("drone"), loop("rain")
	return a
}

func (a *Audio) play(name string, vol float64) {
	if a == nil || a.muted || vol < 0.02 {
		return
	}
	b, ok := a.bufs[name]
	if !ok {
		return
	}
	live := a.players[:0]
	for _, p := range a.players {
		if p.IsPlaying() {
			live = append(live, p)
		} else {
			p.Close()
		}
	}
	a.players = live
	if len(a.players) >= 28 {
		return
	}
	p := a.ctx.NewPlayerFromBytes(b)
	p.SetVolume(math.Min(1, vol))
	p.Play()
	a.players = append(a.players, p)
}

// ambience drives wind, creatures, music and the raid drone.
func (a *Audio) update(g *Game, dt float64) {
	if a == nil {
		return
	}
	mute := 1.0
	if a.muted {
		mute = 0
	}
	cold := clampf((6-g.outT)/16, 0, 1)
	night := 0.0
	if g.isNight() {
		night = 1
	}
	if a.wind != nil {
		a.wind.SetVolume(mute * (0.07 + 0.2*cold + 0.04*night))
	}
	if a.rain != nil {
		v := 0.0
		if g.raining() {
			v = 0.35 * g.precip
		} else if g.snowing() {
			v = 0.05 * g.precip
		}
		a.rain.SetVolume(mute * v)
	}
	wolves := 0
	for _, an := range g.animals {
		if an.Kind == AWolf {
			wolves++
		}
	}
	target := 0.0
	if wolves > 0 {
		target = 0.28
	}
	a.droneVol += (target - a.droneVol) * math.Min(1, dt*0.8)
	if a.drone != nil {
		a.drone.SetVolume(mute * a.droneVol)
	}
	if a.muted || g.paused {
		return
	}
	a.nextNote -= dt
	if a.nextNote <= 0 && wolves == 0 {
		a.nextNote = 2.5 + a.rng.Float64()*5
		hi := 8
		vol := 0.2
		if night == 1 {
			hi, vol = 5, 0.12
		}
		a.play("note"+string(rune('0'+a.rng.Intn(hi))), vol)
		if a.rng.Float64() < 0.35 {
			a.play("note"+string(rune('0'+a.rng.Intn(hi))), vol*0.7)
		}
	}
	a.nextBird -= dt
	if a.nextBird <= 0 {
		a.nextBird = 4 + a.rng.Float64()*8
		h := g.hour()
		switch {
		case h >= 6.5 && h < 19 && g.outT > 2:
			a.play([]string{"bird1", "bird2"}[a.rng.Intn(2)], 0.14)
		case g.isNight() && g.outT > 9:
			a.nextBird = 1.5 + a.rng.Float64()*3
			a.play("cricket", 0.08)
		}
	}
}
