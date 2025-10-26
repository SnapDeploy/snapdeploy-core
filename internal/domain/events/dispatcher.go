package events

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// EventHandler is a function that handles a domain event
type EventHandler func(ctx context.Context, event DomainEvent) error

// Dispatcher dispatches domain events to registered handlers
type Dispatcher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewDispatcher creates a new event dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]EventHandler),
	}
}

// Register registers an event handler for a specific event type
func (d *Dispatcher) Register(eventType string, handler EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Dispatch dispatches an event to all registered handlers
func (d *Dispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	d.mu.RLock()
	handlers, exists := d.handlers[event.EventType()]
	d.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return nil // No handlers registered for this event type
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h EventHandler) {
			defer wg.Done()
			if err := h(ctx, event); err != nil {
				log.Printf("Error handling event %s (ID: %s): %v",
					event.EventType(), event.EventID(), err)
				errChan <- err
			}
		}(handler)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while dispatching event: %v", errs)
	}

	return nil
}

// DispatchAll dispatches multiple events
func (d *Dispatcher) DispatchAll(ctx context.Context, events []DomainEvent) error {
	for _, event := range events {
		if err := d.Dispatch(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
