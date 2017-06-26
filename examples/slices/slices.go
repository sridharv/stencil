package slices

//go:generate stencil
import (
	"strconv"

	"fmt"

	int_slice "github.com/sridharv/stencil/std/slice/T/int"
	str_slice "github.com/sridharv/stencil/std/slice/T/string"
)

// FindInt returns the index in strs containing the string representation of i
func FindInt(strs []string, i int) int {
	return str_slice.Index(strs, strconv.Itoa(i))
}

// FindString returns the index in ints containing the integer value of str
func FindString(ints []int, str string) (int, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("not a number:%s: %v", str, err)
	}
	return int_slice.Index(ints, i), nil
}
