package comp

import (
	"github.com/ide70/ide70/util/log"
	//	"gopkg.in/olebedev/go-duktape.v3"
	"github.com/robertkrimen/otto"
	"strings"
	"fmt"
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

type EventsHandler struct {
	Handlers map[string]*EventHandler
}

type EventHandler struct {
	JsCode string
}

type EventRuntime struct {
	TypeCode string
	ResponseAction ResponseAction
}

type ResponseAction struct {
	compsToRefresh []string
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

func newEventsHandler() *EventsHandler {
	eventsHandler := &EventsHandler{}
	eventsHandler.Handlers = map[string]*EventHandler{}
	return eventsHandler
}

func (esh *EventsHandler) AddHandler(eventType string, handler *EventHandler) {
	esh.Handlers[eventType] = handler
}

func (esh *EventsHandler) ProcessEvent(event *EventRuntime) {
	eventHandler := esh.Handlers[event.TypeCode]
	if eventHandler != nil {
		eventHandler.processEvent()
	}
}

func newEventHandler() *EventHandler {
	eventHandler := &EventHandler{}
	return eventHandler
}

func (eh *EventHandler) processEvent() {
	eventLogger.Info("executing: ", eh.JsCode)
	vm := otto.New()
	vm.Set("wow_key", "wow")
	vm.Set("common_log", func(call otto.FunctionCall) otto.Value {
		right, _ := call.Argument(0).ToString()
		eventLogger.Info(right)
		result, _ := vm.ToValue(2)
		return result
	})
	_, err := vm.Run(eh.JsCode)
	
	/*ctx := duktape.New()
	ctx.PushString("wow!")
	ctx.PutGlobalString("wow_key")
	ctx.PushGlobalGoFunction("common_log", func(c *duktape.Context) int {
		eventLogger.Info(c.SafeToString(-1))
		return 0
	})
	err := ctx.PevalString(eh.JsCode)*/
	
	if err != nil {
		eventLogger.Error("error evaluating script:", eh.JsCode, err.Error())
	}
}
