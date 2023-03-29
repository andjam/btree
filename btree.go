// Package btree implements B-Trees as described in CLRS. B-Trees are balanced
// search trees with an arbitrary branching factor t, t > 2. A high branching
// factor keeps the height of the tree small, which grows with the number of
// keys n, as O(logₜn). The number of nodes in the tree stays small as a result,
// decreasing the peformance penalty of allocating new ones. This makes B-Trees
// ideal for implementing cache efficient insert, delete and sequential access
// operations.
package btree

const (
	t = 512
)

// Comparable defines a total ordering of values of type T. Values within
// BTree are constrained by Comparable to to indicate the order in which they
// are stored.
type Comparable[T any] interface {
	// Compare is called on a value of type T, with another value of type T and
	// indicates the relative order of the two values by returning an int.
	// a.Compare(b) < 0 indicates that value a is less than value b,
	// a.Compare(b) == 0 indicates that a and b match, finally a.Compare(b) > 0
	// indicates that a is greater than b.
	Compare(T) int
}

type BTree[T Comparable[T]] struct {
	root rootNode[T]
}

func NewBTree[T Comparable[T]]() *BTree[T] {
	return &BTree[T]{newRootLeafNode[T]()}
}

// Search searches the tree recursively for the value matching key if such a
// value exists.
func (b BTree[T]) Search(key T) (T, bool) {
	return b.root.search(key)
}

// Insert inserts key into the tree or updates an existing value matching key
// if such a value exists.
func (b *BTree[T]) Insert(key T) {
	if !b.root.isBelowMax() {
		var (
			root    = b.root.asChild()
			newRoot = newRootInternalNode[T]()
		)

		// New values are always placed inside a leaf node. The insert operation
		// recurses down tree in a single pass, searching for the appropriate
		// position in the appropriate leaf node in which to place the value.
		// To guarantee that there is always enough space to place new values;
		// that recursion never descends into a full node; such nodes are split
		// about their median key as they are encountered.
		//
		// Here, the root node of a B-Tree with t = 4 becomes the first child of
		// a new node created from the the median key H, increasing the height
		// of the tree by 1:
		//
		// root
		// (A D F  H L N P)
		// ↓ ↓ ↓ ↓  ↓ ↓ ↓ ↓
		// T₁T₂T₃T₄ T₁T₂T₃T₄
		//
		// The root node originally has and 7 keys (2t-1) and 8 (2t) children.
		// Splitting results in the creation of a new node node which becomes
		// the immediate right sibling of what is now the former root. The two
		// siblings each now have 3 (t-1) keys and 4 (t) children:
		//
		//       newRoot
		//       (H)
		//       ↓ ↓
		// root     sibling
		// (A D F)  (L N P)
		// ↓ ↓ ↓ ↓  ↓ ↓ ↓ ↓
		// T₁T₂T₃T₄ T₁T₂T₃T₄
		medianKey, sibling := root.split()
		newRoot.keys.insert(0, medianKey)
		newRoot.children.insert(0, root)
		newRoot.children.insert(1, sibling)
		b.root = newRoot
	}
	b.root.insertBelowMax(key)
}

// Remove removes the value matching key from the the tree if such a value
// exists, and may result in the shrinking of the tree.
func (b *BTree[T]) Remove(key T) {

	// Like with insertion, removal recurses down the tree in a single pass,
	// rearranging the tree as it goes to maintain its invariants. Unlike
	// insertion, keys can be removed from leaf nodes or internal nodes. Now,
	// care must be taken to ensure that recursion doesn't descend into a node
	// that is too small, rather than one that is too big. This is done by
	// shuffling spare keys between siblings, or merging siblings if necessary.
	b.root.remove(key)
	if !b.root.isAboveMin() {

		// Further, in contrast to the case of insertion into a B-Tree rooted at
		// a full node, where the height of tree increases by 1; removal from a
		// B-Tree may result in an empty internal root node, in which case the
		// first child of the root becomes the new root of the tree.
		//
		// Here, D is removed from a tree with t = 3. The first child of the
		// root becomes the new root of the tree, decreasing the height of the
		// tree by 1.
		//
		//                     root
		//                     (P)
		//                    ↓  ↓
		//     first child             second child
		//     (C        L)            (T     X)
		//     ↓     ↓    ↓            ↓   ↓   ↓
		// (A B) (D E J K) (N 0) (Q R S) (U V) (Y Z)
		//
		// Both children of the root node each with 2 (t-1) keys and 3 (t)
		// children, are too small for recursion to continue. The siblings are
		// merged about P, which moves down from the root to become the median
		// key of the merged node with 5 (2t -1) keys and 6 (t) children, which
		// becomes new root of the tree.
		//
		//     new root
		//     (C       L     P       T     X)
		//     ↓    ↓      ↓      ↓      ↓   ↓
		// (A B) (E J K) (N 0) (Q R S) (U V) (Y Z)
		b.root = b.root.shrink()
	}
}

