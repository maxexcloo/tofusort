package main

import (
	"os"

	"github.com/maxexcloo/tofusort/internal/sorter"
	"github.com/spf13/cobra"
)

var (
	recursive   bool
	sortOptions sorter.Options
)

var rootCmd = &cobra.Command{
	Short:        "Sort OpenTofu/Terraform configuration files alphabetically",
	SilenceUsage: true,
	Use:          "tofusort",
	Long: `tofusort is a tool to sort OpenTofu/Terraform configuration files alphabetically.
It sorts blocks by type and attributes within blocks, keeps comments with their
entries, and formats the result with the HCL formatter.`,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&sortOptions.IdentifiersFirst, "identifiers-first", false, "Put type, name and id first in list-item objects")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
