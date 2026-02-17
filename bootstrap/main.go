// Package main provides an interactive CLI for bootstrapping new Go projects from this template.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

type config struct {
	accountName    string
	repoName       string
	includeBinary  bool
	includeVersion bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := &config{
		includeBinary:  true,
		includeVersion: true,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub Account Name").
				Description("The GitHub account or organization that owns this repository").
				Value(&cfg.accountName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("account name cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Repository Name").
				Description("The name of your new repository").
				Value(&cfg.repoName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("repository name cannot be empty")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include Binary Support?").
				Description("Include goreleaser configuration and binary build workflows").
				Value(&cfg.includeBinary),

			huh.NewConfirm().
				Title("Include Versioning Support?").
				Description("Include release drafter and automated versioning workflows").
				Value(&cfg.includeVersion),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("form error: %w", err)
	}

	fmt.Println("\n🚀 Bootstrapping project with the following configuration:")
	fmt.Printf("  Account: %s\n", cfg.accountName)
	fmt.Printf("  Repository: %s\n", cfg.repoName)
	fmt.Printf("  Binary support: %v\n", cfg.includeBinary)
	fmt.Printf("  Versioning support: %v\n\n", cfg.includeVersion)

	if err := bootstrap(cfg); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	fmt.Println("✅ Bootstrap complete!")
	return nil
}

func bootstrap(cfg *config) error {
	// Change to parent directory to operate on the template root
	if err := os.Chdir(".."); err != nil {
		return fmt.Errorf("failed to change to parent directory: %w", err)
	}

	if !cfg.includeBinary {
		if err := removeBinarySupport(); err != nil {
			return fmt.Errorf("failed to remove binary support: %w", err)
		}
	} else {
		if err := renameCmd(cfg.repoName); err != nil {
			return fmt.Errorf("failed to rename cmd directory: %w", err)
		}
	}

	if !cfg.includeVersion {
		if err := removeVersioningSupport(); err != nil {
			return fmt.Errorf("failed to remove versioning support: %w", err)
		}
	}

	if err := replacePlaceholders(cfg.accountName, cfg.repoName); err != nil {
		return fmt.Errorf("failed to replace placeholders: %w", err)
	}

	if err := cleanupBootstrapFiles(cfg.repoName); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	return nil
}

func removeBinarySupport() error {
	fmt.Println("  Removing binary support files...")

	filesToRemove := []string{
		".goreleaser.yml",
		".github/workflows/release.yml",
	}

	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return removeJustfileBinaryRecipes()
}

func removeJustfileBinaryRecipes() error {
	justfilePath := "justfile"
	content, err := os.ReadFile(justfilePath)
	if err != nil {
		return fmt.Errorf("failed to read justfile: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inBinarySection := false
	inReleaseSection := false
	skipNextEmpty := false

	for _, line := range lines {
		// Detect section starts
		if strings.HasPrefix(line, "# Build ") && strings.HasSuffix(line, " binary") {
			inBinarySection = true
			continue
		}
		if strings.HasPrefix(line, "# Build and release") {
			inReleaseSection = true
			continue
		}

		// If we're in a section and hit a recipe line (starts with non-space)
		if (inBinarySection || inReleaseSection) && len(line) > 0 && line[0] != ' ' && line[0] != '\t' &&
			line[0] != '#' {
			skipNextEmpty = true
			continue
		}

		// Skip lines that are part of the recipe body (indented)
		if (inBinarySection || inReleaseSection) && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		// If we hit an empty line after a section, end the section
		if (inBinarySection || inReleaseSection) && len(strings.TrimSpace(line)) == 0 {
			inBinarySection = false
			inReleaseSection = false
			if skipNextEmpty {
				skipNextEmpty = false
				continue
			}
		}

		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(justfilePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("failed to write justfile: %w", err)
	}

	return nil
}

func removeVersioningSupport() error {
	fmt.Println("  Removing versioning support files...")

	filesToRemove := []string{
		".github/scripts/release-notes.bash",
		".github/release-drafter.yml",
		".github/workflows/release-drafter.yml",
	}

	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return nil
}

func renameCmd(repoName string) error {
	oldPath := "cmd/x-repo-name"
	newPath := filepath.Join("cmd", repoName)

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		// Already renamed or doesn't exist
		return nil
	}

	fmt.Printf("  Renaming %s to %s...\n", oldPath, newPath)
	return os.Rename(oldPath, newPath)
}

func replacePlaceholders(accountName, repoName string) error {
	fmt.Println("  Replacing placeholders in files...")

	// Use find to get all files, excluding .git directory and bootstrap
	cmd := exec.Command("sh", "-c", "find . -type f -not -path './.git/*' -not -path './bootstrap/*'")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, file := range files {
		if file == "" {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		// Check if file contains placeholders
		strContent := string(content)
		if !strings.Contains(strContent, "x-github-account-name") && !strings.Contains(strContent, "x-repo-name") {
			continue
		}

		// Replace placeholders
		strContent = strings.ReplaceAll(strContent, "x-github-account-name", accountName)
		strContent = strings.ReplaceAll(strContent, "x-repo-name", repoName)

		if err := os.WriteFile(file, []byte(strContent), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	return nil
}

func cleanupBootstrapFiles(repoName string) error {
	fmt.Println("  Cleaning up bootstrap files...")

	// Remove directories
	dirsToRemove := []string{
		"bootstrap",
		"test",
	}

	for _, dir := range dirsToRemove {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", dir, err)
		}
	}

	// Remove files
	filesToRemove := []string{
		"gitsync.json",
	}

	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	// Create new README
	readme := fmt.Sprintf("# %s\n\nTODO\n", repoName)
	if err := os.WriteFile("README.md", []byte(readme), 0o644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}
