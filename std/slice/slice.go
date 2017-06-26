// Package slice implements operations on slices.
//
// All operations act on slices of interface{}. Use stencil to specialise to a type.
package slice

import (
	"reflect"
	"sort"
)

// Any returns true if fn is true for any elements of s
func Any(s []interface{}, fn func(interface{}) bool) bool {
	return IndexFunc(s, fn) != -1
}

// Any returns true if fn is true for all elements of s
func All(s []interface{}, fn func(interface{}) bool) bool {
	return IndexFunc(s, func(e interface{}) bool { return !fn(e) }) == -1
}

// IndexFunc returns the index of the first element for which fn returns true.
// If no such element exists it returns -1.
func IndexFunc(s []interface{}, fn func(interface{}) bool) int {
	for i, e := range s {
		if fn(e) {
			return i
		}
	}
	return -1
}

// Index returns the first index of e in s
func Index(s []interface{}, e interface{}) int {
	return IndexFunc(s, func(el interface{}) bool { return el == e })
}

var (
	zero    interface{}
	needsGC = typeNeedsGC(reflect.TypeOf(zero))
)

func typeNeedsGC(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Interface, reflect.Ptr, reflect.Chan, reflect.Slice:
		return true
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			if typeNeedsGC(t.Field(i).Type) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// The following are taken from https://github.com/golang/go/wiki/SliceTricks
//
// Cut, Delete, DeleteUnordered, Push, Pop, Reverse, Insert, InsertSlice

// Cut removes all elements between i and j.
func Cut(a []interface{}, i, j int) []interface{} {
	if !needsGC {
		return append(a[:i], a[j:]...)
	}
	copy(a[i:], a[j:])
	for k, n := len(a)-j+i, len(a); k < n; k++ {
		a[k] = zero
	}
	return a[:len(a)-j+i]
}

// Delete removes the ith element from a and returns the resulting slice.
func Delete(a []interface{}, i int) []interface{} {
	return Cut(a, i, i+1)
}

// DeleteUnordered removes the ith element in a, without preserving order. It can be faster that
// Delete as it results in much fewer copies.
func DeleteUnordered(a []interface{}, i int) []interface{} {
	a[i] = a[len(a)-1]
	a[len(a)-1] = zero
	return a[:len(a)-1]
}

// Insert inserts v in a at index i and returns the new slice
func Insert(a []interface{}, v interface{}, i int) []interface{} {
	a = append(a, zero)
	copy(a[i+1:], a[i:])
	a[i] = v
	return a
}

// InsertSlice inserts v into a at index i and returns the new slice
func InsertSlice(a []interface{}, v []interface{}, i int) []interface{} {
	return append(a[:i], append(v, a[i:]...)...)
}

// Push pushes v on to the end of a, returning an updated slice.
func Push(a []interface{}, v interface{}) []interface{} {
	return append(a, v)
}

// Pop removes the last element from a, returning an updating slice
func Pop(a []interface{}) (interface{}, []interface{}) {
	return a[len(a)-1], a[:len(a)-1]
}

// Reverse reverses a in place.
func Reverse(a []interface{}) {
	for l, r := 0, len(a)-1; l < r; l, r = l+1, r-1 {
		a[l], a[r] = a[r], a[l]
	}
}

type sorter struct {
	a    []interface{}
	less func(a, b interface{}) bool
}

func (s *sorter) Len() int           { return len(s.a) }
func (s *sorter) Less(i, j int) bool { return s.less(s.a[i], s.a[j]) }
func (s *sorter) Swap(i, j int)      { s.a[i], s.a[j] = s.a[j], s.a[i] }

// Sort sorts a using the comparison function less.
func Sort(a []interface{}, less func(a, b interface{}) bool) {
	sort.Sort(&sorter{a, less})
}

// SortStable sorts a stably using the comparison function less.
func SortStable(a []interface{}, less func(a, b interface{}) bool) {
	sort.Stable(&sorter{a, less})
}

// Flatten returns a slice created by adding each element of each slice in slices
func Flatten(slices ...[]interface{}) []interface{} {
	var a []interface{}
	for _, s := range slices {
		a = append(a, s...)
	}
	return a
}
