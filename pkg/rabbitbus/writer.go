package rabbitbus

import (
	"context"

	"github.com/Harardin/rate-limit/pkg/log"

	mq "github.com/rabbitmq/amqp091-go"
)

type Writer struct {
	ch *mq.Channel

	logger log.Logger
}

func (s *Service) NewWriter() (*Writer, error) {
	ch, err := s.conn.Channel()
	if err != nil {
		s.logger.Error("failed to open rabbitmq channel", err)
		return nil, err
	}

	return &Writer{
		ch:     ch,
		logger: s.logger,
	}, nil
}

func (w *Writer) WriteToExchange(ctx context.Context, exchangeName, routingKey string, data []byte) error {
	return w.ch.PublishWithContext(
		ctx,
		exchangeName,
		routingKey,
		false,
		false,
		mq.Publishing{
			DeliveryMode: mq.Persistent,
			ContentType:  "application/json",
			Body:         data,
		},
	)
}

func (w *Writer) WriteToQueue(ctx context.Context, queueName string, data []byte) error {
	return w.ch.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		mq.Publishing{
			DeliveryMode: mq.Persistent,
			ContentType:  "application/json",
			Body:         data,
		},
	)
}
