package skiptree

import "fmt"

// RangeNode defines a value associated with the range [Lower, Upper].
// Key is the lower bound and orders the nodes.
type RangeNode struct {
	Key   float64
	Lower float64
	Upper float64
	Value float64
}

// returns the contents of a RangeNode
func stringify(n *RangeNode) string {
	return fmt.Sprintf("Key:%g {Lower:%g Upper:%g} Value:%g", n.Key, n.Lower, n.Upper, n.Value)
}
