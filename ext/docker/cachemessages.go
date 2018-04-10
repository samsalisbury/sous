package docker

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type (
	cacheHitMessage struct {
		logging.CallerInfo
		logging.Level
		source    sous.SourceID
		imageName string
	}
	cacheMissMessage struct {
		logging.CallerInfo
		logging.Level
		source    sous.SourceID
		imageName string
	}
	cacheErrorMessage struct {
		logging.CallerInfo
		logging.Level
		source sous.SourceID
		err    error
	}
)

func reportCacheHit(logger logging.LogSink, sid sous.SourceID, name string) {
	msg := cacheHitMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      logging.InformationLevel,
		source:     sid,
		imageName:  name,
	}
	logging.Deliver(logger, msg)
}

func (msg cacheHitMessage) MetricsTo(ms logging.MetricsSink) {
	ms.IncCounter("cache-hit", 1)
}

func (msg cacheHitMessage) Message() string {
	return "cache hit"
}

func (msg cacheHitMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousCacheMessageV1)
	msg.CallerInfo.EachField(f)
	f("sous-source-id", fmt.Sprintf("%+v", msg.source))
	f("sous-image-name", msg.imageName)
}

func reportCacheMiss(logger logging.LogSink, sid sous.SourceID, name string) {
	msg := cacheMissMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      logging.InformationLevel,
		source:     sid,
		imageName:  name,
	}
	logging.Deliver(logger, msg)
}

func (msg cacheMissMessage) MetricsTo(ms logging.MetricsSink) {
	ms.IncCounter("cache-miss", 1)
}

func (msg cacheMissMessage) Message() string {
	return "cache miss"
}

func (msg cacheMissMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousCacheMessageV1)
	msg.CallerInfo.EachField(f)
	f("sous-source-id", fmt.Sprintf("%+v", msg.source))
	f("sous-image-name", msg.imageName)
}

func reportCacheError(logger logging.LogSink, sid sous.SourceID, err error) {
	msg := cacheErrorMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		Level:      logging.InformationLevel,
		source:     sid,
		err:        err,
	}
	logging.Deliver(logger, msg)
}

func (msg cacheErrorMessage) MetricsTo(ms logging.MetricsSink) {
	ms.IncCounter("cache-error", 1)
}

func (msg cacheErrorMessage) Message() string {
	return "cache error"
}

func (msg cacheErrorMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousCacheMessageV1)
	msg.CallerInfo.EachField(f)
	f("sous-source-id", fmt.Sprintf("%+v", msg.source))
	f("error", msg.err.Error())
}
