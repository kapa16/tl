package orient

import (
	"image"
	"math"

	"tl/fuel-statement-ocr/internal/align"
	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/preprocess"
)

// Result holds orientation normalization outcome.
type Result struct {
	Image           image.Image
	AppliedRotation int
	ExifOrientation int
	Score           float64
	SecondBestScore float64
}

// Normalize picks the best rotation using multiple anchors.
func Normalize(img image.Image, tmpl *mask.Template, exifOrient int) Result {
	type cand struct {
		rot       int
		img       image.Image
		score     float64
		stripDist float64
		peaks     int
		stripReg  float64
		stripCov  float64
		rowInk    int
	}
	var cands []cand
	maxScore := -1e9
	for _, rot := range rotationCandidates(img) {
		candidate := img
		if rot > 0 {
			candidate = preprocess.Rotate90(img, rot/90)
		}
		score := scoreOrientation(candidate, tmpl)
		if score > maxScore {
			maxScore = score
		}
		alignRes := align.WarpToReference(candidate, tmpl)
		gray := calibrate.ToGray(alignRes.Image)
		tmplEval, _ := mask.LoadByType(tmpl.Type)
		calibrate.AdjustTemplateForCanvas(tmplEval, alignRes.ContentDX, alignRes.ContentDY, alignRes.ContentSX, alignRes.ContentSY)
		strip, peaks := calibrate.LocateReferenceStrip(gray, tmplEval)
		_, stripCov := calibrate.StripTemplateCoverage(gray, strip)
		cands = append(cands, cand{
			rot:       rot,
			img:       candidate,
			score:     score,
			stripDist: stripAnchorDistance(candidate, tmpl),
			peaks:     peaks,
			stripReg:  calibrate.StripDigitRegularity(gray, strip),
			stripCov:  stripCov,
			rowInk:    rowQuantityInkScore(gray, tmplEval),
		})
	}

	bestIdx := 0
	for i, c := range cands {
		best := cands[bestIdx]
		if c.stripCov > best.stripCov+0.12 {
			bestIdx = i
			continue
		}
		if c.stripCov < best.stripCov-0.12 {
			continue
		}
		if c.rowInk > best.rowInk+150 {
			bestIdx = i
			continue
		}
		if c.rowInk < best.rowInk-150 {
			continue
		}
		if c.stripReg > best.stripReg+0.06 {
			bestIdx = i
			continue
		}
		if c.stripReg < best.stripReg-0.06 {
			continue
		}
		if c.peaks > best.peaks {
			bestIdx = i
			continue
		}
		if c.peaks < best.peaks {
			continue
		}
		if c.stripDist < best.stripDist || (c.stripDist == best.stripDist && c.score > best.score) {
			bestIdx = i
		}
	}

	chosen := cands[bestIdx]
	secondBest := -1e9
	for i, c := range cands {
		if i == bestIdx {
			continue
		}
		if c.score > secondBest {
			secondBest = c.score
		}
	}
	return Result{
		Image:           chosen.img,
		AppliedRotation: chosen.rot,
		ExifOrientation: exifOrient,
		Score:           chosen.score,
		SecondBestScore: secondBest,
	}
}

func rowQuantityInkScore(gray *image.Gray, tmpl *mask.Template) int {
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	rowH := tmpl.Table.RowHeight
	if rowH <= 0 {
		return 0
	}
	y0 := int((tmpl.Table.FirstRowY + rowH) * float64(h))
	y1 := int((tmpl.Table.FirstRowY + 2*rowH) * float64(h))
	if y1 <= y0 {
		return 0
	}
	skip := (y1 - y0) * 38 / 100
	y0 += skip
	count := 0
	for _, col := range tmpl.Table.Columns {
		if col.ID != "quantity_liters" && col.ID != "quantity_kg" {
			continue
		}
		cx0, cx1 := col.ColumnXRange()
		ix0 := int(cx0 * float64(w))
		ix1 := int(cx1 * float64(w))
		for y := y0; y < y1; y++ {
			for x := ix0; x < ix1; x++ {
				if gray.GrayAt(x, y).Y < 140 {
					count++
				}
			}
		}
	}
	return count
}

// ScoreAllRotations returns per-rotation anchor scores (for tests and diagnostics).
func ScoreAllRotations(img image.Image, tmpl *mask.Template) map[int]float64 {
	out := make(map[int]float64, 4)
	for _, rot := range rotationCandidates(img) {
		candidate := img
		if rot > 0 {
			candidate = preprocess.Rotate90(img, rot/90)
		}
		out[rot] = scoreOrientation(candidate, tmpl)
	}
	return out
}

// rotationCandidates returns all cardinal rotations to evaluate.
// Always 0/90/180/270: portrait JPEGs may contain sideways content (EXIF=1 but wrong heading).
func rotationCandidates(img image.Image) []int {
	_ = img
	return []int{0, 90, 180, 270}
}