// node represents functionality common to all nodes in the B-tree. All nodes
// implement node in addition to one of rootNode or childNode.
type node[T Comparable[T]] interface {
	isAboveMin() bool   // Returns true if the degree of node is
	isBelowMax() bool   // Returns true if a node is not full
	search(T) (T, bool) // Searches the subtree rooted at a node for a key
	insertBelowMax(T)   // Inserts a key into the subtree rooted at a non-full node
	remove(T)           // Removes a key from the subtree rooted a node
}

type baseLeafNode[T Comparable[T]] struct {
	keys list[T]
}

func newBaseLeafNode[T Comparable[T]]() baseLeafNode[T] {
	return baseLeafNode[T]{newList[T](2*t - 1)}
}

// search searches  a leaf node just reports if the key is contained within its
// local list of keys.
func (n baseLeafNode[T]) search(key T) (outkey T, found bool) {
	i, found := find(n.keys, key)
	if found {
		return n.keys[i], true
	}
	return
}

// insertBelowMax is called to insert a called at the end, the simple case when
// recursion terminates by inserting k into is local key list.
func (n *baseLeafNode[T]) insertBelowMax(k T) {
	i, found := find(n.keys, k)
	if found {
		n.keys[i] = k
		return
	}
	n.keys.insert(i, k)
}

// remove removes the value matching k from the leaf node n such a value exists.
func (n *baseLeafNode[T]) remove(k T) {
	i, found := find(n.keys, k)
	if found {
		n.keys.remove(i)
	}
}

type baseInternalNode[T Comparable[T]] struct {
	keys     list[T]
	children list[childNode[T]]
}

func newBaseInternalNode[T Comparable[T]]() baseInternalNode[T] {
	return baseInternalNode[T]{
		newList[T](2*t - 1),
		newList[childNode[T]](2 * t)}
}

// search recursively searches the subtree rooted at the internal node n for
// for the value matching k.
func (n baseInternalNode[T]) search(k T) (T, bool) {
	i, found := find(n.keys, k)
	if found {
		return n.keys[i], true
	}
	return n.children[i].search(k)
}

// insertBelowMax inserts k into the subtree rooted a the internal node n, or
// updates the value matching k if such a value already exists.
func (n *baseInternalNode[T]) insertBelowMax(k T) {
	i, found := find(n.keys, k)
	if found {
		n.keys[i] = k
		return
	}

	child := n.children[i]
	if !child.isBelowMax() {
		medianKey, newChild := child.split()
		n.keys.insert(i, medianKey)
		n.children.insert(i+1, newChild)

		if k.Compare(n.keys[i]) > 0 {
			child = newChild
		}
	}
	child.insertBelowMax(k)
}

// remove removes k from the subtree rooted at the internal node n.
func (n *baseInternalNode[T]) remove(k T) {
	var (
		i, found = find(n.keys, k)
		child    = n.children[i]
	)

	if found {
		if child.isAboveMin() {
			n.keys[i] = child.deletePred()
			return
		}
		if n.children[i+1].isAboveMin() {
			n.keys[i] = n.children[i+1].deleteSucc()
			return
		}
		child.merge(n.keys.remove(i), n.children[i+1])
		n.children.remove(i + 1)
	} else if child.isAboveMin() {

		// in this case child child is not too small to remove a key from
		// so continue recursion downwards
	} else if i > 0 && n.children[i-1].isAboveMin() {

		// here, child neads to steal a key from one of it's immediate siblings
		//
		//     new root:
		//     (C       L     P       T     X)
		//     ↓    ↓      ↓      ↓      ↓   ↓
		// (A B) (E J K) (N O) (Q R S) (U V) (Y Z)
		//
		//     new root:
		//     (E       L     P       T     X)
		//     ↓    ↓      ↓      ↓      ↓   ↓
		// (A C) (  J K) (N O) (Q R S) (U V) (Y Z)
		stolen := n.keys.remove(i - 1)
		n.keys.insert(i-1, child.shuffleRight(stolen, n.children[i-1]))
	} else if i < len(n.keys) && n.children[i+1].isAboveMin() {
		stolenKey := n.keys.remove(i)
		n.keys.insert(i, child.shuffleLeft(stolenKey, n.children[i+1]))
	} else if i > 0 {

		//                        n
		//                        (P)
		//                        ↓ ↓
		//     n.children[i-1]      child
		//     (C            L)     (T X)
		//     ↓      ↓       ↓
		// (A B) (D  E  J  K) (N O) …
		//
		// P moves down from the root and becomes the median key between cl and tx:
		// merge the root node and continue recursion
		//
		//     n.children[i-1]
		//     (C              L    P T   X)
		//     ↓       ↓         ↓
		// (A B) (✗   E  J K )  (N O)  …
		n.children[i-1].merge(n.keys.remove(i-1), child)
		n.children.remove(i)
		child = n.children[i-1]
	} else if i < len(n.keys) {
		child.merge(n.keys.remove(i), n.children[i+1])
		n.children.remove(i + 1)
	}
	child.remove(k)
}

