package rabbitmq

import (
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/pkg/helper"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	ID          string      `json:"id"`
	Body        []byte      `json:"content"`
	Payload     interface{} `json:"payload"`
	Headers     amqp.Table  `json:"headers,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	ContentType string      `json:"content_type"`
}

type RPCBody struct {
	Pattern string      `json:"pattern"`
	Data    interface{} `json:"data"`
	ID      string      `json:"id"`
}

type PubsubBody struct {
	Pattern string      `json:"type"`
	Data    interface{} `json:"data"`
	ID      string      `json:"id"`
}

type RPCResponse struct {
	IsDisposed bool        `json:"isDisposed,omitempty"`
	Response   interface{} `json:"response,omitempty"`
	ID         string      `json:"id,omitempty"`
	Err        string      `json:"err,omitempty"`
	Status     string      `json:"status,omitempty"`
}

func NewMessage(payload interface{}, headers *amqp.Table) (*Message, error) {
	gid, err := gonanoid.New()
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("msg_%s_%d", gid, time.Now().Unix())

	var body []byte
	var contentType string
	switch v := payload.(type) {
	case string:
		body = []byte(v)
		contentType = "text/plain"
	case []byte:
		body = v
		contentType = "application/octet-stream"
	default:
		body, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
		contentType = "application/json"
	}

	if headers == nil {
		headers = &amqp.Table{}
	}

	return &Message{
		ID:          id,
		Body:        body,
		Payload:     payload,
		Headers:     *headers,
		Timestamp:   time.Now(),
		ContentType: contentType,
	}, nil
}

func (m *Message) GeneratePayload() *amqp.Publishing {
	m.Headers["id"] = m.ID

	return &amqp.Publishing{
		ContentType:  m.ContentType,
		Body:         m.Body,
		MessageId:    m.ID,
		Timestamp:    m.Timestamp,
		DeliveryMode: amqp.Persistent,
		Headers:      m.Headers,
	}
}

func (m *Message) GenerateRPCPayload(queueName, pattern string) *amqp.Publishing {
	v := RPCBody{
		Pattern: pattern,
		Data:    m.Payload,
		ID:      m.ID,
	}
	body, _ := helper.JSONToByte(v)

	return &amqp.Publishing{
		ContentType:   m.ContentType,
		Body:          body,
		MessageId:     m.ID,
		Timestamp:     m.Timestamp,
		DeliveryMode:  amqp.Persistent,
		ReplyTo:       queueName,
		CorrelationId: m.ID,
		Headers:       m.Headers,
	}
}

func (m *Message) GenerateRPCReplyPayload(correlationID string) *amqp.Publishing {
	v, _ := helper.JSONToStruct[RPCResponse](m.Payload)
	v.ID = correlationID
	body, _ := helper.JSONToByte(v)

	return &amqp.Publishing{
		ContentType:   m.ContentType,
		Body:          body,
		Timestamp:     m.Timestamp,
		DeliveryMode:  amqp.Persistent,
		CorrelationId: correlationID,
		Headers:       m.Headers,
	}
}
