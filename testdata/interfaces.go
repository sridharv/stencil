package ifaces

type holder struct {
	data interface{}
}

type Set map[interface{}]struct{}

func (s Set) Add(a interface{}) {
	s[a] = struct{}{}
}

func (s Set) Delete(a interface{}) {
	delete(s, a)
}
