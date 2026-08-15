

### 1. `docscan.py` (Python)

```python
# docscan.py — Python версия

import cv2
import numpy as np
import sys
import os
import argparse
import time
from PIL import Image
from colorama import init, Fore, Style

init(autoreset=True)

class DocumentScanner:
    def __init__(self, threshold=127, adaptive=True, output_format='png'):
        self.threshold = threshold
        self.adaptive = adaptive
        self.output_format = output_format

    def order_points(self, pts):
        """Сортировка точек: верх-левый, верх-правый, ниж-правый, ниж-левый."""
        rect = np.zeros((4, 2), dtype="float32")
        s = pts.sum(axis=1)
        rect[0] = pts[np.argmin(s)]
        rect[2] = pts[np.argmax(s)]
        diff = np.diff(pts, axis=1)
        rect[1] = pts[np.argmin(diff)]
        rect[3] = pts[np.argmax(diff)]
        return rect

    def four_point_transform(self, image, pts):
        """Применяет перспективное преобразование."""
        rect = self.order_points(pts)
        (tl, tr, br, bl) = rect
        widthA = np.linalg.norm(br - bl)
        widthB = np.linalg.norm(tr - tl)
        maxWidth = max(int(widthA), int(widthB))
        heightA = np.linalg.norm(tr - br)
        heightB = np.linalg.norm(tl - bl)
        maxHeight = max(int(heightA), int(heightB))
        dst = np.array([
            [0, 0],
            [maxWidth - 1, 0],
            [maxWidth - 1, maxHeight - 1],
            [0, maxHeight - 1]], dtype="float32")
        M = cv2.getPerspectiveTransform(rect, dst)
        warped = cv2.warpPerspective(image, M, (maxWidth, maxHeight))
        return warped

    def to_black_white(self, image):
        """Преобразует изображение в черно-белый режим."""
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        if self.adaptive:
            # Адаптивная бинаризация
            bw = cv2.adaptiveThreshold(gray, 255, cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
                                       cv2.THRESH_BINARY, 11, 2)
        else:
            # Фиксированный порог
            _, bw = cv2.threshold(gray, self.threshold, 255, cv2.THRESH_BINARY)
        return bw

    def scan_document(self, image_path, output_path=None):
        """Основная функция сканирования документа."""
        image = cv2.imread(image_path)
        if image is None:
            raise ValueError("Не удалось загрузить изображение")

        orig = image.copy()
        ratio = image.shape[0] / 500.0
        image_resized = cv2.resize(image, (int(image.shape[1] / ratio), 500))
        gray = cv2.cvtColor(image_resized, cv2.COLOR_BGR2GRAY)
        gray = cv2.GaussianBlur(gray, (5, 5), 0)
        edged = cv2.Canny(gray, 75, 200)

        contours, _ = cv2.findContours(edged.copy(), cv2.RETR_LIST, cv2.CHAIN_APPROX_SIMPLE)
        contours = sorted(contours, key=cv2.contourArea, reverse=True)[:5]

        screenCnt = None
        for c in contours:
            peri = cv2.arcLength(c, True)
            approx = cv2.approxPolyDP(c, 0.02 * peri, True)
            if len(approx) == 4:
                screenCnt = approx
                break

        if screenCnt is None:
            raise ValueError("Не удалось найти четырёхугольный контур")

        warped = self.four_point_transform(orig, screenCnt.reshape(4, 2) * ratio)
        bw_image = self.to_black_white(warped)

        if output_path is None:
            base = os.path.splitext(image_path)[0]
            output_path = f"{base}_scanned_bw.{self.output_format}"

        cv2.imwrite(output_path, bw_image)
        return output_path, bw_image.shape

    def batch_process(self, input_dir, output_dir=None):
        """Пакетная обработка всех изображений в папке."""
        if not os.path.exists(input_dir):
            print(Fore.RED + f"❌ Папка не найдена: {input_dir}")
            return

        if output_dir and not os.path.exists(output_dir):
            os.makedirs(output_dir)

        extensions = ('.jpg', '.jpeg', '.png', '.bmp', '.tiff', '.tif')
        files = [f for f in os.listdir(input_dir) if f.lower().endswith(extensions)]

        if not files:
            print(Fore.YELLOW + "❌ В папке нет поддерживаемых изображений.")
            return

        print(f"📁 Найдено {len(files)} изображений.")
        success = 0
        for i, f in enumerate(files, 1):
            print(f"\n[{i}/{len(files)}] Обработка: {f}")
            input_path = os.path.join(input_dir, f)
            if output_dir:
                output_path = os.path.join(output_dir, f"scanned_{os.path.splitext(f)[0]}.{self.output_format}")
            else:
                output_path = None
            try:
                start = time.time()
                out_path, shape = self.scan_document(input_path, output_path)
                elapsed = time.time() - start
                print(f"✅ Сохранено: {out_path} ({shape[1]}x{shape[0]}, {elapsed:.2f} сек)")
                success += 1
            except Exception as e:
                print(Fore.RED + f"❌ Ошибка: {e}")

        print(f"\n✅ Готово! Обработано {success} из {len(files)} изображений.")

def main():
    parser = argparse.ArgumentParser(description='Document Scanner (Black & White)')
    parser.add_argument('input', help='Путь к изображению или папке')
    parser.add_argument('--output', '-o', help='Путь для сохранения')
    parser.add_argument('--threshold', '-t', type=int, default=127,
                        help='Порог бинаризации (0-255, по умолч. 127)')
    parser.add_argument('--no-adaptive', action='store_true',
                        help='Отключить адаптивную бинаризацию')
    parser.add_argument('--format', '-f', choices=['png', 'jpg', 'jpeg'], default='png',
                        help='Формат сохранения (по умолч. png)')
    parser.add_argument('--batch', '-b', action='store_true',
                        help='Пакетная обработка папки')
    args = parser.parse_args()

    print(Fore.CYAN + "📄 DocScan B&W (Python)")

    scanner = DocumentScanner(
        threshold=args.threshold,
        adaptive=not args.no_adaptive,
        output_format=args.format
    )

    if args.batch or os.path.isdir(args.input):
        scanner.batch_process(args.input, args.output)
    else:
        if not os.path.exists(args.input):
            print(Fore.RED + f"❌ Файл не найден: {args.input}")
            sys.exit(1)

        start = time.time()
        try:
            out_path, shape = scanner.scan_document(args.input, args.output)
            elapsed = time.time() - start
            print(f"✅ Сохранено: {out_path} ({shape[1]}x{shape[0]}, {elapsed:.2f} сек)")
        except Exception as e:
            print(Fore.RED + f"❌ Ошибка: {e}")
            sys.exit(1)

if __name__ == "__main__":
    main()
