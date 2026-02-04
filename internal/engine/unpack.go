package engine

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash"
	"hash/crc32"
	"io"
	"path/filepath"
	"sync"
)

type bitreader struct {
	r   *bufio.Reader
	acc uint64
	n   int
}

func newBitReader(src *bufio.Reader) *bitreader {
	return &bitreader{
		r: src,
	}
}

// дозаполнить аккумулятор, чтобы не читать каждый бит
func (br *bitreader) fillacc() error {
	for br.n <= 56 {
		b, err := br.r.ReadByte()
		if err != nil {
			return err
		}
		br.acc = (br.acc << 8) | uint64(b)
		br.n += 8
	}

	return nil

}

// расшифровать следующий символ на осноые уже готовых данных
func (br *bitreader) decodeNext(
	mincodes *[65]int,
	symb *[256]byte,
	symboffsets *[65]int,
	counts *[65]int) (byte, error) {

	_ = br.fillacc()
	code := 0
	n, acc := br.n, br.acc

	for l := 1; l < 65; l++ {
		if n <= 0 {
			return 0, io.EOF
		}
		bit := int((acc >> (n - 1)) & 1)
		n--

		code = (code << 1) | bit

		if code < mincodes[l]+counts[l] {
			br.acc = acc
			br.n = n
			offsetInSegment := code - mincodes[l]
			symbol := symb[symboffsets[l]+offsetInSegment]
			return symbol, nil
		}
	}
	return 0, errors.New("code not found")
}

// подготовить данные для восстановления кодов (дерева, но как плоский список)
func prepareData(le [256]byte) ([65]int, [256]byte, [65]int, [65]int) {
	var (
		mincodes    [65]int
		symb        [256]byte
		symboffsets [65]int
		counts      [65]int
	)

	for _, l := range le {
		if l > 0 {
			counts[l]++
		}
	}

	pos := 0
	for i := 1; i < 65; i++ {
		symboffsets[i] = pos
		pos += counts[i]
	}

	var cur [65]int
	for ch, len := range le {
		if len > 0 {
			symb[symboffsets[len]+cur[len]] = byte(ch)
			cur[len]++
		}
	}

	code := 0
	for i := 1; i < 65; i++ {
		mincodes[i] = code
		code = (code + counts[i]) << 1
	}

	return mincodes, symb, symboffsets, counts
}

// открыть файл с архивом (и проверить есть ли такой)
func Open(br *bufio.Reader) error {
	var mg [2]byte // читаем магические байты
	if _, err := io.ReadFull(br, mg[:]); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return errors.New("empty archive file/ can't unpack that")
		}
		return err
	}

	if mg != magic {
		return errors.New("invalid file, not archive or smth")
	}

	return nil
}

var (
	pool = sync.Pool{ // пул для буферов
		New: func() any {
			b := make([]byte, 32*1024) // 32 кБ
			return &b
		},
	}
)

// прочитать имя
func ReadDest(br *bufio.Reader, rpath string) (string, error) {
	var tmpbuf [2]byte
	if _, err := io.ReadFull(br, tmpbuf[:]); err != nil {

		return "", err
	}

	nameSize := binary.LittleEndian.Uint16(tmpbuf[:])
	namebuf := make([]byte, nameSize)
	io.ReadFull(br, namebuf)

	name := string(namebuf)
	destPath := filepath.Join(filepath.Dir(rpath), "unpacked_"+name)

	return destPath, nil
}

func diffHash(hasher hash.Hash32, bitr *bitreader) error {
	var tmpbuf [4]byte
	bitr.n &= ^7
	hi := 0
	for bitr.n >= 8 && hi < 4 {
		tmpbuf[hi] = byte(bitr.acc >> (bitr.n - 8))
		bitr.n -= 8
		hi++
	}

	if hi < 4 {
		if _, err := io.ReadFull(bitr.r, tmpbuf[hi:]); err != nil {
			return errors.New("failed to read hash")
		}
	} else {
		io.Copy(io.Discard, bitr.r)
	}

	originalHash := binary.LittleEndian.Uint32(tmpbuf[:])
	if hasher.Sum32() != originalHash {
		return errors.New("data is corrupted: hash mismatch")
	}

	return nil

}

// распаковать архив в файл с таким же названием
// magic number 2B | nameSize 2B | original name + ext NB | size 8B | N 1B | lengths NB | data ...B  | CRC32 4B
func Decompress(br *bufio.Reader, out io.Writer) error {
	var tmpbuf [8]byte

	if _, err := io.ReadFull(br, tmpbuf[:]); err != nil {
		return err
	}
	originalSize := binary.LittleEndian.Uint64(tmpbuf[:])

	nlen, err := br.ReadByte()
	if err != nil {
		return err
	}

	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr
	defer pool.Put(bufPtr)

	var le [256]byte
	if nlen > 0 {
		l := int(nlen)
		if _, err := io.ReadFull(br, buf[:l*2]); err != nil {
			return err
		}

		for i := 0; i < l; i++ {
			le[buf[i*2]] = buf[i*2+1]
		}
	} else {
		_, err = io.ReadFull(br, le[:])
		if err != nil {
			return err
		}
	}

	hasher := crc32.NewIEEE()

	var outWriter *bufio.Writer
	if bw, ok := out.(*bufio.Writer); ok {
		outWriter = bw
	} else {
		outWriter = bufio.NewWriter(out)
	}

	bitr := newBitReader(br)
	mc, sy, syo, cou := prepareData(le)
	ct := 0
	for i := 0; i < int(originalSize); i++ {
		symbol, err := bitr.decodeNext(&mc, &sy, &syo, &cou)
		if err != nil {
			return errors.New("decoding error: invalid data")
		}

		buf[ct] = symbol
		ct++

		if ct == len(buf) {
			outWriter.Write(buf)
			hasher.Write(buf)
			ct = 0
		}

	}

	if ct > 0 {
		outWriter.Write(buf[:ct])
		hasher.Write(buf[:ct])
	}

	if err := diffHash(hasher, bitr); err != nil {
		return err
	}

	return outWriter.Flush()
}
