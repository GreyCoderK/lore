// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build windows

package notify

import (
	"fmt"
)

// NotifyOSDialog launches a Windows PowerShell WPF dialog for Lore documentation.
// Runs as a detached process — does not block the hook.
func NotifyOSDialog(data DialogData, opts DialogOpts) error {
	opts.defaults()

	script := buildPowerShellScript(data)
	return opts.StartCommand("powershell", []string{"-NoProfile", "-Command", script}, nil)
}

func buildPowerShellScript(data DialogData) string {
	commitMsg := escapeXML(sanitizeForShell(data.CommitMsg))
	prefillWhat := escapeXML(sanitizeForShell(data.PrefillWhat))
	prefillWhy := escapeXML(sanitizeForShell(data.PrefillWhy))
	hash := sanitizeCommitHash(data.CommitHash)
	repoRoot := escapePowerShell(data.RepoRoot)
	lorePath := escapePowerShell(data.LorePath)

	// I18n labels with English fallbacks, XML-escaped for XAML.
	labelWhat := escapeXML(coalesce(data.LabelWhat, "What did you change?"))
	labelWhy := escapeXML(coalesce(data.LabelWhy, "Why?"))
	labelSkip := escapeXML(coalesce(data.LabelSkip, "Skip"))
	labelSave := escapeXML(coalesce(data.LabelSave, "Save"))
	title := escapePowerShell(coalesce(data.LabelTitle, "Lore"))

	// Branch Awareness: optional context line in WPF dialog.
	branchXAML := ""
	if data.Branch != "" || data.Scope != "" {
		ctx := ""
		if data.Branch != "" {
			ctx = "Branch: " + data.Branch
		}
		if data.Scope != "" {
			if ctx != "" {
				ctx += " · "
			}
			ctx += "Scope: " + data.Scope
		}
		branchXAML = fmt.Sprintf(`
    <TextBlock Text="%s" Foreground="Gray" Margin="0,5,0,0"/>`, escapeXML(sanitizeForShell(ctx)))
	}

	return fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework
[xml]$xaml = @"
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation"
        Title="%s — Document commit" Height="420" Width="520"
        WindowStartupLocation="CenterScreen">
  <StackPanel Margin="20">
    <TextBlock Text="Commit: %s" FontWeight="Bold" TextWrapping="Wrap"/>%s
    <TextBlock Text="%s" Margin="0,15,0,5"/>
    <ComboBox Name="DocType" SelectedIndex="1">
      <ComboBoxItem>feature</ComboBoxItem>
      <ComboBoxItem>bugfix</ComboBoxItem>
      <ComboBoxItem>decision</ComboBoxItem>
      <ComboBoxItem>refactor</ComboBoxItem>
      <ComboBoxItem>release</ComboBoxItem>
      <ComboBoxItem>note</ComboBoxItem>
    </ComboBox>
    <TextBlock Text="%s" Margin="0,15,0,5"/>
    <TextBox Name="What" Text="%s"/>
    <TextBlock Text="%s" Margin="0,15,0,5"/>
    <TextBox Name="Why" Text="%s" TextWrapping="Wrap" Height="60"/>
    <StackPanel Orientation="Horizontal" HorizontalAlignment="Right" Margin="0,20,0,0">
      <Button Name="BtnCancel" Content="%s" Width="80" Margin="0,0,10,0"/>
      <Button Name="BtnSave" Content="%s" Width="80" IsDefault="True"/>
    </StackPanel>
  </StackPanel>
</Window>
"@
$reader = New-Object System.Xml.XmlNodeReader $xaml
$window = [Windows.Markup.XamlReader]::Load($reader)
$window.FindName("BtnSave").Add_Click({ $window.DialogResult = $true; $window.Close() })
$window.FindName("BtnCancel").Add_Click({ $window.Close() })
if ($window.ShowDialog()) {
    $type = $window.FindName("DocType").SelectedItem.Content
    $what = $window.FindName("What").Text
    $why  = $window.FindName("Why").Text
    Set-Location '%s'
    & '%s' pending resolve --commit '%s' --type "$type" --what "$what" --why "$why"
}
`,
		escapeXML(title), commitMsg,
		branchXAML,
		escapeXML(coalesce(data.LabelType, "Type:")),
		labelWhat, prefillWhat,
		labelWhy, prefillWhy,
		labelSkip, labelSave,
		repoRoot, lorePath, hash,
	)
}
