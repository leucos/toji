package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/leucos/toji/drivers"
	"gitlab.com/leucos/toji/drivers/everhour"
	"gitlab.com/leucos/toji/drivers/jira"
	"gitlab.com/leucos/toji/internal/config"
	"gitlab.com/leucos/toji/internal/humantime"

	toggl "github.com/jason0x43/go-toggl"
)

var validArgs = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	"mon", "tue", "wed", "thu", "fri", "sat", "sun",
	"today", "yesterday",
	"week", "month", "year",
}

var (
	toDate      string
	dryRun      bool
	utc         bool
	interactive bool
	rollup      bool
	rounding    int
	onlyIssues  []string
)

func init() {
	syncCmd.Flags().StringVarP(&toDate, "to", "t", "", "ending date")
	syncCmd.Flags().BoolVarP(&dryRun, "dryrun", "n", false, "do not update Jira entries")
	syncCmd.Flags().BoolVarP(&utc, "utc", "u", false, "display entries using UTC in the terminal")
	syncCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "asks a comment for each worklog interactively")
	syncCmd.Flags().BoolVarP(&rollup, "rollup", "R", false, "summarize times daily per ticket")
	syncCmd.Flags().IntVarP(&rounding, "rounding", "r", 0, "round rollup times to this value (in minutes)")
	syncCmd.Flags().StringSliceVarP(&onlyIssues, "only", "o", nil, "only update these comma-separated entries")

	syncCmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validArgs, cobra.ShellCompDirectiveDefault
	})

	toggl.DisableLog()
	config.Current.CheckProfile()
}

var syncCmd = &cobra.Command{
	Use:     "sync <start>",
	Short:   "syncs time entries from toggl to jira",
	Example: "toji sync yesterday --to today",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		return doSync(args[0], toDate, dryRun, utc, interactive, rounding, onlyIssues)
	},
	// SilenceUsage: true,
	ValidArgs: validArgs,
}

func doSync(fromDate, toDate string, dryRun, utc, interactive bool, rounding int, onlyIssues []string) error {
	var (
		drv drivers.ConfigurableReplica
		err error
	)

	from, to, err := humantime.ParseTimePair(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("unable to parse time using provided '%s' or '%s': %v", from, to, err)
	}

	if config.Current.Check("jira") {
		slog.Debug("using Jira driver")
		drv, err = jira.New(from, to,
			drivers.WithDryRun(dryRun),
			drivers.WithRoundingMins(rounding),
			drivers.WithTimeZone(time.Local),
			drivers.WithIssues(onlyIssues),
		)
		if err != nil {
			return err
		}
	}

	if config.Current.Check("everhour") {
		fmt.Println("using Everhour driver")
		drv, err = everhour.New(from, to,
			drivers.WithDryRun(dryRun),
			drivers.WithRoundingMins(rounding),
			drivers.WithTimeZone(time.Local),
			drivers.WithIssues(onlyIssues),
		)
		if err != nil {
			return err
		}
	}

	c := make(chan drivers.SyncedEntry, 100)
	slog.Debug("result channel created")

	if rollup {
		go drv.Rollup(c)
	} else {
		go drv.Sync(c)
	}

	slog.Debug("watching channel messages")

	for e := range c {
		fmt.Printf("CHAN: %s\n", e.Message)
	}

	slog.Debug("sync done")
	return nil
}
