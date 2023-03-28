# btree

[![Go Reference](https://pkg.go.dev/badge/github.com/andjam/btree.svg)](https://pkg.go.dev/github.com/andjam/btree)

Package btree implements B-Trees as described in CLRS. B-Trees are balanced
search trees with an arbitrary branching factor t, t > 2. A high branching
factor keeps the height of the tree small, which grows with the number of
keys n, as O(logₜn). The number of nodes in the tree stays small as a result,
decreasing the peformance penalty of allocating new ones. This makes B-Trees
ideal for implementing cache efficient insert, delete and sequential access
operations.
