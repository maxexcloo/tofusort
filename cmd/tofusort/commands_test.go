package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxexcloo/tofusort/internal/parser"
	"github.com/maxexcloo/tofusort/internal/sorter"
)

func TestIsTerraformFile(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"main.tf":            true,
		"main.tf.json":       false,
		"values.tfvars":      true,
		"values.tfvars.json": false,
	}
	for path, expected := range tests {
		if actual := isTerraformFile(path); actual != expected {
			t.Errorf("isTerraformFile(%q) = %t, want %t", path, actual, expected)
		}
	}
}

func TestProcessDirectoryContinuesAfterInvalidFile(t *testing.T) {
	directory := t.TempDir()
	invalidPath := filepath.Join(directory, "invalid.tf")
	validPath := filepath.Join(directory, "valid.tf")
	if err := os.WriteFile(invalidPath, []byte("invalid {"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, []byte("z = 1\na = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	recursive = false
	dryRun = false
	err := processDirectory(directory, parser.New(), sorter.New())
	if err == nil || !strings.Contains(err.Error(), "invalid.tf") {
		t.Fatalf("processDirectory() error = %v, want invalid.tf context", err)
	}
	content, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "a = 2\nz = 1\n" {
		t.Errorf("valid file was not processed after failure:\n%s", content)
	}
}

func TestRunSortReportsAllInputErrors(t *testing.T) {
	err := runSort(nil, []string{"missing-one.tf", "missing-two.tf"})
	if err == nil {
		t.Fatal("runSort() returned nil")
	}
	for _, path := range []string{"missing-one.tf", "missing-two.tf"} {
		if !strings.Contains(err.Error(), path) {
			t.Errorf("runSort() error does not contain %q: %v", path, err)
		}
	}
}

func TestExplicitUnsupportedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.tf.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, run := range []func() error{
		func() error { return runSort(nil, []string{path}) },
		func() error { return runCheck(nil, []string{path}) },
	} {
		if err := run(); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("wanted unsupported file error with context, got %v", err)
		}
	}
}

func TestCheckAndDryRunDoNotMutate(t *testing.T) {
	for _, recurse := range []bool{false, true} {
		t.Run(fmt.Sprint(recurse), func(t *testing.T) {
			recursive = recurse
			dryRun = true
			t.Cleanup(func() { recursive = false; dryRun = false })
			dir := t.TempDir()
			if recurse {
				dir = filepath.Join(dir, "nested")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			inputs := map[string]string{"invalid.tf": "invalid {", "unsorted.tfvars": "z = 1\na = 2\n", "ignored.json": "{}"}
			for name, input := range inputs {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(input), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			root := dir
			if recurse {
				root = filepath.Dir(dir)
			}
			err := runCheck(nil, []string{root})
			if err == nil || !strings.Contains(err.Error(), "invalid.tf") || !strings.Contains(err.Error(), "1 unsorted") {
				t.Fatalf("missing failures: %v", err)
			}
			if err := runSort(nil, []string{root}); err == nil || !strings.Contains(err.Error(), "invalid.tf") {
				t.Fatalf("missing invalid file: %v", err)
			}
			for name, input := range inputs {
				content, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if string(content) != input {
					t.Fatalf("%s was modified", name)
				}
			}
		})
	}
}

func TestSortPreservesModeAndSymlink(t *testing.T) {
	dryRun = false
	dir := t.TempDir()
	target := filepath.Join(dir, "target.tf")
	if err := os.WriteFile(target, []byte("z = 1\na = 2\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.tf")
	if err := os.Symlink("target.tf", link); err != nil {
		t.Fatal(err)
	}
	if err := processFile(link, parser.New(), sorter.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(link); err != nil {
		t.Fatalf("symlink replaced: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode changed: %v", info.Mode())
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "a = 2\nz = 1\n" {
		t.Fatalf("target not sorted: %s", content)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("temporary file leaked: %v", entries)
	}
}
