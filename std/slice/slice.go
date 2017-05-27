package slice

type Element interface{}

// Any returns true if fn is true for any elements of s
func Any(s []Element, fn func(Element) bool) bool {
	return IndexFunc(s, fn) != -1
}

// Any returns true if fn is true for all elements of s
func All(s []Element, fn func(Element) bool) bool {
	return IndexFunc(s, func(e Element) bool { return !fn(e) }) == -1
}

// IndexFunc returns the index of the first element for which fn returns true.
// If no such element exists it returns -1.
func IndexFunc(s []Element, fn func(Element) bool) int {
	for i, e := range s {
		if fn(e) {
			return i
		}
	}
	return -1
}

// Index returns the first index of e in s
func Index(s []Element, e Element) int {
	return IndexFunc(s, func(el Element) bool { return el == e })
}
