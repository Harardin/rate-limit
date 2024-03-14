package rabbitbus

import (
	"context"

	"github.com/Harardin/rate-limit/pkg/log"

	mq "github.com/rabbitmq/amqp091-go"
)

type Reader struct {
	queue    string
	consumer string

	ch *mq.Channel

	m chan msg

	stop chan struct{}

	logger log.Logger
}

func (s *Service) NewReader(ctx context.Context, queueName, consumerName string) (*Reader, error) {
	ch, err := s.conn.Channel()
	if err != nil {
		s.logger.Error("failed to open rabbitmq channel", err)
		return nil, err
	}

	r := &Reader{
		queue:    queueName,
		consumer: consumerName,
		ch:       ch,
		m:        make(chan msg),
		stop:     make(chan struct{}, 1),
		logger:   s.logger,
	}

	// starting rabbitmq reader
	go r.read(ctx, queueName, consumerName)

	return r, nil
}

type msg struct {
	m *mq.Delivery
}

func (r *Reader) read(ctx context.Context, q, c string) {
	defer r.ch.Close()

	d, err := r.ch.Consume(q, c, false, false, false, false, nil)
	if err != nil {
		r.logger.Error("failed to start consuming from channel", err)
		return
	}

	for {
		select {
		case m := <-d:
			r.m <- msg{m: &m}
		case <-r.stop:
			r.logger.Info("stop reading from channel do to manual stop")
			return
		case <-ctx.Done():
			r.logger.Info("stop reading from channel do to exit by context")
			return
		}
	}
}

// Read msg
func (r *Reader) ReceiveMsg() <-chan msg {
	return r.m
}

// Stop stops reading messages from channel and closes channel
func (r *Reader) Stop() {
	r.stop <- struct{}{}
}

// Read Get Message data
func (m *msg) Read() []byte {
	return m.m.Body
}

// Ack acknowledge to rabbit that message was processed
func (m *msg) Ack() error {
	return m.m.Ack(false)
}

// Nack declines msg from rabbit and request it to process it again by another consumer
func (m *msg) Nack() error {
	return m.m.Nack(false, true)
}
