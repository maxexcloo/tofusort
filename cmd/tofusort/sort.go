package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	dryRun    bool
	recursive bool
)

var sortCmd = &cobra.Command{
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSort,
	Short: "Sort OpenTofu/Terraform files alphabetically",
	Use:   "sort [file or directory]",
	Long: `Sort OpenTofu/Terraform configuration files alphabetically.
Sorts blocks by type, then by name within type, and attributes within blocks.`,
}

func init() {
	sortCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without modifying files")
	sortCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
	rootCmd.AddCommand(sortCmd)
}

func runSort(cmd *cobra.Command, args []string) error {
	var errs []error

	for _, path := range args {
		if err := visitPath(path, "process", processFile); err != nil {
			errs = append(errs, fmt.Errorf("failed to process %s: %w", path, err))
		}
	}

	return errors.Join(errs...)
}

func processFile(path string) error {
	content, newContent, err := sortedContent(path)
	if err != nil {
		return err
	}

	if dryRun {
		if !bytes.Equal(content, newContent) {
			fmt.Printf("Would modify: %s\n", path)
		}
		return nil
	}

	if !bytes.Equal(content, newContent) {
		if err := replaceFile(path, newContent); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Sorted: %s\n", path)
	}

	return nil
}

func replaceFile(path string, content []byte) error {
	// Replace the target rather than breaking an explicitly supplied symlink.
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".tofusort-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), target)
}
