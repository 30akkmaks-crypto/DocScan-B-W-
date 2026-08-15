// docscan.java — Java версия

import org.opencv.core.*;
import org.opencv.imgcodecs.Imgcodecs;
import org.opencv.imgproc.Imgproc;
import java.io.*;
import java.nio.file.*;
import java.util.*;

public class docscan {
    static { System.loadLibrary(Core.NATIVE_LIBRARY_NAME); }

    private int threshold;
    private boolean adaptive;
    private String format;

    public docscan(int threshold, boolean adaptive, String format) {
        this.threshold = threshold;
        this.adaptive = adaptive;
        this.format = format;
    }

    private Mat toBlackWhite(Mat image) {
        Mat gray = new Mat();
        Imgproc.cvtColor(image, gray, Imgproc.COLOR_BGR2GRAY);
        Mat bw = new Mat();
        if (adaptive) {
            Imgproc.adaptiveThreshold(gray, bw, 255, Imgproc.ADAPTIVE_THRESH_GAUSSIAN_C,
                                      Imgproc.THRESH_BINARY, 11, 2);
        } else {
            Imgproc.threshold(gray, bw, threshold, 255, Imgproc.THRESH_BINARY);
        }
        return bw;
    }

    public String scanDocument(String inputPath, String outputPath) throws Exception {
        Mat src = Imgcodecs.imread(inputPath);
        if (src.empty()) {
            throw new Exception("Не удалось загрузить изображение");
        }
        Mat orig = src.clone();
        double ratio = src.rows() / 500.0;
        Mat resized = new Mat();
        Imgproc.resize(src, resized, new Size(src.cols() / ratio, 500));

        Mat gray = new Mat();
        Imgproc.cvtColor(resized, gray, Imgproc.COLOR_BGR2GRAY);
        Mat blurred = new Mat();
        Imgproc.GaussianBlur(gray, blurred, new Size(5,5), 0);
        Mat edged = new Mat();
        Imgproc.Canny(blurred, edged, 75, 200);

        List<MatOfPoint> contours = new ArrayList<>();
        Mat hierarchy = new Mat();
        Imgproc.findContours(edged, contours, hierarchy, Imgproc.RETR_LIST, Imgproc.CHAIN_APPROX_SIMPLE);
        contours.sort((a, b) -> Double.compare(Imgproc.contourArea(b), Imgproc.contourArea(a)));
        if (contours.size() > 5) contours = contours.subList(0, 5);

        MatOfPoint2f screenCnt = null;
        for (MatOfPoint cnt : contours) {
            MatOfPoint2f cnt2f = new MatOfPoint2f(cnt.toArray());
            double peri = Imgproc.arcLength(cnt2f, true);
            MatOfPoint2f approx = new MatOfPoint2f();
            Imgproc.approxPolyDP(cnt2f, approx, 0.02 * peri, true);
            if (approx.toArray().length == 4) {
                screenCnt = approx;
                break;
            }
        }
        if (screenCnt == null) {
            throw new Exception("Не найден четырёхугольный контур");
        }

        Mat bw = toBlackWhite(orig);

        if (outputPath == null) {
            String base = inputPath.replaceFirst("\\.[^.]+$", "");
            outputPath = base + "_scanned_bw." + format;
        }

        if (format.equals("png")) {
            Imgcodecs.imwrite(outputPath, bw);
        } else {
            Mat bgr = new Mat();
            Imgproc.cvtColor(bw, bgr, Imgproc.COLOR_GRAY2BGR);
            Imgcodecs.imwrite(outputPath, bgr);
        }

        return outputPath;
    }

    public void batchProcess(String inputDir, String outputDir) throws Exception {
        String[] exts = {".jpg", ".jpeg", ".png", ".bmp", ".tiff"};
        File folder = new File(inputDir);
        List<File> files = new ArrayList<>();
        for (File f : folder.listFiles()) {
            if (f.isFile()) {
                String ext = f.getName().toLowerCase();
                for (String e : exts) {
                    if (ext.endsWith(e)) files.add(f);
                }
            }
        }

        if (files.isEmpty()) {
            System.out.println("❌ В папке нет поддерживаемых изображений.");
            return;
        }

        if (outputDir != null) {
            new File(outputDir).mkdirs();
        }

        System.out.println("📁 Найдено " + files.size() + " изображений.");
        int success = 0;
        for (int i = 0; i < files.size(); i++) {
            System.out.printf("\n[%d/%d] Обработка: %s\n", i+1, files.size(), files.get(i).getName());
            String outPath = null;
            if (outputDir != null) {
                String base = files.get(i).getName().replaceFirst("\\.[^.]+$", "");
                outPath = outputDir + File.separator + "scanned_" + base + "." + format;
            }
            long start = System.currentTimeMillis();
            try {
                String result = scanDocument(files.get(i).getPath(), outPath);
                double elapsed = (System.currentTimeMillis() - start) / 1000.0;
                System.out.printf("✅ Сохранено: %s (%.2f сек)\n", result, elapsed);
                success++;
            } catch (Exception e) {
                System.out.println("❌ Ошибка: " + e.getMessage());
            }
        }
        System.out.printf("\n✅ Готово! Обработано %d из %d изображений.\n", success, files.size());
    }

    public static void main(String[] args) throws Exception {
        String input = null;
        String output = null;
        int threshold = 127;
        boolean adaptive = true;
        String format = "png";
        boolean batch = false;

        for (int i = 0; i < args.length; i++) {
            if (args[i].equals("--output") || args[i].equals("-o")) output = args[++i];
            else if (args[i].equals("--threshold") || args[i].equals("-t")) threshold = Integer.parseInt(args[++i]);
            else if (args[i].equals("--no-adaptive")) adaptive = false;
            else if (args[i].equals("--format") || args[i].equals("-f")) format = args[++i];
            else if (args[i].equals("--batch") || args[i].equals("-b")) batch = true;
            else if (input == null) input = args[i];
        }

        if (input == null) {
            System.out.println("Usage: java docscan <image> [--threshold 127] [--format png] [--batch]");
            System.exit(1);
        }

        System.out.println("\u001B[36m📄 DocScan B&W (Java)\u001B[0m");

        docscan scanner = new docscan(threshold, adaptive, format);

        if (batch || Files.isDirectory(Paths.get(input))) {
            scanner.batchProcess(input, output);
        } else {
            long start = System.currentTimeMillis();
            try {
                String result = scanner.scanDocument(input, output);
                double elapsed = (System.currentTimeMillis() - start) / 1000.0;
                System.out.printf("✅ Сохранено: %s (%.2f сек)\n", result, elapsed);
            } catch (Exception e) {
                System.out.println("\u001B[31m❌ Ошибка: " + e.getMessage() + "\u001B[0m");
                System.exit(1);
            }
        }
    }
}
