package routing

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"time"
)

type WatchMsgFunc func()

type WatchMsg struct {
	Handle WatchMsgFunc
}

type Events struct {
	watchCh chan WatchMsg
}

func NewEvents() *Events {
	return &Events{
		watchCh: make(chan WatchMsg, 1000),
	}
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

	hcInterval := time.Second * 10
	timer := time.NewTimer(hcInterval)
	for {
		select {
		case <-timer.C:
			r.doHealthCheck()
			timer.Reset(hcInterval)
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
