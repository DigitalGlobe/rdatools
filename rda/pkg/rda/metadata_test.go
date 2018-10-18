package rda

import (
	"testing"
)

func TestImageGeoreferencingApply(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}
	xGeo, yGeo := gt.Apply(10.0, 1.0)
	if xGeo != 11.0 || yGeo != 19.0 {
		t.Fatalf("Expected Apply(10.0, 1.0) = (11.0, 19.0), got (%f, %f)", xGeo, yGeo)
	}
}

func TestImageGeoreferencingInvert(t *testing.T) {
	gt := ImageGeoreferencing{
		TranslateX: 10.0,
		ScaleX:     0.1,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igt := ImageGeoreferencing{
		TranslateX: -100.0,
		ScaleX:     10.0,
		ShearX:     0.0,
		TranslateY: 20.0,
		ShearY:     0.0,
		ScaleY:     -1.0,
	}

	igtc, err := gt.Invert()
	if err != nil {
		t.Fatalf("failed to invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse, %+v != %+v", igtc, igt)
	}

	igtc, err = gt.hardInvert()
	if err != nil {
		t.Fatalf("failed to hard invert, err: %+v", err)
	}
	if igtc != igt {
		t.Fatalf("bad inverse on hard invert, %+v != %+v", igtc, igt)
	}
}
