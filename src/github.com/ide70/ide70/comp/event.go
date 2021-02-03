package comp

import (
	"github.com/ide70/ide70/util/log"
	//	"gopkg.in/olebedev/go-duktape.v3"
	"fmt"
	"github.com/robertkrimen/otto"
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
	eraApplyToParent        // Apply changes to parent window
	eraScrollDownComp
)

type UnitRuntimeEventsHandler struct {
	Vm      *otto.Otto
	exMutex *sync.RWMutex
	Unit    *UnitRuntime
}

type CompDefEventsHandler struct {
	Handlers map[string]*EventHandler
}

type EventHandler struct {
	JsCode string
}

type EventRuntime struct {
	TypeCode       string
	UnitRuntime    *UnitRuntime
	ResponseAction *ResponseAction
}

func NewEventRuntime(unit *UnitRuntime, typeCode string) *EventRuntime {
	er := &EventRuntime{}
	er.UnitRuntime = unit
	er.TypeCode = typeCode
	er.ResponseAction = newResponseAction()
	return er
}

type ResponseAction struct {
	compsToRefresh []string
}

func newResponseAction() *ResponseAction {
	ra := &ResponseAction{}
	return ra
}

func (ra *ResponseAction) Encode() string {
	if len(ra.compsToRefresh) > 0 {
		return fmt.Sprintf("%d,%s", eraDirtyComps, strings.Join(ra.compsToRefresh, ","))
	}
	return fmt.Sprintf("%d", eraNoAction)
}

func (ra *ResponseAction) SetCompRefresh(comp *CompRuntime) {
	ra.compsToRefresh = append(ra.compsToRefresh, comp.Sid())
}

func newUnitRuntimeEventsHandler(unit *UnitRuntime) *UnitRuntimeEventsHandler {
	eventsHandler := &UnitRuntimeEventsHandler{}
	eventsHandler.Unit = unit
	vm := otto.New()
	vm.Set("wow_key", "wow")
	vm.Set("common_log", func(call otto.FunctionCall) otto.Value {
		right, _ := call.Argument(0).ToString()
		eventLogger.Info(right)
		result, _ := vm.ToValue(2)
		return result
	})
	vm.Set("CompByCr", func(call otto.FunctionCall) otto.Value {
		childRefId, _ := call.Argument(0).ToString()
		c := unit.CompByChildRefId[childRefId]
		eventLogger.Info("CompByCr:", c)
		result, ev := vm.ToValue(c)
		if ev != nil {
			eventLogger.Error("error converting result:", ev.Error())
		}
		eventLogger.Info("result:", result)
		return result
	})
	vm.Set("CompSetProp", func(call otto.FunctionCall) otto.Value {
		compValueIf, _ := call.Argument(0).Export()
		c := compValueIf.(*CompRuntime)
		propKey, _ := call.Argument(1).ToString()
		propValue, _ := call.Argument(2).ToString()
		eventLogger.Info("CompValue:", c)
		eventLogger.Info("PropKey:", propKey)
		eventLogger.Info("PropValue:", propValue)
		eventVal, _ := vm.Get("currentEvent")
		eventIf, _ := eventVal.Export()
		event := eventIf.(*EventRuntime)
		eventLogger.Info("currentEvent:", event)
		
		c.State[propKey] = propValue
		event.ResponseAction.SetCompRefresh(c)
		
		result, _ := vm.ToValue(0)
		return result
	})
	eventsHandler.exMutex = &sync.RWMutex{}
	eventsHandler.Vm = vm

	return eventsHandler
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
	_, err := eh.Vm.Run(jsCode)
	if err != nil {
		eventLogger.Error("error evaluating script:", jsCode, err.Error())
	}
}
