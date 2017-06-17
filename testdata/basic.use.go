package use

import (
	f32_basic "basic/int/float32"
	"fmt"
)

func CheckMax() {
	var f float32
	f = f32_basic.Max(10.0, 11.0)
	fmt.Println(f)
}
