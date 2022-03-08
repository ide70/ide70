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

func (dc *DateCtx) Parse(layout, value string) time.Time {
	t, _ := time.Parse(layout, value)
	return t
}

func (dc *DateCtx) FormatTime(t *time.Time, format string) string {
	translation := DateFormatExchange.NewPatternTranslation()
	return t.Format(translation.JavaToGoFormat(format))
}
