package main

import (
	"os"

	"gitlab.com/leucos/toji/cmd"
)

func init() {
	// cobra.OnInitialize(initDriver, checkRuntime)

	// rootCmd.PersistentFlags().StringVar(&socket, "socket", "", "docker or containerd daemon socket path (defaults /run/containerd/containerd.sock (containerd) or /var/run/docker.sock (docker))")
}

// Run executes Toji
func main() {
	if err := cmd.Run(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}
}
