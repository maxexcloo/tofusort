package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/tofusort/internal/parser"
	"github.com/yourusername/tofusort/internal/sorter"
)

var checkCmd = &cobra.Command{
	Use:   "check [file or directory]",
	Short: "Check if OpenTofu/Terraform files are sorted",
	Long: `Check if OpenTofu/Terraform configuration files are already sorted.
Returns exit code 0 if all files are sorted, 1 if any files need sorting.
Useful for CI/CD pipelines to enforce sorted configuration files.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	p := parser.New()
	s := sorter.New()

	var unsortedFiles []string

	for _, path := range args {
		files, err := checkPath(path, p, s)
		if err != nil {
			return fmt.Errorf("failed to check %s: %w", path, err)
		}
		unsortedFiles = append(unsortedFiles, files...)
	}

	if len(unsortedFiles) > 0 {
		for _, file := range unsortedFiles {
			fmt.Printf("Not sorted: %s\n", file)
		}
		fmt.Printf("\nFound %d unsorted file(s). Run 'tofusort sort' to fix.\n", len(unsortedFiles))
		os.Exit(1)
	}

	fmt.Println("All files are sorted!")
	return nil
}

func checkPath(path string, p *parser.Parser, s *sorter.Sorter) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		return checkDirectory(path, p, s)
	}

	unsorted, err := checkFile(path, p, s)
	if err != nil {
		return nil, err
	}

	if unsorted {
		return []string{path}, nil
	}

	return nil, nil
}

func checkDirectory(dir string, p *parser.Parser, s *sorter.Sorter) ([]string, error) {
	var unsortedFiles []string

	if recursive {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if isTerraformFile(path) {
				unsorted, err := checkFile(path, p, s)
				if err != nil {
					return err
				}
				if unsorted {
					unsortedFiles = append(unsortedFiles, path)
				}
			}

			return nil
		})
		return unsortedFiles, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if isTerraformFile(path) {
			unsorted, err := checkFile(path, p, s)
			if err != nil {
				return nil, err
			}
			if unsorted {
				unsortedFiles = append(unsortedFiles, path)
			}
		}
	}

	return unsortedFiles, nil
}

func checkFile(path string, p *parser.Parser, s *sorter.Sorter) (bool, error) {
	if !isTerraformFile(path) {
		return false, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	file, err := p.ParseFile(content)
	if err != nil {
		return false, fmt.Errorf("failed to parse file: %w", err)
	}

	s.SortFile(file)
	newContent := p.FormatFile(file)

	return string(content) != string(newContent), nil
}
