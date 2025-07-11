package utils

import "fmt"

func FormatHourTimestamp(hour int) string {
	if hour == 0 {
		return "12:00 AM"
	} else if hour < 12 {
		return fmt.Sprintf("%d:00 AM", hour)
	} else if hour == 12 {
		return "12:00 PM"
	} else {
		return fmt.Sprintf("%d:00 PM", hour-12)
	}
}
