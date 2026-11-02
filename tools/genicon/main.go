// genicon draws the 番茄钟 app icon: a ripe tomato with a calyx,
// stem and highlight, supersampled for smooth edges.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const S = 1024

func lerp(a, b, t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return a + (b-a)*t
}

func angNorm(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// sample returns the RGBA of the point (x, y) in unit coordinates.
func sample(x, y float64) (r, g, b, a float64) {
	// calyx: five green spikes fanning from the top of the body
	cx, cy := 0.5, 0.305
	dx, dy := x-cx, y-cy
	d := math.Hypot(dx, dy)
	if d <= 0.14 && d > 1e-6 {
		ang := math.Atan2(dy, dx)
		for k := range 5 {
			ca := -math.Pi/2 + float64(k)*2*math.Pi/5
			half := 0.24*(1-d/0.14) + 0.03
			if math.Abs(angNorm(ang-ca)) <= half {
				return 74, 140, 66, 255
			}
		}
	}
	// stem
	if x >= 0.486 && x <= 0.514 && y >= 0.15 && y <= 0.31 {
		return 74, 140, 66, 255
	}
	// body: a red ellipse with a vertical gradient
	ex := (x - 0.5) / 0.36
	ey := (y - 0.62) / 0.33
	if ex*ex+ey*ey <= 1 {
		t := (y - 0.30) / 0.64
		r = lerp(236, 198, t)
		g = lerp(96, 50, t)
		b = lerp(78, 44, t)
		// soft highlight
		hx := (x - 0.375) / 0.10
		hy := (y - 0.50) / 0.065
		if q := hx*hx + hy*hy; q <= 1 {
			ha := 0.30 * (1 - q)
			r, g, b = lerp(r, 255, ha), lerp(g, 255, ha), lerp(b, 255, ha)
		}
		return r, g, b, 255
	}
	return 0, 0, 0, 0
}

func main() {
	const ss = 4 // supersample 4x4 for smooth edges
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	for y := range S {
		for x := range S {
			var r, g, b, a float64
			for sy := range ss {
				for sx := range ss {
					fx := (float64(x) + float64(sx)/ss) / S
					fy := (float64(y) + float64(sy)/ss) / S
					sr, sg, sb, sa := sample(fx, fy)
					r, g, b, a = r+sr, g+sg, b+sb, a+sa
				}
			}
			n := float64(ss * ss)
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n),
			})
		}
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
