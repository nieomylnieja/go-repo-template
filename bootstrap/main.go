// Package main provides an interactive CLI for bootstrapping new Go projects from this template.
package main

import (
	"errors"
	"fmt"
	"io"
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
	cfg, err := loadConfig(os.Stdin, os.Stdout)
	if err != nil {
		return err
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

func loadConfig(in io.Reader, out io.Writer) (*config, error) {
	if accountName := os.Getenv("BOOTSTRAP_ACCOUNT"); accountName != "" {
		return loadConfigFromEnv(accountName)
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return nil, fmt.Errorf("interactive mode requires a TTY; set BOOTSTRAP_ACCOUNT and BOOTSTRAP_REPO: %w", err)
	}
	_ = f.Close() // probe only, never written to; close error is not actionable
	return loadConfigInteractive(in, out)
}

func loadConfigFromEnv(accountName string) (*config, error) {
	repoName := os.Getenv("BOOTSTRAP_REPO")
	if repoName == "" {
		return nil, errors.New("BOOTSTRAP_REPO environment variable is required when BOOTSTRAP_ACCOUNT is set")
	}
	accountName = strings.TrimSpace(accountName)
	repoName = strings.TrimSpace(repoName)
	if err := validateName(accountName, "BOOTSTRAP_ACCOUNT"); err != nil {
		return nil, err
	}
	if err := validateName(repoName, "BOOTSTRAP_REPO"); err != nil {
		return nil, err
	}
	return &config{
		accountName:    accountName,
		repoName:       repoName,
		includeBinary:  os.Getenv("BOOTSTRAP_NO_BINARY") != "true",
		includeVersion: os.Getenv("BOOTSTRAP_NO_VERSIONING") != "true",
	}, nil
}

func loadConfigInteractive(in io.Reader, out io.Writer) (*config, error) {
	cfg := &config{}

	accountInput := huh.NewInput().
		Title("GitHub Account Name").
		Description("The GitHub account or organization that owns this repository").
		Value(&cfg.accountName).
		Validate(func(s string) error {
			return validateName(strings.TrimSpace(s), "account name")
		})
	repoInput := huh.NewInput().
		Title("Repository Name").
		Description("The name of your new repository").
		Value(&cfg.repoName).
		Validate(func(s string) error {
			return validateName(strings.TrimSpace(s), "repository name")
		})
	binaryConfirm := huh.NewConfirm().
		Title("Include Binary Support?").
		Description("Include goreleaser configuration and binary build workflows").
		Value(&cfg.includeBinary)
	versionConfirm := huh.NewConfirm().
		Title("Include Versioning Support?").
		Description("Include release drafter and automated versioning workflows").
		Value(&cfg.includeVersion)

	form := huh.NewForm(
		huh.NewGroup(accountInput, repoInput),
		huh.NewGroup(binaryConfirm, versionConfirm),
	).
		WithInput(in).
		WithOutput(out).
		WithAccessible(os.Getenv("ACCESSIBLE") != "")
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}
	cfg.accountName = strings.TrimSpace(cfg.accountName)
	cfg.repoName = strings.TrimSpace(cfg.repoName)
	return cfg, nil
}

func validateName(name, field string) error {
	if name == "" {
		return fmt.Errorf("%s cannot be empty", field)
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("%s contains invalid characters", field)
	}
	return nil
}

func bootstrap(cfg *config) error {
	// The bootstrap tool runs from bootstrap/; chdir to the project root.
	if err := os.Chdir(".."); err != nil {
		return fmt.Errorf("failed to change to parent directory: %w", err)
	}

	if _, err := os.Stat("go.mod"); err != nil {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			cwd = fmt.Sprintf("<unknown: %v>", cwdErr)
		}
		return fmt.Errorf("expected go.mod in %s: %w (are you running from the bootstrap directory?)", cwd, err)
	}
	if _, err := os.Stat(".git"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "  Warning: .git directory not found, this might not be a git repository\n")
		} else {
			fmt.Fprintf(os.Stderr, "  Warning: could not check for .git directory: %v\n", err)
		}
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
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return removeJustfileBinaryRecipes()
}

func removeJustfileBinaryRecipes() error {
	return removeJustfileRecipes(func(line string) bool {
		return (strings.HasPrefix(line, "# Build ") && strings.HasSuffix(line, " binary")) ||
			strings.HasPrefix(line, "# Build and release")
	})
}

