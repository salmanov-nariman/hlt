package broker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)


type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr: kafka.TCP(brokers...),
		Topic: topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &Producer{writer: w}
}

func (p *Producer) Publish(ctx context.Context, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Kafka Producer] Error marshaling event: %v", err)
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
	})
	if err != nil {
		log.Printf("[Kafka Producer] Error sending event to broker: %v", err)
		return err
	}
	return nil
}

func (p *Producer) Close() {
	p.writer.Close()
}