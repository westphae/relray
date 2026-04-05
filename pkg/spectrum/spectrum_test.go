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
	// Monochromatic 550nm shifted by factor 2 -> 1100nm (now within extended range)
	m := Monochromatic(550, 1.0)
	shifted := m.Shift(2.0)
	// Energy should be at ~1100nm band, not in the visible range
	_, y, _ := shifted.ToXYZ()
	if y > eps {
		t.Errorf("550nm redshifted to 1100nm should have zero visible luminance, got Y=%v", y)
	}
	// But total energy should be nonzero (it's in the IR now)
	var total float64
	for _, v := range shifted {
		total += v
	}
	if total < eps {
		t.Error("550nm redshifted to 1100nm should have IR energy in extended range")
	}
}

func TestShiftBlueshift(t *testing.T) {
	// Monochromatic 600nm shifted by factor 0.5 -> 300nm (now within extended range)
	m := Monochromatic(600, 1.0)
	shifted := m.Shift(0.5)
	// Energy should be at ~300nm band, not in the visible range
	_, y, _ := shifted.ToXYZ()
	if y > eps {
		t.Errorf("600nm blueshifted to 300nm should have zero visible luminance, got Y=%v", y)
	}
	// But total energy should be nonzero (it's in the UV now)
	var total float64
	for _, v := range shifted {
		total += v
	}
	if total < eps {
		t.Error("600nm blueshifted to 300nm should have UV energy in extended range")
	}
}

func TestShiftIRIntoVisible(t *testing.T) {
	// Key test: monochromatic 1000nm (invisible IR) blueshifted by factor 0.5 -> 500nm (visible green)
	m := Monochromatic(1000, 1.0)
	// Verify it's invisible
	_, y, _ := m.ToXYZ()
	if y > eps {
		t.Errorf("1000nm should be invisible, got Y=%v", y)
	}
	// Blueshift by factor 0.5: lambda_obs = lambda_emit * 0.5
	shifted := m.Shift(0.5)
	_, y, _ = shifted.ToXYZ()
	if y < 0.01 {
		t.Errorf("1000nm blueshifted to 500nm should be visible, got Y=%v", y)
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

func TestBlackbodySunlike(t *testing.T) {
	// 5778K blackbody (solar temperature) should appear roughly white/warm
	bb := Blackbody(5778, 1.0)
	_, y, _ := bb.ToXYZ()
	if math.Abs(y-1.0) > 0.01 {
		t.Errorf("Blackbody(5778, 1.0) Y = %v, want ~1.0", y)
	}
	// Should have nonzero values across the visible spectrum
	if bb[0] <= 0 || bb[40] <= 0 || bb[80] <= 0 {
		t.Error("Blackbody should have power across entire visible range")
	}
}

func TestBlackbodyHot(t *testing.T) {
	// 10000K should be bluish (more power at short wavelengths)
	bb := Blackbody(10000, 1.0)
	x, _, z := bb.ToXYZ()
	// High color temperature = high Z relative to X
	if z <= x {
		t.Errorf("10000K blackbody should be blue-biased: X=%v Z=%v", x, z)
	}
}

func TestBlackbodyCool(t *testing.T) {
	// 3000K should be reddish (more power at long wavelengths)
	bb := Blackbody(3000, 1.0)
	x, _, z := bb.ToXYZ()
	// Low color temperature = high X relative to Z
	if x <= z {
		t.Errorf("3000K blackbody should be red-biased: X=%v Z=%v", x, z)
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
