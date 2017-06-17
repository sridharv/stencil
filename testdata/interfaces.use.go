package use

import (
	int_set "ifaces/interface/int"
)

func CountUnique(args ...int) int {
	s := int_set.Set{}
	for _, a := range args {
		s.Add(a)
	}
	return len(s)
}
