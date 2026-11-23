//go:build windows

package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestPowerShellScriptsParse: the scripts the Windows sessions run are
// PowerShell that Windows PowerShell reads without an error.
func TestPowerShellScriptsParse(t *testing.T) {
	const check = `$errs = $null
[void][System.Management.Automation.Language.Parser]::ParseInput($env:SHOWCASE__PS_SCRIPT, [ref]$null, [ref]$errs)
foreach ($e in $errs) { $e.Message }
exit $errs.Count`
	for name, script := range map[string]string{"keys": keysScript, "top": topScript} {
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", check)
		cmd.Env = append(os.Environ(), "SHOWCASE__PS_SCRIPT="+script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: %v\n%s", name, err, out)
		}
	}
}
