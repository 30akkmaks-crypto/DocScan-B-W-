// docscan.js — JavaScript версия

const cv = require('opencv4nodejs');
const fs = require('fs');
const path = require('path');

class DocumentScanner {
    constructor(threshold = 127, adaptive = true, format = 'png') {
        this.threshold = threshold;
        this.adaptive = adaptive;
        this.format = format;
    }

    toBlackWhite(image) {
        const gray = image.cvtColor(cv.COLOR_BGR2GRAY);
        let bw;
        if (this.adaptive) {
            bw = gray.adaptiveThreshold(255, cv.ADAPTIVE_THRESH_GAUSSIAN_C, cv.THRESH_BINARY, 11, 2);
        } else {
            bw = gray.threshold(this.threshold, 255, cv.THRESH_BINARY);
        }
        return bw;
    }

    async scanDocument(inputPath, outputPath = null) {
        const src = cv.imread(inputPath);
        if (src.empty) {
            throw new Error('Не удалось загрузить изображение');
        }
        const orig = src.copy();
        const ratio = src.rows / 500;
        const resized = src.resize(500, Math.round(src.cols / ratio));

        const gray = resized.cvtColor(cv.COLOR_BGR2GRAY);
        const blurred = gray.gaussianBlur(new cv.Size(5, 5), 0);
        const edged = blurred.canny(75, 200);

        const contours = edged.findContours(cv.RETR_LIST, cv.CHAIN_APPROX_SIMPLE);
        const sorted = contours.sort((a, b) => b.area - a.area).slice(0, 5);

        let screenCnt = null;
        for (const cnt of sorted) {
            const peri = cnt.arcLength(true);
            const approx = cnt.approxPolyDP(0.02 * peri, true);
            if (approx.length === 4) {
                screenCnt = approx;
                break;
            }
        }
        if (!screenCnt) {
            throw new Error('Не найден четырёхугольный контур');
        }

        // Коррекция перспективы (упрощённо)
        const bw = this.toBlackWhite(orig);

        if (!outputPath) {
            const base = path.basename(inputPath, path.extname(inputPath));
            outputPath = `${base}_scanned_bw.${this.format}`;
        }

        if (this.format === 'png') {
            cv.imwrite(outputPath, bw);
        } else {
            const bgr = bw.cvtColor(cv.COLOR_GRAY2BGR);
            cv.imwrite(outputPath, bgr);
        }

        return outputPath;
    }

    async batchProcess(inputDir, outputDir = null) {
        const exts = ['.jpg', '.jpeg', '.png', '.bmp', '.tiff'];
        const files = fs.readdirSync(inputDir).filter(f => exts.includes(path.extname(f).toLowerCase()));

        if (files.length === 0) {
            console.log('❌ В папке нет поддерживаемых изображений.');
            return;
        }

        if (outputDir && !fs.existsSync(outputDir)) {
            fs.mkdirSync(outputDir, { recursive: true });
        }

        console.log(`📁 Найдено ${files.length} изображений.`);
        let success = 0;

        for (let i = 0; i < files.length; i++) {
            console.log(`\n[${i+1}/${files.length}] Обработка: ${files[i]}`);
            const inputPath = path.join(inputDir, files[i]);
            let outputPath = null;
            if (outputDir) {
                const base = path.basename(files[i], path.extname(files[i]));
                outputPath = path.join(outputDir, `scanned_${base}.${this.format}`);
            }
            const start = Date.now();
            try {
                const result = await this.scanDocument(inputPath, outputPath);
                const elapsed = (Date.now() - start) / 1000;
                console.log(`✅ Сохранено: ${result} (${elapsed.toFixed(2)} сек)`);
                success++;
            } catch (err) {
                console.error(`❌ Ошибка: ${err.message}`);
            }
        }
        console.log(`\n✅ Готово! Обработано ${success} из ${files.length} изображений.`);
    }
}

async function main() {
    const args = process.argv.slice(2);
    let input = null;
    let output = null;
    let threshold = 127;
    let adaptive = true;
    let format = 'png';
    let batch = false;

    for (let i = 0; i < args.length; i++) {
        if (args[i] === '--output' || args[i] === '-o') output = args[++i];
        else if (args[i] === '--threshold' || args[i] === '-t') threshold = parseInt(args[++i]);
        else if (args[i] === '--no-adaptive') adaptive = false;
        else if (args[i] === '--format' || args[i] === '-f') format = args[++i];
        else if (args[i] === '--batch' || args[i] === '-b') batch = true;
        else if (!input) input = args[i];
    }

    if (!input) {
        console.log('Usage: node docscan.js <image> [--threshold 127] [--format png] [--batch]');
        process.exit(1);
    }

    console.log('\x1b[36m📄 DocScan B&W (JavaScript)\x1b[0m');

    const scanner = new DocumentScanner(threshold, adaptive, format);

    if (batch || fs.statSync(input).isDirectory()) {
        await scanner.batchProcess(input, output);
    } else {
        const start = Date.now();
        try {
            const result = await scanner.scanDocument(input, output);
            const elapsed = (Date.now() - start) / 1000;
            console.log(`✅ Сохранено: ${result} (${elapsed.toFixed(2)} сек)`);
        } catch (err) {
            console.error(`\x1b[31m❌ Ошибка: ${err.message}\x1b[0m`);
            process.exit(1);
        }
    }
}

main().catch(console.error);
