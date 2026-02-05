package engine

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"sync"
)

// райтер чтобы писать побитово
type bitwriter struct {
	dest *bufio.Writer
	acc  uint64
	n    uint8
}

// создать новый битовый райтер
func newBitWriter(f *bufio.Writer) *bitwriter {
	return &bitwriter{dest: f}
}

// записать битовую посл-ность v длины l в w
func (bw *bitwriter) write(v uint64, l uint8) error {
	bw.acc <<= l
	bw.acc |= v & ((1 << l) - 1)
	bw.n += l

	for bw.n >= 8 {
		bw.n -= 8

		if err := bw.dest.WriteByte(byte(bw.acc >> bw.n)); err != nil {
			return err
		}
	}
	return nil
}

// завершить запись в w (дополнить нулями до 8)
func (bw *bitwriter) flush() error {
	if bw.n > 0 {
		byteToWrite := byte(bw.acc << (8 - bw.n))

		if err := bw.dest.WriteByte(byteToWrite); err != nil {
			return err
		}
		bw.n, bw.acc = 0, 0
	}
	return nil
}

// буффер пул для того, чтобы каждый раз не аллоцировались новые
var (
	magic      = [2]byte{67, 67}
	bufferPool = sync.Pool{
		New: func() any {
			b := make([]byte, 64*4096)
			return &b
		},
	}
	treePool = sync.Pool{
		New: func() any {
			return &Tree{}
		},
	}
)

// функция собирает кучу и дерево на основе файла (по 1 проходу), и вызывает pack (2 проход по файлу),
// она вызывается из мейна
func Compress(filename string, size uint64, inputFile io.Reader, outputFile io.Writer) error {
	if size == 0 {
		return errors.New("empty input.")
	}

	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufferPool.Put(bufPtr)

	fr := CalcFreq(inputFile, buf)

	newtree := treePool.Get().(*Tree)
	newtree.reset()
	defer treePool.Put(newtree)

	tree, i := newtree.buildTree(fr)

	lens, codes, used := tree.prepare(i)
	if sk, ok := inputFile.(io.ReadSeeker); ok {
		_, err := sk.Seek(0, 0)
		if err != nil {
			return err
		}
	} else {
		return errors.New("file is not seekable. can't archive that.")
	}

	var outputWriter *bufio.Writer
	if nw, ok := outputFile.(*bufio.Writer); ok {
		outputWriter = nw
	} else {
		outputWriter = bufio.NewWriter(outputFile)
	}

	return pack(inputFile, outputWriter, &lens, &codes, used, size, []byte(filename), buf)
}

// magic number 2B | nameSize 2B | original name + ext NB | size 8B | N 1B | lengths NB | data ...B  | CRC32 4B
//
//	^-- пакуем данные в файл в w вот с такой структурой --^
func pack(input io.Reader, w *bufio.Writer, le *[256]byte, c *[256]uint64, used uint16, size uint64, filename []byte, buf []byte) error {
	_, err := w.Write(magic[:]) // магические байты

	var tmp [8]byte
	binary.LittleEndian.PutUint16(tmp[:2], uint16(len(filename)))

	if _, err := w.Write(tmp[:2]); err != nil {
		return err
	}

	w.Write(filename)
	binary.LittleEndian.PutUint64(tmp[:], size)

	if _, err := w.Write(tmp[:]); err != nil {
		return err
	}

	if used < 32 {
		w.WriteByte(uint8(used))
		for i := 0; i < 256; i++ {
			if le[i] > 0 {
				tmp[0], tmp[1] = byte(i), le[i]
				if _, err := w.Write(tmp[:2]); err != nil {
					return err
				}
			}
		}
	} else {
		w.WriteByte(0)
		_, err = w.Write(le[:])
	}

	if err != nil {
		return err
	}

	hasher := crc32.NewIEEE() // новый хэшер
	bw := newBitWriter(w)
	for {
		n, err := input.Read(buf)

		if n > 0 {
			for _, b := range buf[:n] {
				if err := bw.write(c[b], le[b]); err != nil {
					return err
				}
			}
			hasher.Write(buf[:n])
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}
	}

	if err := bw.flush(); err != nil {
		return err
	}
	binary.Write(w, binary.LittleEndian, hasher.Sum32())

	return w.Flush()
}
