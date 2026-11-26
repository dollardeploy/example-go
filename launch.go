package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"math"
	"math/rand"
)

// A night launch, rendered as a tone map and quantised with an 8x8 Bayer ordered dither
// to three ink levels. Unlike a browser canvas, the server renders the whole loop once at
// startup into an animated GIF per theme, then serves the bytes from memory.
const (
	launchW       = 160
	launchH       = 96
	ground        = 80
	padX          = 108
	cycleMs       = 7500.0 // one launch loop
	igniteMs      = 900.0  // engines light, rocket shakes
	liftoffMs     = 1900.0
	simStepMs     = 1000.0 / 60
	frameEvery    = 4 // keep every 4th simulated step: 15 fps
	gifFrameDelay = 7 // in 1/100 s, ~= frameEvery * simStepMs
)

var rocket = []string{
	"....#....",
	"...###...",
	"..#####..",
	"..#####..",
	"..##.##..",
	"..#...#..",
	"..##.##..",
	"..#####..",
	"..#####..",
	"..#####..",
	"..#####..",
	".#######.",
	"##.###.##",
	"#..###..#",
}

var (
	rocketW = len(rocket[0])
	rocketH = len(rocket)
	restY   = float64(ground - 1 - rocketH)
	racks   = []int{16, 40}
)

// Background first, then three ink levels from weakest to strongest.
var (
	launchPaletteLight = color.Palette{rgb(0xf4f2f0), rgb(0xb5b3b0), rgb(0x585755), rgb(0x090909)}
	launchPaletteDark  = color.Palette{rgb(0x050506), rgb(0x242cc7), rgb(0x4a69e8), rgb(0xd4e4ff)}
)

func rgb(v uint32) color.Color {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}

var bayer = func() [64]float64 {
	m := [][]int{{0}}
	for len(m) < 8 {
		n := len(m)
		next := make([][]int, n*2)
		for i := range next {
			next[i] = make([]int, n*2)
		}
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				v := m[y][x] * 4
				next[y][x] = v
				next[y][x+n] = v + 2
				next[y+n][x] = v + 3
				next[y+n][x+n] = v + 1
			}
		}
		m = next
	}
	var out [64]float64
	for y := range 8 {
		for x := range 8 {
			out[y*8+x] = (float64(m[y][x]) + 0.5) / 64
		}
	}
	return out
}()

// Sky, ground and structures never change: compute their tones once.
func backdrop() []float64 {
	tone := make([]float64, launchW*launchH)
	set := func(x, y int, v float64) { tone[y*launchW+x] = v }
	for y := range launchH {
		for x := range launchW {
			var v float64
			switch {
			case y < ground:
				v = 0.11 * math.Pow(float64(y)/ground, 3)
			case y == ground:
				v = 0.5
			default:
				v = 0.3 - float64(y-ground)/float64(launchH-ground)*0.2
			}
			set(x, y, v)
		}
	}
	for x := padX - 8; x <= padX+8; x++ {
		set(x, ground-1, 0.7)
	}
	for y := ground - 24; y < ground; y++ {
		set(padX-13, y, 0.55)
		set(padX-10, y, 0.55)
		if y%3 == 0 {
			for x := padX - 12; x < padX-10; x++ {
				set(x, y, 0.4)
			}
		}
	}
	for _, rx := range racks {
		for y := ground - 29; y < ground; y++ {
			for x := rx; x < rx+18; x++ {
				edge := x == rx || x == rx+17 || (y-(ground-29))%7 == 0
				if edge {
					set(x, y, 0.55)
				} else {
					set(x, y, 0)
				}
			}
		}
	}
	return tone
}

type particle struct{ x, y, vx, vy, life, max float64 }

