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

// загрузить данные для тестов
func loadData() []byte {
	data, _ := os.ReadFile("../../testdata/nci") // здесь можно вставить путь у к любому файлу
	// data := bytes.Repeat([]byte("ThisDataEquals16B"), 64*1024) // пока просто повторяющиеся байты (1 Мб)
	return data
}

func BenchmarkUnpack(b *testing.B) {
	raw := loadData()

	var compbuf bytes.Buffer
	compbuf.Grow(len(raw))
	Compress("test.txt", uint64(len(raw)), bytes.NewReader(raw), &compbuf) // 1 раз сжимаем данные и создаем ридеры

	br := bufio.NewReader(&compbuf)
	cbytes := compbuf.Bytes()
	inputReader := bytes.NewReader(nil)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		inputReader.Reset(cbytes) // очищаем буферы перед очередным проходом
		br.Reset(inputReader)

		Open(br) // просто скипаем эти 2 функции
		ReadDest(br, "")

		err := Decompress(br, io.Discard)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.SetBytes(int64(len(raw)))
}

func BenchmarkPack(b *testing.B) {
	raw := loadData()
	le := uint64(len(raw))

	input := bytes.NewReader(nil)
	br := bufio.NewReader(input)

	b.ReportAllocs()
	b.SetBytes(int64(le))

	b.ResetTimer()

	for b.Loop() {
		input.Reset(raw)
		br.Reset(input)

		_ = Compress("test.data", le, input, io.Discard)
	}
}

func BenchmarkGzipUnpack(b *testing.B) {
	raw := loadData()

	var compbuf bytes.Buffer
	gw := gzip.NewWriter(&compbuf)
	gw.Write(raw)
	gw.Close()
	cbytes := compbuf.Bytes()

	inputReader := bytes.NewReader(nil)
	gr, _ := gzip.NewReader(bytes.NewReader(cbytes))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		inputReader.Reset(cbytes)
		_ = gr.Reset(inputReader)

		// Распаковываем в никуда
		_, err := io.Copy(io.Discard, gr)
		if err != nil {
			b.Fatal(err)
		}
		_ = gr.Close()
	}

	b.SetBytes(int64(len(raw)))
}

func BenchmarkGzipPack(b *testing.B) {
	raw := loadData()
	le := int64(len(raw))

	gw := gzip.NewWriter(io.Discard)

	b.ReportAllocs()
	b.SetBytes(le)
	b.ResetTimer()

	for b.Loop() {
		gw.Reset(io.Discard)

		_, err := gw.Write(raw)
		if err != nil {
			b.Fatal(err)
		}

		err = gw.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
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
