package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// notificationsEnabled can be set to false during tests to suppress real OS notifications.
var notificationsEnabled = true

// showNotification displays a native OS notification with the fix results.
func showNotification(success, errors, total int) {
	if !notificationsEnabled {
		return
	}
	var message string

	switch {
	case errors == 0:
		message = fmt.Sprintf("Fixed %d file(s)", success)
	case success == 0:
		message = fmt.Sprintf("Failed to fix %d file(s)", errors)
	default:
		message = fmt.Sprintf("Fixed %d/%d file(s), %d error(s)", success, total, errors)
	}

	switch runtime.GOOS {
	case "windows":
		notifyWindows(message)
	case "darwin":
		notifyDarwin(message)
	default:
		notifyLinux(message)
	}
}

// notifyWindows sends a balloon tip notification via PowerShell's NotifyIcon.
func notifyWindows(message string) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.Visible = $true
$n.ShowBalloonTip(3000, 'AudioInk', '%s', 'Info')
Start-Sleep -Seconds 4
$n.Dispose()
`, escapePowerShell(message))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	_ = cmd.Start()
}

// notifyDarwin sends a notification via osascript on macOS.
func notifyDarwin(message string) {
	script := fmt.Sprintf(`display notification "%s" with title "AudioInk"`, escapeAppleScript(message))
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Start()
}

// notifyLinux sends a notification via notify-send on Linux.
func notifyLinux(message string) {
	cmd := exec.Command("notify-send", "AudioInk", message, "-t", "3000")
	_ = cmd.Start()
}

// escapePowerShell escapes single quotes in a string for safe embedding in
// a PowerShell single-quoted string literal.
func escapePowerShell(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// escapeAppleScript escapes backslashes and double quotes for safe embedding
// in an AppleScript double-quoted string literal.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
