package ifaces

type holder struct {
	data interface{}
}

type orderedPair struct {
	first  interface{}
	second interface{}
}

type Set map[interface{}]struct{}

func (s Set) Add(a interface{}) {
	s[a] = struct{}{}
}

func (s Set) Delete(a interface{}) {
	delete(s, a)
}
