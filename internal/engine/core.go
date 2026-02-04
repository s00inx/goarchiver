package engine

import (
	"io"
	"sort"
)

// узел для дерева
type Node struct {
	Freq int   // 8Б частота (ну то есть макс частота может быть 2 миллиарда)
	l, r int32 // индексы 4+4 Б

	Char byte // 1Б на сам символ + 7 паддинг = 24Б
}

// дерево для кодов
// в дата оно хранится, максимально узлов в дереве - 2n - 1 = 511, тк n = 256 (всего 256 значений байт)
type Tree struct {
	data [512]Node
	next int32
}

// бинарная куча - плоский массив для узлов + указатель на дерево, которое будет построено
type Heap struct {
	tree *Tree
	data []int32
}

// выделяем новую ноду
func (t *Tree) newNode(ch byte, freq int) int32 {
	i := t.next
	t.data[i] = Node{Char: ch, Freq: freq, l: -1, r: -1}
	t.next++

	return i
}

// подготовить каноническое дерево
func (t *Tree) prepare(rootI int32) ([256]byte, [256]uint64, uint16) {
	var le [256]byte
	getLengths(t, rootI, 0, &le)

	type item struct {
		char byte
		l    byte
	}

	var sorteddata [256]item
	var used uint16

	for c, l := range le {
		if l > 0 {
			sorteddata[used] = item{byte(c), l}
			used++
		}
	}

	sorted := sorteddata[:used]
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].l != sorted[j].l {
			return sorted[i].l < sorted[j].l
		}
		return sorted[i].char < sorted[j].char
	})

	var codes [256]uint64
	var cur uint64
	var lastLen byte

	for _, it := range sorted {
		if lastLen > 0 {
			cur = (cur + 1) << (it.l - lastLen)
		}
		codes[it.char] = cur
		lastLen = it.l
	}

	return le, codes, used
}

// рекурсивно посчитать длины кодов (так быстрее, учитывая то что их всего 256)
func getLengths(t *Tree, ni int32, depth byte, le *[256]byte) {
	if ni == -1 {
		return
	}

	n := &t.data[ni]

	if n.l == -1 && n.r == -1 {
		le[n.Char] = depth
		return
	}

	getLengths(t, n.l, depth+1, le)
	getLengths(t, n.r, depth+1, le)
}

// передаем само дерево и индекс корня
func (tree *Tree) buildTree(fr [256]int) (*Tree, int32) {
	h := Heap{
		tree: tree,
		data: make([]int32, 0, 256),
	}

	for i, v := range fr {
		if v > 0 {
			h.insert(
				tree.newNode(byte(i), v),
			)
		}
	}

	if len(h.data) == 1 {
		onlynodei := h.data[0] // Берем индекс из кучи, а не 0
		db := byte(0)
		if tree.data[onlynodei].Char == 0 {
			db = 1
		}
		n := tree.newNode(db, 0)
		h.insert(n)
	}

	for len(h.data) > 1 {
		l := h.pop()
		r := h.pop()

		parent := tree.newNode(0, tree.data[l].Freq+tree.data[r].Freq)
		tree.data[parent].l = l
		tree.data[parent].r = r

		h.insert(parent)
	}

	return tree, h.pop()
}

// вставить значение в кучу
func (h *Heap) insert(k int32) {
	h.data = append(h.data, k)
	h.siftUp(len(h.data) - 1)
}

// взять минимум из кучи (корень)
func (h *Heap) pop() int32 {
	i := h.data[0]
	li := len(h.data) - 1

	h.data[0] = h.data[li]
	h.data = h.data[:li]

	h.siftDown(0)
	return i
}

// просеять кучу вниз
func (h *Heap) siftDown(i int) {
	n := len(h.data)
	for {
		l, r := 2*i+1, 2*i+2
		j := i

		if l < n && h.tree.data[h.data[l]].Freq < h.tree.data[h.data[j]].Freq {
			j = l
		}
		if r < n && h.tree.data[h.data[r]].Freq < h.tree.data[h.data[j]].Freq {
			j = r
		}

		if j == i {
			break
		}

		h.data[i], h.data[j] = h.data[j], h.data[i]
		i = j
	}
}

func (h *Heap) siftUp(i int) {
	for i > 0 {
		dt := h.data
		p := (i - 1) / 2
		if h.tree.data[dt[i]].Freq < h.tree.data[dt[p]].Freq {
			dt[i], dt[p] = dt[p], dt[i]
			i = p
		} else {
			break
		}
	}
}

func (t *Tree) reset() {
	t.next = 0
}

// подсчитать частоту символов тексте
func CalcFreq(text io.Reader, buf []byte) [256]int {
	var freqmap [256]int

	for {
		n, err := text.Read(buf)
		for _, ch := range buf[:n] {
			freqmap[ch]++
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
	}

	return freqmap
}
