package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmYesNo prompts the user with "(y/N)" and returns true only on "y"/"Y".
// If skipConfirm is true, returns true without prompting.
func ConfirmYesNo(prompt string, skipConfirm bool) bool {
	if skipConfirm {
		return true
	}
	fmt.Printf("%s (y/N) ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	return strings.EqualFold(input, "y")
}

// ConfirmTypeFull prompts the user to type a specific word (like "yes") to confirm.
// If skipConfirm is true, returns true without prompting.
func ConfirmTypeFull(prompt string, expected string, skipConfirm bool) bool {
	if skipConfirm {
		return true
	}
	fmt.Printf("%s Type '%s' to confirm: ", prompt, expected)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	return input == expected
}
