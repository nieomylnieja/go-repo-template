package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"
)

//go:embed testdata/expected-justfile-with-binary
var expectedJustfileWithBinary string

//go:embed testdata/expected-justfile-no-binary
var expectedJustfileNoBinary string

const (
	testAccountName = "test-account"
	testRepoName    = "test-repo"
)

func TestBootstrap_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes bootstrap directory", func(t *testing.T) {
		assert.NoDirExists(t, filepath.Join(tmpDir, "bootstrap"))
	})

	t.Run("removes test directory", func(t *testing.T) {
		assert.NoDirExists(t, filepath.Join(tmpDir, "test"))
	})
	t.Run("renames cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assert.NoDirExists(t, oldPath)
		assert.DirExists(t, newPath)
	})
	t.Run("replaces repo and account names in files", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		assert.NotContains(t, goMod, "x-github-account-name")
		assert.NotContains(t, goMod, "x-repo-name")
		assert.Contains(t, goMod, testAccountName)
		assert.Contains(t, goMod, testRepoName)

		justfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		assert.NotContains(t, justfile, "x-repo-name")

		golangciYml := readFile(t, filepath.Join(tmpDir, ".golangci.yml"))
		assert.NotContains(t, golangciYml, "x-github-account-name")
		assert.NotContains(t, golangciYml, "x-repo-name")
		assert.Contains(t, golangciYml, testAccountName)
		assert.Contains(t, golangciYml, testRepoName)

		featureRequest := readFile(t, filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE", "feature_request.md"))
		assert.NotContains(t, featureRequest, "x-repo-name")
		assert.Contains(t, featureRequest, testRepoName)
	})
	t.Run("creates new README.md", func(t *testing.T) {
		readme := readFile(t, filepath.Join(tmpDir, "README.md"))
		expectedContent := "# " + testRepoName + "\n\nTODO\n"
		assert.Equal(t, expectedContent, readme, "README.md content incorrect")
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("keeps versioning-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "pr-autolabeler.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))
	})

	t.Run("keeps build and release recipes in justfile", func(t *testing.T) {
		actualJustfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		expectedJustfile := getExpectedJustfile(t, true)
		assert.Equal(t, expectedJustfile, actualJustfile, "justfile content differs from expected")
	})
}

func TestBootstrap_NoBinaryFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes binary-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("removes build and release recipes from justfile", func(t *testing.T) {
		actualJustfile := readFile(t, filepath.Join(tmpDir, "justfile"))
		expectedJustfile := getExpectedJustfile(t, false)
		assert.Equal(t, expectedJustfile, actualJustfile, "justfile content differs from expected")
	})
	t.Run("does not rename cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assert.NoDirExists(t, newPath)
		assert.DirExists(t, oldPath)
	})
	t.Run("still replaces placeholders", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		assert.NotContains(t, goMod, "x-github-account-name")
		assert.NotContains(t, goMod, "x-repo-name")
	})
	t.Run("output shows binary support disabled", func(t *testing.T) {
		assert.Contains(t, output, "Binary support: false")
	})
}

func TestBootstrap_NoVersioningFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-versioning", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes versioning-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "pr-autolabeler.yml"))
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assert.FileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assert.FileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("output shows versioning support disabled", func(t *testing.T) {
		assert.Contains(t, output, "Versioning support: false")
	})
}

func TestBootstrap_BothFlags(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", "--no-versioning", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed. Output: %s", output)

	t.Run("removes binary-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	})
	t.Run("removes versioning-related files", func(t *testing.T) {
		assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
	})
	t.Run("output indicates both options were processed", func(t *testing.T) {
		assert.Contains(t, output, "Binary support: false")
		assert.Contains(t, output, "Versioning support: false")
	})
}

func TestBootstrap_FlagsInDifferentOrder(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, "--no-versioning", "--no-binary", testAccountName, testRepoName)
	require.NoError(t, err, "Bootstrap failed with flags in different order")

	assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	assert.NoFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
}

