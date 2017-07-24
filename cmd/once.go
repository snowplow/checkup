package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/snowplow/checkup"
	"github.com/spf13/cobra"
)

var onceCmd = &cobra.Command{
	Use:   "once",
	Short: "Run checks once and exit",
	Long: `The once subcommand runs checkups once and exits.

The result of each check is saved to storage if configured.
Additionally, if a notifier is configured, it will be
called to analyze and potentially notify you of any
problems.

Either storage or notifier (or both) have to be configured.
If both are missing the app will terminate.

Example:

  $ checkup once`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var results []checkup.Result
		notify = true

		if len(args) != 0 {
			fmt.Println(cmd.Long)
			os.Exit(1)
		}

		c := loadCheckup()
		if len(c.Checkers) == 0 {
			log.Fatal("no checkers configured")
		}
		if c.Storage != nil {
			err = c.CheckAndStore()
		} else if c.Notifier == nil {
			log.Fatal("neither storage nor notifier configured")
		} else {
			results, err = c.Check()
		}

		if err != nil {
			log.Fatal(err)
		}

		unhealthyCount := 0
		for _, result := range results {
			if !result.Healthy {
				unhealthyCount++
			}
		}

		if unhealthyCount > 0 {
			log.Printf("Found %d unhealthy endpoints", unhealthyCount)
		}
	},
}

func init() {
	RootCmd.AddCommand(onceCmd)
}
