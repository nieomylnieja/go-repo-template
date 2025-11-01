package test

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

//go:embed testdata/expected-makefile-with-binary
var expectedMakefileWithBinary string

//go:embed testdata/expected-makefile-no-binary
var expectedMakefileNoBinary string

const (
	testAccountName = "test-account"
	testRepoName    = "test-repo"
)

func TestBootstrap_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, testAccountName, testRepoName)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v\nOutput: %s", err, output)
	}

	t.Run("removes bootstrap directory", func(t *testing.T) {
		assertDirNotExists(t, filepath.Join(tmpDir, "bootstrap"))
	})

	t.Run("removes test directory", func(t *testing.T) {
		assertDirNotExists(t, filepath.Join(tmpDir, "test"))
	})
	t.Run("renames cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assertDirNotExists(t, oldPath)
		assertDirExists(t, newPath)
	})
	t.Run("replaces repo and account names in files", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		assertNotContains(t, goMod, "x-github-account-name")
		assertNotContains(t, goMod, "x-repo-name")
		assertContains(t, goMod, testAccountName)
		assertContains(t, goMod, testRepoName)

		makefile := readFile(t, filepath.Join(tmpDir, "Makefile"))
		assertNotContains(t, makefile, "x-repo-name")

		golangciYml := readFile(t, filepath.Join(tmpDir, ".golangci.yml"))
		assertNotContains(t, golangciYml, "x-github-account-name")
		assertNotContains(t, golangciYml, "x-repo-name")
		assertContains(t, golangciYml, testAccountName)
		assertContains(t, golangciYml, testRepoName)

		featureRequest := readFile(t, filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE", "feature_request.md"))
		assertNotContains(t, featureRequest, "x-repo-name")
		assertContains(t, featureRequest, testRepoName)
	})
	t.Run("creates new README.md", func(t *testing.T) {
		readme := readFile(t, filepath.Join(tmpDir, "README.md"))
		expectedContent := "# " + testRepoName + "\n\nTODO\n"
		if readme != expectedContent {
			t.Errorf("README.md content incorrect.\nExpected: %q\nGot: %q", expectedContent, readme)
		}
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assertFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assertFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("keeps versioning-related files", func(t *testing.T) {
		assertFileExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assertFileExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assertFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))
	})

	t.Run("keeps build and release targets in Makefile", func(t *testing.T) {
		actualMakefile := readFile(t, filepath.Join(tmpDir, "Makefile"))
		expectedMakefile := getExpectedMakefile(t, true)

		if actualMakefile != expectedMakefile {
			t.Errorf(
				"Makefile content differs from expected.\nExpected:\n%s\n\nGot:\n%s",
				expectedMakefile,
				actualMakefile,
			)
		}
	})
}

func TestBootstrap_NoBinaryFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", testAccountName, testRepoName)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v\nOutput: %s", err, output)
	}

	t.Run("removes binary-related files", func(t *testing.T) {
		assertFileNotExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assertFileNotExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("removes build and release targets from Makefile", func(t *testing.T) {
		actualMakefile := readFile(t, filepath.Join(tmpDir, "Makefile"))
		expectedMakefile := getExpectedMakefile(t, false)

		if actualMakefile != expectedMakefile {
			t.Errorf(
				"Makefile content differs from expected.\nExpected:\n%s\n\nGot:\n%s",
				expectedMakefile,
				actualMakefile,
			)
		}
	})
	t.Run("does not rename cmd directory", func(t *testing.T) {
		oldPath := filepath.Join(tmpDir, "cmd", "x-repo-name")
		newPath := filepath.Join(tmpDir, "cmd", testRepoName)

		assertDirNotExists(t, newPath)
		assertDirExists(t, oldPath)
	})
	t.Run("still replaces placeholders", func(t *testing.T) {
		goMod := readFile(t, filepath.Join(tmpDir, "go.mod"))
		if strings.Contains(goMod, "x-github-account-name") || strings.Contains(goMod, "x-repo-name") {
			t.Error("go.mod still contains placeholders")
		}
	})
	t.Run("output indicates flag was processed", func(t *testing.T) {
		assertContains(t, output, "--no-binary flag is set")
	})
}

func TestBootstrap_NoVersioningFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-versioning", testAccountName, testRepoName)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v\nOutput: %s", err, output)
	}

	t.Run("removes versioning-related files", func(t *testing.T) {
		assertFileNotExists(t, filepath.Join(tmpDir, ".github", "scripts", "release-notes.bash"))
		assertFileNotExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
		assertFileNotExists(t, filepath.Join(tmpDir, ".github", "workflows", "release-drafter.yml"))
	})
	t.Run("keeps binary-related files", func(t *testing.T) {
		assertFileExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
		assertFileExists(t, filepath.Join(tmpDir, ".github", "workflows", "release.yml"))
	})
	t.Run("output indicates flag was processed", func(t *testing.T) {
		assertContains(t, output, "--no-versioning flag is set")
	})
}

