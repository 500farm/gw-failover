package main

import "time"

// FIXME synchronization
type PingCounter struct {
	lifeTime       time.Duration
	maxExpectedRtt time.Duration
	startTime      time.Time
	upSince        *time.Time

	requests    []time.Time
	replies     []time.Time
	lastRequest time.Time
	lastReply   time.Time
}

type PingCounterStats struct {
	upTime     time.Duration
	downTime   time.Duration
	replyRatio float64
}

func NewPingCounter(lifeTime time.Duration) *PingCounter {
	return &PingCounter{
		lifeTime:       lifeTime,
		maxExpectedRtt: 100 * time.Millisecond,
		startTime:      time.Now(),
	}
}

func (c *PingCounter) AddRequest() {
	now := time.Now()
	c.requests = append(c.requests, now)
	c.cleanup()
	c.lastRequest = now
}

func (c *PingCounter) AddReply() {
	now := time.Now()
	c.replies = append(c.replies, now)
	c.cleanup()
	c.lastReply = now
}

func (c *PingCounter) Stats(downThreshold time.Duration) PingCounterStats {
	dt := c.downTime(downThreshold)
	return PingCounterStats{
		downTime:   dt,
		upTime:     c.upTime(),
		replyRatio: c.replyRatio(),
	}
}

func (c *PingCounter) downTime(threshold time.Duration) time.Duration {
	if !c.lastRequest.IsZero() && c.lastRequest.After(c.lastReply) {
		var elapsed time.Duration
		if !c.lastReply.IsZero() {
			elapsed = time.Since(c.lastReply)
		} else {
			elapsed = time.Since(c.startTime)
		}
		if elapsed > threshold+c.maxExpectedRtt {
			c.upSince = nil
			return elapsed.Truncate(time.Millisecond)
		}
	}
	if c.upSince == nil {
		t := time.Now()
		c.upSince = &t
	}
	return 0
}

func (c *PingCounter) upTime() time.Duration {
	if c.upSince == nil {
		return 0
	}
	return time.Since(*c.upSince).Truncate(time.Millisecond)
}

func (c *PingCounter) replyRatio() float64 {
	if c.upSince == nil {
		return 0
	}
	now := time.Now()
	numRequests := countSamples(c.requests, *c.upSince, now.Add(-c.maxExpectedRtt))
	numReplies := countSamples(c.replies, *c.upSince, now)
	if numRequests > 0 {
		if r := float64(numReplies) / float64(numRequests); r < 1 {
			return r
		}
	}
	return 1
}

func (c *PingCounter) cleanup() {
	if c.isTimeToCleanup() {
		now := time.Now()
		from := now.Add(-c.lifeTime)
		to := now.Add(time.Second)
		c.requests = filterSamples(c.requests, from, to)
		c.replies = filterSamples(c.replies, from, to)
	}
}

func (c *PingCounter) isTimeToCleanup() bool {
	return len(c.requests) > 0 &&
		time.Since(c.requests[0]) > 2*c.lifeTime
}

func filterSamples(samples []time.Time, from time.Time, to time.Time) []time.Time {
	result := make([]time.Time, 0, len(samples))
	for _, t := range samples {
		if t.After(from) && t.Before(to) {
			result = append(result, t)
		}
	}
	return result
}

func countSamples(samples []time.Time, from time.Time, to time.Time) int {
	result := 0
	for _, t := range samples {
		if t.After(from) && t.Before(to) {
			result++
		}
	}
	return result
}
