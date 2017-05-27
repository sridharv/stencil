// Package num provides numeric utilities intended to be used with stencil
//
// As an example, to use a version of num specialized for int32 use
//
//     import "github.com/sridharv/stencil/std/num/Number/int32"
package num


type Number float64

// Max returns the largest number in n
func Max(n...Number) Number {
	if len(n) == 0 {
		return 0
	}
	max := n[0]
	for _, e := range n[1:] {
		if max < e {
			max = e
		}
	}
	return max
}

// Min returns the smallest number in n
func Min(n...Number) Number {
	if len(n) == 0 {
		return 0
	}
	min := n[0]
	for _, e := range n[1:] {
		if min > e {
			min = e
		}
	}
	return min
}

// Sum returns the sum of all numbers in n
func Sum(n...Number) Number {
	var s Number
	for _, e := range n {
		s += e
	}
	return s
}
