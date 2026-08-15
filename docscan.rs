// docscan.rs — Rust версия

use image::{GenericImageView, ImageBuffer, Luma};
use std::env;
use std::fs;
use std::path::Path;
use std::time::Instant;

struct Scanner {
    threshold: u8,
    adaptive: bool,
    format: String,
}

impl Scanner {
    fn new(threshold: u8, adaptive: bool, format: &str) -> Self {
        Scanner {
            threshold,
            adaptive,
            format: format.to_string(),
        }
    }

    fn to_black_white(&self, img: &image::DynamicImage) -> ImageBuffer<Luma<u8>, Vec<u8>> {
        let gray = img.to_luma8();
        let (width, height) = gray.dimensions();
        let mut bw = ImageBuffer::new(width, height);

        if self.adaptive {
            // Упрощённая адаптивная бинаризация
            let block_size = 11;
            let offset = 2;
            for y in 0..height {
                for x in 0..width {
                    let mut sum = 0u32;
                    let mut count = 0u32;
                    for dy in -(block_size/2)..=(block_size/2) {
                        for dx in -(block_size/2)..=(block_size/2) {
                            let nx = x as i32 + dx;
                            let ny = y as i32 + dy;
                            if nx >= 0 && nx < width as i32 && ny >= 0 && ny < height as i32 {
                                let pixel = gray.get_pixel(nx as u32, ny as u32);
                                sum += pixel[0] as u32;
                                count += 1;
                            }
                        }
                    }
                    let threshold = (sum / count) as u8;
                    let val = if gray.get_pixel(x, y)[0] > threshold.saturating_sub(offset) {
                        255
                    } else {
                        0
                    };
                    bw.put_pixel(x, y, Luma([val]));
                }
            }
        } else {
            for y in 0..height {
                for x in 0..width {
                    let pixel = gray.get_pixel(x, y);
                    let val = if pixel[0] > self.threshold { 255 } else { 0 };
                    bw.put_pixel(x, y, Luma([val]));
                }
            }
        }
        bw
    }

    fn scan_document(&self, input_path: &str, output_path: Option<&str>) -> Result<String, Box<dyn std::error::Error>> {
        let img = image::open(input_path)?;
        let (w, h) = img.dimensions();

        // Упрощённо: обрезаем и преобразуем
        let bw = self.to_black_white(&img);

        let out_path = if let Some(path) = output_path {
            path.to_string()
        } else {
            let base = Path::new(input_path).file_stem().unwrap().to_str().unwrap();
            format!("{}_scanned_bw.{}", base, self.format)
        };

        if self.format == "png" {
            bw.save(&out_path)?;
        } else {
            let bgr = image::ImageBuffer::from_fn(w, h, |x, y| {
                let val = bw.get_pixel(x, y)[0];
                image::Rgb([val, val, val])
            });
            bgr.save(&out_path)?;
        }

        Ok(out_path)
    }

    fn batch_process(&self, input_dir: &str, output_dir: Option<&str>) -> Result<(), Box<dyn std::error::Error>> {
        let exts = [".jpg", ".jpeg", ".png", ".bmp", ".tiff"];
        let entries = fs::read_dir(input_dir)?;
        let mut files = Vec::new();

        for entry in entries {
            let entry = entry?;
            let path = entry.path();
            if path.is_file() {
                if let Some(ext) = path.extension() {
                    let ext_str = format!(".{}", ext.to_str().unwrap_or("").to_lowercase());
                    if exts.contains(&ext_str.as_str()) {
                        files.push(path);
                    }
                }
            }
        }

        if files.is_empty() {
            println!("❌ В папке нет поддерживаемых изображений.");
            return Ok(());
        }

        if let Some(dir) = output_dir {
            fs::create_dir_all(dir)?;
        }

        println!("📁 Найдено {} изображений.", files.len());
        let mut success = 0;

        for (i, file) in files.iter().enumerate() {
            println!("\n[{}/{}] Обработка: {}", i+1, files.len(), file.file_name().unwrap().to_str().unwrap());
            let out_path = if let Some(dir) = output_dir {
                let base = file.file_stem().unwrap().to_str().unwrap();
                Some(format!("{}/scanned_{}.{}", dir, base, self.format))
            } else {
                None
            };
            let start = Instant::now();
            match self.scan_document(file.to_str().unwrap(), out_path.as_deref()) {
                Ok(result) => {
                    let elapsed = start.elapsed().as_secs_f64();
                    println!("✅ Сохранено: {} ({:.2} сек)", result, elapsed);
                    success += 1;
                }
                Err(e) => println!("❌ Ошибка: {}", e),
            }
        }
        println!("\n✅ Готово! Обработано {} из {} изображений.", success, files.len());
        Ok(())
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = env::args().collect();
    let mut input = None;
    let mut output = None;
    let mut threshold = 127;
    let mut adaptive = true;
    let mut format = "png".to_string();
    let mut batch = false;

    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--output" | "-o" => { output = Some(args[i+1].clone()); i += 2; }
            "--threshold" | "-t" => { threshold = args[i+1].parse().unwrap_or(127); i += 2; }
            "--no-adaptive" => { adaptive = false; i += 1; }
            "--format" | "-f" => { format = args[i+1].clone(); i += 2; }
            "--batch" | "-b" => { batch = true; i += 1; }
            _ => {
                if input.is_none() {
                    input = Some(args[i].clone());
                }
                i += 1;
            }
        }
    }

    if input.is_none() {
        println!("Usage: cargo run -- <image> [--threshold 127] [--format png] [--batch]");
        return Ok(());
    }

    println!("\x1b[36m📄 DocScan B&W (Rust)\x1b[0m");

    let scanner = Scanner::new(threshold as u8, adaptive, &format);

    let input_path = input.unwrap();

    if batch || Path::new(&input_path).is_dir() {
        scanner.batch_process(&input_path, output.as_deref())?;
    } else {
        let start = Instant::now();
        match scanner.scan_document(&input_path, output.as_deref()) {
            Ok(result) => {
                let elapsed = start.elapsed().as_secs_f64();
                println!("✅ Сохранено: {} ({:.2} сек)", result, elapsed);
            }
            Err(e) => {
                println!("\x1b[31m❌ Ошибка: {}\x1b[0m", e);
                std::process::exit(1);
            }
        }
    }

    Ok(())
}
