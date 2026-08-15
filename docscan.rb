# docscan.rb — Ruby версия

require 'rmagick'
require 'optparse'

class DocumentScanner
  def initialize(threshold: 127, adaptive: true, format: 'png')
    @threshold = threshold
    @adaptive = adaptive
    @format = format
  end

  def to_black_white(image)
    # Преобразование в ч/б
    if @adaptive
      # Используем адаптивную бинаризацию через ImageMagick
      image = image.quantize(2, Magick::GRAYColorspace)
    else
      image = image.threshold(@threshold * Magick::QuantumRange / 255)
    end
    image
  end

  def scan_document(input_path, output_path = nil)
    img = Magick::Image.read(input_path).first
    raise "Не удалось загрузить изображение" if img.nil?

    # Упрощённое обнаружение документа и коррекция
    # В реальном проекте нужно использовать контуры, здесь упрощённо
    bw = to_black_white(img)

    if output_path.nil?
      base = File.basename(input_path, '.*')
      output_path = "#{base}_scanned_bw.#{@format}"
    end

    if @format == 'png'
      bw.write(output_path) { self.format = 'png' }
    else
      bw.write(output_path) { self.format = 'jpeg' }
    end

    output_path
  end

  def batch_process(input_dir, output_dir = nil)
    exts = %w[.jpg .jpeg .png .bmp .tiff]
    files = Dir.entries(input_dir).select { |f| exts.include?(File.extname(f).downcase) }

    if files.empty?
      puts "❌ В папке нет поддерживаемых изображений."
      return
    end

    FileUtils.mkdir_p(output_dir) if output_dir

    puts "📁 Найдено #{files.size} изображений."
    success = 0

    files.each_with_index do |f, i|
      puts "\n[#{i+1}/#{files.size}] Обработка: #{f}"
      input_path = File.join(input_dir, f)
      output_path = nil
      if output_dir
        base = File.basename(f, '.*')
        output_path = File.join(output_dir, "scanned_#{base}.#{@format}")
      end
      start = Time.now
      begin
        result = scan_document(input_path, output_path)
        elapsed = Time.now - start
        puts "✅ Сохранено: #{result} (#{elapsed.round(2)} сек)"
        success += 1
      rescue => e
        puts "❌ Ошибка: #{e.message}"
      end
    end

    puts "\n✅ Готово! Обработано #{success} из #{files.size} изображений."
  end
end

def main
  options = { threshold: 127, adaptive: true, format: 'png', batch: false }
  input = nil
  output = nil

  OptionParser.new do |opts|
    opts.banner = "Usage: ruby docscan.rb <image> [--threshold 127] [--format png] [--batch]"
    opts.on("--output PATH", "-o", "Путь для сохранения") { |v| output = v }
    opts.on("--threshold N", "-t", Integer, "Порог бинаризации") { |v| options[:threshold] = v }
    opts.on("--no-adaptive", "Отключить адаптивную бинаризацию") { options[:adaptive] = false }
    opts.on("--format FORMAT", "-f", "Формат сохранения") { |v| options[:format] = v }
    opts.on("--batch", "-b", "Пакетная обработка") { options[:batch] = true }
  end.parse!

  input = ARGV[0]
  unless input
    puts "Usage: ruby docscan.rb <image> [--threshold 127] [--format png] [--batch]"
    exit 1
  end

  puts "\e[36m📄 DocScan B&W (Ruby)\e[0m"

  scanner = DocumentScanner.new(**options)

  if options[:batch] || File.directory?(input)
    scanner.batch_process(input, output)
  else
    unless File.exist?(input)
      puts "\e[31m❌ Файл не найден: #{input}\e[0m"
      exit 1
    end
    start = Time.now
    begin
      result = scanner.scan_document(input, output)
      elapsed = Time.now - start
      puts "✅ Сохранено: #{result} (#{elapsed.round(2)} сек)"
    rescue => e
      puts "\e[31m❌ Ошибка: #{e.message}\e[0m"
      exit 1
    end
  end
end

main if __FILE__ == $0
