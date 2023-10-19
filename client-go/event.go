package main

import (
	"fmt"
	"sync"
)

type Event struct {
	eventLock     sync.Mutex
	eventHandlers []func(message string)
}

func NewEvent() *Event {
	return &Event{
		eventLock:     sync.Mutex{},
		eventHandlers: make([]func(message string), 0),
	}
}

func (e *Event) Emit(message string) {
	e.eventLock.Lock()
	defer e.eventLock.Unlock()
	for _, handler := range e.eventHandlers {
		go handler(message)
	}
}

func (e *Event) Add(handler func(message string)) {
	e.eventLock.Lock()
	defer e.eventLock.Unlock()
	e.eventHandlers = append(e.eventHandlers, handler)
}

func (e *Event) Remove(handlerToRemove func(message string)) {
	e.eventLock.Lock()
	defer e.eventLock.Unlock()
	for i, handler := range e.eventHandlers {
		if fmt.Sprintf("%p", handler) == fmt.Sprintf("%p", handlerToRemove) {
			e.eventHandlers = append(e.eventHandlers[:i], e.eventHandlers[i+1:]...)
			break
		}
	}
}
