package bbox

import (
	"fmt"
	"testing"
)

func TestNewBoundingBox(t *testing.T) {
	// 5km radius in CGH airport
	bbox := NewBoundingBox(-23.627238, -46.655919, 5000)

	fmt.Println(bbox)
}
