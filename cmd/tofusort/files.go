package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/maxexcloo/tofusort/internal/sorter"
	"github.com/spf13/cobra"
)

var (
	recursive           bool
	sortOptions         sorter.Options
	terraformExtensions = []string{".tf", ".tfvars"}
)

func addRecursiveFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process directories recursively")
}

// visitPath returns errors that already name the file or directory involved.
func visitPath(path, action string, visit func(string) error) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", action, err)
	}
	if !info.IsDir() {
		if err := visit(path); err != nil {
			return fmt.Errorf("failed to %s %s: %w", action, path, err)
		}
		return nil
	}
	return visitDirectory(path, action, visit)
}

func visitDirectory(dir, action string, visit func(string) error) error {
	var errs []error
	visitFile := func(path string) {
		if isTerraformFile(path) {
			if err := visit(path); err != nil {
				errs = append(errs, fmt.Errorf("failed to %s %s: %w", action, path, err))
			}
		}
	}

	if recursive {
		// WalkDir does not follow even an explicitly supplied directory symlink.
		target, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory %s: %w", dir, err)
		}
		err = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if d != nil && d.IsDir() && path != target && (d.Name() == ".git" || d.Name() == ".terraform") {
				return fs.SkipDir
			}
			// Report paths under the directory as supplied, not its resolved target.
			if rel, relErr := filepath.Rel(target, path); relErr == nil {
				path = filepath.Join(dir, rel)
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to access %s: %w", path, err))
			} else if !d.IsDir() {
				visitFile(path)
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			visitFile(filepath.Join(dir, entry.Name()))
		}
	}
	return errors.Join(errs...)
}

func sortedContent(path string) ([]byte, []byte, error) {
	if !isTerraformFile(path) {
		return nil, nil, fmt.Errorf("unsupported file format: %s (expected %s)", path, strings.Join(terraformExtensions, " or "))
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}
	file, err := parser.ParseFile(content)
	if err != nil {
		return nil, nil, err
	}
	if err := sorter.SortFile(file, sortOptions); err != nil {
		return nil, nil, err
	}
	return content, file.Source, nil
}

func isTerraformFile(path string) bool {
	return slices.Contains(terraformExtensions, strings.ToLower(filepath.Ext(path)))
}
