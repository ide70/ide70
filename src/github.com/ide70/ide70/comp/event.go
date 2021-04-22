package comp

import (
	"fmt"
	"github.com/ide70/ide70/store"
	"github.com/ide70/ide70/util/log"
	"github.com/robertkrimen/otto"
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
)

const EvtUnitPrefix = "onUnit"
const ParamPassParamID = "ppi" // Event type parameter name

const (
	EvtUnitCreate = "onUnitCreate"
)

type UnitRuntimeEventsHandler struct {
	Vm      *otto.Otto
	exMutex *sync.RWMutex
	Unit    *UnitRuntime
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
	JsCode string
}

type EventRuntime struct {
	TypeCode       string
	ValueStr       string
	UnitRuntime    *UnitRuntime
	Comp           *CompRuntime
	ResponseAction *ResponseAction
	Session        *Session
}

type Attr struct {
	Key   string
	Value string
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
	compsToRefresh []string
	attrsToRefresh map[string][]Attr
	propsToRefresh map[string][]Attr
	loadUnit       string
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
	if ra.loadUnit != "" {
		sb.WriteString(fmt.Sprintf("%d,%s", eraReloadWin, ra.loadUnit))
	}
	if len(ra.compsToRefresh) > 0 {
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

	if sb.Len() > 0 {
		return sb.String()
	}
	return fmt.Sprintf("%d", eraNoAction)
}

func (ra *ResponseAction) SetCompRefresh(comp *CompRuntime) {
	ra.compsToRefresh = append(ra.compsToRefresh, strconv.FormatInt(comp.Sid(), 10))
}

func (ra *ResponseAction) SetCompAttrRefresh(comp *CompRuntime, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10)
	ra.attrsToRefresh[id] = append(ra.attrsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetCompPropRefresh(comp *CompRuntime, key, value string) {
	id := strconv.FormatInt(comp.Sid(), 10)
	ra.propsToRefresh[id] = append(ra.propsToRefresh[id], Attr{Key: key, Value: value})
}

func (ra *ResponseAction) SetLoadUnit(unitName string) {
	ra.loadUnit = unitName
}

type CompRuntimeSW struct {
	c     *CompRuntime
	event *EventRuntime
}

func (cSW *CompRuntimeSW) SetProp(key, value string) *CompRuntimeSW {
	cSW.c.State[key] = value
	eventLogger.Info("property", key, "set to", value)
	return cSW
}

func (cSW *CompRuntimeSW) GetProp(key string) interface{} {
	return cSW.c.State[key]
}

func (cSW *CompRuntimeSW) RefreshHTMLProp(key, value string) *CompRuntimeSW {
	cSW.event.ResponseAction.SetCompPropRefresh(cSW.c, key, value)
	return cSW
}

func (cSW *CompRuntimeSW) RefreshHTMLAttr(key, value string) *CompRuntimeSW {
	cSW.event.ResponseAction.SetCompAttrRefresh(cSW.c, key, value)
	return cSW
}

func (cSW *CompRuntimeSW) Refresh() {
	cSW.event.ResponseAction.SetCompRefresh(cSW.c)
}

func passParametersToMap(passParameters ...interface{}) map[string]interface{} {
	if len(passParameters) == 0 {
		return nil
	}
	switch ppT := passParameters[0].(type) {
	case int64, string:
		ppMap := map[string]interface{}{}
		ppMap["id"] = ppT
		return ppMap
	case map[string]interface{}:
		return ppT
	}
	return nil
}

func (e *EventRuntime) LoadUnit(unitName string, passParameters ...interface{}) {
	logger.Info("LoadUnit", unitName, passParameters)
	passParametersMap := passParametersToMap(passParameters...)
	logger.Info("passParametersMap", passParametersMap)
	if passParametersMap != nil {
		id := e.Session.SetPassParameters(passParametersMap)
		unitName = fmt.Sprintf("%s?%s=%s", unitName, ParamPassParamID, id)
	}
	logger.Info("unitName", unitName)
	e.ResponseAction.SetLoadUnit(unitName)
}

func newUnitRuntimeEventsHandler(unit *UnitRuntime, passParams map[string]interface{}) *UnitRuntimeEventsHandler {
	eventsHandler := &UnitRuntimeEventsHandler{}
	eventsHandler.Unit = unit
	vm := otto.New()
	vm.Set("PassParams", passParams)
	vm.Set("common_log", func(call otto.FunctionCall) otto.Value {
		right, _ := call.Argument(0).ToString()
		eventLogger.Info(right)
		result, _ := vm.ToValue(2)
		return result
	})
	vm.Set("CompByCr", func(call otto.FunctionCall) otto.Value {
		childRefId, _ := call.Argument(0).ToString()
		c := unit.CompByChildRefId[childRefId]
		eventVal, _ := vm.Get("currentEvent")
		eventIf, _ := eventVal.Export()
		event := eventIf.(*EventRuntime)
		cSW := &CompRuntimeSW{c: c, event: event}
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
	e.UnitRuntime.EventsHandler.runJs(e, eh.JsCode)
}

func (eh *UnitRuntimeEventsHandler) runJs(e *EventRuntime, jsCode string) {
	eh.exMutex.Lock()
	defer eh.exMutex.Unlock()
	eh.Vm.Set("currentEvent", e)
	defer eh.Vm.Set("currentEvent", nil)
	eventLogger.Info("executing: ", jsCode)
	eventLogger.Info("event: ", e)
	_, err := eh.Vm.Run(jsCode)
	if err != nil {
		eventLogger.Error("error evaluating script:", jsCode, err.Error())
	}
}

func (er *EventRuntime) DBCtx() *store.DatabaseContext {
	return er.UnitRuntime.Application.Connectors.MainDB
}