// childNode represents the functionality of all nodes which are not the root
// node of the B-tree.
type childNode[T Comparable[T]] interface {
	node[T]
	asRoot() rootNode[T]            // Reconstructs the node as a rootNode
	split() (T, childNode[T])       // Splits node the node, creating a sibling
	merge(T, childNode[T])          // Merges node with a sibling
	deletePred() T                  // Deletes the last key in the subtree
	deleteSucc() T                  // Deletes the first key in the subtree
	shuffleLeft(T, childNode[T]) T  // Shuffles keys around, stealing from the right
	shuffleRight(T, childNode[T]) T // Shuffles keys around, stealing from the left
}

// childLeafNode implements childNode interface, representing a leaf node which
// is not the root of the B-tree.
type childLeafNode[T Comparable[T]] struct {
	baseLeafNode[T]
}

func newChildLeafNode[T Comparable[T]]() *childLeafNode[T] {
	return &childLeafNode[T]{newBaseLeafNode[T]()}
}
func (n childLeafNode[T]) isAboveMin() bool {
	return len(n.keys) > t-1
}
func (n childLeafNode[T]) isBelowMax() bool {
	return len(n.keys) < 2*t-1
}
func (n childLeafNode[T]) asRoot() rootNode[T] {
	return &rootLeafNode[T]{n.baseLeafNode}
}

// split splits node n in to two, returning the median key and newly created
// sibling node intended to sperate the nodes in the parent.
func (n *childLeafNode[T]) split() (T, childNode[T]) {
	sibling := newChildLeafNode[T]()
	sibling.keys.splice(0, t, &n.keys)
	return n.keys.remove(t - 1), sibling
}

// merge merges what is intended to be sibling nodes in order around their
// median key
func (n *childLeafNode[T]) merge(medianKey T, m childNode[T]) {
	sibling := m.(*childLeafNode[T])
	n.keys.insert(len(n.keys), medianKey)
	n.keys.splice(len(n.keys), 0, &sibling.keys)
}

// deletePred deletes the sucessor of some key which is the first key of the
// sub tree rooted at n.
func (n *childLeafNode[T]) deletePred() T {
	return n.keys.remove(len(n.keys) - 1)
}

// deleteSucc deletes the sucessor of some key which is the first key in the
// sub tree rooted at n.
func (n *childLeafNode[T]) deleteSucc() T {
	return n.keys.remove(0)
}

func (n *childLeafNode[T]) shuffleLeft(stolenKey T, m childNode[T]) T {
	sibling := m.(*childLeafNode[T])
	n.keys.insert(len(n.keys), stolenKey)
	return sibling.keys.remove(0)
}

func (n *childLeafNode[T]) shuffleRight(stolenKey T, m childNode[T]) T {
	sibling := m.(*childLeafNode[T])
	n.keys.insert(0, stolenKey)
	return sibling.keys.remove(len(sibling.keys) - 1)
}

// childLeafNode implements childNode interface, representing an internal node
// which is not the root of the B-tree.
type childInternalNode[T Comparable[T]] struct {
	baseInternalNode[T]
}

func newChildInternalNode[T Comparable[T]]() *childInternalNode[T] {
	return &childInternalNode[T]{newBaseInternalNode[T]()}
}

func (n childInternalNode[T]) isAboveMin() bool {
	return len(n.keys) > t-1
}
func (n childInternalNode[T]) isBelowMax() bool {
	return len(n.keys) < 2*t-1
}
func (n childInternalNode[T]) asRoot() rootNode[T] {
	return &rootInternalNode[T]{n.baseInternalNode}
}

