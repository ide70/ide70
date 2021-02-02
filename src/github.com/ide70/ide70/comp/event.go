package comp

import (
	"github.com/ide70/ide70/util/log"
//	"gopkg.in/olebedev/go-duktape.v3"
)

var eventLogger = log.Logger{"event"}

type EventsHandler struct {
	Handlers map[string]*EventHandler
}

type EventHandler struct {
	JsCode string
}

func newEventsHandler() *EventsHandler {
	eventsHandler := &EventsHandler{}
	eventsHandler.Handlers = map[string]*EventHandler{}
	return eventsHandler
}

func (esh *EventsHandler) AddHandler(eventType string, handler *EventHandler) {
	esh.Handlers[eventType] = handler
}

func (esh *EventsHandler) ProcessEvent(eventType string) {
	eventHandler := esh.Handlers[eventType]
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
	/*ctx := duktape.New()
	ctx.PushString("wow!")
	ctx.PutGlobalString("wow_key")
	ctx.PushGlobalGoFunction("common_log", func(c *duktape.Context) int {
		eventLogger.Info(c.SafeToString(-1))
		return 0
	})
	err := ctx.PevalString(eh.JsCode)
	if err != nil {
		eventLogger.Error("error evaluating script:", eh.JsCode, err.Error())
	}*/
}
