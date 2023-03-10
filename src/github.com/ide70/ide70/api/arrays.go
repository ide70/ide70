package api

import (
	"regexp"
	"strings"
	//"github.com/robertkrimen/otto"
	//"reflect"
)

type Arrays struct {
}

type ITableW struct {
	t ITable
}

type SIMapW struct {
	m SIMap
}

type RowToInsert struct {
	row SIMap
	itablew *ITableW
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

func (as *Arrays) NewSIMap() SIMap {
	return SIMap{}
}

func (as *Arrays) NewValueMapping() *ValueMapping {
	return &ValueMapping{m: map[interface{}]interface{}{}, rm: map[interface{}]interface{}{}}
}

func (t ITable) Change() *ITableW {
	return &ITableW{t: t}
}

func (m SIMap) Change() *SIMapW {
	return &SIMapW{m: m}
}

func (m *SIMapW) Put(key string, value interface{}) *SIMapW {
	m.m.Put(key, value)
	return m
}

func (m *SIMapW) Finalize() SIMap {
	return m.m
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

func (tw *ITableW) RowToInsert() *RowToInsert {
	return &RowToInsert{itablew: tw, row: SIMap{}}
}

type ScanElement struct {
	scanExpr string
	target string
}

type ScanConfig struct {
	m *SIMap
	elements []*ScanElement
}

func (m SIMap) ScanConfig() *ScanConfig {
	return &ScanConfig{m :&m}
}

func (scfg *ScanConfig) AddScanElement(scanExpr, target string) *ScanConfig {
	scfg.elements = append(scfg.elements, &ScanElement{scanExpr: scanExpr, target:target})
	return scfg
}

func (scfg *ScanConfig) ScanToTable() ITable {
	t := ITable{}
	return t
}

func (m SIMap) HasKey(keyIf interface{}) bool {
	key := IAsString(keyIf)
	_,has := m[key]
	return has
}

func (m SIMap) DeleteKeys(keys IArray) {
	for _,key := range keys {
		m.Delete(IAsString(key))
	}
}

func (m SIMap) NewMapByKeys(keys IArray) SIMap{
	nm := SIMap{}
	for _,keyI := range keys {
		key := IAsString(keyI)
		nm[key] = m[key]
	}
	return nm
}

func (m SIMap) Clear() SIMap{
	for k,_ := range m {
		delete(m,k)
	}
	return m
}

func (m SIMap) AppendMap(am SIMap) SIMap{
	for k,v := range am {
		m[k] = v
	}
	return m
}

/*func (m SIMap) ForEach(f otto.FunctionCall) {
	arg0 := call.Argument(0)
	logger.Debug("is fn:", arg0.IsFunction());
	val, _ := call.Otto.ToValue("hello")
	valThis, _ := call.Otto.ToValue(nil)
	arg0.Call(valThis, val)
}*/

func (r *RowToInsert) AddCol(col string, v interface{}) *RowToInsert {
	r.row[col] = v
	return r
}

func (r *RowToInsert) InsertAt0() *ITableW {
	logger.Debug("RowToInsert before:", r.itablew.t)
	r.itablew.t = append(ITable{r.row}, r.itablew.t...)
	logger.Debug("RowToInsert after:", r.itablew.t)
	return r.itablew
}

func (r *RowToInsert) Append() *ITableW {
	r.itablew.t = append(r.itablew.t, r.row)
	return r.itablew
}

func ensureRow(tw *ITableW) int {
	if len(tw.t) == 0 {
		tw.t = append(tw.t, SIMap{})
	}
	return len(tw.t) - 1
}

func (tw *ITableW) Finalize() ITable {
	logger.Debug("finalize, length:", len(tw.t))
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
	s := IAsString(i)
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