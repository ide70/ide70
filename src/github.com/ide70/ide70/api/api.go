package api

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"github.com/ide70/ide70/loader"
	"github.com/ide70/ide70/util/log"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

var logger = log.Logger{"api"}

type API struct {
}

var ApiInst = &API{}

func (a *API) Logger() *log.Logger {
	return ApiLogger()
}

func (a *API) DateCtx() *DateCtx {
	return &DateCtx{}
}

func (a *API) LoadCtx() *LoadContext {
	return &LoadContext{}
}

type SIMap map[string]interface{}

type IArray []interface{}

type ITable []SIMap

type ValueMapping struct {
	m  map[interface{}]interface{}
	rm map[interface{}]interface{}
}

func (vm *ValueMapping) Put(k, v interface{}) *ValueMapping {
	vm.m[k] = v
	vm.rm[v] = k
	return vm
}

func (vm *ValueMapping) Get(k interface{}) interface{} {
	return vm.m[k]
}

func (vm *ValueMapping) GetReverse(k interface{}) interface{} {
	return vm.rm[k]
}

type Interface struct {
	I interface{}
}

func (i Interface) AsMap() SIMap {
	switch iT := i.I.(type) {
	case SIMap:
		return iT
	}
	return IAsSIMap(i.I)
}

func (i Interface) AsArray() IArray {
	return IAsArr(i.I)
}

func (i Interface) AsITable() ITable {
	switch iT := i.I.(type) {
	case ITable:
		return iT
	case []interface{}:
		rt := ITable{}
		for _, e := range iT {
			switch eT := e.(type) {
			case SIMap:
				rt = append(rt, eT)
			case map[string]interface{}:
				rt = append(rt, eT)
			}
		}
		return rt
	}
	return ITable{}
}

func (i Interface) AsHTable() *HTable {
	switch iT := i.I.(type) {
	case *HTable:
		return iT
	}
	return nil
}

func (i Interface) AsValueMapping() *ValueMapping {
	switch iT := i.I.(type) {
	case *ValueMapping:
		return iT
	}
	return nil
}

func (i Interface) AsString() String {
	return String{s: IAsString(i.I)}
}

func (i Interface) AsInt64() int64 {
	return IAsInt64(i.I)
}

func (i Interface) AsQueryDef() *QueryDef {
	switch iT := i.I.(type) {
	case *QueryDef:
		return iT
	}
	return nil
}

func (i Interface) AsDBO() *DataBaseObject {
	switch iT := i.I.(type) {
	case *DataBaseObject:
		return iT
	case DataBaseObject:
		return &iT
	}
	return nil
}

func (i Interface) AsBinaryData() *BinaryData {
	switch iT := i.I.(type) {
	case *BinaryData:
		return iT
	case BinaryData:
		return &iT
	}
	return nil
}

func (i Interface) AsUpdateSet() *DboUpdateSet {
	switch iT := i.I.(type) {
	case *DboUpdateSet:
		return iT
	case DboUpdateSet:
		return &iT
	}
	return nil
}

func (m SIMap) Delete(key string) {
	if m == nil {
		return
	}
	delete(m, key)
}

func (m SIMap) Put(key string, value interface{}) {
	if m == nil {
		return
	}
	m[key] = value
}

func (m SIMap) GetForConvert(key string) Interface {
	if m == nil {
		return Interface{nil}
	}
	return Interface{m[key]}
}

func (m SIMap) Get(key string) interface{} {
	if m == nil {
		return nil
	}
	return m[key]
}

func (m SIMap) GetDbId() interface{} {
	return m.Get("_id")
}

func (m SIMap) GetOneColumn() interface{} {
	if len(m) != 1 {
		return nil
	}
	for _, v := range m {
		return v
	}
	return nil
}

func (m SIMap) GetOneColumnForConvert() Interface {
	if len(m) != 1 {
		return Interface{nil}
	}
	for _, v := range m {
		return Interface{v}
	}
	return Interface{nil}
}

func (m SIMap) KeyList() IArray {
	a := []interface{}{}
	for k, _ := range m {
		a = append(a, k)
	}
	return a
}

func (m SIMap) ValueList() IArray {
	a := []interface{}{}
	for _, v := range m {
		a = append(a, v)
	}
	return a
}

