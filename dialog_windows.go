//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func escapeForPS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func promptManualEntry(filename string) ManualEntry {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Application]::EnableVisualStyles()

$form = New-Object System.Windows.Forms.Form
$form.Text = 'AudioInk — Manual Entry'
$form.Size = New-Object System.Drawing.Size(420, 280)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.MinimizeBox = $false
$form.TopMost = $true

$lblFile = New-Object System.Windows.Forms.Label
$lblFile.Text = '%s'
$lblFile.Location = New-Object System.Drawing.Point(16, 16)
$lblFile.Size = New-Object System.Drawing.Size(370, 20)
$lblFile.Font = New-Object System.Drawing.Font('Segoe UI', 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($lblFile)

$lblArtist = New-Object System.Windows.Forms.Label
$lblArtist.Text = 'Artist:'
$lblArtist.Location = New-Object System.Drawing.Point(16, 52)
$lblArtist.Size = New-Object System.Drawing.Size(60, 20)
$form.Controls.Add($lblArtist)

$txtArtist = New-Object System.Windows.Forms.TextBox
$txtArtist.Location = New-Object System.Drawing.Point(80, 50)
$txtArtist.Size = New-Object System.Drawing.Size(300, 24)
$form.Controls.Add($txtArtist)

$lblTitle = New-Object System.Windows.Forms.Label
$lblTitle.Text = 'Title:'
$lblTitle.Location = New-Object System.Drawing.Point(16, 92)
$lblTitle.Size = New-Object System.Drawing.Size(60, 20)
$form.Controls.Add($lblTitle)

$txtTitle = New-Object System.Windows.Forms.TextBox
$txtTitle.Location = New-Object System.Drawing.Point(80, 90)
$txtTitle.Size = New-Object System.Drawing.Size(300, 24)
$form.Controls.Add($txtTitle)

$btnSubmit = New-Object System.Windows.Forms.Button
$btnSubmit.Text = 'Submit'
$btnSubmit.Location = New-Object System.Drawing.Point(190, 140)
$btnSubmit.Size = New-Object System.Drawing.Size(90, 32)
$btnSubmit.DialogResult = [System.Windows.Forms.DialogResult]::OK
$form.Controls.Add($btnSubmit)
$form.AcceptButton = $btnSubmit

$btnSkip = New-Object System.Windows.Forms.Button
$btnSkip.Text = 'Skip'
$btnSkip.Location = New-Object System.Drawing.Point(290, 140)
$btnSkip.Size = New-Object System.Drawing.Size(90, 32)
$btnSkip.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
$form.Controls.Add($btnSkip)
$form.CancelButton = $btnSkip

$result = $form.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
    $a = $txtArtist.Text.Trim()
    $t = $txtTitle.Text.Trim()
    if ($a -eq '' -and $t -eq '') {
        Write-Output 'SKIP'
    } else {
        Write-Output "$a|$t"
    }
} else {
    Write-Output 'SKIP'
}
`, escapeForPS(filename))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	out, err := cmd.Output()
	if err != nil {
		logger.Printf("    dialog error: %v", err)
		return ManualEntry{Skipped: true}
	}
	return parseDialogOutput(string(out))
}
