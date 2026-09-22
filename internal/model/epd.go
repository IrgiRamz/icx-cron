package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Standard 6-field parser with mandatory seconds: [Second] [Minute] [Hour] [Dom] [Month] [Dow]
var standardParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ParseCronExpression normalizes Easycron 5-field/6-field expressions to 6-field with seconds
func ParseCronExpression(cronExpr string) (cron.Schedule, string, error) {
	fields := strings.Fields(strings.TrimSpace(cronExpr))
	if len(fields) == 0 {
		return nil, "", fmt.Errorf("empty cron expression")
	}

	var normalized string

	switch len(fields) {
	case 5:
		// Standard 5-field cron: [Minute] [Hour] [Dom] [Month] [Dow]
		// Prepend '0' for seconds
		normalized = "0 " + strings.Join(fields, " ")
	case 6:
		// Easycron 6-field format: [Minute] [Hour] [Dom] [Month] [Dow] [Year/*]
		// Or 6-field with seconds: [Second] [Minute] [Hour] [Dom] [Month] [Dow]
		// If 6th field is '*' or a 4-digit Year (e.g. '2026'), it's Easycron format [Min Hour Dom Month Dow Year/*]
		if fields[5] == "*" || len(fields[5]) == 4 {
			// Drop 6th field (Year/*) and prepend '0' for seconds -> [0] [Min] [Hour] [Dom] [Month] [Dow]
			normalized = "0 " + strings.Join(fields[:5], " ")
		} else {
			// Treat as [Second] [Minute] [Hour] [Dom] [Month] [Dow]
			normalized = strings.Join(fields, " ")
		}
	default:
		normalized = strings.Join(fields, " ")
	}

	sched, err := standardParser.Parse(normalized)
	if err != nil {
		// Fallback: try parsing with SecondOptional parser
		altParser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedAlt, errAlt := altParser.Parse(cronExpr)
		if errAlt == nil {
			return schedAlt, cronExpr, nil
		}
		return nil, "", err
	}

	return sched, normalized, nil
}

// CalculateEPD calculates exact executions per day (EPD) using 24-hour simulation
func CalculateEPD(cronExpr string) int {
	sched, _, err := ParseCronExpression(cronExpr)
	if err != nil {
		return 1 // Fallback default
	}

	// 24-hour simulation from 00:00:00 today to 23:59:59 today
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(-1 * time.Second)
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	count := 0
	curr := startTime

	for {
		next := sched.Next(curr)
		if next.IsZero() || !next.Before(endTime) {
			break
		}
		count++
		curr = next
	}

	if count == 0 {
		return 1
	}
	return count
}

func (j *Job) GetEPD() int64 {
	return int64(CalculateEPD(j.CronExpression))
}
