package set_example

//go:generate stencil
import (
	string_set "github.com/deckarep/golang-set/interface/string"
)

func Common(list1, list2 []string) []string {
	s1 := string_set.NewSetFromSlice(list1)
	intersection := s1.Intersect(string_set.NewSetFromSlice(list2))
	return intersection.ToSlice()
}
