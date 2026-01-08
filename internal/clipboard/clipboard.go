// Package clipboard provides platform-specific clipboard operations.
package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CopyHTML attempts to copy HTML content to the system clipboard.
func CopyHTML(htmlContent string) error {
	switch runtime.GOOS {
	case "linux":
		return copyHTMLLinux(htmlContent)
	case "darwin":
		return copyHTMLMacOS(htmlContent)
	case "windows":
		return copyHTMLWindows(htmlContent)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func copyHTMLLinux(htmlContent string) error {
	// Try different clipboard tools in order of preference
	htmlTools := [][]string{
		{"wl-copy", "--type", "text/html"},                        // Wayland HTML
		{"xclip", "-selection", "clipboard", "-t", "text/html"},   // X11 HTML
		{"xsel", "--clipboard", "--input", "--type", "text/html"}, // X11 alternative HTML
	}

	for _, tool := range htmlTools {
		if isCommandAvailable(tool[0]) {
			cmd := exec.Command(tool[0], tool[1:]...)
			cmd.Stdin = strings.NewReader(htmlContent)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	// Fallback: try to copy as plain text
	textTools := [][]string{
		{"wl-copy"},                          // Wayland
		{"xclip", "-selection", "clipboard"}, // X11
		{"xsel", "--clipboard", "--input"},   // X11 alternative
	}

	for _, tool := range textTools {
		if isCommandAvailable(tool[0]) {
			cmd := exec.Command(tool[0], tool[1:]...)
			cmd.Stdin = strings.NewReader(htmlContent)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable clipboard tool found (tried: wl-copy, xclip, xsel)")
}

func copyHTMLMacOS(htmlContent string) error {
	// Use osascript to copy HTML with formatting
	script := fmt.Sprintf(`osascript -e 'set the clipboard to "%s" as «class HTML»'`,
		strings.ReplaceAll(htmlContent, `"`, `\"`))

	cmd := exec.Command("sh", "-c", script)
	return cmd.Run()
}

func copyHTMLWindows(htmlContent string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::SetText(@"
%s
"@, [System.Windows.Forms.TextDataFormat]::Html)`, htmlContent)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
