package main

import (
	"encoding/base64"
	"encoding/binary"
	"runtime"
	"unicode/utf16"
)

// On Windows the programs in the showcase's PTYs are ones Windows has: cmd
// and Windows PowerShell. There is no editor that runs in a terminal there,
// so vi only says so.
func init() {
	if runtime.GOOS == "windows" {
		sessions = windowsSessions
	}
}

var windowsSessions = []session{
	{"shell", "your shell", nil},
	{"keys", "each key as Windows hands it to the program (PowerShell)", powershell(keysScript)},
	{"top", "a full-screen program (PowerShell)", powershell(topScript)},
	{"vi", "an editor that wants Esc: Windows has none in a terminal", []string{"cmd", "/c",
		"echo Windows has no editor that runs in a terminal, so vi is not here.& pause"}},
}

// keysScript prints each key as the program gets it, as cat -v does; Ctrl-C
// ends it.
const keysScript = `'Each key as Windows hands it to the program; Ctrl-C ends.'
while ($true) {
	$k = [Console]::ReadKey($true)
	'{0,-14} U+{1:X4}  {2}' -f $k.Key, [int]$k.KeyChar, $k.Modifiers
}`

// topScript is a full-screen program, as top: the processes by CPU on the
// alternate screen, every second; q ends it.
const topScript = `$e = [char]27
[Console]::Write("$e[?1049h$e[?25l")
try {
	while ($true) {
		$n = [Math]::Max([Console]::WindowHeight - 5, 1)
		$list = Get-Process | Sort-Object CPU -Descending | Select-Object -First $n Id, ProcessName, CPU |
			Format-Table -AutoSize | Out-String -Width ([Console]::WindowWidth)
		[Console]::Write("$e[H$e[2J" + "Processes by CPU, every second; q ends" + $list)
		$until = (Get-Date).AddSeconds(1)
		while ((Get-Date) -lt $until) {
			if ([Console]::KeyAvailable -and [Console]::ReadKey($true).KeyChar -eq 'q') { return }
			Start-Sleep -Milliseconds 50
		}
	}
} finally {
	[Console]::Write("$e[?25h$e[?1049l")
}`

// powershell runs script in Windows PowerShell, handed over encoded so that
// no quote in it needs escaping.
func powershell(script string) []string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	return []string{"powershell", "-NoProfile", "-EncodedCommand", base64.StdEncoding.EncodeToString(b)}
}

// slowStartCommand is a shell that takes a while to connect.
func slowStartCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/k", "echo connecting to slow-host...& timeout /t 3 /nobreak >nul& echo connected"}
	}
	return []string{"sh", "-c", `echo "connecting to slow-host…"; sleep 3; echo connected; exec "${SHELL:-sh}"`}
}

// copyKeyCommand pretends to copy the key to host.
func copyKeyCommand(host string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo Copying ~/.ssh/id_ed25519.pub to " + host + "& echo (the showcase only pretends)& timeout /t 2 /nobreak >nul"}
	}
	return []string{"sh", "-c", "echo Copying ~/.ssh/id_ed25519.pub to " + host + "; echo '(the showcase only pretends)'; sleep 2"}
}
