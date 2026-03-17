// Package main provides an interactive CLI for bootstrapping new Go projects from this template.
package main

import (
	"fmt"
	"io/fs"
	"os"
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

	// Check for non-interactive mode via environment variables (for testing)
	if accountName := os.Getenv("BOOTSTRAP_ACCOUNT"); accountName != "" {
		repoName := os.Getenv("BOOTSTRAP_REPO")
		if repoName == "" {
			return fmt.Errorf("BOOTSTRAP_REPO environment variable is required when BOOTSTRAP_ACCOUNT is set")
		}

		cfg.accountName = strings.TrimSpace(accountName)
		cfg.repoName = strings.TrimSpace(repoName)
		cfg.includeBinary = os.Getenv("BOOTSTRAP_NO_BINARY") != "true"
		cfg.includeVersion = os.Getenv("BOOTSTRAP_NO_VERSIONING") != "true"

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

	// Interactive mode - check if /dev/tty is accessible
	// In non-interactive environments (tests, CI), /dev/tty won't be available
	if _, err := os.Open("/dev/tty"); err != nil {
		return fmt.Errorf("interactive mode requires a TTY. Use BOOTSTRAP_ACCOUNT and BOOTSTRAP_REPO environment variables for non-interactive usage")
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

	// Trim whitespace from user input to prevent invalid paths/module names
	cfg.accountName = strings.TrimSpace(cfg.accountName)
	cfg.repoName = strings.TrimSpace(cfg.repoName)

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
	// Change to parent directory to operate on the template root.
	// This affects all subsequent file operations in this process.
	// The bootstrap tool is expected to run from bootstrap/ subdirectory.
	if err := os.Chdir(".."); err != nil {
		return fmt.Errorf("failed to change to parent directory: %w", err)
	}

	// Validate we're in the expected location by checking for marker files
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("after changing directory, expected go.mod file not found - are you running from the bootstrap directory?")
	}
	if _, err := os.Stat(".git"); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: .git directory not found, this might not be a git repository\n")
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

// removeJustfileBinaryRecipes removes specific recipe sections from justfile.
// A "section" consists of: comment line, recipe name line, indented body, and trailing empty line.
// This parses justfile text format which uses indentation to denote recipe bodies.
// WARNING: Section detection is tightly coupled to justfile comment format.
// If justfile section comments change, update the detection patterns accordingly.
func removeJustfileBinaryRecipes() error {
	justfilePath := "justfile"

	// Get original file permissions to preserve them
	info, err := os.Stat(justfilePath)
	if err != nil {
		return fmt.Errorf("failed to stat justfile: %w", err)
	}

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
		// Detect section starts by matching exact comment patterns.
		// These patterns are specific to the current justfile structure.
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
	// Preserve original file permissions
	if err := os.WriteFile(justfilePath, []byte(newContent), info.Mode().Perm()); err != nil {
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
	// oldPath is hardcoded to match the template's placeholder directory.
	// This must be kept in sync with the template structure.
	oldPath := "cmd/x-repo-name"
	newPath := filepath.Join("cmd", repoName)

	_, err := os.Stat(oldPath)
	if os.IsNotExist(err) {
		// Check if new path already exists (already renamed in a previous run)
		if _, err := os.Stat(newPath); err == nil {
			fmt.Printf("  Directory %s already exists, skipping rename\n", newPath)
			return nil
		}
		// Neither old nor new path exists - this might be an error in the template
		return fmt.Errorf("expected directory %s does not exist", oldPath)
	}
	if err != nil {
		return fmt.Errorf("failed to check %s: %w", oldPath, err)
	}

	fmt.Printf("  Renaming %s to %s...\n", oldPath, newPath)
	return os.Rename(oldPath, newPath)
}

func replacePlaceholders(accountName, repoName string) error {
	fmt.Println("  Replacing placeholders in files...")

	return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git and bootstrap directories
		if d.IsDir() {
			if path == ".git" || path == "bootstrap" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get file info to preserve permissions
		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: Could not stat %s: %v\n", path, err)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			// Some files might be binary or have permission issues
			fmt.Fprintf(os.Stderr, "  Warning: Could not read %s: %v\n", path, err)
			return nil
		}

		// Check if file contains placeholders
		strContent := string(content)
		if !strings.Contains(strContent, "x-github-account-name") && !strings.Contains(strContent, "x-repo-name") {
			return nil
		}

		// Replace placeholders
		strContent = strings.ReplaceAll(strContent, "x-github-account-name", accountName)
		strContent = strings.ReplaceAll(strContent, "x-repo-name", repoName)

		// Preserve original file permissions
		if err := os.WriteFile(path, []byte(strContent), info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		return nil
	})
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

	// Create new README with standard non-executable file permissions
	readme := fmt.Sprintf("# %s\n\nTODO\n", repoName)
	//nolint:gosec // G306: 0o644 is intentional for non-executable text files
	if err := os.WriteFile("README.md", []byte(readme), 0o644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}
