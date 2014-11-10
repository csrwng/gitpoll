package main

import (
	"fmt"
	"os"
	"time"

	"github.com/csrwng/gitpoll/pkg/hook"
	"github.com/spf13/cobra"
)

func main() {
	var openshiftEndpoint string
	pollCmd := &cobra.Command{
		Use:  "poll",
		Long: "Git poller is a tool to poll Openshift Origin build configurations and launch builds based on those configurations",
		Run: func(c *cobra.Command, args []string) {
			hook.Start(openshiftEndpoint)
			t := time.Tick(1 * time.Minute)
			for _ = range t {
				fmt.Println("Watching...")
			}
		},
	}
	pollCmd.Flags().StringVarP(&openshiftEndpoint, "endpoint", "e", defaultEndpoint(), "Set openshift v3 master endpoint")
	pollCmd.Execute()
}

func defaultEndpoint() string {
	endpoint := os.Getenv("KUBERNETES_MASTER")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	return endpoint
}
