package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	OrderCreated      = "ORDER_CREATED"
	OrderConfirmed    = "ORDER_CONFIRMED"
	OrderCancelled    = "ORDER_CANCELLED"
	InventoryReserved = "INVENTORY_RESERVED"
	InventoryReleased = "INVENTORY_RELEASED"
	InventoryFailed   = "INVENTORY_FAILED"
	PaymentCompleted  = "PAYMENT_COMPLETED"
	PaymentFailed     = "PAYMENT_FAILED"
	PaymentRefunded   = "PAYMENT_REFUNDED"
	NotificationSent  = "NOTIFICATION_SENT"
)

// Kafka topics
const (
	TopicOrderCreated      = "order.created"
	TopicOrderConfirmed    = "order.confirmed"
	TopicOrderCancelled    = "order.cancelled"
	TopicInventoryReserved = "inventory.reserved"
	TopicInventoryReleased = "inventory.released"
	TopicInventoryFailed   = "inventory.failed"
	TopicPaymentCompleted  = "payment.completed"
	TopicPaymentFailed     = "payment.failed"
	TopicPaymentRefunded   = "payment.refunded"
	TopicNotificationSent  = "notification.sent"
)

// Event is the standard envelope for all Kafka messages.
type Event struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateID string          `json:"aggregate_id"`
	Timestamp   time.Time       `json:"timestamp"`
	Version     int             `json:"version"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    EventMetadata   `json:"metadata"`
}

// EventMetadata contains tracing and source information.
type EventMetadata struct {
	TraceID       string `json:"trace_id"`
	Source        string `json:"source"`
	CorrelationID string `json:"correlation_id"`
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType string, aggregateID string, source string, payload interface{}) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Event{
		EventID:     uuid.New().String(),
		EventType:   eventType,
		AggregateID: aggregateID,
		Timestamp:   time.Now().UTC(),
		Version:     1,
		Payload:     payloadBytes,
		Metadata: EventMetadata{
			Source: source,
		},
	}, nil
}

// Marshal serializes the event to JSON bytes.
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserializes JSON bytes into an Event.
func Unmarshal(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// DecodePayload unmarshals the event payload into the target struct.
func (e *Event) DecodePayload(target interface{}) error {
	return json.Unmarshal(e.Payload, target)
}
