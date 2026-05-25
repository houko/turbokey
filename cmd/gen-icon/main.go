// Command gen-icon renders the TurboKey app icon (rounded blue/indigo tile with a
// white lightning bolt) and writes cmd/turbokey/icon.ico (multi-size, for the exe
// file icon) and internal/ui/icon.png (256px, embedded for window/tray icons).
// This is a build-time generator, not part of the app.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const master = 1536 // high-res master, divisible by all target sizes

func dist(ax, ay, bx, by float64) float64 {
	dx, dy := ax-bx, ay-by
	return math.Sqrt(dx*dx + dy*dy)
}

func insideRounded(px, py, left, top, right, bottom, r float64) bool {
	if px < left || px > right || py < top || py > bottom {
		return false
	}
	switch {
	case px < left+r && py < top+r:
		return dist(px, py, left+r, top+r) <= r
	case px > right-r && py < top+r:
		return dist(px, py, right-r, top+r) <= r
	case px < left+r && py > bottom-r:
		return dist(px, py, left+r, bottom-r) <= r
	case px > right-r && py > bottom-r:
		return dist(px, py, right-r, bottom-r) <= r
	}
	return true
}

func inPoly(px, py float64, poly [][2]float64) bool {
	in := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if (yi > py) != (yj > py) && px < (xj-xi)*(py-yi)/(yj-yi)+xi {
			in = !in
		}
		j = i
	}
	return in
}

func lerp(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t) }

func renderMaster(bolt color.NRGBA) *image.NRGBA {
	W := float64(master)
	img := image.NewNRGBA(image.Rect(0, 0, master, master))
	m, r := 96.0, 336.0
	left, top, right, bottom := m, m, W-m, W-m
	// vertical gradient: blue -> indigo
	tR, tG, tB := uint8(0x5b), uint8(0x8d), uint8(0xfb)
	bR, bG, bB := uint8(0x43), uint8(0x38), uint8(0xca)
	// lightning bolt polygon (normalized), scaled to master
	norm := [][2]float64{{0.58, 0.08}, {0.30, 0.54}, {0.47, 0.54}, {0.40, 0.92}, {0.72, 0.42}, {0.53, 0.42}}
	poly := make([][2]float64, len(norm))
	for i, p := range norm {
		poly[i] = [2]float64{p[0] * W, p[1] * W}
	}
	for y := 0; y < master; y++ {
		for x := 0; x < master; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			if !insideRounded(px, py, left, top, right, bottom, r) {
				continue
			}
			t := (py - top) / (bottom - top)
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			c := color.NRGBA{lerp(tR, bR, t), lerp(tG, bG, t), lerp(tB, bB, t), 0xff}
			if inPoly(px, py, poly) {
				c = bolt
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// downscale box-averages src to size×size with alpha-weighted color (clean edges
// against the transparent background).
func downscale(src *image.NRGBA, size int) *image.NRGBA {
	f := master / size
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var rr, gg, bb, aa int
			for dy := 0; dy < f; dy++ {
				for dx := 0; dx < f; dx++ {
					c := src.NRGBAAt(x*f+dx, y*f+dy)
					a := int(c.A)
					rr += int(c.R) * a
					gg += int(c.G) * a
					bb += int(c.B) * a
					aa += a
				}
			}
			var out color.NRGBA
			if aa > 0 {
				out = color.NRGBA{uint8(rr / aa), uint8(gg / aa), uint8(bb / aa), uint8(aa / (f * f))}
			}
			dst.SetNRGBA(x, y, out)
		}
	}
	return dst
}

func pngBytes(img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func writeICO(path string, sizes []int, pngs map[int][]byte) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))
	offset := 6 + 16*len(sizes)
	for _, s := range sizes {
		data := pngs[s]
		dim := byte(s)
		if s >= 256 {
			dim = 0
		}
		buf.WriteByte(dim) // width
		buf.WriteByte(dim) // height
		buf.WriteByte(0)   // palette
		buf.WriteByte(0)   // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))  // planes
		binary.Write(&buf, binary.LittleEndian, uint16(32)) // bpp
		binary.Write(&buf, binary.LittleEndian, uint32(len(data)))
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(data)
	}
	for _, s := range sizes {
		buf.Write(pngs[s])
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
}

func main() {
	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}
	yellow := color.NRGBA{0xfb, 0xbf, 0x24, 0xff} // brighter, "active" feel

	mOff := renderMaster(white)
	mOn := renderMaster(yellow)

	sizes := []int{16, 32, 48, 256}
	pngsOff := map[int][]byte{}
	for _, s := range sizes {
		pngsOff[s] = pngBytes(downscale(mOff, s))
	}
	writeICO("cmd/turbokey/icon.ico", sizes, pngsOff)
	if err := os.WriteFile("internal/ui/icon.png", pngsOff[256], 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile("internal/ui/icon_on.png", pngBytes(downscale(mOn, 256)), 0644); err != nil {
		panic(err)
	}
	println("wrote icon.ico, icon.png, icon_on.png")
}
