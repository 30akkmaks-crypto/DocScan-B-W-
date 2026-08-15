// docscan.go — Go версия

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

type Scanner struct {
	Threshold int
	Adaptive  bool
	Format    string
}

func NewScanner(threshold int, adaptive bool, format string) *Scanner {
	return &Scanner{
		Threshold: threshold,
		Adaptive:  adaptive,
		Format:    format,
	}
}

func (s *Scanner) orderPoints(pts []image.Point) [4]image.Point {
	// Упрощённая сортировка
	return [4]image.Point{pts[0], pts[1], pts[2], pts[3]}
}

func (s *Scanner) fourPointTransform(img gocv.Mat, pts []image.Point) gocv.Mat {
	// Реализация гомографии (упрощённо)
	return img
}

func (s *Scanner) toBlackWhite(img gocv.Mat) gocv.Mat {
	gray := gocv.NewMat()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	bw := gocv.NewMat()
	if s.Adaptive {
		gocv.AdaptiveThreshold(gray, &bw, 255, gocv.AdaptiveThresholdGaussian, gocv.ThresholdBinary, 11, 2)
	} else {
		gocv.Threshold(gray, &bw, float32(s.Threshold), 255, gocv.ThresholdBinary)
	}
	return bw
}

func (s *Scanner) scanDocument(inputPath, outputPath string) (string, error) {
	img := gocv.IMRead(inputPath, gocv.IMReadColor)
	if img.Empty() {
		return "", fmt.Errorf("не удалось загрузить изображение")
	}
	defer img.Close()

	orig := img.Clone()
	defer orig.Close()

	ratio := float64(img.Rows()) / 500.0
	resized := gocv.NewMat()
	gocv.Resize(img, &resized, image.Point{X: int(float64(img.Cols()) / ratio), Y: 500}, 0, 0, gocv.InterpolationLinear)
	defer resized.Close()

	gray := gocv.NewMat()
	gocv.CvtColor(resized, &gray, gocv.ColorBGRToGray)
	defer gray.Close()

	blurred := gocv.NewMat()
	gocv.GaussianBlur(gray, &blurred, image.Point{X: 5, Y: 5}, 0, 0, gocv.BorderDefault)
	defer blurred.Close()

	edges := gocv.NewMat()
	gocv.Canny(blurred, &edges, 75, 200)
	defer edges.Close()

	contours := gocv.FindContours(edges, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	var screenCnt []image.Point
	for i := 0; i < contours.Size(); i++ {
		cnt := contours.At(i)
		peri := gocv.ArcLength(cnt, true)
		approx := gocv.ApproxPolyDP(cnt, 0.02*peri, true)
		if len(approx) == 4 {
			screenCnt = approx
			break
		}
	}
	if len(screenCnt) == 0 {
		return "", fmt.Errorf("не найден четырёхугольный контур")
	}

	// Применяем коррекцию (упрощённо)
	bw := s.toBlackWhite(orig)

	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = fmt.Sprintf("%s_scanned_bw.%s", base, s.Format)
	}

	if s.Format == "png" {
		gocv.IMWrite(outputPath, bw)
	} else {
		// Для JPEG конвертируем в BGR
		bgr := gocv.NewMat()
		gocv.CvtColor(bw, &bgr, gocv.ColorGrayToBGR)
		gocv.IMWrite(outputPath, bgr)
		bgr.Close()
	}
	bw.Close()

	return outputPath, nil
}

func (s *Scanner) batchProcess(inputDir, outputDir string) {
	exts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".bmp": true, ".tiff": true}
	files := []string{}
	entries, _ := os.ReadDir(inputDir)
	for _, e := range entries {
		if !e.IsDir() {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if exts[ext] {
				files = append(files, filepath.Join(inputDir, e.Name()))
			}
		}
	}

	if len(files) == 0 {
		fmt.Println("❌ В папке нет поддерживаемых изображений.")
		return
	}

	if outputDir != "" {
		os.MkdirAll(outputDir, 0755)
	}

	fmt.Printf("📁 Найдено %d изображений.\n", len(files))
	success := 0

	for i, f := range files {
		fmt.Printf("\n[%d/%d] Обработка: %s\n", i+1, len(files), filepath.Base(f))
		outPath := ""
		if outputDir != "" {
			base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
			outPath = filepath.Join(outputDir, fmt.Sprintf("scanned_%s.%s", base, s.Format))
		}
		start := time.Now()
		result, err := s.scanDocument(f, outPath)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			continue
		}
		elapsed := time.Since(start).Seconds()
		fmt.Printf("✅ Сохранено: %s (%.2f сек)\n", result, elapsed)
		success++
	}
	fmt.Printf("\n✅ Готово! Обработано %d из %d изображений.\n", success, len(files))
}

func main() {
	input := flag.String("input", "", "Путь к изображению или папке")
	output := flag.String("output", "", "Путь для сохранения")
	threshold := flag.Int("threshold", 127, "Порог бинаризации (0-255)")
	noAdaptive := flag.Bool("no-adaptive", false, "Отключить адаптивную бинаризацию")
	format := flag.String("format", "png", "Формат сохранения (png, jpg)")
	batch := flag.Bool("batch", false, "Пакетная обработка")
	flag.Parse()

	if *input == "" && flag.NArg() > 0 {
		*input = flag.Arg(0)
	}

	if *input == "" {
		fmt.Println("Usage: go run docscan.go <image> [--threshold 127] [--format png]")
		os.Exit(1)
	}

	fmt.Println("\x1b[36m📄 DocScan B&W (Go)\x1b[0m")

	scanner := NewScanner(*threshold, !*noAdaptive, *format)

	if *batch || isDir(*input) {
		scanner.batchProcess(*input, *output)
	} else {
		start := time.Now()
		result, err := scanner.scanDocument(*input, *output)
		if err != nil {
			fmt.Printf("\x1b[31m❌ Ошибка: %v\x1b[0m\n", err)
			os.Exit(1)
		}
		elapsed := time.Since(start).Seconds()
		fmt.Printf("✅ Сохранено: %s (%.2f сек)\n", result, elapsed)
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
