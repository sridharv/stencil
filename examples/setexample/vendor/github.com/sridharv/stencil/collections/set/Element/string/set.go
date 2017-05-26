package set

// Element is the type of element held by the set.

// Of returns a set containing all elements of e
func Of(e ...string) Set {
	s := Set{}
	s.AddAll(e...)
	return s
}

// Set is a set of type Element
type Set map[string]struct{}

// Add adds e to the set s
func (s Set) Add(e string) { s[e] = struct{}{} }

// Remove removes e from the set s
func (s Set) Remove(e string) { delete(s, e) }

// Intersection returns a new set which is the intersection of s and o
func (s Set) Intersection(o Set) Set {
	r := Set{}
	for k := range s {
		if _, ok := o[k]; ok {
			r[k] = struct{}{}
		}
	}
	return r
}

// AddAll adds all elements in e to the set s
func (s Set) AddAll(e ...string) {
	for _, elem := range e {
		s[elem] = struct{}{}
	}
}

// AsSlice returns the elements of s as a slice
func (s Set) AsSlice() []string {
	r, i := make([]string, len(s)), 0
	for k := range s {
		r[i] = k
		i++
	}
	return r
}
