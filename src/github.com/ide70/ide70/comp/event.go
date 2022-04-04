package comp

import (
	"fmt"
	"github.com/ide70/ide70/api"
	"github.com/ide70/ide70/util/file"
	"github.com/ide70/ide70/util/log"
	"github.com/robertkrimen/otto"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var eventLogger = log.Logger{"event"}

// Event response actions (client actions to take after processing an event).
const (
	eraNoAction      = iota // Event processing OK and no action required
	eraReloadWin            // Window name to be reloaded
	eraDirtyComps           // There are dirty components which needs to be refreshed
	eraFocusComp            // Focus a compnent
	eraDirtyAttrs           // There are dirty component attributes which needs to be refreshed
	eraDirtyProps           // There are dirty component DOM properties which needs to be refreshed
	eraApplyToParent        // Apply changes to parent window
	eraScrollDownComp
	eraCompFuncExecute
	eraForwardToParent
)

const EvtUnitPrefix = "onUnit"
const ParamPassParamID = "ppi" // Event type parameter name
const loadUnitSelf = "."
const PathUnitById = "ubi"

const (
	EvtUnitCreate        = "onUnitCreate"
	EvtBeforeCompRefresh = "beforeCompRefresh"
)

type UnitRuntimeEventsHandler struct {
	Vm      *otto.Otto
	exMutex *sync.RWMutex
	Unit    *UnitRuntime
}

type VmBase struct {
}

type CompDefEventsHandler struct {
	Handlers map[string]*EventHandler
}

type UnitDefEventsHandler struct {
	Handlers map[string][]*CompEventHandler
}

type CompEventHandler struct {
	EventHandler *EventHandler
	CompDef      *CompDef
}

type EventHandler struct {
	JsCode      string
	PropertyKey string
}

type EventRuntime struct {
	TypeCode       string
	ValueStr       string
	UnitRuntime    *UnitRuntime
	Comp           *CompRuntime
	ResponseAction *ResponseAction
	Session        *Session
	MouseWX        int64
	MouseWY        int64
	Params         map[string]interface{}
}

type Attr struct {
	Key   string
	Value string
}

type JSFuncCall struct {
	Comp     string
	FuncName string
	Args     []string
}

type EventForward struct {
	To        *CompRuntime
	EventType string
	Params    map[string]interface{}
}

type ParentForward struct {
	SID       int64
	EventType string
}

type LoadUnit struct {
	unit       string
	passParams map[string]interface{}
	targetCr   string
}

type SessionWrapper interface {
	SetAuthUser(userName string)
	SetAuthRole(role string)
	AuthUser() string
	AuthRole() string
	IsAuthenticated() bool
	ClearAuthentication()
}

func NewEventRuntime(sess *Session, unit *UnitRuntime, comp *CompRuntime, typeCode string, valueStr string) *EventRuntime {
	er := &EventRuntime{}
	er.Session = sess
	er.UnitRuntime = unit
	er.Comp = comp
	er.TypeCode = typeCode
	er.ValueStr = valueStr
	er.ResponseAction = newResponseAction()
	return er
}

type ResponseAction struct {
	compsToRefresh     []string
	attrsToRefresh     map[string][]Attr
	propsToRefresh     map[string][]Attr
	compFuncsToExecute []JSFuncCall
	loadUnit           *LoadUnit
	forward            []*EventForward
	applyToParent      bool
	parentForward      []*ParentForward
}

func newResponseAction() *ResponseAction {
	ra := &ResponseAction{}
	ra.attrsToRefresh = map[string][]Attr{}
	ra.propsToRefresh = map[string][]Attr{}
	return ra
}

func addSep(sb *strings.Builder, sep string) {
	if sb.Len() > 0 {
		sb.WriteString(sep)
	}
}

func (ra *ResponseAction) Encode() string {
	var sb strings.Builder
	if ra.applyToParent {
		sb.WriteString(fmt.Sprintf("%d", eraApplyToParent))
	}
	if ra.loadUnit != nil {
		addSep(&sb, "|")
		unit := ra.loadUnit.unit
		if unit == loadUnitSelf {
			unit = ""
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s", eraReloadWin, unit, ra.loadUnit.targetCr))
	}
	if len(ra.compsToRefresh) > 0 {
		addSep(&sb, "|")
		sb.WriteString(fmt.Sprintf("%d,%s", eraDirtyComps, strings.Join(ra.compsToRefresh, ",")))
	}

	if len(ra.attrsToRefresh) > 0 {
		addSep(&sb, "|")
		sb.WriteString(fmt.Sprintf("%d", eraDirtyAttrs))
		for sid, attrs := range ra.attrsToRefresh {
			for _, attr := range attrs {
				sb.WriteString(fmt.Sprintf(",%s,%s,%s", sid, attr.Key, attr.Value))
			}
		}
	}

	if len(ra.propsToRefresh) > 0 {
		addSep(&sb, "|")
		sb.WriteString(fmt.Sprintf("%d", eraDirtyProps))
		for sid, attrs := range ra.propsToRefresh {
			for _, attr := range attrs {
				sb.WriteString(fmt.Sprintf(",%s,%s,%s", sid, attr.Key, attr.Value))
			}
		}
	}

	if len(ra.compFuncsToExecute) > 0 {
		for _, compFuncCall := range ra.compFuncsToExecute {
			addSep(&sb, "|")
			sb.WriteString(fmt.Sprintf("%d", eraCompFuncExecute))
			sb.WriteString(fmt.Sprintf(",%s,%s", compFuncCall.Comp, compFuncCall.FuncName))
			for _, arg := range compFuncCall.Args {
				sb.WriteString(fmt.Sprintf(",%s", arg))
			}
		}
	}

	for _, parentForward := range ra.parentForward {
		addSep(&sb, "|")
		sb.WriteString(fmt.Sprintf("%d,%s,%d", eraForwardToParent, parentForward.EventType, parentForward.SID))
	}

	if sb.Len() > 0 {
		return sb.String()
	}
	return fmt.Sprintf("%d", eraNoAction)
}

func (ra *ResponseAction) ApplyToParent(applyToParent bool) {
	ra.applyToParent = applyToParent
}

func (ra *ResponseAction) SetCompRefresh(comp *CompRuntime) {
	ra.compsToRefresh = append(ra.compsToRefresh, strconv.FormatInt(comp.Sid(), 10))
}

func (ra *ResponseAction) SetSubCompRefresh(comp *CompRuntime, idPostfix string) {
	id := strconv.FormatInt(comp.Sid(), 10) + idPostfix
	ra.compsToRefresh = append(ra.compsToRefresh, id)
}

func (ra *ResponseAction) SetCompAttrRefresh(comp *CompRuntime, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10)
	ra.attrsToRefresh[id] = append(ra.attrsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetSubAttrRefresh(comp *CompRuntime, idPostfix, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10) + "-" + idPostfix
	ra.attrsToRefresh[id] = append(ra.attrsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetForwardEvent(comp *CompRuntime, eventType string) {
	ra.forward = append(ra.forward, &EventForward{To: comp, EventType: eventType, Params: map[string]interface{}{}})
}

func (ra *ResponseAction) AddForwardEventParam(key string, value interface{}) {
	last := len(ra.forward) - 1
	if last >= 0 {
		ra.forward[last].Params[key] = value
	}
}

func (ra *ResponseAction) AddForwardEventParams(params map[string]interface{}) {
	last := len(ra.forward) - 1
	if last >= 0 {
		ra.forward[last].Params = params
	}
}

func (ra *ResponseAction) AddParentForwardEvent(sid int64, eventType string) {
	ra.parentForward = append(ra.parentForward, &ParentForward{SID: sid, EventType: eventType})
}

func (ra *ResponseAction) SetCompPropRefresh(comp *CompRuntime, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10)
	ra.propsToRefresh[id] = append(ra.propsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetSubPropRefresh(comp *CompRuntime, idPostfix, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10) + "-" + idPostfix
	ra.propsToRefresh[id] = append(ra.propsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetCompFuncExecute(comp *CompRuntime, funcName string, args ...string) {
	id := strconv.FormatInt(comp.Sid(), 10)
	ra.compFuncsToExecute = append(ra.compFuncsToExecute, JSFuncCall{Comp: id, FuncName: funcName, Args: args})
}

func (ra *ResponseAction) SetSubCompFuncExecute(comp *CompRuntime, idPostfix, funcName string, args ...string) {
	id := strconv.FormatInt(comp.Sid(), 10) + "-" + idPostfix
	ra.compFuncsToExecute = append(ra.compFuncsToExecute, JSFuncCall{Comp: id, FuncName: funcName, Args: args})
}

func (ra *ResponseAction) initLoadUnit() {
	if ra.loadUnit == nil {
		ra.loadUnit = &LoadUnit{passParams: map[string]interface{}{}}
	}
}

func (ra *ResponseAction) SetLoadUnit(unitName string) {
	ra.initLoadUnit()
	ra.loadUnit.unit = unitName
}

func (ra *ResponseAction) SetLoadUnitToTarget(unitName string, target *CompRuntime) {
	ra.initLoadUnit()
	ra.loadUnit.unit = unitName
	ra.loadUnit.targetCr = strconv.FormatInt(target.Sid(), 10)
	ra.loadUnit.passParams["parentCr"] = target.CompDef.ChildRefId()
}

func (ra *ResponseAction) AddLoadUnitParam(key string, value interface{}) {
	ra.initLoadUnit()
	ra.loadUnit.passParams[key] = value
}

type CompCtx struct {
	c     *CompRuntime
	event *EventRuntime
}

func (cSW *CompCtx) SetProp(key string, value interface{}) *CompCtx {
	cSW.c.State[key] = value
	eventLogger.Info("property", key, "set to", value)
	return cSW
}

func (cSW *CompCtx) RemoveProp(key string) *CompCtx {
	delete(cSW.c.State, key)
	eventLogger.Info("property", key, "removed")
	return cSW
}

func (cSW *CompCtx) CreateMapProp(key string) *CompCtx {
	cSW.c.State[key] = map[string]interface{}{}
	return cSW
}

func (cSW *CompCtx) GetProp(key string) interface{} {
	return cSW.c.State[key]
}

func (cSW *CompCtx) HasProp(key string) bool {
	return cSW.c.State[key] != nil
}

func (cSW *CompCtx) Props() api.SIMap {
	return cSW.c.State
}

func (c *CompRuntime) GetProp(key string) interface{} {
	return c.State[key]
}

func (cSW *CompCtx) GetPropToCast(key string) api.Interface {
	return api.Interface{I:cSW.c.State[key]}
}

func (cSW *CompCtx) ParentContext() api.Interface {
	return api.Interface{I: cSW.Comp().GetProp("parentContext")}
}

func (cSW *CompCtx) ApplyToParent() *CompCtx {
	cSW.event.ResponseAction.ApplyToParent(true)
	return cSW
}

func (cSW *CompCtx) RefreshHTMLProp(key, value string) *CompCtx {
	cSW.event.ResponseAction.SetCompPropRefresh(cSW.c, key, value)
	return cSW
}

func (cSW *CompCtx) RefreshHTMLAttr(key, value string) *CompCtx {
	cSW.event.ResponseAction.SetCompAttrRefresh(cSW.c, key, value)
	return cSW
}

func (cSW *CompCtx) RefreshSubHTMLAttr(idPostfix, key, value string) *CompCtx {
	cSW.event.ResponseAction.SetSubAttrRefresh(cSW.c, idPostfix, key, value)
	return cSW
}

func (cSW *CompCtx) RefreshSubHTMLProp(idPostfix, key, value string) *CompCtx {
	cSW.event.ResponseAction.SetSubPropRefresh(cSW.c, idPostfix, key, value)
	return cSW
}

func (cSW *CompCtx) ForwardEvent(eventType string) *CompCtx {
	if eventType == "" {
		eventType = cSW.event.TypeCode
	}
	logger.Info("cSW.c", cSW.c.ChildRefId())
	cSW.event.ResponseAction.SetForwardEvent(cSW.c, eventType)
	return cSW
}

func (cSW *CompCtx) AddForwardParam(key string, value interface{}) *CompCtx {
	cSW.event.ResponseAction.AddForwardEventParam(key, value)
	return cSW
}

func (cSW *CompCtx) AddForwardParams(params map[string]interface{}) *CompCtx {
	cSW.event.ResponseAction.AddForwardEventParams(params)
	return cSW
}

func (cSW *CompCtx) Comp() *CompRuntime {
	return cSW.c
}

func (cSW *CompCtx) ForwardToParent(parentCompCr, eventType string) *CompCtx {
	if eventType == "" {
		eventType = cSW.event.TypeCode
	}
	unit := cSW.event.UnitRuntime
	parentUnit := unit.GetParent(cSW.event.Session)
	if parentUnit == nil {
		return cSW
	}
	comp := parentUnit.CompByChildRefId[parentCompCr]
	if comp == nil {
		return cSW
	}
	cSW.event.ResponseAction.AddParentForwardEvent(comp.Sid(), eventType)
	return cSW
}

func (cSW *CompCtx) ForwardToParentComp(parentComp *CompRuntime, eventType string) *CompCtx {
	//logger.Info("ForwardToParentComp pc:", parentComp)
	logger.Info("ForwardToParentComp:", parentComp.Sid(), eventType)
	cSW.event.ResponseAction.AddParentForwardEvent(parentComp.Sid(), eventType)
	return cSW
}

func (cSW *CompCtx) FuncExecute(funcName string, args ...string) *CompCtx {
	cSW.event.ResponseAction.SetCompFuncExecute(cSW.c, funcName, args...)
	return cSW
}

func (cSW *CompCtx) SubCompFuncExecute(idPrefix, funcName string, args ...string) *CompCtx {
	cSW.event.ResponseAction.SetSubCompFuncExecute(cSW.c, idPrefix, funcName, args...)
	return cSW
}

func (cSW *CompCtx) Refresh() {
	cSW.event.ResponseAction.SetCompRefresh(cSW.c)
}

func (cSW *CompCtx) RefreshSubComp(idPostfix string) {
	cSW.event.ResponseAction.SetSubCompRefresh(cSW.c, idPostfix)
}

func (cSW *CompCtx) GetParentComp() *CompCtx {
	return &CompCtx{c: cSW.c.State["parentComp"].(*CompRuntime), event: cSW.event}
}

func reloadUnit(e *EventRuntime, unit *UnitRuntime) {
	unitPath := fmt.Sprintf("%s/%s", PathUnitById, unit.getID())
	logger.Info("Reload existing unit:", unitPath)
	e.ResponseAction.SetLoadUnit(unitPath)
}

func (e *EventRuntime) CurrentComp() *CompCtx {
	return &CompCtx{c: e.Comp, event: e}
}

func (e *EventRuntime) EventKey() string {
	return e.ValueStr
}

func (e *EventRuntime) ParentComp() *CompCtx {
	return &CompCtx{c: e.Comp.State["parentComp"].(*CompRuntime), event: e}
}

func (e *EventRuntime) ParentContext() interface{} {
	return e.Comp.State["parentContext"]
}

func (e *EventRuntime) LoadUnit(unitName string) {
	e.LoadUnitToTarget(unitName, nil)
}

func (e *EventRuntime) setPassParamteres() {
	if e.ResponseAction.loadUnit != nil {
		id := e.Session.SetPassParameters(e.ResponseAction.loadUnit.passParams, e.UnitRuntime)
		unitName := fmt.Sprintf("%s?%s=%s", e.ResponseAction.loadUnit.unit, ParamPassParamID, id)
		e.ResponseAction.SetLoadUnit(unitName)
	}
}

func (e *EventRuntime) LoadUnitToTarget(unitName string, target *CompRuntime) {
	if target == nil {
		logger.Info("LoadUnit unitName", unitName)
		e.ResponseAction.SetLoadUnit(unitName)
	} else {
		logger.Info("LoadUnit unitName and target", unitName, target.ChildRefId())
		e.ResponseAction.SetLoadUnitToTarget(unitName, target)
	}
}

func (cSW *CompCtx) LoadUnitInto(unitName string) *CompCtx {
	cSW.event.LoadUnitToTarget(unitName, cSW.c)
	return cSW
}

func (cSW *CompCtx) LoadUnit(unitName string) *CompCtx {
	cSW.event.LoadUnit(unitName)
	return cSW
}

func (cSW *CompCtx) AddPassParam(key string, value interface{}) *CompCtx {
	cSW.event.ResponseAction.AddLoadUnitParam(key, value)
	return cSW
}

func (cSW *CompCtx) GeneratedChildren() []*CompCtx {
	children := []*CompCtx{}
	for _, child := range cSW.c.GenChildren {
		children = append(children, &CompCtx{c: child, event: cSW.event})
	}
	return children
}

func (e *EventRuntime) GetSession() SessionWrapper {
	return e.Session
}

func (e *EventRuntime) LoadParent() {
	logger.Info("Loadparent")
	unit := e.UnitRuntime
	parentUnit := unit.GetParent(e.Session)
	if parentUnit == nil {
		return
	}
	reloadUnit(e, parentUnit)
	e.Session.DeleteUnit(unit)
}

func (e *EventRuntime) CloseLayer() {
	logger.Info("CloseLayer")
	unit := e.UnitRuntime
	//parentUnit := unit.GetParent(e.Session)
	// iframe-et megkeresni, src-t üresre állítani, elrejteni
	// ide70js parent operations

	e.Session.DeleteUnit(unit)
}

func (e *EventRuntime) ClearAuthentication() {
	logger.Info("Logout")
	e.Session.ClearAuthentication()
}

func (e *EventRuntime) ReloadUnit() {
	reloadUnit(e, e.UnitRuntime)
}

func (e *EventRuntime) CompProps() api.SIMap {
	return e.Comp.State
}

func (e *EventRuntime) GetParam(key string) interface{} {
	logger.Info("GetParam:", e.Params)
	if e.Params == nil {
		return nil
	}
	logger.Info("GetParam:", key, e.Params[key])
	return e.Params[key]
}

func (e *EventRuntime) GetUnit() *UnitCtx {
	return &UnitCtx{e.UnitRuntime}
}

func (vm *VmBase) Event() *EventRuntime {
	return nil
}

func (vm *VmBase) CompCtx() *CompCtx {
	return nil
}

func (vm *VmBase) Api() *api.API {
	return nil
}

func (vm *VmBase) CompByCr(compName string) *CompCtx {
	return nil
}

func (vm *VmBase) common_log(text string) {
}

func (vm *VmBase) PassParams() map[string]interface{} {
	return nil
}

func newUnitRuntimeEventsHandler(unit *UnitRuntime) *UnitRuntimeEventsHandler {
	eventsHandler := &UnitRuntimeEventsHandler{}
	eventsHandler.Unit = unit
	vm := otto.New()
	vm.Set("PassParams", unit.PassContext.Params)
	vm.Set("common_log", func(call otto.FunctionCall) otto.Value {
		right, _ := call.Argument(0).ToString()
		eventLogger.Info("EXE: " + right)
		result, _ := vm.ToValue(2)
		return result
	})
	vm.Set("CompByCr", func(call otto.FunctionCall) otto.Value {
		childRefId, _ := call.Argument(0).ToString()
		eventVal, _ := vm.Get("currentEvent")
		eventIf, _ := eventVal.Export()
		event := eventIf.(*EventRuntime)
		var c *CompRuntime = nil
		if childRefId == "" {
			c = event.Comp
		} else {
			c = unit.CompByChildRefId[childRefId]
		}
		if c == nil {
			result, ev := vm.ToValue(nil)
			if ev != nil {
				eventLogger.Error("error converting result:", ev.Error())
			}
			return result
		}
		cSW := &CompCtx{c: c, event: event}
		result, ev := vm.ToValue(cSW)
		if ev != nil {
			eventLogger.Error("error converting result:", ev.Error())
		}
		return result
	})
	vm.Set("Event", func(call otto.FunctionCall) otto.Value {
		eventVal, _ := vm.Get("currentEvent")
		return eventVal
	})
	vm.Set("CompCtx", func(call otto.FunctionCall) otto.Value {
		eventVal, _ := vm.Get("currentEvent")
		eventIf, _ := eventVal.Export()
		event := eventIf.(*EventRuntime)
		co, _ := vm.ToValue(&CompCtx{c: event.Comp, event: event})
		return co
	})
	vm.Set("Api", func(call otto.FunctionCall) otto.Value {
		result, _ := vm.ToValue(api.ApiInst)
		return result
	})
	eventsHandler.exMutex = &sync.RWMutex{}
	eventsHandler.Vm = vm

	return eventsHandler
}

func newUnitDefEventsHandler() *UnitDefEventsHandler {
	eventsHandler := &UnitDefEventsHandler{}
	eventsHandler.Handlers = map[string][]*CompEventHandler{}
	return eventsHandler
}

func (esh *UnitDefEventsHandler) AddHandler(eventType string, handler *CompEventHandler) {
	esh.Handlers[eventType] = append(esh.Handlers[eventType], handler)
}

func newEventsHandler() *CompDefEventsHandler {
	eventsHandler := &CompDefEventsHandler{}
	eventsHandler.Handlers = map[string]*EventHandler{}
	return eventsHandler
}

func (esh *CompDefEventsHandler) AddHandler(eventType string, handler *EventHandler) {
	esh.Handlers[eventType] = handler
}

func ProcessCompEvent(e *EventRuntime) {
	for {
		logger.Info("compDef.eh:", e.Comp.CompDef.EventsHandler)
		e.Comp.CompDef.EventsHandler.ProcessEvent(e)
		if len(e.ResponseAction.forward) > 0 {
			logger.Info("event forward to type:" + e.ResponseAction.forward[0].EventType)
			e.Comp = e.ResponseAction.forward[0].To
			e.TypeCode = e.ResponseAction.forward[0].EventType
			e.Params = e.ResponseAction.forward[0].Params
			logger.Info("with params:", e.Params)
			e.ResponseAction.forward = e.ResponseAction.forward[1:]
			continue
		}
		break
	}
	e.setPassParamteres()
}

func (esh *CompDefEventsHandler) ProcessEvent(event *EventRuntime) {
	eventHandler := esh.Handlers[event.TypeCode]
	if eventHandler != nil {
		eventHandler.processEvent(event)
	}
}

func newEventHandler() *EventHandler {
	eventHandler := &EventHandler{}
	return eventHandler
}

func (eh *EventHandler) processEvent(e *EventRuntime) {
	calcResult := e.UnitRuntime.EventsHandler.runJs(e, eh.JsCode)
	if eh.PropertyKey != "" {
		logger.Info("calc result for", eh.PropertyKey, "is", calcResult, "type:", reflect.TypeOf(calcResult))
		e.Comp.State[eh.PropertyKey] = calcResult
		logger.Info("calc result done");
	}
}

func (eh *UnitRuntimeEventsHandler) runJs(e *EventRuntime, jsCode string) interface{} {
	defer func() {
		r := recover()
		if r != nil {
			eventLogger.Info("Vm Run panic:", r)
		}
	}()
	eh.exMutex.Lock()
	defer eh.exMutex.Unlock()
	eh.Vm.Set("currentEvent", e)
	defer eh.Vm.Set("currentEvent", nil)
	//eventLogger.Info("executing: ", jsCode)
	//eventLogger.Info("event: ", e)

	eventLogger.Info("Vm.Run: ", jsCode)
	value, err := eh.Vm.Run(jsCode)
	eventLogger.Info("Vm.Run end")
	if err != nil {
		eventLogger.Error("error evaluating script:", jsCode, err.Error())
	}
	valueIf, err := value.Export()
	if err != nil {
		eventLogger.Error("error converting result:", jsCode, err.Error())
	}
	return valueIf
}

func (er *EventRuntime) DBCtx() *api.DatabaseContext {
	return er.UnitRuntime.Application.Connectors.MainDB
}

func (er *EventRuntime) FileCtx() *file.FileContext {
	return er.UnitRuntime.Application.Connectors.FileContext
}

func (er *EventRuntime) LoadCtx() *api.LoadContext {
	return er.UnitRuntime.Application.Connectors.LoadContext
}

type UnitCtx struct {
	unit *UnitRuntime
}

func (unit *UnitCtx) CollectStored() api.SIMap {
	return unit.unit.CollectStored()
}

func (unit *UnitCtx) InitializeStored(data api.SIMap) {
	unit.unit.InitializeStored(data)
}

func (unit *UnitCtx) GetPassParams() api.SIMap {
	return unit.unit.PassContext.Params
}
