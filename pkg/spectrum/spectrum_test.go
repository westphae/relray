package spectrum

import (
	"math"
	"testing"
)

const eps = 1e-6

func TestD65NormalizedY(t *testing.T) {
	d := D65()
	_, y, _ := d.ToXYZ()
	if math.Abs(y-1.0) > 0.01 {
		t.Errorf("D65 Y = %v, want ~1.0", y)
	}
}

func TestD65ToSRGB(t *testing.T) {
	// D65 (normalized to Y=1) should convert to roughly white in sRGB.
	d := D65()
	x, y, z := d.ToXYZ()
	r, g, b := XYZToSRGB(x, y, z)
	// All channels should be high (near 255)
	if r < 200 || g < 200 || b < 200 {
		t.Errorf("D65 sRGB = (%d, %d, %d), expected near-white", r, g, b)
	}
}

func TestShiftIdentity(t *testing.T) {
	d := D65()
	shifted := d.Shift(1.0)
	for i := range d {
		if math.Abs(shifted[i]-d[i]) > eps {
			t.Errorf("Shift(1.0) changed band %d: %v -> %v", i, d[i], shifted[i])
			break
		}
	}
}

func TestShiftRedshift(t *testing.T) {
	// Monochromatic 550nm shifted by factor 2 -> 1100nm (outside visible range -> zero)
	m := Monochromatic(550, 1.0)
	shifted := m.Shift(2.0)
	var total float64
	for _, v := range shifted {
		total += v
	}
	if total > eps {
		t.Errorf("550nm redshifted to 1100nm should be zero, got total %v", total)
	}
}

func TestShiftBlueshift(t *testing.T) {
	// Monochromatic 600nm shifted by factor 0.5 -> 300nm (outside visible range -> zero)
	m := Monochromatic(600, 1.0)
	shifted := m.Shift(0.5)
	var total float64
	for _, v := range shifted {
		total += v
	}
	if total > eps {
		t.Errorf("600nm blueshifted to 300nm should be zero, got total %v", total)
	}
}

func TestMonochromatic(t *testing.T) {
	m := Monochromatic(550, 1.0)
	// Should have nonzero values near band 34 (550nm = 380 + 34*5)
	idx := int(BandIndex(550))
	if m[idx] <= 0 {
		t.Errorf("monochromatic 550nm should have power at band %d", idx)
	}
	// Bands far away should be zero
	if m[0] != 0 {
		t.Errorf("monochromatic 550nm should be zero at 380nm, got %v", m[0])
	}
}

func TestFromRGBRed(t *testing.T) {
	r := FromRGB(1, 0, 0)
	x, y, z := r.ToXYZ()
	rr, gg, bb := XYZToLinearRGB(x, y, z)
	// Red channel should dominate
	if rr <= gg || rr <= bb {
		t.Errorf("FromRGB(1,0,0) should have red dominant, got linear RGB (%v, %v, %v)", rr, gg, bb)
	}
}
