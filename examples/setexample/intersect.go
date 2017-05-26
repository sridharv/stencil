package set_example

import (
	string_set "github.com/sridharv/stencil/collections/set/Element/string"
)

func Common(list1, list2 []string) []string {
	return string_set.Of(list1...).Intersection(string_set.Of(list2...)).AsSlice()
}
