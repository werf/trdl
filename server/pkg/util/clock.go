package util

import "time"

type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

type SystemClock struct{}

func (c *SystemClock) Now() time.Time {
	return time.Now()
}

func (c *SystemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func NewFixedClock(nowTime time.Time) *FixedClock {
	return &FixedClock{NowTime: nowTime}
}

type FixedClock struct {
	NowTime time.Time
}

func (c *FixedClock) Now() time.Time {
	return c.NowTime
}

func (c *FixedClock) Since(t time.Time) time.Duration {
	return c.NowTime.Sub(t)
}
