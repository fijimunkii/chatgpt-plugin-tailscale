//go:build linux

package remote

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSystemSummaryCommandRunsOnLinux(t *testing.T) {
	command := SystemSummaryCommand()
	output, err := exec.Command("sh", "-c", command.Shell()).CombinedOutput()
	if err != nil {
		t.Fatalf("system summary failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, prefix := range []string{
		"hostname\t",
		"kernel\t",
		"uptime_seconds\t",
		"load_average\t",
		"memory_total_kib\t",
		"boot_id\t",
	} {
		if !strings.Contains(text, prefix) {
			t.Fatalf("summary missing %q:\n%s", prefix, text)
		}
	}
}
