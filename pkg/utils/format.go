package utils

import (
	"fmt"
	"time"
)

func FormatPeriod(start, end time.Time) string {
	return fmt.Sprintf("%s - %s",
		start.Format("2006-01-02 15:04"),
		end.Format("2006-01-02 15:04"))
}
