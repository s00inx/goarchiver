package internal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kfcemployee/goarchiver/internal/engine"
)

// путь к файлу --> архивация --> путь к архиву или ошибка
func PackFile(path string, odir string) (string, error) {
	// открываем файл
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("error opening a file: %w", err)
	}
	defer file.Close()

	// обрабатываем пустой файл (частный случай)
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("error getting file metadata: %w", err)
	}

	// даже если файл пустой создаем новый файл для результата
	base, ext := filepath.Base(path), filepath.Ext(path)
	fn := base[:len(base)-len(ext)] + "_zipped.arc"
	var outpath string
	if odir != "" {
		outpath = filepath.Join(odir, fn)
	} else {
		dir := filepath.Dir(path)
		outpath = filepath.Join(dir, fn)
	}
	out, err := os.Create(outpath)
	if err != nil {
		return "", fmt.Errorf("error creating a result file: %w", err)
	}
	defer func() {
		out.Close()
	}()

	// обрабатываем пустой файл и выходим
	if stat.Size() == 0 {
		return outpath, nil
	}

	err = engine.Compress(base, uint64(stat.Size()), file, out)
	if err != nil {
		out.Close()
		os.Remove(outpath)
		return "", fmt.Errorf("error compressing a file: %w", err)
	}

	if err := out.Close(); err != nil {
		return "", fmt.Errorf("error closing result file: %w", err)
	}
	return outpath, nil
}

// распаковать файл (архив --> ошибка или создать файл)
func UnpackFile(srcpath string) error {
	f, err := os.Open(srcpath)
	if err != nil {
		return fmt.Errorf("error opening a file.")
	}

	r := bufio.NewReader(f) // делаем новый ридер на файл 1 раз

	err = engine.Open(r)
	if err != nil {
		return err
	}

	dest, err := engine.ReadDest(r, srcpath)
	if err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}

	if err = engine.Decompress(r, out); err != nil {
		os.Remove(dest)
		return err
	}

	return nil
}
