package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tofusort",
	Short: "Sort Terraform/OpenTofu configuration files alphabetically",
	Long: `tofusort is a tool to sort Terraform/OpenTofu configuration files alphabetically.
It sorts blocks by type, attributes within blocks, and preserves comments and formatting.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
