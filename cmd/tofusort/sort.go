package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/tofusort/internal/parser"
	"github.com/yourusername/tofusort/internal/sorter"
)

var (
	recursive bool
	dryRun    bool
)

var sortCmd = &cobra.Command{
	Use:   "sort [file or directory]",
	Short: "Sort OpenTofu/Terraform files alphabetically",
	Long: `Sort OpenTofu/Terraform configuration files alphabetically.
Sorts blocks by type, then by name within type, and attributes within blocks.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSort,
}

func init() {
	sortCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
	sortCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be changed without modifying files")
	rootCmd.AddCommand(sortCmd)
}

func runSort(cmd *cobra.Command, args []string) error {
	p := parser.New()
	s := sorter.New()

	for _, path := range args {
		if err := processPath(path, p, s); err != nil {
			return fmt.Errorf("failed to process %s: %w", path, err)
		}
	}

	return nil
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
	if recursive {
		return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if isTerraformFile(path) {
				return processFile(path, p, s)
			}

			return nil
		})
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
				return err
			}
		}
	}

	return nil
}

func processFile(path string, p *parser.Parser, s *sorter.Sorter) error {
	if !isTerraformFile(path) {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	file, err := p.ParseFile(content)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	s.SortFile(file)

	newContent := p.FormatFile(file)

	if dryRun {
		if string(content) != string(newContent) {
			fmt.Printf("Would modify: %s\n", path)
		}
		return nil
	}

	if string(content) != string(newContent) {
		if err := os.WriteFile(path, newContent, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Sorted: %s\n", path)
	}

	return nil
}

func isTerraformFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".tfvars"
}
