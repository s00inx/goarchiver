package engine

// файл с бенчмарками и тестами

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"testing"
)

func BenchmarkUnpack(b *testing.B) {
	cdata, _ := os.ReadFile("../../testdata/mozilla_zipped.arc")

	// сразу создаем тут и редер и райтер чтобы переиспользовать их в бенчмарках
	inputReader := bytes.NewReader(cdata)
	br := bufio.NewReader(inputReader)

	b.ReportAllocs()
	b.ResetTimer() // теперь сбрасываем таймер и начинаем замер!
	for i := 0; i < b.N; i++ {
		inputReader.Reset(cdata) // очищаем буферы перед очередным проходом
		br.Reset(inputReader)

		Open(br) // просто скипаем эти 2 функции
		ReadDest(br, "")

		err := Decompress(br, io.Discard) // анпак работает с уже созданными ридером, что позволяет не расходовать память на его создание
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(cdata)))
}

func BenchmarkGzipUnpack(b *testing.B) {
	cdata, _ := os.ReadFile("../../testdata/mozilla.gz")

	r, _ := gzip.NewReader(bytes.NewReader(cdata))

	inputReader := bytes.NewReader(cdata)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		inputReader.Reset(cdata)

		_ = r.Reset(inputReader)

		_, err := io.Copy(io.Discard, r)
		if err != nil {
			b.Fatal(err)
		}

		_ = r.Close()
	}

	b.SetBytes(int64(len(cdata)))
}

func BenchmarkBytesPack(b *testing.B) {
	data := bytes.Repeat([]byte("ThisDataEquals16B"), 64*1024) // 64 кБ
	b.SetBytes(int64(len(data)))
	input := bytes.NewReader(data)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.Reset(data)

		err := Compress("test.txt", uint64(len(data)), input, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func loadData() []byte {
	data, _ := os.ReadFile("../../testdata/reymont")
	return data
}

func BenchmarkFilePack(b *testing.B) {
	raw := loadData()
	le := uint64(len(raw))

	b.ReportAllocs()
	b.SetBytes(int64(le))
	b.ResetTimer()

	input := bytes.NewReader(raw)
	for i := 0; i < b.N; i++ {
		input.Reset(raw)
		_ = Compress("test.data", le, input, io.Discard)
	}
}

func BenchmarkGzipFilePack(b *testing.B) {
	raw := loadData()

	w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)

	b.ReportAllocs()
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Reset(io.Discard)
		_, _ = w.Write(raw)
		_ = w.Close()
	}
}

func runRoundTripTest(t *testing.T, name string, originalData []byte, wantErr bool, errtext string) {
	t.Helper()
	var compressed bytes.Buffer
	err := Compress(name, uint64(len(originalData)), bytes.NewReader(originalData), &compressed)
	if err != nil {
		t.Fatalf("Сжатие не удалось: %v", err)
	}

	compressedBytes := compressed.Bytes()
	inputReader := bytes.NewReader(compressedBytes)
	br := bufio.NewReader(inputReader)

	if err := Open(br); err != nil {
		if !checkError(t, name, wantErr, err, errtext) {
			t.Fatalf("Open не удалось: %v", err)
		}
	}
	if _, err := ReadDest(br, ""); err != nil {
		if !checkError(t, name, wantErr, err, errtext) {
			t.Fatalf("ReadDest не удалось: %v", err)
		}
	}

	var decompressed bytes.Buffer
	if err := Decompress(br, &decompressed); err != nil {
		if !checkError(t, name, wantErr, err, errtext) {
			t.Fatalf("Распаковка не удалась: %v", err)
		}

	}

	if !bytes.Equal(originalData, decompressed.Bytes()) {
		t.Errorf("Данные не совпадают! Тест: %s", name)
	}
}

func checkError(t *testing.T, name string, wantErr bool, err error, errText string) bool {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Errorf("[%s] Ожидалась ошибка, но её не было", name)

		}
		if errText != "" && !strings.Contains(err.Error(), errText) {
			t.Errorf("[%s] Получена не та ошибка. Ожидали: %s, получили: %v", name, errText, err)
		}
		return true
	} else if err != nil {
		t.Fatalf("[%s] Непредвиденная ошибка: %v", name, err)
		return true
	}

	return false
}

func TestPackUnpackCycle(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errText string
	}{
		{
			name:    "Short Phrase",
			data:    []byte("если вы это читаете пусть у вас будет хороший день!"),
			wantErr: false,
			errText: "",
		},
		{
			name:    "SingleChar",
			data:    []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
			wantErr: false,
			errText: "",
		},
		{
			name:    "Repeated",
			data:    []byte(bytes.Repeat([]byte("abc123"), 1000)),
			wantErr: false,
			errText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRoundTripTest(t, tt.name, tt.data, tt.wantErr, tt.errText)
		})
	}
}
