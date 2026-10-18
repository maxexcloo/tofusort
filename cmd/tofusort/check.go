package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCheck,
	Short: "Check if OpenTofu/Terraform files are sorted",
	Use:   "check [file or directory]",
	Long: `Check if OpenTofu/Terraform configuration files are already sorted.
Returns exit code 0 if all files are sorted, 1 if any files need sorting
or any input is invalid or unreadable.
Useful for CI/CD pipelines to enforce sorted configuration files.`,
}

func init() {
	checkCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	var unsortedFiles []string
	var errs []error

	for _, path := range args {
		err := visitPath(path, "check", func(path string) error {
			content, newContent, err := sortedContent(path)
			if err != nil {
				return err
			}
			if !bytes.Equal(content, newContent) {
				unsortedFiles = append(unsortedFiles, path)
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(unsortedFiles) > 0 {
		for _, file := range unsortedFiles {
			fmt.Printf("Not sorted: %s\n", file)
		}
		command := "tofusort sort"
		if sortOptions.IdentifiersFirst {
			command += " --identifiers-first"
		}
		errs = append(
			errs,
			fmt.Errorf(
				"found %d unsorted file(s); run '%s' to fix",
				len(unsortedFiles),
				command,
			),
		)
	}

	if len(errs) == 0 {
		fmt.Println("All files are sorted!")
	}
	return errors.Join(errs...)
}
