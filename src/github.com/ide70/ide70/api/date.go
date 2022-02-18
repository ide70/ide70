package api

import (
	"time"
)

type DateCtx struct{}

func (dc *DateCtx) Now() time.Time {
	return time.Now()
}

func (dc *DateCtx) Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) time.Time {
	return time.Date(year, month, day, hour, min, sec, nsec, loc)
}

func (dc *DateCtx) Parse(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}
