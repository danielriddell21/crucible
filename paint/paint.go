// Package paint holds the small colour and framebuffer operations the
// family's software renderers share: scaling a colour's brightness for
// distance shading, and compositing a translucent wash over a finished
// frame. Each game keeps its own shading curve — how brightness falls off
// with distance is art direction — and calls these to apply the result.
//
// The operations work on [image/color.RGBA] values and raw RGBA byte
// buffers (the pixel layout of an Ebiten image), so they stay display-free
// and reusable by any front-end that composes frames in software.
package paint

import "image/color"

// Scale multiplies a colour's red, green and blue channels by k and returns
// an opaque colour. Values are truncated toward zero, matching a direct
// float-to-byte conversion; k is expected in [0, 1] for shading but is not
// clamped.
func Scale(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * k),
		G: uint8(float64(c.G) * k),
		B: uint8(float64(c.B) * k),
		A: 255,
	}
}

// ScaleBytes is [Scale] packed into an opaque RGBA quad, for writing
// straight into a byte framebuffer without an intermediate colour value.
func ScaleBytes(c color.RGBA, k float64) [4]byte {
	return [4]byte{
		uint8(float64(c.R) * k),
		uint8(float64(c.G) * k),
		uint8(float64(c.B) * k),
		255,
	}
}

// BlendOver alpha-composites the solid colour c over every pixel of an RGBA
// framebuffer in place, with c contributing a fraction alpha and the
// existing pixel the remainder. It is the full-screen tint used for damage
// flashes and power-up washes. Bytes past the last whole pixel are left
// untouched.
func BlendOver(fb []byte, c color.RGBA, alpha float64) {
	keep := 1 - alpha
	ri, gi, bi := float64(c.R)*alpha, float64(c.G)*alpha, float64(c.B)*alpha
	for i := 0; i+3 < len(fb); i += 4 {
		fb[i] = uint8(float64(fb[i])*keep + ri)
		fb[i+1] = uint8(float64(fb[i+1])*keep + gi)
		fb[i+2] = uint8(float64(fb[i+2])*keep + bi)
	}
}
