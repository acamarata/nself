package ui

import (
	"fmt"
	"os"
	"strings"
)

// CommandHeader prints a 60-char wide double-line box with title and subtitle.
// Every command starts with this header.
func CommandHeader(title, subtitle string) {
	w := 60
	border := strings.Repeat("═", w-2)

	fmt.Printf("%s%s%s\n", C(Blue, "╔"), C(Blue, border), C(Blue, "╗"))
	fmt.Printf("%s %s %s\n", C(Blue, "║"), padRight(C(Bold, title), title, w-4), C(Blue, "║"))
	fmt.Printf("%s %s %s\n", C(Blue, "║"), padRight(C(Dim, subtitle), subtitle, w-4), C(Blue, "║"))
	fmt.Printf("%s%s%s\n", C(Blue, "╚"), C(Blue, border), C(Blue, "╝"))
}

// CommandHeaderStderr prints the command header to stderr.
// Use this for commands whose stdout output is machine-readable (URLs, values, JSON).
func CommandHeaderStderr(title, subtitle string) {
	w := 60
	border := strings.Repeat("═", w-2)
	fmt.Fprintf(os.Stderr, "%s%s%s\n", C(Blue, "╔"), C(Blue, border), C(Blue, "╗"))
	fmt.Fprintf(os.Stderr, "%s %s %s\n", C(Blue, "║"), padRight(C(Bold, title), title, w-4), C(Blue, "║"))
	fmt.Fprintf(os.Stderr, "%s %s %s\n", C(Blue, "║"), padRight(C(Dim, subtitle), subtitle, w-4), C(Blue, "║"))
	fmt.Fprintf(os.Stderr, "%s%s%s\n", C(Blue, "╚"), C(Blue, border), C(Blue, "╝"))
}

// SummaryBox prints a green double-line box with a star-decorated title and bulleted items.
func SummaryBox(title string, items []string) {
	w := 60
	border := strings.Repeat("═", w-2)

	starTitle := iconStar + " " + title + " " + iconStar
	innerW := w - 4 // space inside ║ _ content _ ║

	fmt.Println()
	fmt.Printf("%s%s%s\n", C(Green, "╔"), C(Green, border), C(Green, "╗"))
	fmt.Printf("%s %s %s\n", C(Green, "║"), padRight(C(Bold, starTitle), starTitle, innerW), C(Green, "║"))
	fmt.Printf("%s%s%s\n", C(Green, "╠"), C(Green, border), C(Green, "╣"))
	for _, item := range items {
		bulletItem := "  " + IconBullet + " " + item
		fmt.Printf("%s %s %s\n", C(Green, "║"), padRight(bulletItem, bulletItem, innerW), C(Green, "║"))
	}
	fmt.Printf("%s%s%s\n", C(Green, "╚"), C(Green, border), C(Green, "╝"))
	fmt.Println()
}

// Separator prints a 60-char dim horizontal line.
func Separator() {
	fmt.Printf("%s\n", C(Dim, strings.Repeat("─", 60)))
}

// padRight pads a styled string to width based on the plain text length.
// styled is the string with ANSI codes, plain is the same string without codes.
func padRight(styled, plain string, width int) string {
	padding := width - len([]rune(plain))
	if padding <= 0 {
		return styled
	}
	return styled + strings.Repeat(" ", padding)
}
