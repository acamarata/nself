package ui

import "fmt"

// Bullet prints an indented bullet point with a blue bullet icon.
func Bullet(item string) {
	fmt.Printf("  %s %s\n", C(Blue, IconBullet), item)
}

// Checked prints a green checkmark list item.
func Checked(item string) {
	fmt.Printf("  %s %s\n", C(Green, IconSuccess), item)
}

// Unchecked prints a dimmed pending list item.
func Unchecked(item string) {
	fmt.Printf("  %s %s\n", C(Dim, "[ ]"), item)
}
