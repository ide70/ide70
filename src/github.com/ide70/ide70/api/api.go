package api

import (
	"github.com/ide70/ide70/util/log"
	"github.com/ide70/ide70/dataxform"
	"github.com/ide70/ide70/loader"
	"sort"
	"strings"
	"reflect"
	"runtime"
)

var logger = log.Logger{"api"}

type SIMap map[string]interface{}

type IArray []interface{}

type ITable []SIMap

type Interface struct {
	I interface{}
}

func (i Interface) AsMap() SIMap {
	return dataxform.IAsSIMap(i.I)
}

func (i Interface) AsArray() IArray {
	return dataxform.IAsArr(i.I)
}

func (i Interface) AsString() string {
	return dataxform.IAsString(i.I)
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

func (m SIMap) Get(key string) Interface {
	if m == nil {
		return Interface{nil}
	}
	return Interface{m[key]}
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
	sort.SliceStable(a, func(i, j int) bool {
		aiStr := dataxform.IAsString(a[i])
		ajStr := dataxform.IAsString(a[j])
		return aiStr < ajStr
	})
	return a
}

func (a IArray) PrintDBG() {
	logger.Info("IArray:",a)
}

func (t ITable) PrintDBG() {
	logger.Info("ITable:",t)
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
	pc *ProcessContext
	subject ITable
}

type ArrProcessContext struct {
	pc *ProcessContext
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
	logger.Info("opSubject", opSubject)
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
	logger.Info("ProcessContext()")
	pc := &TblProcessContext{pc: &ProcessContext{}, subject: t}
	logger.Info("ProcessContext() end")
	return pc
}

func (pcT *TblProcessContext) AsStr() *TblProcessContextStr {
	logger.Info("AsStr()")
	pc := pcT.pc
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{ProcessContextFunc: opAsString})
	logger.Info("AsStr() end")
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
	logger.Info("Get()")
	logger.Info("pcT", pcT)
	pc := pcT.pc
	logger.Info("pc.ops", pc.ops)
	pc.ops = append(pc.ops, ProcessContextFuncWithArgs{opMapGet, []interface{}{key}})
	logger.Info("End()")
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
	logger.Info("Process()")
	pc := pcT.pc
	for idx, elem := range pcT.subject {
		logger.Info("elem:", elem)
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
	logger.Info("Process() end")
	return pcT.subject
}

func getFunctionName(i interface{}) string {
    return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (t ITable) GroupBy(col, groupCol string) ITable {
	dt := ITable{}
	grpBy := map[interface{}]ITable{}
	for _,row := range t {
		by := row[col]
		grp := grpBy[by]
		if grp == nil {
			grp = ITable{}
		}
		subRow := SIMap{}
		for k,v := range row {
			if k != col {
				subRow[k]=v
			}
		}
		grpBy[by] = append(grp, subRow)
	}
	grps := IArray{}
	for k,_ := range grpBy {
		grps = append(grps, k)
	}
	grpsSorted := grps.Sort()
	for _,grp := range grpsSorted {
		dRow := SIMap{}
		dRow[col] = grp
		dRow[groupCol] = grpBy[grp]
		dt = append(dt, dRow)
	}
	return dt
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
