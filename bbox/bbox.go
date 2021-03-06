package bbox

import (
	"fmt"
	"math"
)

type BoundingBox struct {
	MinLatitude  float64
	MinLongitude float64
	MaxLatitude  float64
	MaxLongitude float64
}

var earthCircumference float64 = 40075000

func (b BoundingBox) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", b.MinLatitude, b.MinLongitude, b.MaxLatitude, b.MaxLongitude)
}

func NewBoundingBox(latitude float64, longitude float64, radius float64) BoundingBox {
	dY := (360 * radius) / earthCircumference
	dX := dY * math.Cos(rad(latitude))

	return BoundingBox{
		MinLatitude:  latitude - dY,
		MinLongitude: longitude - dX,
		MaxLatitude:  latitude + dY,
		MaxLongitude: longitude + dX,
	}
}

func rad(deg float64) float64 {
	return deg * math.Pi / 180
}
