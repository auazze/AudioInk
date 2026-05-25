package main

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
	"strings"
)

// showNotification surfaces the outcome of a CLI auto-fix to the user.
// Without this, right-clicking files and picking Auto-fix would silently
// either succeed or fail with no visual feedback.
//
// Platform strategies:
//   - Windows: Burnt Toast style notification via PowerShell (no extra deps).
//   - macOS:   osascript display notification.
//   - Linux:   notify-send if available, otherwise no-op.
func showNotification(success, errors, total int) {
	if total == 0 {
		return
	}

	title := "AudioInk"
	var msg string
	switch {
	case errors == 0:
		msg = fmt.Sprintf("Fixed %d file(s)", success)
	case success == 0:
		msg = fmt.Sprintf("Failed: %d / %d", errors, total)
	default:
		msg = fmt.Sprintf("Fixed %d, failed %d (of %d)", success, errors, total)
	}

	switch goruntime.GOOS {
	case "windows":
		notifyWindows(title, msg)
	case "darwin":
		notifyMacOS(title, msg)
	case "linux":
		notifyLinux(title, msg)
	}
}

// notifyWindows uses PowerShell + a Windows Forms balloon tip — works on
// every Windows version since 7 without requiring BurntToast or the
// Microsoft.Toolkit.Uwp.Notifications PowerShell module to be installed.
// Disappears after ~5 seconds on its own.
func notifyWindows(title, msg string) {
	// Escape single quotes for PowerShell single-quoted strings.
	t := strings.ReplaceAll(title, "'", "''")
	m := strings.ReplaceAll(msg, "'", "''")

	script := fmt.Sprintf(`
[void] [System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms')
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.Visible = $true
$n.BalloonTipTitle = '%s'
$n.BalloonTipText = '%s'
$n.ShowBalloonTip(5000)
Start-Sleep -Seconds 6
$n.Dispose()
`, t, m)

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	if err := cmd.Start(); err != nil {
		logger.Printf("notify (windows): %v", err)
		return
	}
	// Detach — we don't need to wait for the popup to clear.
	go func() { _ = cmd.Wait() }()
}

func notifyMacOS(title, msg string) {
	t := strings.ReplaceAll(title, `"`, `\"`)
	m := strings.ReplaceAll(msg, `"`, `\"`)
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, m, t)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		logger.Printf("notify (macos): %v", err)
	}
}

func notifyLinux(title, msg string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return // notify-send not installed — skip silently
	}
	cmd := exec.Command("notify-send", title, msg)
	if err := cmd.Run(); err != nil {
		logger.Printf("notify (linux): %v", err)
	}
}
