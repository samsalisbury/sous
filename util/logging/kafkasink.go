package logging

import (
	"log"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/nyarly/spies"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type (
	kafkaSink interface {
		live() bool
		id() string
		shouldSend(lvl Level) bool
		send(lvl Level, entry *logrus.Entry) error
		closedown()
	}

	liveKafkaSink struct {
		idstring     string
		defaultTopic string
		level        Level
		formatter    logrus.Formatter
		producer     sarama.AsyncProducer
		exit         sync.WaitGroup
	}

	kafkaSinkSpy struct {
		spy *spies.Spy
	}
)

func newLiveKafkaSink(
	id string,
	level Level,
	formatter logrus.Formatter,
	brokers []string,
	defaultTopic string,
	injectHostname bool) (*liveKafkaSink, error) {

	var err error
	var producer sarama.AsyncProducer
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	kafkaConfig.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	if producer, err = sarama.NewAsyncProducer(brokers, kafkaConfig); err != nil {
		return nil, err
	}

	sink := &liveKafkaSink{
		idstring:     id,
		defaultTopic: defaultTopic,
		level:        level,
		formatter:    formatter,
		producer:     producer,
	}

	sink.exit.Add(1)

	go func() {
		defer sink.exit.Done()
		for err := range producer.Errors() {
			val := err.Msg.Value.(sarama.ByteEncoder)
			len := val.Length()
			sVal := string(val[:len])
			maxLen := 5000
			if len > maxLen {
				len = maxLen
			}
			log.Printf("Failed to send log entry to Kafka: %v\n%s", err, sVal[:len-1])
		}
	}()

	return sink, nil
}

func (sink *liveKafkaSink) live() bool {
	return sink != nil
}

func (sink *liveKafkaSink) id() string {
	if sink == nil {
		return ""
	}
	return sink.idstring
}

func (sink *liveKafkaSink) shouldSend(lvl Level) bool {
	if sink == nil {
		return false
	}
	return lvl <= sink.level
}

func (sink *liveKafkaSink) send(lvl Level, entry *logrus.Entry) error {
	if sink == nil {
		return nil
	}
	var partitionKey sarama.ByteEncoder
	var b []byte
	var err error

	if !sink.shouldSend(lvl) {
		return nil
	}

	uuid, present := entry.Data["@uuid"]
	if !present {
		return errors.Errorf("Required field @uuid absent")
	}
	uuidStr, stringType := uuid.(string)
	if !stringType {
		return errors.Errorf("Required field @uuid was type %T, not string", uuid)
	}
	partitionKey = sarama.ByteEncoder(uuidStr)

	entry.Level = lvl.logrusLevel()
	if b, err = sink.formatter.Format(entry); err != nil {
		return err
	}
	value := sarama.ByteEncoder(b)

	topic := sink.defaultTopic
	sink.producer.Input() <- &sarama.ProducerMessage{
		Key:   partitionKey,
		Topic: topic,
		Value: value,
	}
	return nil
}

func (sink *liveKafkaSink) closedown() {
	if sink == nil {
		return
	}
	sink.producer.AsyncClose()
	sink.exit.Wait()
}

func newKafkaSinkSpy() (kafkaSinkSpy, *spies.Spy) {
	spy := spies.NewSpy()
	return kafkaSinkSpy{spy: spy}, spy
}

func (spy kafkaSinkSpy) live() bool {
	res := spy.spy.Called()
	return res.GetOr(0, true).(bool)
}

func (spy kafkaSinkSpy) id() string {
	res := spy.spy.Called()
	return res.String(0)
}

func (spy kafkaSinkSpy) shouldSend(lvl Level) bool {
	res := spy.spy.Called(lvl)
	return res.GetOr(0, true).(bool)
}

func (spy kafkaSinkSpy) send(lvl Level, entry *logrus.Entry) error {
	res := spy.spy.Called(lvl, entry)
	return res.Error(0)
}
func (spy kafkaSinkSpy) closedown() {
	spy.spy.Called()
}
