package sous

import (
	"github.com/opentable/sous/util/logging"
)

type loggingProcessor struct {
	ls logging.LogSink
}

func (log loggingProcessor) HandlePairs(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	log.doLog(dp)
	return dp, nil
}

func (log loggingProcessor) doLog(dp *DeployablePair) {
	msg := &deployableMessage{
		pairmessage: NewDeployablePairSubmessage(dp),
		callerInfo:  logging.GetCallerInfo(),
	}
	msg.callerInfo.ExcludeMe()

	logging.Deliver(msg, log.ls)
}

func (log loggingProcessor) HandleResolution(rez *DiffResolution) {
	msg := &diffRezMessage{
		resolution: rez,
		callerInfo: logging.GetCallerInfo(logging.NotHere()),
	}
	logging.Deliver(msg, log.ls)
}
