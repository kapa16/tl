package recognize

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"tl/qrcoderecognizer/internal/preprocess"
)

type binarizerKind int

const (
	binarizerHybrid binarizerKind = iota
	binarizerGlobal
)

type attempt struct {
	label      string
	img        image.Image
	tryHarder  bool
	binarizer  binarizerKind
}

// FromFile распознаёт QR на изображении, перебирая несколько стратегий предобработки.
func FromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	for _, a := range buildAttempts(img) {
		text, err := decodeOnce(a)
		if err == nil && text != "" {
			return text, nil
		}
	}
	return "", fmt.Errorf("QR code not found")
}

func buildAttempts(img image.Image) []attempt {
	attempts := []attempt{
		{label: "raw", img: img, binarizer: binarizerHybrid},
		{label: "raw+tryharder", img: img, tryHarder: true, binarizer: binarizerHybrid},
		{label: "raw+global", img: img, binarizer: binarizerGlobal},
		{label: "raw+global+tryharder", img: img, tryHarder: true, binarizer: binarizerGlobal},
	}

	for _, factor := range []float64{1.3, 1.4, 1.5, 2.0} {
		enhanced := preprocess.EnhanceContrast(img, factor)
		attempts = append(attempts,
			attempt{label: fmt.Sprintf("contrast%.1f", factor), img: enhanced, binarizer: binarizerHybrid},
			attempt{label: fmt.Sprintf("contrast%.1f+tryharder", factor), img: enhanced, tryHarder: true, binarizer: binarizerHybrid},
		)
	}

	for _, thr := range []uint8{128, 140, 150, 160} {
		bw := preprocess.Threshold(img, thr)
		attempts = append(attempts,
			attempt{label: fmt.Sprintf("threshold%d", thr), img: bw, binarizer: binarizerHybrid},
			attempt{label: fmt.Sprintf("threshold%d+tryharder", thr), img: bw, tryHarder: true, binarizer: binarizerHybrid},
		)
	}

	for i, crop := range preprocess.CornerCrops(img) {
		attempts = append(attempts,
			attempt{label: fmt.Sprintf("corner%d", i), img: crop, tryHarder: true, binarizer: binarizerHybrid},
		)
		enhanced := preprocess.EnhanceContrast(crop, 1.5)
		attempts = append(attempts,
			attempt{label: fmt.Sprintf("corner%d+contrast1.5", i), img: enhanced, tryHarder: true, binarizer: binarizerHybrid},
		)
	}

	return attempts
}

func decodeOnce(a attempt) (string, error) {
	bmp, err := newBinaryBitmap(a.img, a.binarizer)
	if err != nil {
		return "", err
	}

	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_POSSIBLE_FORMATS: []gozxing.BarcodeFormat{gozxing.BarcodeFormat_QR_CODE},
	}
	if a.tryHarder {
		hints[gozxing.DecodeHintType_TRY_HARDER] = true
	}

	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bmp, hints)
	if err != nil {
		return "", err
	}
	return result.GetText(), nil
}

func newBinaryBitmap(img image.Image, kind binarizerKind) (*gozxing.BinaryBitmap, error) {
	source := gozxing.NewLuminanceSourceFromImage(img)
	var binarizer gozxing.Binarizer
	switch kind {
	case binarizerGlobal:
		binarizer = gozxing.NewGlobalHistgramBinarizer(source)
	default:
		binarizer = gozxing.NewHybridBinarizer(source)
	}
	return gozxing.NewBinaryBitmap(binarizer)
}
