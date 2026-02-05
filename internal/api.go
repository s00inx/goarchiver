package internal

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"

	"github.com/kfcemployee/goarchiver/internal/engine"
)

// путь к файлу --> архивация --> путь к архиву или ошибка
func PackFile(path string, odir string) error {
	// открываем файл
	file, err := os.Open(path)
	if err != nil {
		return errors.New("error opening a file: " + err.Error())
	}
	defer file.Close()

	// обрабатываем пустой файл (частный случай)
	stat, err := file.Stat()
	if err != nil {
		return errors.New("error getting a file metadata: " + err.Error())
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
		return errors.New("error creating a file: " + err.Error())
	}
	defer func() {
		out.Close()
	}()

	err = engine.Compress(base, uint64(stat.Size()), file, out)
	if err != nil {
		out.Close()
		os.Remove(outpath)
		return errors.New("error comressing: " + err.Error())
	}

	if err := out.Close(); err != nil {
		return errors.New("error closing a file: " + err.Error())
	}
	return nil
}

// распаковать файл (архив --> ошибка или создать файл)
func UnpackFile(srcpath string) error {
	f, err := os.Open(srcpath)
	if err != nil {
		return errors.New("error opening a file: " + err.Error())
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