func TestBootstrap_MissingArguments(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectedErr: "interactive mode requires a TTY",
		},
		{
			name:        "only account name",
			args:        []string{testAccountName},
			expectedErr: "BOOTSTRAP_REPO environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			copyProject(t, tmpDir)

			output, err := runBootstrap(t, tmpDir, tt.args...)
			assert.Error(t, err, "Expected bootstrap to fail with missing arguments")
			assert.Contains(t, output, tt.expectedErr)
		})
	}
}

func TestBootstrap_FlagAfterPositionalArgs(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, testAccountName, testRepoName, "--no-binary")
	require.NoError(t, err, "Bootstrap should handle flags after positional args")
	assert.NoFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
}

func TestLoadConfigInteractive_BothEnabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, true, true)
	require.NoError(t, err)
	assert.Equal(t, "my-account", cfg.accountName)
	assert.Equal(t, "my-repo", cfg.repoName)
	assert.True(t, cfg.includeBinary)
	assert.True(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_BinaryDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, false, true)
	require.NoError(t, err)
	assert.False(t, cfg.includeBinary)
	assert.True(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_VersioningDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, true, false)
	require.NoError(t, err)
	assert.True(t, cfg.includeBinary)
	assert.False(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_BothDisabled(t *testing.T) {
	cfg, err := runLoadConfigInteractive(t, false, false)
	require.NoError(t, err)
	assert.False(t, cfg.includeBinary)
	assert.False(t, cfg.includeVersion)
}

func TestLoadConfigInteractive_WhitespaceTrimmed(t *testing.T) {
	stdin, stdout, cancel := huhtest.NewResponder().
		AddResponse("GitHub Account Name", "  my-account  ").
		AddResponse("Repository Name", "  my-repo  ").
		AddConfirm("Include Binary Support?", huhtest.ConfirmAffirm).
		AddConfirm("Include Versioning Support?", huhtest.ConfirmAffirm).
		Start(t, 30*time.Second)
	defer cancel()

	cfg, err := loadConfigInteractive(stdin, stdout)
	require.NoError(t, err)
	assert.Equal(t, "my-account", cfg.accountName)
	assert.Equal(t, "my-repo", cfg.repoName)
}

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		account        string
		repo           string
		noBinary       string
		noVersioning   string
		wantErr        bool
		wantAccount    string
		wantRepo       string
		wantBinary     bool
		wantVersioning bool
	}{
		{
			name:    "missing BOOTSTRAP_REPO returns error",
			account: "my-account",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "whitespace-only BOOTSTRAP_REPO returns error",
			account: "my-account",
			repo:    "   ",
			wantErr: true,
		},
		{
			name:    "whitespace-only BOOTSTRAP_ACCOUNT returns error",
			account: "   ",
			repo:    "my-repo",
			wantErr: true,
		},
		{
			name:           "whitespace is trimmed from account and repo",
			account:        "  my-account  ",
			repo:           "  my-repo  ",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_BINARY=true disables binary",
			account:        "my-account",
			repo:           "my-repo",
			noBinary:       "true",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     false,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_BINARY not set enables binary",
			account:        "my-account",
			repo:           "my-repo",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
		{
			name:           "BOOTSTRAP_NO_VERSIONING=true disables versioning",
			account:        "my-account",
			repo:           "my-repo",
			noVersioning:   "true",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: false,
		},
		{
			name:           "BOOTSTRAP_NO_VERSIONING not set enables versioning",
			account:        "my-account",
			repo:           "my-repo",
			wantAccount:    "my-account",
			wantRepo:       "my-repo",
			wantBinary:     true,
			wantVersioning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BOOTSTRAP_ACCOUNT", tt.account)
			t.Setenv("BOOTSTRAP_REPO", tt.repo)
			t.Setenv("BOOTSTRAP_NO_BINARY", tt.noBinary)
			t.Setenv("BOOTSTRAP_NO_VERSIONING", tt.noVersioning)

			cfg, err := loadConfigFromEnv(tt.account)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAccount, cfg.accountName)
			assert.Equal(t, tt.wantRepo, cfg.repoName)
			assert.Equal(t, tt.wantBinary, cfg.includeBinary)
			assert.Equal(t, tt.wantVersioning, cfg.includeVersion)
		})
	}
}

