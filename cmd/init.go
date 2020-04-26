package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "inits configuration",
	Example: "toji init",
	// Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doInit()
	},
	SilenceUsage: true,
}

func init() {
	// joinCmd.Flags().BoolVar(&cpu, "cpu", false, "cpu cgroup")
	// joinCmd.Flags().BoolVar(&cpuset, "cpuset", false, "cpuset cgroup")
	// joinCmd.Flags().BoolVar(&devices, "devices", false, "devices cgroup")
	// joinCmd.Flags().BoolVar(&freezer, "freezer", false, "freezer cgroup")
	// joinCmd.Flags().BoolVar(&hugetlb, "hugetlb", false, "hugetlb cgroup")
	// joinCmd.Flags().BoolVar(&memory, "memory", false, "memory cgroup")
	// joinCmd.Flags().BoolVar(&net, "net", false, "net cgroup")
	// joinCmd.Flags().BoolVar(&perfevent, "perfevent", false, "perfevent cgroup")
	// joinCmd.Flags().BoolVar(&pids, "pids", false, "pids cgroup")
}

func doInit() error {
	var err error
	if _, err = os.Stat(configFile); err == nil {
		fmt.Fprintf(os.Stderr, "Config file %s already exists\n", configFile)
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	d := filepath.Dir(configFile)

	err = os.MkdirAll(d, 0700)
	if err != nil {
		return err
	}

	// cfg := Configuration{}
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Toggle token: ")
	togglToken, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	fmt.Printf("Jira username: ")
	jiraUser, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	fmt.Printf("Jira token: ")
	jiraToken, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	fmt.Printf("Jira URL: ")
	jiraURL, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	viper.Set("toggle.token", strings.TrimSpace(togglToken))
	viper.Set("jira.username", strings.TrimSpace(jiraUser))
	viper.Set("jira.token", strings.TrimSpace(jiraToken))
	viper.Set("jira.url", strings.TrimSpace(jiraURL))

	err = viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("unable to write configuration: %v", err)
	}

	return nil
}
