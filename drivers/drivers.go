package drivers

import (
	"log/slog"
	"time"
)

type EntryType int

const (
	Simple EntryType = iota
	Rollup
	Skip
)

type SyncedEntry struct {
	Date     string
	Duration int64
	Type     EntryType
	Message  string
}

type BaseOptions struct {
	DryRun   bool
	TimeZone *time.Location
	Issues   []string
}

type OptionFunc func(c *BaseOptions) error

// type Configurable interface {
// 	WithDryRun(bool) Option
// 	WithRoundingMins(int) Option
// 	WithTimeZone(time.Location) Option
// 	WithIssues([]string) Option
// }

type Replica interface {
	Sync(c chan SyncedEntry)
	// Rollup(c chan SyncedEntry)
}

type ConfigurableReplica interface {
	Replica
	// Configurable
}

func WithDryRun(b bool) OptionFunc {
	slog.Debug("setting dryrun", "b", b)
	return func(base *BaseOptions) error {
		base.DryRun = b
		return nil
	}
}

// func WithRoundingMins(i int) OptionFunc {
// 	slog.Debug("setting Rounding", "i", i)
// 	return func(base *BaseOptions) error {
// 		base.Rounding = i
// 		return nil
// 	}
// }

func WithTimeZone(t *time.Location) OptionFunc {
	slog.Debug("setting tz", "t", t)
	return func(base *BaseOptions) error {
		base.TimeZone = t
		return nil
	}
}

func WithIssues(s []string) OptionFunc {
	slog.Debug("setting issues", "s", s)
	return func(base *BaseOptions) error {
		base.Issues = s
		return nil
	}
}
