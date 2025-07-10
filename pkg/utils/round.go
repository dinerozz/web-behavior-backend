package utils

import (
	"fmt"
	"strconv"
)

func RoundToTwoDecimals(value float64) float64 {
	rounded, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return rounded
}
