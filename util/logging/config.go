package logging

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Config captures outside configuration for a root LogSet
type Config struct {
	Kafka struct {
		Enabled      bool
		DefaultLevel string `env:"SOUS_KAFKA_LOG_LEVEL"`
		Topic        string `env:"SOUS_KAFKA_TOPIC"`
		Brokers      []string
		BrokerList   string `env:"SOUS_KAFKA_BROKERS"`
	}
}

func (cfg Config) getBrokers() []string {
	if len(cfg.Kafka.Brokers) != 0 {
		return cfg.Kafka.Brokers
	}
	return strings.Split(cfg.Kafka.BrokerList, ",")
}

func (cfg Config) getKafkaLevels() []logrus.Level {
	switch {
	default:
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		}
	case "Critial":
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		}
	case "Warning":
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		}
	case "Information":
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
		}
	case "Debug":
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		}
	case "ExtraDebug1":
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		}
	}
}

func (cfg Config) useKafka() bool {
	return cfg.Kafka.Enabled
}

// Validate asserts the validity of the logging configuration
func (cfg Config) Validate() error {
	if err := cfg.validateKafka(); cfg.useKafka() && err != nil {
		return err
	}
}

func (cfg Config) validateKafka() error {
	switch cfg.Kafka.DefaultLevel {
	default:
		return errors.Errorf("default Kafka log level unrecognized: configured as %q", cfg.Kafka.DefaultLevel)
	case "":
		return errors.Errorf("default Kafka log level empty")
	case "Critical", "Warning", "Information", "Debug", "ExtraDebug1":
	}

	switch {
	default:
		return nil
	case len(cfg.getBrokers()) == 0:
		return errors.Errorf("no brokers specified for kafka")
	case cfg.Kafka.Topic == "":
		return errors.Errorf("no Kafka topic configured")
	}
}
