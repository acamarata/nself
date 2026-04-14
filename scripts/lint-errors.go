//go:build ignore

// lint-errors scans user-facing Go source files for bare fmt.Errorf calls
// that should use the structured errs.New/Newf/Wrap pattern instead.
//
// Usage: go run scripts/lint-errors.go [./cmd/commands/ ...]
//
// Exit codes:
//
//	0  No violations found
//	1  Violations found
//	2  Usage error
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// bareErrorfRe matches lines containing fmt.Errorf that are NOT inside comments.
var bareErrorfRe = regexp.MustCompile(`\bfmt\.Errorf\b`)

// allowedPaths are directories where bare fmt.Errorf is acceptable
// (internal packages, tests, scripts).
var allowedSuffixes = []string{
	"_test.go",
}

// allowedDirs are directory basenames where bare fmt.Errorf is allowed.
var allowedDirs = map[string]bool{
	"scripts": true,
}

type violation struct {
	File    string
	Line    int
	Content string
}

func main() {
	dirs := os.Args[1:]
	if len(dirs) == 0 {
		dirs = []string{"./cmd/commands/"}
	}

	var violations []violation

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if allowedDirs[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			for _, suffix := range allowedSuffixes {
				if strings.HasSuffix(path, suffix) {
					return nil
				}
			}

			found, scanErr := scanFile(path)
			if scanErr != nil {
				fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, scanErr)
				return nil
			}
			violations = append(violations, found...)
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error walking %s: %v\n", dir, err)
			os.Exit(2)
		}
	}

	if len(violations) == 0 {
		fmt.Println("lint-errors: no bare fmt.Errorf found in user-facing paths")
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "lint-errors: found %d bare fmt.Errorf call(s) in user-facing code:\n\n", len(violations))
	for _, v := range violations {
		fmt.Fprintf(os.Stderr, "  %s:%d: %s\n", v.File, v.Line, strings.TrimSpace(v.Content))
	}
	fmt.Fprintf(os.Stderr, "\nReplace with errs.New(code, msg), errs.Newf(code, fmt, ...), or errs.Wrap(code, msg, err)\n")
	os.Exit(1)
}

func scanFile(path string) ([]violation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []violation
	scanner := bufio.NewScanner(f)
	lineNum := 0
	inBlockComment := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track block comments.
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlockComment = true
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}

		// Skip single-line comments.
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if bareErrorfRe.MatchString(line) {
			results = append(results, violation{
				File:    path,
				Line:    lineNum,
				Content: line,
			})
		}
	}

	return results, scanner.Err()
}
