package api

import (
	"time"
	"github.com/ide70/ide70/dataxform"
	"github.com/newm4n/go-dfe"
)

type DateCtx struct{}

func (dc *DateCtx) Now() time.Time {
	return time.Now()
}

func (dc *DateCtx) Date(year int, month time.Month, day, hour, min, sec, nsec int, loc *time.Location) time.Time {
	return time.Date(year, month, day, hour, min, sec, nsec, loc)
}

func (dc *DateCtx) PureDate(yearI, monthI, dayI interface{}) time.Time {
	year := dataxform.IAsInt(yearI)
	month := dataxform.IAsInt(monthI)
	day := dataxform.IAsInt(dayI)
	logger.Info("PureDate", year,month,day);
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func (dc *DateCtx) PureTime(hourI, minuteI, secondI interface{}) time.Time {
	hour := dataxform.IAsInt(hourI)
	minute := dataxform.IAsInt(minuteI)
	second := dataxform.IAsInt(secondI)
	logger.Info("PureTime", hour,minute,second);
	return time.Date(0, 1, 1, hour, minute, second, 0, time.UTC)
}

func (dc *DateCtx) SetSecond(t time.Time, secondI interface{}) time.Time {
	second := dataxform.IAsInt(secondI)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), second, t.Nanosecond(), t.Location())
}

func (dc *DateCtx) SetMinute(t time.Time, minuteI interface{}) time.Time {
	minute := dataxform.IAsInt(minuteI)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, t.Second(), t.Nanosecond(), t.Location())
}

func (dc *DateCtx) SetHour(t time.Time, hourI interface{}) time.Time {
	hour := dataxform.IAsInt(hourI)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func (dc *DateCtx) Parse(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}

func (dc *DateCtx) FormatTime(t *time.Time, format string) string {
	translation := DateFormatExchange.NewPatternTranslation()
	return t.Format(translation.JavaToGoFormat(format))
}