func (a IArray) Sort() IArray {
	if len(a) > 0 && IIsInt(a[0]) {
		sort.SliceStable(a, func(i, j int) bool {
			ai := IAsInt64(a[i])
			aj := IAsInt64(a[j])
			return ai < aj
		})
		return a
	}

	sort.SliceStable(a, func(i, j int) bool {
		aiStr := IAsString(a[i])
		ajStr := IAsString(a[j])
		return aiStr < ajStr
		return ByCaseLess(aiStr, ajStr)
	})
	return a
}

func ByCaseLess(si, sj string) bool {
	iRunes := []rune(si)
	jRunes := []rune(sj)

	max := len(iRunes)
	if max > len(jRunes) {
		max = len(jRunes)
	}

	for idx := 0; idx < max; idx++ {
		ir := iRunes[idx]
		jr := jRunes[idx]

		lir := unicode.ToLower(ir)
		ljr := unicode.ToLower(jr)

		if lir != ljr {
			return lir < ljr
		}

		// the lowercase runes are the same, so compare the original
		if ir != jr {
			return ir < jr
		}
	}

	// If the strings are the same up to the length of the shortest string,
	// the shorter string comes first
	return len(iRunes) < len(jRunes)
}

func (a IArray) PrintDBG() {
	logger.Debug("IArray:", a)
}

func (t ITable) PrintDBG() {
	logger.Debug("ITable:", t)
}

func (t ITable) Get(idxIf interface{}) SIMap {
	idx := IAsInt(idxIf)
	if idx < 0 || idx >= len(t) {
		return SIMap{}
	}
	return t[idx]
}

func (a IArray) ToTable(dstCol string) ITable {
	t := ITable{}
	for _, e := range a {
		t = append(t, SIMap{dstCol: e})
	}
	return t
}

type ProcessContext struct {
	ops       []ProcessContextFuncWithArgs
	actIdx    int
	actArgs   []interface{}
	opSubject interface{}
	actElem   interface{}
}

type TblProcessContext struct {
	pc      *ProcessContext
	subject ITable
}

type ArrProcessContext struct {
	pc      *ProcessContext
	subject IArray
}

type TblProcessContextStr struct {
	pc *TblProcessContext
}

type OpResult int

const (
	OK = iota + 1
	FAIL
)

type ProcessContextFunc func(pc *ProcessContext) OpResult

type ProcessContextFuncWithArgs struct {
	ProcessContextFunc
	args []interface{}
}

func opElem(pcA *ArrProcessContext) OpResult {
	pc := pcA.pc
	pc.opSubject = pc.actElem
	return OK
}

func opAsString(pc *ProcessContext) OpResult {
	switch pc.opSubject.(type) {
	case string:
		return OK
	}
	return FAIL
}

func opSToken(pc *ProcessContext) OpResult {
	opSubject := pc.opSubject.(string)
	idx := pc.actArgs[0].(int)
	sep := pc.actArgs[1].(string)
	tokens := strings.Split(opSubject, sep)
	if idx < len(tokens) {
		pc.opSubject = tokens[idx]
	} else {
		pc.opSubject = ""
	}
	return OK
}

func opFromSToken(pc *ProcessContext) OpResult {
	opSubject := pc.opSubject.(string)
	logger.Debug("opSubject", opSubject)
	idx := pc.actArgs[0].(int)
	sep := pc.actArgs[1].(string)
	tokens := strings.Split(opSubject, sep)
	if idx < len(tokens) {
		pc.opSubject = strings.Join(tokens[idx:], sep)
	} else {
		pc.opSubject = ""
	}
	return OK
}

func opMapSet(pc *ProcessContext) OpResult {
	key := pc.actArgs[0].(string)
	actMap := pc.actElem.(SIMap)
	actMap[key] = pc.opSubject
	return OK
}

func opMapGet(pc *ProcessContext) OpResult {
	key := pc.actArgs[0].(string)
	actMap := pc.actElem.(SIMap)
	pc.opSubject = actMap[key]
	return OK
}

func (a IArray) ProcessContext() *ArrProcessContext {
	return &ArrProcessContext{pc: &ProcessContext{}, subject: a}
}

func (t ITable) ProcessContext() *TblProcessContext {
	logger.Debug("ProcessContext()")
	pc := &TblProcessContext{pc: &ProcessContext{}, subject: t}
	logger.Debug("ProcessContext() end")
	return pc
}

