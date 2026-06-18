// Утилита распознавания QR-кодов на базе gozxing (порт ZXing).
//
// Использование:
//
//	QRCodeRecognizer.exe <путь_к_изображению>           — текст QR в stdout, код 0
//	QRCodeRecognizer.exe --orientation <путь_к_изображению> — угол поворота по часовой (0|90|180|270), код 0
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func main() {
	orientationMode := false
	args := os.Args[1:]
	if len(args) >= 1 && args[0] == "--orientation" {
		orientationMode = true
		args = args[1:]
	}

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: QRCodeRecognizer [--orientation] <image-path>")
		os.Exit(2)
	}

	img, err := loadImage(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if orientationMode {
		rotation, ok := detectUprightRotation(img)
		if !ok {
			fmt.Fprintln(os.Stderr, "QR not found")
			os.Exit(1)
		}
		fmt.Print(rotation)
		return
	}

	text, err := decodeQRText(img)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(text)
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}

func decodeQRText(img image.Image) (string, error) {
	result, err := decodeQR(img)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func decodeQR(img image.Image) (*gozxing.Result, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, err
	}
	reader := qrcode.NewQRCodeReader()
	return reader.Decode(bmp, nil)
}

// detectUprightRotation возвращает угол поворота по часовой стрелке для выравнивания бланка.
// Ориентация определяется по направлению верхнего ребра QR (точки finder TL→TR), а не по углу листа.
func detectUprightRotation(img image.Image) (int, bool) {
	if rotation, ok := rotationFromQREdge(img); ok {
		return rotation, true
	}
	for _, deg := range []int{90, 180, 270} {
		rotated := rotateImage(img, deg)
		if rotation, ok := rotationFromQREdge(rotated); ok && rotation == 0 {
			return deg, true
		}
	}
	return 0, false
}

// rotationFromQREdge вычисляет поворот по направлению ребра TL→TR символа QR.
// Выровненный бланк: верхнее ребро QR идёт слева направо (угол 0°).
func rotationFromQREdge(img image.Image) (int, bool) {
	result, err := decodeQR(img)
	if err != nil {
		return 0, false
	}

	points := result.GetResultPoints()
	if len(points) < 3 {
		return 0, false
	}

	// ZXing/gozxing для QR: [0]=BL, [1]=TL, [2]=TR finder patterns.
	pTopLeft := points[1]
	pTopRight := points[2]
	dx := pTopRight.GetX() - pTopLeft.GetX()
	dy := pTopRight.GetY() - pTopLeft.GetY()
	adx := math.Abs(dx)
	ady := math.Abs(dy)

	const slopeRatio = 0.55
	if adx >= ady*slopeRatio {
		if dx > 0 {
			return 0, true
		}
		return 180, true
	}
	if dy > 0 {
		return 90, true
	}
	return 270, true
}

func normalizeRotation(angle int) int {
	angle %= 360
	if angle < 0 {
		angle += 360
	}
	switch angle {
	case 0, 90, 180, 270:
		return angle
	default:
		return 0
	}
}

func rotateImage(img image.Image, degrees int) image.Image {
	switch normalizeRotation(degrees) {
	case 0:
		return img
	case 90:
		return rotate90CW(img)
	case 180:
		return rotate180(img)
	case 270:
		return rotate90CW(rotate180(img))
	default:
		return img
	}
}

func rotate90CW(img image.Image) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}

func rotate180(img image.Image) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, h-1-y, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}
