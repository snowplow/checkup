package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/sourcegraph/checkup"
	"github.com/spf13/cobra"
)

var configFile string
var storeResults bool
var silent bool
var notify bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "checkup",
	Short: "Perform checks on your services and sites",
	Long: `Checkup is distributed, lock-free, self-hosted health
checks and status pages.

Checkup will always look for a checkup.json file in
the current working directory by default and use it.
You can specify a different file location using the
--config/-c flag.

Running checkup without any arguments will invoke
a single checkup and print results to stdout. To
store the results of the check, use --store.

To send notification, use --notify. It also can be
used in conjunction with --silent to prevent
printing out the results to console`,

	Run: func(cmd *cobra.Command, args []string) {
		allHealthy := true
		c := loadCheckup()

		if storeResults {
			if c.Storage == nil {
				log.Fatal("no storage configured")
			}
		}

		results, err := c.Check()
		if err != nil {
			log.Fatal(err)
		}

		if storeResults {
			err := c.Storage.Store(results)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		for _, result := range results {
			if !silent {
				fmt.Println(result)
			} else if !notify {
				log.Fatal("--silent is to be used along with --notify")
			}
			if !result.Healthy {
				allHealthy = false
			}
		}

		if !allHealthy {
			os.Exit(1)
		}
	},
}

func loadCheckup() checkup.Checkup {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	var c checkup.Checkup
	err = json.Unmarshal(configBytes, &c)
	if err != nil {
		log.Fatal(err)
	}
	if !notify {
		c.Notifier = nil
	}

	return c
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.Println("Checkup started")
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "checkup.json", "JSON config file")
	RootCmd.Flags().BoolVar(&storeResults, "store", false, "Store results")
	RootCmd.Flags().BoolVar(&silent, "silent", false, "Do not print results")
	RootCmd.Flags().BoolVar(&notify, "notify", false, "Send notificaion")
}
