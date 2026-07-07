package skiptree

import (
	"fmt"
	"slices"
	"strings"
)

// ItemSkipTree holds range nodes ordered by their lower bound and answers
// range-membership queries. It replaces the earlier hand-rolled binary search
// tree: formulas insert ranges in ascending key order, which degenerated that
// tree into a linked list, so a sorted slice with binary search is both
// simpler and faster.
type ItemSkipTree struct {
	nodes []RangeNode
}

// Insert adds the range [lower, upper] with its associated value.
// The lower bound is the key. Inserting an existing key replaces its node.
func (st *ItemSkipTree) Insert(lower float64, upper float64, value float64) {
	n := RangeNode{Key: lower, Lower: lower, Upper: upper, Value: value}
	i, found := slices.BinarySearchFunc(st.nodes, lower, func(rn RangeNode, key float64) int {
		switch {
		case rn.Key < key:
			return -1
		case rn.Key > key:
			return 1
		default:
			return 0
		}
	})
	if found {
		st.nodes[i] = n
		return
	}
	st.nodes = slices.Insert(st.nodes, i, n)
}

// Remove removes the node whose key (lower bound) is `key`.
func (st *ItemSkipTree) Remove(key float64) {
	for i, n := range st.nodes {
		if n.Key == key {
			st.nodes = slices.Delete(st.nodes, i, i+1)
			return
		}
	}
}

// Search returns the value of the range containing rr. Adjacent ranges share
// boundary points; on a shared boundary the range with the smaller key wins,
// matching the traversal order of the original tree.
func (st *ItemSkipTree) Search(rr float64) (value float64, found bool) {
	// find the first node with Lower > rr; candidates are to its left
	i, _ := slices.BinarySearchFunc(st.nodes, rr, func(rn RangeNode, key float64) int {
		if rn.Lower <= key {
			return -1
		}
		return 1
	})
	// walk left over nodes still covering rr to prefer the smaller key
	match := -1
	for j := i - 1; j >= 0 && st.nodes[j].Lower <= rr; j-- {
		if rr <= st.nodes[j].Upper {
			match = j
		}
	}
	if match < 0 {
		return 0.0, false
	}
	return st.nodes[match].Value, true
}

// String prints a visual representation of the ranges in key order.
func (st *ItemSkipTree) String() {
	fmt.Println("------------------------------------------------")
	for _, n := range st.nodes {
		fmt.Println("---[ " + stringify(&n))
	}
	fmt.Println("------------------------------------------------")
}

// Describe returns the ranges in key order as a single string.
func (st *ItemSkipTree) Describe() string {
	var b strings.Builder
	for _, n := range st.nodes {
		b.WriteString(stringify(&n))
		b.WriteString("\n")
	}
	return b.String()
}
