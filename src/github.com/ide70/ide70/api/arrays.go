package api

import (
"github.com/ide70/ide70/dataxform"
	"regexp"
	"strings"
)

type Arrays struct {
}

type ITableW struct {
	t ITable
}

type TableCriterion interface {
	isTrue(i interface {}) bool
}

type TableLike struct {
	re *regexp.Regexp
}


func (a *API) Arrays() *Arrays {
	return &Arrays{}
}

func (as *Arrays) NewITableW() *ITableW {
	return &ITableW{t: ITable{}}
}

func (tw *ITableW) AddCol(col string, v interface{}) *ITableW {
	row := ensureRow(tw)
	tw.t[row][col] = v
	return tw
}

func (tw *ITableW) AddEmptyRow() *ITableW {
	tw.t = append(tw.t, SIMap{})
	return tw
}

func ensureRow(tw *ITableW) int {
	if len(tw.t) == 0 {
		tw.t = append(tw.t, SIMap{})
	}
	return len(tw.t) - 1
}

func (tw *ITableW) Finalize() ITable {
	logger.Info("finalize, length:", len(tw.t))
	return tw.t
}

func (t ITable) FilterColumn(column string, criterion TableCriterion) ITable{
	res := ITable{}
	for _,m := range t {
		if criterion.isTrue(m[column]) {
			res = append(res, m)
		}
	}
	return res
}

func (as *Arrays) Like(like string) TableCriterion {
	return TableLike{re: convertLikeToRegex(like)}
}

func (l TableLike) isTrue(i interface{}) bool {
	s := dataxform.IAsString(i)
	return l.re.MatchString(s)
}

func convertLikeToRegex(like string) *regexp.Regexp {
	like = strings.ReplaceAll(like, "%", "\\w+")
	re, err := regexp.Compile(like)
	if err != nil {
		logger.Error("compiling regex:", err.Error())
		return regexp.MustCompile("$a")
	}
	return re
}