// removeJustfileRecipes removes recipe sections from the justfile whose comment header
// matches isSectionHeader. Each section is: comment line, recipe name line, indented
// body lines, and a trailing empty line.
//
// WARNING: Section detection is tightly coupled to the justfile comment format.
// If justfile section comments change, update the callers accordingly.
func removeJustfileRecipes(isSectionHeader func(string) bool) error {
	justfilePath := "justfile"

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
	inSection := false
	skipNextEmpty := false

	for _, line := range lines {
		if !inSection && isSectionHeader(line) {
			inSection = true
			skipNextEmpty = false
			continue
		}

		if inSection && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '#' {
			skipNextEmpty = true
			continue
		}

		if inSection && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		if inSection && len(strings.TrimSpace(line)) == 0 {
			inSection = false
			if skipNextEmpty {
				skipNextEmpty = false
				continue
			}
		}

		newLines = append(newLines, line)
	}

	if inSection {
		return fmt.Errorf("justfile section was never closed (no trailing blank line)")
	}

	newContent := strings.Join(newLines, "\n")
	//nolint:gosec // G306: preserving original file permissions is intentional
	if err := os.WriteFile(justfilePath, []byte(newContent), info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to write justfile: %w", err)
	}

	return nil
}

func removeJustfileRecipeDependency(recipeName, dependencyName string) error {
	justfilePath := "justfile"

	info, err := os.Stat(justfilePath)
	if err != nil {
		return fmt.Errorf("failed to stat justfile: %w", err)
	}

	content, err := os.ReadFile(justfilePath)
	if err != nil {
		return fmt.Errorf("failed to read justfile: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	recipePrefix := recipeName + ":"
	changed := false

	for i, line := range lines {
		if !strings.HasPrefix(line, recipePrefix) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != recipePrefix {
			continue
		}

		dependencies := fields[1:]
		filteredDependencies := dependencies[:0]
		for _, dependency := range dependencies {
			if dependency != dependencyName {
				filteredDependencies = append(filteredDependencies, dependency)
			}
		}
		if len(filteredDependencies) == len(dependencies) {
			continue
		}

		newLine := recipePrefix
		if len(filteredDependencies) > 0 {
			newLine += " " + strings.Join(filteredDependencies, " ")
		}
		lines[i] = newLine
		changed = true
	}

	if !changed {
		return nil
	}

	newContent := strings.Join(lines, "\n")
	//nolint:gosec // G306: preserving original file permissions is intentional
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
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
	if errors.Is(err, fs.ErrNotExist) {
		if _, newErr := os.Stat(newPath); newErr == nil {
			fmt.Printf("  Directory %s already exists, skipping rename\n", newPath)
			return nil
		} else if !errors.Is(newErr, fs.ErrNotExist) {
			return fmt.Errorf("failed to check %s: %w", newPath, newErr)
		}
		return fmt.Errorf("expected directory %s does not exist", oldPath)
	}
	if err != nil {
		return fmt.Errorf("failed to check %s: %w", oldPath, err)
	}

	fmt.Printf("  Renaming %s to %s...\n", oldPath, newPath)
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename %s to %s: %w", oldPath, newPath, err)
	}
	return nil
}

func replacePlaceholders(accountName, repoName string) error {
	fmt.Println("  Replacing placeholders in files...")

	return filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking %s: %w", path, err)
		}

		if d.IsDir() {
			if path == ".git" || path == "bootstrap" {
				return filepath.SkipDir
			}
			return nil
		}

		// WalkDir does not follow symlinks; skip them to avoid modifying files outside the project tree.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		//nolint:gosec // G304: path comes from WalkDir over a known project root
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		strContent := string(content)
		if !strings.Contains(strContent, "x-github-account-name") && !strings.Contains(strContent, "x-repo-name") {
			return nil
		}

		strContent = strings.ReplaceAll(strContent, "x-github-account-name", accountName)
		strContent = strings.ReplaceAll(strContent, "x-repo-name", repoName)

		//nolint:gosec // G306: preserving original file permissions; symlinks already skipped above
		if err := os.WriteFile(path, []byte(strContent), info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		return nil
	})
}

func cleanupBootstrapFiles(repoName string) error {
	fmt.Println("  Cleaning up bootstrap files...")

	dirsToRemove := []string{
		"bootstrap",
		"test",
	}
	for _, dir := range dirsToRemove {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", dir, err)
		}
	}

	filesToRemove := []string{
		"gitsync.json",
	}
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	if err := removeJustfileRecipes(func(line string) bool {
		return line == "# Bootstrap the project from the template" ||
			line == "# Run bootstrap tests"
	}); err != nil {
		return fmt.Errorf("failed to remove bootstrap recipes from justfile: %w", err)
	}

	if err := removeJustfileRecipeDependency("test", "test-bootstrap"); err != nil {
		return fmt.Errorf("failed to remove bootstrap test dependency from justfile: %w", err)
	}

	readme := fmt.Sprintf("# %s\n\nTODO\n", repoName)
	//nolint:gosec // G306: 0o644 is intentional for non-executable text files
	if err := os.WriteFile("README.md", []byte(readme), 0o644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}