func getExpectedJustfile(t *testing.T, includeBinary bool) string {
	t.Helper()

	if includeBinary {
		return expectedJustfileWithBinary
	}
	return expectedJustfileNoBinary
}

func runBootstrap(t *testing.T, tmpDir string, args ...string) (string, error) {
	t.Helper()

	// Build the Go bootstrap binary
	bootstrapDir := filepath.Join(tmpDir, "bootstrap")
	buildCmd := exec.Command("go", "build", "-o", "bootstrap-bin")
	buildCmd.Dir = bootstrapDir
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build bootstrap binary. Output: %s", output)

	// Parse arguments to extract flags and set environment variables
	accountName := ""
	repoName := ""
	noBinary := "false"
	noVersioning := "false"

	for _, arg := range args {
		switch arg {
		case "--no-binary":
			noBinary = "true"
		case "--no-versioning":
			noVersioning = "true"
		default:
			if accountName == "" {
				accountName = arg
			} else if repoName == "" {
				repoName = arg
			}
		}
	}

	// Run the bootstrap binary from the bootstrap directory (like the old script)
	// The tool expects to run from bootstrap/ and will cd .. to the project root
	bootstrapBin := filepath.Join(bootstrapDir, "bootstrap-bin")
	//nolint:gosec // G204: bootstrapBin is a test binary path, not user input
	cmd := exec.Command(bootstrapBin)
	cmd.Dir = bootstrapDir // Run from bootstrap/ directory

	// Only set environment variables if values are provided
	// This allows testing the error case when required args are missing
	env := os.Environ()
	if accountName != "" || repoName != "" {
		env = append(env,
			"BOOTSTRAP_ACCOUNT="+accountName,
			"BOOTSTRAP_REPO="+repoName,
			"BOOTSTRAP_NO_BINARY="+noBinary,
			"BOOTSTRAP_NO_VERSIONING="+noVersioning,
		)
	}
	cmd.Env = env

	output, err = cmd.CombinedOutput()
	return string(output), err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	//nolint:gosec // G304: path is a test file path within t.TempDir()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file %s", path)
	return string(content)
}

func runLoadConfigInteractive(t *testing.T, includeBinary, includeVersion bool) (*config, error) {
	t.Helper()

	includeBinaryResponse := huhtest.ConfirmNegative
	if includeBinary {
		includeBinaryResponse = huhtest.ConfirmAffirm
	}
	includeVersionResponse := huhtest.ConfirmNegative
	if includeVersion {
		includeVersionResponse = huhtest.ConfirmAffirm
	}

	stdin, stdout, cancel := huhtest.NewResponder().
		AddResponse("GitHub Account Name", "my-account").
		AddResponse("Repository Name", "my-repo").
		AddConfirm("Include Binary Support?", includeBinaryResponse).
		AddConfirm("Include Versioning Support?", includeVersionResponse).
		Start(t, 30*time.Second)
	defer cancel()

	return loadConfigInteractive(stdin, stdout)
}

func copyProject(t *testing.T, dst string) {
	t.Helper()
	// We're in the bootstrap directory, so copy the parent (project root)
	src := filepath.Join(findModuleRoot(t), "..")
	src, err := filepath.Abs(src)
	require.NoError(t, err)
	err = os.CopyFS(dst, os.DirFS(src))
	require.NoError(t, err, "Failed to copy directory %s to %s", src, dst)
}

var (
	moduleRoot         string
	findModuleRootOnce sync.Once
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	findModuleRootOnce.Do(func() {
		// We're in bootstrap/, so just return current directory
		dir, err := os.Getwd()
		require.NoError(t, err, "Failed to get working directory")
		moduleRoot = filepath.Clean(dir)
	})
	return moduleRoot
}
