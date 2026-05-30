// git_complain — Create a GitHub issue from a code snippet in one command.
//
// Usage:
//
//	git-complain <file> <start>-<end> "<issue title>"
//
// Example:
//
//	git-complain main.py 42-55 "This loop throws a NoneType error"
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// extensionToLang maps a lowercase file extension to its Markdown fence language tag.
var extensionToLang = map[string]string{
	".py":    "python",
	".js":    "javascript",
	".jsx":   "jsx",
	".ts":    "typescript",
	".tsx":   "tsx",
	".go":    "go",
	".rs":    "rust",
	".c":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".java":  "java",
	".kt":    "kotlin",
	".swift": "swift",
	".rb":    "ruby",
	".php":   "php",
	".cs":    "csharp",
	".sh":    "bash",
	".bash":  "bash",
	".zsh":   "bash",
	".yaml":  "yaml",
	".yml":   "yaml",
	".toml":  "toml",
	".json":  "json",
	".html":  "html",
	".css":   "css",
	".scss":  "scss",
	".sql":   "sql",
	".md":    "markdown",
	".r":     "r",
	".lua":   "lua",
	".ex":    "elixir",
	".exs":   "elixir",
	".hs":    "haskell",
	".tf":    "hcl",
	".dart":  "dart",
}

func usage() {
	fmt.Fprintf(os.Stderr, `git-complain — create a GitHub issue from a code snippet

Usage:
  git-complain <file> <start>-<end> "<title>"

Arguments:
  <file>          Path to the source file
  <start>-<end>   Inclusive 1-indexed line range  (e.g. 42-55)
  <title>         Issue title (quote if it contains spaces)

Examples:
  git-complain main.py 42-55 "This loop throws a NoneType error"
  git-complain src/auth.ts 1-20 "Missing null check on user object"

Requirements:
  GitHub CLI (gh) must be installed and authenticated.
  Install: https://cli.github.com/
  Auth:    gh auth login
`)
}

// fatal prints a formatted error to stderr and exits with code 1.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// parseLineRange parses "X-Y" into (start, end) as 1-indexed integers.
func parseLineRange(s string) (int, int, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("line range must be formatted as 'X-Y' (e.g. '42-55'), got: %q", s)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil || start < 1 {
		return 0, 0, fmt.Errorf("start line must be a positive integer, got: %q", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil || end < 1 {
		return 0, 0, fmt.Errorf("end line must be a positive integer, got: %q", parts[1])
	}

	if end < start {
		return 0, 0, fmt.Errorf("end line (%d) must be >= start line (%d)", end, start)
	}

	return start, end, nil
}

// extractLines reads the file at path and returns lines [start, end] (1-indexed, inclusive).
// Indentation is preserved. end is clamped to the actual file length.
func extractLines(path string, start, end int) (string, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("cannot open %q: %w", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum >= start && lineNum <= end {
			lines = append(lines, scanner.Text())
		}
		if lineNum > end {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", 0, fmt.Errorf("error reading %q: %w", path, err)
	}

	if start > lineNum {
		return "", 0, fmt.Errorf(
			"start line %d exceeds file length (%d lines)", start, lineNum,
		)
	}

	// Clamp reported end to actual lines read.
	actualEnd := end
	if lineNum < end {
		fmt.Fprintf(os.Stderr,
			"warning: end line %d exceeds file length (%d); clamping to %d.\n",
			end, lineNum, lineNum,
		)
		actualEnd = lineNum
	}

	return strings.Join(lines, "\n"), actualEnd, nil
}

// buildMarkdownBody composes the GitHub issue body.
func buildMarkdownBody(title, filePath string, start, end int, snippet string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	lang := extensionToLang[ext] // empty string is fine — fence still works

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", title)
	fmt.Fprintf(&sb, "**File:** `%s` &nbsp;·&nbsp; **Lines:** %d–%d\n\n", filePath, start, end)
	fmt.Fprintf(&sb, "```%s\n", lang)
	fmt.Fprintf(&sb, "%s\n", snippet)
	fmt.Fprintf(&sb, "```\n\n")
	fmt.Fprintf(&sb, "---\n")
	fmt.Fprintf(&sb, "*Created with [git-complain](https://github.com)*")

	return sb.String()
}

// checkGHInstalled aborts with a friendly message if `gh` is not on PATH.
func checkGHInstalled() {
	if _, err := exec.LookPath("gh"); err != nil {
		fatal(
			"GitHub CLI ('gh') not found on PATH.\n" +
				"  Install it from https://cli.github.com/ and run 'gh auth login'.",
		)
	}
}

// createGitHubIssue calls `gh issue create` via subprocess.
func createGitHubIssue(title, body string) {
	cmd := exec.Command("gh", "issue", "create", "--title", title, "--body", body)
	cmd.Stdin = os.Stdin // allow gh to prompt interactively if needed

	out, err := cmd.Output()
	if err != nil {
		// Unpack stderr from ExitError for a precise message.
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.ToLower(strings.TrimSpace(string(exitErr.Stderr)))
		}

		switch {
		case strings.Contains(stderr, "auth") || strings.Contains(stderr, "login"):
			fatal("GitHub CLI is not authenticated.\n  Run 'gh auth login' and try again.")
		case strings.Contains(stderr, "not a git repository"):
			fatal("Current directory is not inside a Git repository.\n  Navigate to your project root and try again.")
		case stderr != "":
			fatal("'gh issue create' failed:\n  %s", stderr)
		default:
			fatal("'gh issue create' failed: %v", err)
		}
	}

	url := strings.TrimSpace(string(out))
	if url != "" {
		fmt.Printf("✓ Issue created: %s\n", url)
	} else {
		fmt.Println("✓ Issue created successfully.")
	}
}

func main() {
	// Print usage when called with no arguments or --help / -h.
	if len(os.Args) == 1 {
		usage()
		os.Exit(0)
	}
	if os.Args[1] == "--help" || os.Args[1] == "-h" {
		usage()
		os.Exit(0)
	}

	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "error: expected exactly 3 arguments: <file> <start>-<end> \"<title>\"")
		fmt.Fprintln(os.Stderr, "Run 'git-complain --help' for usage.")
		os.Exit(1)
	}

	filePath := os.Args[1]
	lineRangeStr := os.Args[2]
	title := os.Args[3]

	// 1. Validate file exists and is a regular file.
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		fatal("file not found: %q", filePath)
	} else if err != nil {
		fatal("cannot stat %q: %v", filePath, err)
	}
	if !info.Mode().IsRegular() {
		fatal("%q is not a regular file", filePath)
	}

	// 2. Parse line range.
	start, end, err := parseLineRange(lineRangeStr)
	if err != nil {
		fatal("%v", err)
	}

	// 3. Extract snippet.
	snippet, actualEnd, err := extractLines(filePath, start, end)
	if err != nil {
		fatal("%v", err)
	}

	// 4. Build Markdown body.
	body := buildMarkdownBody(title, filePath, start, actualEnd, snippet)

	// 5. Verify gh is available.
	checkGHInstalled()

	// 6. Create the issue.
	fmt.Printf("Creating issue %q from %s:%d-%d …\n", title, filePath, start, actualEnd)
	createGitHubIssue(title, body)
}