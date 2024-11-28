package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/leucos/toji/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "toji",
	Short: "CLI to add your Toggl entries to your Jira tickets",
}

var (
	configFile     string
	currentProfile string
	logLevel       string
	// Version of current binary
	Version string
	// BuildDate of current binary
	BuildDate string
)

func init() {
	// cobra.OnInitialize(config.Init(configFile, currentProfile)())
	cobra.OnInitialize(func() {
		config.Init(configFile, currentProfile)
	})

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", config.Current.Guess(), "configuration file")
	rootCmd.PersistentFlags().StringVarP(&currentProfile, "profile", "p", "", "profile to use")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "loglevel", "l", "info", "log level")
}

// Run the CLI
func Run() error {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(versionCmd)

	// Run command
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func parseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}

func setupLogging() {
	level, err := parseLevel(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level: %s\n", logLevel)
		os.Exit(1)
	}

	slog.Info("setting log level", "level", logLevel)

	opts := &slog.HandlerOptions{
		Level: level,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}