// renderLaunch simulates one full launch cycle and returns it as an animated GIF.
func renderLaunch(palette color.Palette) []byte {
	base := backdrop()
	tone := make([]float64, len(base))
	rnd := rand.New(rand.NewSource(7))

	type star struct {
		x, y int
		p    float64
	}
	stars := make([]star, 34)
	for i := range stars {
		stars[i] = star{rnd.Intn(launchW), rnd.Intn(ground - 45), rnd.Float64() * 6}
	}
	// Exhaust light falling on the ground around the pad, precomputed falloff.
	glow := make([]float64, launchW*launchH)
	for i := range glow {
		x, y := i%launchW, i/launchW
		if y >= ground-30 {
			glow[i] = math.Exp(-math.Hypot(float64(x-padX), float64(y-ground)*1.8) / 20)
		}
	}

	var particles []particle
	anim := &gif.GIF{LoopCount: 0}

	for step := 0; float64(step)*simStepMs < cycleMs; step++ {
		now := float64(step) * simStepMs
		s := math.Max(0, now-liftoffMs) / 1000
		ry := restY - 9*s*s
		shake := 0
		if now > igniteMs && now < liftoffMs+500 {
			shake = int(math.Round(math.Sin(now / 22)))
		}
		rx := padX - rocketW/2 + shake
		firing := now > igniteMs && ry > -float64(rocketH)

		if firing {
			for range 5 {
				m := 20 + rnd.Float64()*40
				particles = append(particles, particle{
					x: padX + (rnd.Float64()-0.5)*3, y: ry + float64(rocketH),
					vx: (rnd.Float64() - 0.5) * 0.4, vy: 1.2 + rnd.Float64()*1.4,
					life: m, max: m,
				})
			}
		}
		alive := particles[:0]
		for _, p := range particles {
			p.x += p.vx
			p.y += p.vy
			if p.y >= ground-2 && p.vy > 0 {
				// exhaust hits the pad and billows sideways
				p.y = ground - 2 - rnd.Float64()*4
				p.vy = -rnd.Float64() * 0.12
				dir := 1.0
				if p.x < padX {
					dir = -1
				}
				p.vx = dir * (0.5 + rnd.Float64()*1.6)
			}
			p.life--
			if p.life > 0 && p.x >= 0 && p.x < launchW {
				alive = append(alive, p)
			}
		}
		particles = alive

		if step%frameEvery != 0 {
			continue
		}

		copy(tone, base)
		max := func(x, y int, v float64) {
			if x < 0 || x >= launchW || y < 0 || y >= launchH {
				return
			}
			if i := y*launchW + x; v > tone[i] {
				tone[i] = v
			}
		}

		for _, st := range stars {
			max(st.x, st.y, 0.45+0.35*math.Sin(now/700+st.p))
		}

		if near := math.Max(0, 1-(restY-ry)/70); firing && near > 0 {
			g := near * (0.55 + 0.15*rnd.Float64())
			for i := (ground - 30) * launchW; i < len(tone); i++ {
				tone[i] += glow[i] * g
			}
		}

		// blinking server lights
		for r, rx0 := range racks {
			for u := range 4 {
				y := ground - 25 + u*7
				on := (int(now/180)+u*3+r*5)%7 != 0
				v := 0.2
				if on {
					v = 1
				}
				max(rx0+3, y, v)
				for x := rx0 + 7; x < rx0+15; x += 2 {
					max(x, y, 0.35)
				}
			}
		}

		for _, p := range particles {
			v := p.life / p.max
			x, y := int(p.x), int(p.y)
			max(x, y, v)
			max(x+1, y, v*0.75)
			max(x, y+1, v*0.75)
			max(x-1, y, v*0.5)
		}

		// rocket body, lit from the left
		top := int(math.Round(ry))
		for y, row := range rocket {
			for x := range rocketW {
				if row[x] == '#' {
					max(rx+x, top+y, 1-float64(x)*0.045)
				}
			}
		}

		// quantise: tone -> background or one of 3 ink levels via ordered dither
		frame := image.NewPaletted(image.Rect(0, 0, launchW, launchH), palette)
		levels := float64(len(palette) - 1)
		for i, t := range tone {
			t = math.Min(t, 1) * levels
			level := int(t)
			x, y := i%launchW, i/launchW
			if t-float64(level) > bayer[(y&7)<<3|x&7] {
				level++
			}
			frame.Pix[i] = uint8(level)
		}
		anim.Image = append(anim.Image, frame)
		anim.Delay = append(anim.Delay, gifFrameDelay)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, anim); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
