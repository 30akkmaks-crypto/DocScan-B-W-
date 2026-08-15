<?php
// docscan.php — PHP версия

class DocumentScanner {
    private $threshold;
    private $adaptive;
    private $format;

    public function __construct($threshold = 127, $adaptive = true, $format = 'png') {
        $this->threshold = $threshold;
        $this->adaptive = $adaptive;
        $this->format = $format;
    }

    public function toBlackWhite($image) {
        // Преобразование в ч/б через Imagick
        $img = new Imagick();
        $img->readImageBlob($image);

        // Преобразование в градации серого
        $img->transformImageColorspace(Imagick::COLORSPACE_GRAY);

        if ($this->adaptive) {
            // Адаптивная бинаризация через случайный порог
            // В реальном проекте нужна более сложная реализация
            $img->thresholdImage($this->threshold * Imagick::getQuantum() / 255);
        } else {
            $img->thresholdImage($this->threshold * Imagick::getQuantum() / 255);
        }

        return $img;
    }

    public function scanDocument($inputPath, $outputPath = null) {
        if (!file_exists($inputPath)) {
            throw new Exception("Файл не найден: $inputPath");
        }

        $imageData = file_get_contents($inputPath);
        $img = new Imagick();
        $img->readImageBlob($imageData);

        // Упрощённое обнаружение контуров (в реальном проекте используйте OpenCV)
        // Здесь просто преобразуем в ч/б
        $bw = $this->toBlackWhite($imageData);

        if ($outputPath === null) {
            $base = pathinfo($inputPath, PATHINFO_FILENAME);
            $outputPath = "{$base}_scanned_bw.{$this->format}";
        }

        if ($this->format === 'png') {
            $bw->setImageFormat('png');
        } else {
            $bw->setImageFormat('jpeg');
        }

        $bw->writeImage($outputPath);
        $bw->clear();

        return $outputPath;
    }

    public function batchProcess($inputDir, $outputDir = null) {
        if (!is_dir($inputDir)) {
            echo "❌ Папка не найдена: $inputDir\n";
            return;
        }

        $exts = ['jpg', 'jpeg', 'png', 'bmp', 'tiff'];
        $files = array_diff(scandir($inputDir), ['.', '..']);
        $images = [];

        foreach ($files as $f) {
            $ext = strtolower(pathinfo($f, PATHINFO_EXTENSION));
            if (in_array($ext, $exts)) {
                $images[] = $f;
            }
        }

        if (empty($images)) {
            echo "❌ В папке нет поддерживаемых изображений.\n";
            return;
        }

        if ($outputDir && !is_dir($outputDir)) {
            mkdir($outputDir, 0755, true);
        }

        echo "📁 Найдено " . count($images) . " изображений.\n";
        $success = 0;

        foreach ($images as $i => $f) {
            echo "\n[" . ($i+1) . "/" . count($images) . "] Обработка: $f\n";
            $inputPath = $inputDir . DIRECTORY_SEPARATOR . $f;
            $outputPath = null;
            if ($outputDir) {
                $base = pathinfo($f, PATHINFO_FILENAME);
                $outputPath = $outputDir . DIRECTORY_SEPARATOR . "scanned_$base.{$this->format}";
            }
            $start = microtime(true);
            try {
                $result = $this->scanDocument($inputPath, $outputPath);
                $elapsed = microtime(true) - $start;
                echo "✅ Сохранено: $result (" . number_format($elapsed, 2) . " сек)\n";
                $success++;
            } catch (Exception $e) {
                echo "❌ Ошибка: " . $e->getMessage() . "\n";
            }
        }

        echo "\n✅ Готово! Обработано $success из " . count($images) . " изображений.\n";
    }
}

function main($argv) {
    $input = null;
    $output = null;
    $threshold = 127;
    $adaptive = true;
    $format = 'png';
    $batch = false;

    for ($i = 1; $i < count($argv); $i++) {
        if ($argv[$i] == '--output' || $argv[$i] == '-o') {
            $output = $argv[++$i];
        } elseif ($argv[$i] == '--threshold' || $argv[$i] == '-t') {
            $threshold = (int)$argv[++$i];
        } elseif ($argv[$i] == '--no-adaptive') {
            $adaptive = false;
        } elseif ($argv[$i] == '--format' || $argv[$i] == '-f') {
            $format = $argv[++$i];
        } elseif ($argv[$i] == '--batch' || $argv[$i] == '-b') {
            $batch = true;
        } elseif ($input === null) {
            $input = $argv[$i];
        }
    }

    if ($input === null) {
        echo "Usage: php docscan.php <image> [--threshold 127] [--format png] [--batch]\n";
        exit(1);
    }

    echo "\033[36m📄 DocScan B&W (PHP)\033[0m\n";

    $scanner = new DocumentScanner($threshold, $adaptive, $format);

    if ($batch || is_dir($input)) {
        $scanner->batchProcess($input, $output);
    } else {
        if (!file_exists($input)) {
            echo "\033[31m❌ Файл не найден: $input\033[0m\n";
            exit(1);
        }
        $start = microtime(true);
        try {
            $result = $scanner->scanDocument($input, $output);
            $elapsed = microtime(true) - $start;
            echo "✅ Сохранено: $result (" . number_format($elapsed, 2) . " сек)\n";
        } catch (Exception $e) {
            echo "\033[31m❌ Ошибка: " . $e->getMessage() . "\033[0m\n";
            exit(1);
        }
    }
}

$argc = $_SERVER['argc'] ?? 0;
$argv = $_SERVER['argv'] ?? [];
main($argv);
?>
