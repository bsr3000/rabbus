package rabbus

import (
	"sync"
	"testing"
	"time"
)

func TestRabbusListen(t *testing.T) {
	r, err := NewRabbus(Config{
		Dsn:      "amqp://guest:guest@localhost:5672",
		Attempts: 1,
		Timeout:  time.Second * 2,
		Durable:  true,
	})
	if err != nil {
		t.Fail()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	if err := r.Listen(ListenConfig{
		ExchangeName: "test_ex",
		ExchangeType: "topic",
		RoutingKey:   "test_key",
		QueueName:    "test_q",
		HandlerFunc: func(d *Delivery) {
			d.Ack(true)
			wg.Done()
		},
	}); err != nil {
		t.Errorf("Expected to listen message %s", err)
	}

	r.Emit() <- &Message{
		ExchangeName: "test_ex",
		ExchangeType: "topic",
		RoutingKey:   "test_key",
		Payload:      []byte(`foo`),
		DeliveryMode: Transient,
	}

	go func() {
		for {
			select {
			case <-r.EmitOk():
			case <-r.EmitErr():
				t.Errorf("Expected to emit message")
				wg.Done()
			}
		}
	}()

	wg.Wait()
}

func TestRabbusListen_Validate(t *testing.T) {
	r, err := NewRabbus(Config{
		Dsn:      "amqp://guest:guest@localhost:5672",
		Attempts: 1,
		Timeout:  time.Second * 2,
	})
	if err != nil {
		t.Fail()
	}

	if err := r.Listen(ListenConfig{}); err == nil {
		t.Errorf("Expected to validate ExchangeName")
	}

	if err = r.Listen(ListenConfig{ExchangeName: "foo"}); err == nil {
		t.Errorf("Expected to validate ExchangeType")
	}

	if err = r.Listen(ListenConfig{
		ExchangeName: "foo",
		ExchangeType: "foo",
	}); err == nil {
		t.Errorf("Expected to validate QueueName")
	}

	if err = r.Listen(ListenConfig{
		ExchangeName: "foo",
		ExchangeType: "foo",
		QueueName:    "foo",
	}); err == nil {
		t.Errorf("Expected to validate HandlerFunc")
	}
}

func TestRabbusClose(t *testing.T) {
	r, err := NewRabbus(Config{
		Dsn:      "amqp://guest:guest@localhost:5672",
		Attempts: 1,
		Timeout:  time.Second * 2,
	})
	if err != nil {
		t.Fail()
	}

	r.Close()
}

func TestRabbus_reconnect(t *testing.T) {
	r, err := NewRabbus(Config{
		Dsn:      "amqp://guest:guest@localhost:5672",
		Attempts: 1,
		Timeout:  time.Second * 2,
	})
	if err != nil {
		t.Fail()
	}

	r.Close()
	r.reconnect()

	r.Emit() <- &Message{
		ExchangeName: "foo",
		ExchangeType: "direct",
		RoutingKey:   "foo",
		Payload:      []byte(`foo`),
		DeliveryMode: Transient,
	}

loop:
	for {
		select {
		case <-r.EmitOk():
			break loop
		case <-r.EmitErr():
			t.Errorf("Expected to emit message")
			break loop
		}
	}
}