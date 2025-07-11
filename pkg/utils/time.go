package utils

import (
	"log"
	"time"
)

var (
	AlmatyLocation *time.Location
)

func init() {
	var err error
	AlmatyLocation, err = time.LoadLocation("Asia/Almaty")
	if err != nil {
		log.Printf("Ошибка загрузки таймзоны Asia/Almaty: %v", err)
		// Fallback на фиксированную таймзону +6
		AlmatyLocation = time.FixedZone("ALMT", 5*60*60)
	}
}

func NowAlmaty() time.Time {
	return time.Now().In(AlmatyLocation)
}

func ToAlmaty(t time.Time) time.Time {
	return t.In(AlmatyLocation)
}

func FormatAlmaty(t time.Time, layout string) string {
	return t.In(AlmatyLocation).Format(layout)
}

func FormatAlmatyDefault(t time.Time) string {
	return FormatAlmaty(t, "2006-01-02 15:04:05 MST")
}

func ParseAlmaty(layout, value string) (time.Time, error) {
	t, err := time.ParseInLocation(layout, value, AlmatyLocation)
	return t, err
}

func UTCToAlmaty(utcTime time.Time) time.Time {
	return utcTime.In(AlmatyLocation)
}

func AlmatyToUTC(almatyTime time.Time) time.Time {
	return almatyTime.UTC()
}
