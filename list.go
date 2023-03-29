package btree

type list[T any] []T

func newList[T any](capacity int) list[T] {
	return make(list[T], 0, capacity)
}

func (l *list[T]) splice(i, j int, m *list[T]) {
	l.insertTo(i, m.removeFrom(j, len(*m))...)
}

func (l *list[T]) insertTo(i int, items ...T) {
	var (
		insertedList list[T] = items
		newLen               = len(insertedList) + len(*l)
		j                    = len(insertedList) + i
	)
	*l = (*l)[:newLen]

	if newLen > j {
		copy((*l)[j:], (*l)[i:])
	}
	copy((*l)[i:], insertedList)
}

func (l *list[T]) insert(i int, item T) {
	l.insertTo(i, item)
}

func (l *list[T]) removeFrom(i, j int) list[T] {
	var (
		removedListLen = j - i
		removedList    = make(list[T], removedListLen)
		newLen         = len(*l) - removedListLen
	)
	copy(removedList, (*l)[i:j])
	copy((*l)[i:], (*l)[j:])
	*l = (*l)[:newLen]
	return removedList
}

func (l *list[T]) remove(i int) T {
	return l.removeFrom(i, i+1)[0]
}

func find[T Comparable[T]](l list[T], item T) (int, bool) {
	var (
		low  = 0
		high = len(l)
	)
	for low < high {
		var (
			between  = (low + high) / 2
			compared = item.Compare(l[between])
		)
		if compared < 0 {
			high = between
			continue
		}
		if compared > 0 {
			low = between + 1
			continue
		}
		return between, true
	}
	return low, false
}
