// docscan.cs — C# версия

using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using OpenCvSharp;

class DocumentScanner {
    private int threshold;
    private bool adaptive;
    private string format;

    public DocumentScanner(int threshold, bool adaptive, string format) {
        this.threshold = threshold;
        this.adaptive = adaptive;
        this.format = format;
    }

    private Mat ToBlackWhite(Mat image) {
        Mat gray = new Mat();
        Cv2.CvtColor(image, gray, ColorConversionCodes.BGR2GRAY);
        Mat bw = new Mat();
        if (adaptive) {
            Cv2.AdaptiveThreshold(gray, bw, 255, AdaptiveThresholdTypes.GaussianC, ThresholdTypes.Binary, 11, 2);
        } else {
            Cv2.Threshold(gray, bw, threshold, 255, ThresholdTypes.Binary);
        }
        return bw;
    }

    public string ScanDocument(string inputPath, string outputPath) {
        Mat src = Cv2.ImRead(inputPath);
        if (src.Empty()) {
            throw new Exception("Не удалось загрузить изображение");
        }
        Mat orig = src.Clone();
        double ratio = src.Rows / 500.0;
        Mat resized = new Mat();
        Cv2.Resize(src, resized, new Size(src.Cols / ratio, 500));

        Mat gray = new Mat();
        Cv2.CvtColor(resized, gray, ColorConversionCodes.BGR2GRAY);
        Mat blurred = new Mat();
        Cv2.GaussianBlur(gray, blurred, new Size(5, 5), 0);
        Mat edged = new Mat();
        Cv2.Canny(blurred, edged, 75, 200);

        var contours = Cv2.FindContours(edged, RetrievalModes.List, ContourApproximationModes.ApproxSimple);
        var sorted = contours.OrderByDescending(c => Cv2.ContourArea(c)).Take(5).ToList();

        Point[][] screenCnt = null;
        foreach (var cnt in sorted) {
            var peri = Cv2.ArcLength(cnt, true);
            var approx = Cv2.ApproxPolyDP(cnt, 0.02 * peri, true);
            if (approx.Length == 4) {
                screenCnt = approx;
                break;
            }
        }
        if (screenCnt == null) {
            throw new Exception("Не найден четырёхугольный контур");
        }

        Mat bw = ToBlackWhite(orig);

        if (string.IsNullOrEmpty(outputPath)) {
            string baseName = Path.GetFileNameWithoutExtension(inputPath);
            outputPath = $"{baseName}_scanned_bw.{format}";
        }

        if (format == "png") {
            Cv2.ImWrite(outputPath, bw);
        } else {
            Mat bgr = new Mat();
            Cv2.CvtColor(bw, bgr, ColorConversionCodes.GRAY2BGR);
            Cv2.ImWrite(outputPath, bgr);
        }

        return outputPath;
    }

    public void BatchProcess(string inputDir, string outputDir) {
        var exts = new[] { ".jpg", ".jpeg", ".png", ".bmp", ".tiff" };
        var files = Directory.GetFiles(inputDir).Where(f => exts.Contains(Path.GetExtension(f).ToLower())).ToList();

        if (files.Count == 0) {
            Console.WriteLine("❌ В папке нет поддерживаемых изображений.");
            return;
        }

        if (!string.IsNullOrEmpty(outputDir)) {
            Directory.CreateDirectory(outputDir);
        }

        Console.WriteLine($"📁 Найдено {files.Count} изображений.");
        int success = 0;

        for (int i = 0; i < files.Count; i++) {
            Console.WriteLine($"\n[{i+1}/{files.Count}] Обработка: {Path.GetFileName(files[i])}");
            string outPath = null;
            if (!string.IsNullOrEmpty(outputDir)) {
                string baseName = Path.GetFileNameWithoutExtension(files[i]);
                outPath = Path.Combine(outputDir, $"scanned_{baseName}.{format}");
            }
            var start = DateTime.Now;
            try {
                string result = ScanDocument(files[i], outPath);
                var elapsed = (DateTime.Now - start).TotalSeconds;
                Console.WriteLine($"✅ Сохранено: {result} ({elapsed:F2} сек)");
                success++;
            } catch (Exception e) {
                Console.WriteLine($"❌ Ошибка: {e.Message}");
            }
        }
        Console.WriteLine($"\n✅ Готово! Обработано {success} из {files.Count} изображений.");
    }

    public static void Main(string[] args) {
        string input = null;
        string output = null;
        int threshold = 127;
        bool adaptive = true;
        string format = "png";
        bool batch = false;

        for (int i = 0; i < args.Length; i++) {
            if (args[i] == "--output" || args[i] == "-o") output = args[++i];
            else if (args[i] == "--threshold" || args[i] == "-t") threshold = int.Parse(args[++i]);
            else if (args[i] == "--no-adaptive") adaptive = false;
            else if (args[i] == "--format" || args[i] == "-f") format = args[++i];
            else if (args[i] == "--batch" || args[i] == "-b") batch = true;
            else if (input == null) input = args[i];
        }

        if (input == null) {
            Console.WriteLine("Usage: dotnet run <image> [--threshold 127] [--format png] [--batch]");
            return;
        }

        Console.WriteLine("\u001B[36m📄 DocScan B&W (C#)\u001B[0m");

        var scanner = new DocumentScanner(threshold, adaptive, format);

        if (batch || Directory.Exists(input)) {
            scanner.BatchProcess(input, output);
        } else {
            var start = DateTime.Now;
            try {
                string result = scanner.ScanDocument(input, output);
                var elapsed = (DateTime.Now - start).TotalSeconds;
                Console.WriteLine($"✅ Сохранено: {result} ({elapsed:F2} сек)");
            } catch (Exception e) {
                Console.WriteLine($"\u001B[31m❌ Ошибка: {e.Message}\u001B[0m");
            }
        }
    }
}
