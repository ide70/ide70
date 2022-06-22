package api

import (
	"github.com/ide70/ide70/dataxform"
	"github.com/newm4n/go-dfe"
	"time"
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
	logger.Info("PureDate", year, month, day)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func (dc *DateCtx) PureTime(hourI, minuteI, secondI interface{}) time.Time {
	hour := dataxform.IAsInt(hourI)
	minute := dataxform.IAsInt(minuteI)
	second := dataxform.IAsInt(secondI)
	logger.Info("PureTime", hour, minute, second)
	return time.Date(0, 1, 1, hour, minute, second, 0, time.UTC)
}

func (dc *DateCtx) SetSecond(tI interface{}, secondI interface{}) time.Time {
	t := dc.AsTime(tI)
	second := dataxform.IAsInt(secondI)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), second, t.Nanosecond(), t.Location())
}

func (dc *DateCtx) SetMinute(tI interface{}, minuteI interface{}) time.Time {
	t := dc.AsTime(tI)
	minute := dataxform.IAsInt(minuteI)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, t.Second(), t.Nanosecond(), t.Location())
}

func (dc *DateCtx) SetHour(tI interface{}, hourI interface{}) time.Time {
	t := dc.AsTime(tI)	
	hour := dataxform.IAsInt(hourI)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func (dc *DateCtx) Parse(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}

func (dc *DateCtx) FormatTime(tI interface{}, format string) string {
	t := dc.AsTime(tI)
	if t == nil {
		return ""
	}
	translation := DateFormatExchange.NewPatternTranslation()
	return t.Format(translation.JavaToGoFormat(format))
}

func (dc *DateCtx) AsTime(i interface{}) *time.Time {
	switch iT := i.(type) {
	case time.Time:
		return &iT
	case *time.Time:
		return iT
	case string:
		t, err := time.Parse(time.RFC3339, iT)
		if err != nil {
			logger.Error(err.Error())
		} else {
			return &t
		}
	}
	return nil
}