func TestBootstrap_BothFlags(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--no-binary", "--no-versioning", testAccountName, testRepoName)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v\nOutput: %s", err, output)
	}

	t.Run("removes binary-related files", func(t *testing.T) {
		assertFileNotExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	})
	t.Run("removes versioning-related files", func(t *testing.T) {
		assertFileNotExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
	})
	t.Run("output indicates both flags were processed", func(t *testing.T) {
		assertContains(t, output, "--no-binary flag is set")
		assertContains(t, output, "--no-versioning flag is set")
	})
}

func TestBootstrap_FlagsInDifferentOrder(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, "--no-versioning", "--no-binary", testAccountName, testRepoName)
	if err != nil {
		t.Fatalf("Bootstrap failed with flags in different order: %v", err)
	}

	assertFileNotExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
	assertFileNotExists(t, filepath.Join(tmpDir, ".github", "release-drafter.yml"))
}

func TestBootstrap_MissingArguments(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no arguments",
			args: []string{},
		},
		{
			name: "only account name",
			args: []string{testAccountName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			copyProject(t, tmpDir)

			output, err := runBootstrap(t, tmpDir, tt.args...)
			if err == nil {
				t.Error("Expected bootstrap to fail with missing arguments, but it succeeded")
			}
			assertContains(t, output, "Usage:")
		})
	}
}

func TestBootstrap_UnknownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	output, err := runBootstrap(t, tmpDir, "--unknown-flag", testAccountName, testRepoName)
	if err == nil {
		t.Error("Expected bootstrap to fail with unknown flag, but it succeeded")
	}
	assertContains(t, output, "Unknown option")
}

func TestBootstrap_FlagAfterPositionalArgs(t *testing.T) {
	tmpDir := t.TempDir()
	copyProject(t, tmpDir)

	_, err := runBootstrap(t, tmpDir, testAccountName, testRepoName, "--no-binary")
	if err != nil {
		t.Fatalf("Bootstrap should handle flags after positional args: %v", err)
	}
	assertFileNotExists(t, filepath.Join(tmpDir, ".goreleaser.yml"))
}

func getExpectedMakefile(t *testing.T, includeBinary bool) string {
	t.Helper()

	if includeBinary {
		return expectedMakefileWithBinary
	}
	return expectedMakefileNoBinary
}

func runBootstrap(t *testing.T, tmpDir string, args ...string) (string, error) {
	t.Helper()

	bootstrapScript := filepath.Join(tmpDir, "bootstrap", "init.bash")
	cmd := exec.Command(bootstrapScript, args...)
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected string to contain %q", needle)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected string to not contain %q", needle)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if !fileExists(path) {
		t.Errorf("expected file to exist: %s", path)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if fileExists(path) {
		t.Errorf("expected file to not exist: %s", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	if !dirExists(path) {
		t.Errorf("expected directory to exist: %s", path)
	}
}

func assertDirNotExists(t *testing.T, path string) {
	t.Helper()
	if dirExists(path) {
		t.Errorf("expected directory to not exist: %s", path)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

func copyProject(t *testing.T, dst string) {
	t.Helper()
	src := findModuleRoot(t)
	err := os.CopyFS(dst, os.DirFS(src))
	if err != nil {
		t.Fatalf("Failed to copy directory %s to %s: %v", src, dst, err)
	}
}

var (
	moduleRoot         string
	findModuleRootOnce sync.Once
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	findModuleRootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		dir = filepath.Clean(dir)
		for {
			if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
				moduleRoot = dir
				return
			}
			d := filepath.Dir(dir)
			if d == dir {
				break
			}
			dir = d
		}
	})
	return moduleRoot
}
