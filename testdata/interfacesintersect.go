package ifaces

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
