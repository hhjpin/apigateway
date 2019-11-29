package routing

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"time"
)

type HealthCheckMsg int

type WatchMsgFunc func()

type WatchMsg struct {
	Handle WatchMsgFunc
}

type Events struct {
	watchCh       chan WatchMsg
	healthCheckCh chan HealthCheckMsg
}

func NewEvents() *Events {
	return &Events{
		healthCheckCh: make(chan HealthCheckMsg),
		watchCh:       make(chan WatchMsg, 1000),
	}
}

func (r *Table) PushHealthCheckEvent() {
	r.events.healthCheckCh <- 1
}

func (r *Table) PushWatchEvent(msg WatchMsg) {
	r.events.watchCh <- msg
}

func (r *Table) HandleEvent() {
	defer func() {
		logger.Debugf("handle event panic")
		if err := recover(); err != nil {
			stack := utils.Stack(3)
			logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s", utils.TimeFormat(time.Now()), err, stack)
		}
		go r.HandleEvent()
	}()

	for {
		select {
		case <-r.events.healthCheckCh:
			r.doHealthCheck()
		case msg := <-r.events.watchCh:
			r.handleWatchEvent(&msg)
		}
	}
}

//do more watch event
func (r *Table) handleWatchEvent(msg *WatchMsg) {
	msg.Handle()
	for {
		select {
		case msg := <-r.events.watchCh:
			msg.Handle()
		default:
			return
		}
	}
}
