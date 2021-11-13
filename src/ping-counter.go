package main

import (
	"sync"
	"time"
)

type PingCounter struct {
	lifeTime       time.Duration
	maxExpectedRtt time.Duration
	startTime      time.Time
	upSince        time.Time
	lastRequest    time.Time
	lastReply      time.Time
	lock           sync.Mutex
}

type PingCounterStats struct {
	waitTime time.Duration
	downTime time.Duration
	upTime   time.Duration
}

func NewPingCounter(lifeTime time.Duration) *PingCounter {
	return &PingCounter{
		lifeTime:       lifeTime,
		maxExpectedRtt: 100 * time.Millisecond,
		startTime:      time.Now(),
	}
}

func (c *PingCounter) AddRequest() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastRequest = time.Now()
}

func (c *PingCounter) AddReply() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.lastReply = time.Now()
}

func (c *PingCounter) Stats(replyTimeout time.Duration) PingCounterStats {
	c.lock.Lock()
	defer c.lock.Unlock()

	dt, ut := c.downUpTimes(replyTimeout)
	return PingCounterStats{
		waitTime: c.waitTime().Truncate(time.Millisecond),
		downTime: dt.Truncate(time.Millisecond),
		upTime:   ut.Truncate(time.Millisecond),
	}
}

func (c *PingCounter) waitTime() time.Duration {
	if !c.lastRequest.IsZero() && c.lastRequest.After(c.lastReply) {
		if !c.lastReply.IsZero() {
			return time.Since(c.lastReply)
		}
		return time.Since(c.startTime)
	}
	return 0
}

func (c *PingCounter) downUpTimes(replyTimeout time.Duration) (time.Duration, time.Duration) {
	wait := c.waitTime()
	if wait > replyTimeout+c.maxExpectedRtt {
		// up -> down
		c.upSince = time.Time{}
		return wait, 0
	}
	if c.upSince.IsZero() {
		// down -> up
		c.upSince = time.Now()
		return 0, 0
	}
	// already up
	return 0, time.Since(c.upSince)
}
