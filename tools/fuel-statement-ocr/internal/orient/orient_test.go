package orient

import (
	"image"
	"testing"
)

func TestRotationCandidatesAllCardinal(t *testing.T) {
	portrait := image.NewRGBA(image.Rect(0, 0, 100, 200))
	landscape := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for _, img := range []image.Image{portrait, landscape} {
		cands := rotationCandidates(img)
		if len(cands) != 4 {
			t.Fatalf("expected 4 rotations, got %d", len(cands))
		}
		want := map[int]bool{0: true, 90: true, 180: true, 270: true}
		for _, c := range cands {
			if !want[c] {
				t.Fatalf("unexpected rotation %d", c)
			}
		}
	}
}
