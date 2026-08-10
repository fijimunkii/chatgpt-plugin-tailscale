package remote

import (
	"strings"
	"testing"
)

func TestOperatorCommandQuotesWorkingDirectoryAndInterpretsCommand(t *testing.T) {
	command, err := OperatorCommand("printf '%s\\n' hello && touch marker", "/tmp/work dir; echo nope")
	if err != nil {
		t.Fatalf("OperatorCommand: %v", err)
	}
	if command.Operation() != "operator_exec" {
		t.Fatalf("operation = %q", command.Operation())
	}
	if !strings.Contains(command.Shell(), "cd '/tmp/work dir; echo nope'") {
		t.Fatalf("working directory was not quoted: %q", command.Shell())
	}
	if !strings.Contains(command.Shell(), "/bin/sh -lc") || !strings.Contains(command.Shell(), "touch marker") {
		t.Fatalf("command is not executed through the operator shell: %q", command.Shell())
	}
}

func TestOperatorCommandRejectsEmptyAndNUL(t *testing.T) {
	for _, input := range []string{"", "   ", "echo ok\x00echo bad"} {
		if _, err := OperatorCommand(input, ""); err == nil {
			t.Fatalf("expected command %q to be rejected", input)
		}
	}
}
