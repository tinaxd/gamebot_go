package gamebot

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strconv"
	"time"
)

var (
	ErrParseTargetTime = errors.New("failed to parse string as TargetTimeFormat")

	TargetTimeFormatRe = regexp.MustCompile("^([0-9]{2})([0-9]{2})$")
)

type TargetTimeFormat struct {
	Hour   uint
	Minute uint
}

func ParseTargetTimeFormat(text string) (TargetTimeFormat, error) {
	parts := TargetTimeFormatRe.FindStringSubmatch(text)
	if parts == nil {
		return TargetTimeFormat{}, ErrParseTargetTime
	}

	hour, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || hour > 23 {
		return TargetTimeFormat{}, ErrParseTargetTime
	}

	minute, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil || minute > 59 {
		return TargetTimeFormat{}, ErrParseTargetTime
	}

	return TargetTimeFormat{
		Hour:   uint(hour),
		Minute: uint(minute),
	}, nil
}

func CalculateNextDay(now time.Time, targetTime TargetTimeFormat) time.Time {
	nowLocal := now.Local()

	// Get current date
	year, month, day := nowLocal.Date()

	// Create target time on current date
	target := time.Date(year, month, day, int(targetTime.Hour), int(targetTime.Minute), 0, 0, now.Location())

	// If target time is earlier in the day or equal to current time, move to next day
	if target.Before(nowLocal) || target.Equal(nowLocal) {
		target = target.AddDate(0, 0, 1)
	}

	return target
}

type Notifier struct {
	handler    func()
	targetTime time.Time
}

func NewNotifier(targetTime time.Time, handler func()) *Notifier {
	return &Notifier{
		handler:    handler,
		targetTime: targetTime,
	}
}

func (n *Notifier) WaitAndDo(ctx context.Context) {
	duration := time.Until(n.targetTime)
	timer := time.NewTimer(duration)
	log.Printf("notifier will be fired in %.1f seconds", duration.Seconds())

	select {
	case <-timer.C:
		// Time has come
		log.Printf("notifier fired")
		n.handler()
	case <-ctx.Done():
		// Canceled
		timer.Stop()
	}
}
