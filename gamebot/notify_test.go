package gamebot_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tinaxd/gamebot/gamebot"
)

func TestParseTargetTimeFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected gamebot.TargetTimeFormat
		hasError bool
	}{
		{
			name:     "Valid time 0000",
			input:    "0000",
			expected: gamebot.TargetTimeFormat{Hour: 0, Minute: 0},
			hasError: false,
		},
		{
			name:     "Valid time 2359",
			input:    "2359",
			expected: gamebot.TargetTimeFormat{Hour: 23, Minute: 59},
			hasError: false,
		},
		{
			name:     "Valid time 0930",
			input:    "0930",
			expected: gamebot.TargetTimeFormat{Hour: 9, Minute: 30},
			hasError: false,
		},
		{
			name:     "Invalid format - has colon",
			input:    "12:45",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
		{
			name:     "Invalid format - letters",
			input:    "abcd",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
		{
			name:     "Invalid hour - too large",
			input:    "2400",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
		{
			name:     "Invalid minute - too large",
			input:    "2360",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
		{
			name:     "Invalid format - extra parts",
			input:    "123456",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: gamebot.TargetTimeFormat{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gamebot.ParseTargetTimeFormat(tt.input)

			if tt.hasError {
				assert.Error(t, err)
				assert.Equal(t, gamebot.ErrParseTargetTime, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCalculateNextDay(t *testing.T) {
	// Create JST location
	jst := time.FixedZone("JST", 9*60*60)

	// Base case: current time 18:00
	now := time.Date(2023, 5, 10, 18, 0, 0, 0, jst)

	tests := []struct {
		name       string
		targetTime gamebot.TargetTimeFormat
		expected   time.Time
		comment    string
	}{
		{
			name:       "Target is later same day",
			targetTime: gamebot.TargetTimeFormat{Hour: 18, Minute: 5},
			expected:   time.Date(2023, 5, 10, 18, 5, 0, 0, jst),
			comment:    "18:05 should be same day",
		},
		{
			name:       "Target is later same day evening",
			targetTime: gamebot.TargetTimeFormat{Hour: 23, Minute: 0},
			expected:   time.Date(2023, 5, 10, 23, 0, 0, 0, jst),
			comment:    "23:00 should be same day",
		},
		{
			name:       "Target is earlier so next day",
			targetTime: gamebot.TargetTimeFormat{Hour: 15, Minute: 0},
			expected:   time.Date(2023, 5, 11, 15, 0, 0, 0, jst),
			comment:    "15:00 should be next day",
		},
		{
			name:       "Target is midnight so next day",
			targetTime: gamebot.TargetTimeFormat{Hour: 0, Minute: 0},
			expected:   time.Date(2023, 5, 11, 0, 0, 0, 0, jst),
			comment:    "00:00 should be next day",
		},
		{
			name:       "Target is same time so next day",
			targetTime: gamebot.TargetTimeFormat{Hour: 18, Minute: 0},
			expected:   time.Date(2023, 5, 11, 18, 0, 0, 0, jst),
			comment:    "18:00 should be next day",
		},
		{
			name:       "Target is one minute later",
			targetTime: gamebot.TargetTimeFormat{Hour: 18, Minute: 1},
			expected:   time.Date(2023, 5, 10, 18, 1, 0, 0, jst),
			comment:    "18:01 should be same day",
		},
		{
			name:       "Target is one minute earlier",
			targetTime: gamebot.TargetTimeFormat{Hour: 17, Minute: 59},
			expected:   time.Date(2023, 5, 11, 17, 59, 0, 0, jst),
			comment:    "17:59 should be next day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gamebot.CalculateNextDay(now, tt.targetTime)
			assert.Equal(t, tt.expected, result, tt.comment)
		})
	}

	// Test with different current times
	otherTimes := []struct {
		name        string
		currentTime time.Time
		targetTime  gamebot.TargetTimeFormat
		expected    time.Time
	}{
		{
			name:        "Current 00:00, Target 12:00 - same day",
			currentTime: time.Date(2023, 5, 10, 0, 0, 0, 0, jst),
			targetTime:  gamebot.TargetTimeFormat{Hour: 12, Minute: 0},
			expected:    time.Date(2023, 5, 10, 12, 0, 0, 0, jst),
		},
		{
			name:        "Current 23:59, Target 00:00 - next day",
			currentTime: time.Date(2023, 5, 10, 23, 59, 0, 0, jst),
			targetTime:  gamebot.TargetTimeFormat{Hour: 0, Minute: 0},
			expected:    time.Date(2023, 5, 11, 0, 0, 0, 0, jst),
		},
		{
			name:        "Current 12:00, Target 12:00 - next day",
			currentTime: time.Date(2023, 5, 10, 12, 0, 0, 0, jst),
			targetTime:  gamebot.TargetTimeFormat{Hour: 12, Minute: 0},
			expected:    time.Date(2023, 5, 11, 12, 0, 0, 0, jst),
		},
		{
			name:        "End of month rollover",
			currentTime: time.Date(2023, 5, 31, 23, 0, 0, 0, jst),
			targetTime:  gamebot.TargetTimeFormat{Hour: 1, Minute: 0},
			expected:    time.Date(2023, 6, 1, 1, 0, 0, 0, jst),
		},
		{
			name:        "End of year rollover",
			currentTime: time.Date(2023, 12, 31, 23, 0, 0, 0, jst),
			targetTime:  gamebot.TargetTimeFormat{Hour: 1, Minute: 0},
			expected:    time.Date(2024, 1, 1, 1, 0, 0, 0, jst),
		},
	}

	for _, tt := range otherTimes {
		t.Run(tt.name, func(t *testing.T) {
			result := gamebot.CalculateNextDay(tt.currentTime, tt.targetTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}