// split splits node n in to two, returning the median key and newly created
// sibling node intended to sperate the nodes in the parent.
func (n *childInternalNode[T]) split() (T, childNode[T]) {
	sibling := newChildInternalNode[T]()
	sibling.children.splice(0, t, &n.children)
	sibling.keys.splice(0, t, &n.keys)
	return n.keys.remove(t - 1), sibling
}

// merge merges what is intended to be sibling nodes in order around their
// median key.
func (n *childInternalNode[T]) merge(medianKey T, m childNode[T]) {
	sibling := m.(*childInternalNode[T])
	n.keys.insert(len(n.keys), medianKey)
	n.keys.splice(len(n.keys), 0, &sibling.keys)
	n.children.splice(len(n.children), 0, &sibling.children)
}

// deletePred deletes the sucessor of some key key which is the first key
// of the sub tree rooted at n.
func (n childInternalNode[T]) deletePred() T {
	var (
		i     = 0
		child = n.children[i]
	)
	if child.isAboveMin() {
		return child.deletePred()
	}

	right := n.children[i+1]
	if right.isAboveMin() {
		key := n.keys.remove(i + 1)
		n.keys.insert(i+1, child.shuffleLeft(key, right))
		return child.deletePred()
	}
	n.children.remove(i + 1)
	child.merge(n.keys.remove(i), right)
	return child.deletePred()
}

// deleteSucc deletes the sucessor of some key key which is the first key
// in the sub tree rooted at n.
func (n childInternalNode[T]) deleteSucc() T {
	var (
		i     = len(n.keys)
		child = n.children[i]
	)
	if child.isAboveMin() {
		return child.deleteSucc()
	}

	left := n.children[i-1]
	if left.isAboveMin() {
		key := n.keys.remove(i - 1)
		n.keys.insert(i-1, child.shuffleRight(key, left))
		return child.deleteSucc()
	}
	left.merge(n.keys.remove(i-1), child)
	return left.deleteSucc()
}

func (n *childInternalNode[T]) shuffleLeft(stolenKey T, m childNode[T]) T {
	sibling := m.(*childInternalNode[T])
	n.keys.insert(len(n.keys), stolenKey)
	n.children.insert(len(n.children), sibling.children.remove(0))
	return sibling.keys.remove(0)
}

func (n *childInternalNode[T]) shuffleRight(stolenKey T, m childNode[T]) T {
	sibling := m.(*childInternalNode[T])
	n.keys.insert(0, stolenKey)
	n.children.insert(0, sibling.children.remove(len(sibling.keys)))
	return sibling.keys.remove(len(sibling.keys) - 1)
}

// rootNode represents the functionality of the root node of the tree
type rootNode[T Comparable[T]] interface {
	node[T]
	shrink() rootNode[T]   // Shrinks the subtree when root node is empty
	asChild() childNode[T] // Reconstructs the root node as a child node
}

// rootLeafNode implements rootNode interface, representing a leaf node which
// is the root of the B-tree.
type rootLeafNode[T Comparable[T]] struct {
	baseLeafNode[T]
}

func newRootLeafNode[T Comparable[T]]() *rootLeafNode[T] {
	return &rootLeafNode[T]{newBaseLeafNode[T]()}
}
func (n rootLeafNode[T]) isAboveMin() bool {
	return len(n.keys) > 0
}
func (n rootLeafNode[T]) isBelowMax() bool {
	return len(n.keys) < 2*t-1
}
func (n rootLeafNode[T]) shrink() rootNode[T] {
	return &n
}
func (n rootLeafNode[T]) asChild() childNode[T] {
	return &childLeafNode[T]{n.baseLeafNode}
}

// rootInternalNode implements rootNode interface, representing an internal
// node which is root of the B-tree.
type rootInternalNode[T Comparable[T]] struct {
	baseInternalNode[T]
}

func newRootInternalNode[T Comparable[T]]() *rootInternalNode[T] {
	return &rootInternalNode[T]{newBaseInternalNode[T]()}
}
func (n rootInternalNode[T]) isAboveMin() bool {
	return len(n.keys) > 0
}
func (n rootInternalNode[T]) isBelowMax() bool {
	return len(n.keys) < 2*t-1
}
func (n rootInternalNode[T]) shrink() rootNode[T] {
	return n.children[0].asRoot()
}
func (n rootInternalNode[T]) asChild() childNode[T] {
	return &childInternalNode[T]{n.baseInternalNode}
}
