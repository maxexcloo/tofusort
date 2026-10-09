package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/maxexcloo/tofusort/internal/sorter"
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
	p := parser.New()
	s := sorter.New()
	var errs []error

	for _, path := range args {
		if err := processPath(path, p, s); err != nil {
			errs = append(errs, fmt.Errorf("failed to process %s: %w", path, err))
		}
	}

	return errors.Join(errs...)
}

func processPath(path string, p *parser.Parser, s *sorter.Sorter) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return processDirectory(path, p, s)
	}

	return processFile(path, p, s)
}

func processDirectory(dir string, p *parser.Parser, s *sorter.Sorter) error {
	var errs []error

	if recursive {
		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to access %s: %w", path, err))
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if isTerraformFile(path) {
				if err := processFile(path, p, s); err != nil {
					errs = append(errs, fmt.Errorf("failed to process %s: %w", path, err))
				}
			}

			return nil
		})
		if walkErr != nil {
			errs = append(errs, walkErr)
		}
		return errors.Join(errs...)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if isTerraformFile(path) {
			if err := processFile(path, p, s); err != nil {
				errs = append(errs, fmt.Errorf("failed to process %s: %w", path, err))
			}
		}
	}

	return errors.Join(errs...)
}

func processFile(path string, p *parser.Parser, s *sorter.Sorter) error {
	if !isTerraformFile(path) {
		return fmt.Errorf("unsupported file format: %s (expected .tf or .tfvars)", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	file, err := p.ParseFile(content)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	if err := s.SortFile(file); err != nil {
		return err
	}

	newContent := p.FormatFile(file)

	if dryRun {
		if string(content) != string(newContent) {
			fmt.Printf("Would modify: %s\n", path)
		}
		return nil
	}

	if string(content) != string(newContent) {
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

func isTerraformFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".tfvars"
}
