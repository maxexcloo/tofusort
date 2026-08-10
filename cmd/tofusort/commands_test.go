package main

import (
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