func (pcT *TblProcessContext) AsStr() *TblProcessContextStr {
	logger.Debug("AsStr()")
	pc := pcT.pc
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{ProcessContextFunc: opAsString})
	logger.Debug("AsStr() end")
	return &TblProcessContextStr{pc: pcT}
}

func (pcStr *TblProcessContextStr) Token(idx int, separator string) *TblProcessContextStr {
	pc := pcStr.pc.pc
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{opSToken, []interface{}{idx, separator}})
	return pcStr
}

func (pcStr *TblProcessContextStr) FromToken(idx int, separator string) *TblProcessContextStr {
	pc := pcStr.pc.pc
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{opFromSToken, []interface{}{idx, separator}})
	return pcStr
}

func (pc *TblProcessContextStr) Set(key string) *TblProcessContext {
	return pc.pc.Set(key)
}

func (pcT *TblProcessContext) Set(key string) *TblProcessContext {
	pc := pcT.pc
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{opMapSet, []interface{}{key}})
	return pcT
}

func (pcT *TblProcessContext) Get(key string) *TblProcessContext {
	logger.Debug("Get()")
	logger.Debug("pcT", pcT)
	pc := pcT.pc
	logger.Debug("pc.ops", pc.ops)
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{opMapGet, []interface{}{key}})
	logger.Debug("End()")
	return pcT
}

/*func (pc *ArrProcessContext) Process() IArray {
	aD := IArray{}
	for idx, elem := range pc.subject {
		pc.actIdx = idx
		pc.actElem = elem
		pc.destElem = nil
		for _, opWA := range pc.ops {
			pc.actArgs = opWA.args
			if opWA.ProcessContextFunc(pc.ProcessContext) == FAIL {
				break
			}
		}
		if pc.destElem != nil {
			aD = append(aD, pc.destElem)
		}
	}
	return aD
}*/

