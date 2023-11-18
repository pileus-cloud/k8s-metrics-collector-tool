package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
)

func init() {
  rootCmd.AddCommand(versionCmd)
}

var version = "1.0.0"

var versionCmd = &cobra.Command{
  Use:   "version",
  Short: "Print the version number",
  Long:  ``,
  Run: func(cmd *cobra.Command, args []string) {
    fmt.Printf("Prom agent cli tool version %s\n", version)
  },
}