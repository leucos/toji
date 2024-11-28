package humantime

import (
	"fmt"
	"time"
)

var week map[string]time.Weekday = map[string]time.Weekday{
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sunday":    time.Sunday,
	"mon":       time.Monday,
	"tue":       time.Tuesday,
	"wed":       time.Wednesday,
	"thu":       time.Thursday,
	"fri":       time.Friday,
	"sat":       time.Saturday,
	"sun":       time.Sunday,
}

func ParseTimePair(s string, e string) (time.Time, time.Time, error) {
	var (
		start, end time.Time
	)

	start, err := parseStart(s)
	if err != nil {
		return start, end, err
	}

	end, err = parseEnd(e)
	if err != nil {
		return start, end, err
	}

	if start.After(end) {
		return start, end, fmt.Errorf("end date (%s) is before start date (%s)", end, start)
	}

	return start, end, nil
}

func parseStart(s string) (time.Time, error) {
	start := time.Now()

	switch s {
	case "today":
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "yesterday":
		start = start.AddDate(0, 0, -1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "week", "monday":
		for start.Weekday() != time.Monday { // iterate back to Monday
			start = start.AddDate(0, 0, -1)
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	case "month":
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)
	case "year":
		start = time.Date(start.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	default: // we got a weekday or a date
		if d, ok := week[s]; ok {
			for start.Weekday() != d { // iterate back to requested day
				start = start.AddDate(0, 0, -1)
			}
			start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
			break
		}
		if d, err := time.Parse("200601021504", s); err == nil {
			start = d
			break
		}
		// If only YYYYmmDD is specified start at 00h00
		// time.Truncate can't be used since it works only for UTC
		if d, err := time.Parse("20060102", s); err == nil {
			start = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
			break
		}
		return start, fmt.Errorf("unable to parse start date (%s)", start)
	}
	return start, nil
}

func parseEnd(e string) (time.Time, error) {
	end := time.Now()

	switch e {
	case "today":
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
	case "yesterday":
		end = end.AddDate(0, 0, -1)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
	case "week":
		for end.Weekday() != time.Monday { // iterate back to Monday
			end = end.AddDate(0, 0, -1)
		}
		end = time.Date(end.Year(), end.Month(), end.Day()+6, 23, 59, 59, 0, time.Local)
	case "month":
		end = time.Date(end.Year(), end.Month()+1, 1, 23, 59, 59, 0, time.Local)
		end = end.AddDate(0, 0, -1)
	case "year":
		end = time.Date(end.Year()+1, 1, 1, 23, 59, 59, 0, time.Local)
		end = end.AddDate(0, 0, -1)
	default: // we got a weekday or a date
		if d, ok := week[e]; ok {
			for end.Weekday() != d { // iterate back to requested day
				end = end.AddDate(0, 0, -1)
			}
			end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)
			break
		}
		if d, err := time.Parse("200601021504", e); err == nil {
			end = d
			break
		}
		// If only YYYYmmDD is specified start at 00h00
		// time.Truncate can't be used since it works only for UTC
		if d, err := time.Parse("20060102", e); err == nil {
			end = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 0, 0, time.Local)
			break
		}
		return end, fmt.Errorf("unable to parse end date (%s)", end)
	}
	return end, nil
}