func (pcT *TblProcessContext) Process() ITable {
	logger.Debug("Process()")
	pc := pcT.pc
	for idx, elem := range pcT.subject {
		logger.Debug("elem:", elem)
		pc.actIdx = idx
		pc.actElem = elem
		for _, opWA := range pc.ops {
			pc.actArgs = opWA.args
			if opWA.ProcessContextFunc(pc) == FAIL {
				logger.Warning("Process() op failed:", getFunctionName(opWA.ProcessContextFunc))
				break
			}
		}
	}
	logger.Debug("Process() end")
	return pcT.subject
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (t ITable) Len() int {
	return len(t)
}

type HTable struct {
	header ITable
	data   ITable
}

func (t ITable) GroupBy(col, groupCol string) ITable {
	dt := ITable{}
	grpBy := map[interface{}]ITable{}
	for _, row := range t {
		by := row[col]
		grp := grpBy[by]
		if grp == nil {
			grp = ITable{}
		}
		subRow := SIMap{}
		for k, v := range row {
			if k != col {
				subRow[k] = v
			}
		}
		grpBy[by] = append(grp, subRow)
	}
	grps := IArray{}
	for k, _ := range grpBy {
		grps = append(grps, k)
	}
	grpsSorted := grps.Sort()
	for _, grp := range grpsSorted {
		dRow := SIMap{}
		dRow[col] = grp
		dRow[groupCol] = grpBy[grp]
		dt = append(dt, dRow)
	}
	return dt
}

func (t ITable) AscBy(col string) ITable {
	dt := ITable{}
	byColVal := map[interface{}]ITable{}
	for _, row := range t {
		val := row[col]
		byColValTbl := byColVal[val]
		if byColValTbl == nil {
			byColValTbl = ITable{}
		}
		byColVal[val] = append(byColValTbl, row)
	}
	colVals := IArray{}
	for k, _ := range byColVal {
		colVals = append(colVals, k)
	}
	logger.Info("colVals:", colVals)
	colValsSorted := colVals.Sort()
	logger.Info("colValS:", colValsSorted)
	for _, colVal := range colValsSorted {
		dt = dt.AddTable(byColVal[colVal])
	}
	return dt
}

type SSet map[string]bool

func (set SSet) ToSortedArray() IArray {
	a := IArray{}
	for k, _ := range set {
		a = append(a, k)
	}
	return a.Sort()
}

func (t ITable) MatrixView(colVert, colHoriz, colCell string) *HTable {
	hd := ITable{}
	dt := ITable{}
	horizNamesSet := SSet{}
	vertNamesSet := SSet{}
	cellMap := map[string]map[string]interface{}{}
	for _, row := range t {
		horizVal := IAsString(row[colHoriz])
		vertVal := IAsString(row[colVert])
		horizNamesSet[horizVal] = true
		vertNamesSet[vertVal] = true
		if _, has := cellMap[vertVal]; !has {
			cellMap[vertVal] = map[string]interface{}{}
		}
		cellMap[vertVal][horizVal] = row[colCell]
	}
	horizNames := horizNamesSet.ToSortedArray()
	vertNames := vertNamesSet.ToSortedArray()
	firstHdrCol := SIMap{}
	firstHdrCol["header"] = colVert + "/" + colHoriz
	firstHdrCol["key"] = colVert
	hd = append(hd, firstHdrCol)
	for _, horizName := range horizNames {
		hdrCol := SIMap{}
		hdrCol["header"] = horizName
		hdrCol["key"] = horizName
		hd = append(hd, hdrCol)
	}
	for _, vertName := range vertNames {
		row := SIMap{}
		row[colVert] = vertName
		for _, horizName := range horizNames {
			row[IAsString(horizName)] = cellMap[IAsString(vertName)][IAsString(horizName)]
		}
		dt = append(dt, row)
	}
	return &HTable{header: hd, data: dt}
}

func (ht *HTable) Header() ITable {
	return ht.header
}

func (ht *HTable) Data() ITable {
	return ht.data
}

func (ht *HTable) SetHeader(t ITable) {
	ht.header = t
}

func (ht *HTable) SetData(t ITable) {
	ht.data = t
}

func (t ITable) ReoderBy(col string, order ITable) ITable {
	dt := ITable{}
	byColVal := map[interface{}]SIMap{}
	for _, row := range t {
		colVal := row[col]
		byColVal[colVal] = row
	}
	for _, row := range order {
		colVal := row[col]
		if connRow, has := byColVal[colVal]; has {
			dt = append(dt, connRow)
		}
	}
	return dt
}

func (t ITable) ColReplaceVal(col string, oldValue, newValue interface{}) ITable {
	for _, row := range t {
		colVal := row[col]
		if IAsString(colVal) == IAsString(oldValue) {
			row[col] = newValue
		}
	}
	return t
}

func (t ITable) ColAddWithVal(col string, value interface{}) ITable {
	for _, row := range t {
		row[col] = value
	}
	return t
}

func (t ITable) ColView(col string) ITable {
	dt := ITable{}
	for _, row := range t {
		dstRow := SIMap{}
		dstRow[col] = row[col]
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) MergeView(t2 ITable) ITable {
	dt := ITable{}
	for idx, row := range t {
		dstRow := SIMap{}
		for k, v := range row {
			dstRow[k] = v
		}
		if idx < t2.Len() {
			t2row := t2[idx]
			for k, v := range t2row {
				dstRow[k] = v
			}
		}
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) LeftJoinView(tcol, t2col string, t2 ITable) ITable {
	dt := ITable{}
	keyMap2 := map[interface{}]interface{}{}
	for _, row := range t2 {
		keyMap2[row[t2col]] = row
	}
	for _, row := range t {
		dstRow := SIMap{}
		for k, v := range row {
			dstRow[k] = v
		}
		if row[tcol] != nil {
			row2 := keyMap2[row[tcol]].(SIMap)
			if row2 != nil {
				for k, v := range row2 {
					if k != t2col {
						dstRow[k] = v
					}
				}
			}
		}
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) InnerJoinView(tcol, t2col string, t2 ITable) ITable {
	dt := ITable{}
	keyMap2 := map[interface{}]interface{}{}
	for _, row := range t2 {
		keyMap2[row[t2col]] = row
	}
	for _, row := range t {
		dstRow := SIMap{}
		if row[tcol] != nil {
			row2 := keyMap2[row[tcol]].(SIMap)
			if row2 != nil {
				for k, v := range row2 {
					if k != t2col {
						dstRow[k] = v
					}
				}
				for k, v := range row {
					dstRow[k] = v
				}
				dt = append(dt, dstRow)
			}
		}
	}
	return dt
}

func (t ITable) ColRenameView(colsrc, coldst string) ITable {
	dt := ITable{}
	for _, row := range t {
		dstRow := SIMap{}
		dstRow[coldst] = row[colsrc]
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) ColsView(cols ...string) ITable {
	dt := ITable{}
	for _, row := range t {
		dstRow := SIMap{}
		for _, col := range cols {
			dstRow[col] = row[col]
		}
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) RemapColView(col string, vm *ValueMapping, reverse bool) ITable {
	dt := ITable{}
	m := map[interface{}]interface{}{}
	if reverse {
		m = vm.rm
	} else {
		m = vm.m
	}
	
	for _, row := range t {
		dstRow := SIMap{}
		for ck, cv := range row {
			if ck == col && m[cv] != nil {
				dstRow[ck] = m[cv]
			} else {
				dstRow[ck] = cv
			}
		}
		dt = append(dt, dstRow)
	}
	return dt
}

func (t ITable) Distinct() ITable {
	dt := ITable{}
	rowMap := map[string]bool{}
	for _, row := range t {
		hash := Hash(prepareHash(row))
		if _, has := rowMap[hash]; !has {
			rowMap[hash] = true
			dt = append(dt, row)
		}
	}
	return dt
}

type MapVal struct {
	Keys   []string
	Values []interface{}
}

func prepareHash(m SIMap) MapVal {
	keys := []string{}
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := IArray{}
	for _, k := range keys {
		values = append(values, m[k])
	}
	return MapVal{Keys: keys, Values: values}
}

func Hash(s interface{}) string {
	var b bytes.Buffer
	gob.NewEncoder(&b).Encode(s)
	hash := md5.Sum(b.Bytes())
	return hex.EncodeToString(hash[:])
}

func (t ITable) ColJoinBySeparator(srcCol1, srcCol2, dstCol, separator string) ITable {
	for _, row := range t {
		srcColVal1 := IAsString(row[srcCol1])
		srcColVal2 := IAsString(row[srcCol2])
		if IsEmpty(srcColVal1) || IsEmpty(srcColVal2) {
			row[dstCol] = srcColVal1 + srcColVal2
		} else {
			row[dstCol] = srcColVal1 + separator + srcColVal2
		}
	}
	return t
}

type FilterExpr interface {
	Match(SIMap) bool
}

type ITFilter struct {
	t ITable
	f FilterExpr
}

type LikeFilter struct {
	col string
	re  *regexp.Regexp
}

func (f *LikeFilter) Match(row SIMap) bool {
	return f.re.MatchString(IAsString(row[f.col]))
}

func (t ITable) ExprLikeContains(col, sub string) *ITFilter {
	return &ITFilter{t: t, f: &LikeFilter{col: col, re: regexp.MustCompile(".*" + sub + ".*")}}
}

func (t ITable) ExprLikePrefix(col, sub string) *ITFilter {
	return &ITFilter{t: t, f: &LikeFilter{col: col, re: regexp.MustCompile(sub + ".*")}}
}

func (f *ITFilter) Select() ITable {
	dt := ITable{}
	for _, row := range f.t {
		if f.f.Match(row) {
			dt = append(dt, row)
		}
	}
	return dt
}

func (t ITable) RowsSelectByVal(col string, val interface{}) ITable {
	dt := ITable{}
	for _, row := range t {
		colVal := row[col]
		if IAsString(colVal) == IAsString(val) {
			dt = append(dt, row)
		}
	}
	return dt
}

func (t ITable) Copy() ITable {
	dt := ITable{}
	for _, row := range t {
		dt = append(dt, SIMapLightCopy(row))
	}
	return dt
}

func (t ITable) AddTable(at ITable) ITable {
	for _, row := range at {
		t = append(t, row)
	}
	return t
}

type LoadContext struct {
}

func (loadCtx *LoadContext) LoadYamlAsMap(fileName, folderPrefix string) SIMap {
	ty := loader.GetTemplatedYaml(fileName, "ide70/"+folderPrefix+"/")
	if ty == nil || ty.Def == nil {
		return nil
	}
	return ty.Def
}

func (loadCtx *LoadContext) LoadDictAsMap(dictName string) SIMap {
	res := SIMap{}
	fileAsTemplatedYaml := loader.GetTemplatedYaml(dictName, "ide70/dcfg/dict/")
	if fileAsTemplatedYaml != nil {
		fileData := fileAsTemplatedYaml.Def
		itemList := SIMapGetByKeyAsList(fileData, "items")
		for _, itemIf := range itemList {
			itemData := IAsSIMap(itemIf)
			code := SIMapGetByKeyAsString(itemData, "code")
			descr := SIMapGetByKeyAsString(itemData, "descr")
			res[code] = descr
		}
	}
	return res
}