func scoreOrientation(img image.Image, tmpl *mask.Template) float64 {
	gray := calibrate.ToGray(img)

	var total float64
	exp := tmpl.Anchors.DigitReferenceStrip

	total += stripAnchorScore(gray, tmpl) * 10.0

	topQ := stripQuality(gray, exp.X, exp.Y, exp.W, exp.H)
	total += topQ * 2.0

	mirrorY := 1.0 - exp.Y - exp.H
	if mirrorY > 0 {
		bottomQ := stripQuality(gray, exp.X, mirrorY, exp.W, exp.H)
		if bottomQ > topQ+0.02 {
			total -= 6.0
		}
	}

	qr := tmpl.Anchors.QRTopLeft
	total += darkRegionScore(gray, qr.X, qr.Y, qr.W, qr.H) * 2.0

	if tmpl.DocumentTitle != nil {
		r := tmpl.DocumentTitle.SearchRegion
		total += darkRegionScore(gray, r.X, r.Y, r.W, r.H) * 1.5
	}

	w, h := imageioBounds(img)
	total += horizontalEdgeScore(gray, int(float64(h)*0.16), int(float64(h)*0.88)) * 1.0
	if tmpl.CanonicalOrientation == "portrait" && h > w {
		total += 1.5
	}

	total += float64(rowQuantityInkScore(gray, tmpl)) * 0.002

	return total
}

// stripAnchorScore is 0..1 — how well the printed 1234567890 strip matches the template anchor.
func stripAnchorScore(gray *image.Gray, tmpl *mask.Template) float64 {
	exp := tmpl.Anchors.DigitReferenceStrip
	sx, sy, _, _, ok := calibrate.FindPrintedDigitStripNear(gray, exp.X, exp.Y, exp.W, exp.H)
	if !ok {
		return stripQuality(gray, exp.X, exp.Y, exp.W, exp.H) * 0.4
	}
	dx := math.Abs(sx - exp.X)
	dy := math.Abs(sy - exp.Y)
	peaks := len(calibrate.StripDigitRects(gray, mask.Rect{X: sx, Y: sy, W: exp.W, H: exp.H}))
	score := 1.0 - math.Min(1.0, dx*3+dy*8)
	if peaks >= 8 {
		score += 0.15
	}
	return score
}

func stripAnchorDistance(img image.Image, tmpl *mask.Template) float64 {
	gray := calibrate.ToGray(img)
	exp := tmpl.Anchors.DigitReferenceStrip
	sx, sy, _, _, ok := calibrate.FindPrintedDigitStripNear(gray, exp.X, exp.Y, exp.W, exp.H)
	if !ok {
		return 1e9
	}
	dx := math.Abs(sx - exp.X)
	dy := math.Abs(sy - exp.Y)
	dist := dx*3 + dy*8
	peaks := len(calibrate.StripDigitRects(gray, mask.Rect{X: sx, Y: sy, W: exp.W, H: exp.H}))
	if peaks < 7 {
		dist += 0.5
	}
	return dist
}

func horizontalEdgeScore(gray *image.Gray, y0, y1 int) float64 {
	w := gray.Bounds().Dx()
	if y1 > gray.Bounds().Dy() {
		y1 = gray.Bounds().Dy()
	}
	var best int
	for y := y0; y < y1; y++ {
		var sum int
		for x := 1; x < w; x += 2 {
			g0 := int(gray.GrayAt(x-1, y).Y)
			g1 := int(gray.GrayAt(x, y).Y)
			if g0-g1 > 20 || g1-g0 > 20 {
				sum++
			}
		}
		if sum > best {
			best = sum
		}
	}
	if w == 0 {
		return 0
	}
	r := float64(best) / float64(w/2)
	if r > 0.15 {
		return 0.15
	}
	return r
}

func stripQuality(gray *image.Gray, x, y, w, h float64) float64 {
	width, height := gray.Bounds().Dx(), gray.Bounds().Dy()
	x0 := int(x * float64(width))
	y0 := int(y * float64(height))
	x1 := int((x + w) * float64(width))
	y1 := int((y + h) * float64(height))
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	var dark, total int
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			total++
			if gray.GrayAt(xx, yy).Y < 140 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	r := float64(dark) / float64(total)
	if r < 0.04 || r > 0.55 {
		return r * 0.3
	}
	return r
}

func darkRegionScore(gray *image.Gray, x, y, w, h float64) float64 {
	width, height := gray.Bounds().Dx(), gray.Bounds().Dy()
	x0 := int(x * float64(width))
	y0 := int(y * float64(height))
	x1 := int((x + w) * float64(width))
	y1 := int((y + h) * float64(height))
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	var dark, total int
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			total++
			if gray.GrayAt(xx, yy).Y < 160 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	r := float64(dark) / float64(total)
	if r < 0.02 {
		return r
	}
	if r > 0.7 {
		return 0.7 - (r-0.7)*0.5
	}
	return r
}

func imageioBounds(img image.Image) (int, int) {
	b := img.Bounds()
	return b.Dx(), b.Dy()
}
