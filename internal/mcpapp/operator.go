package mcpapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fijimunkii/chatgpt-plugin-tailscale/internal/remote"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const operatorWarning = "Operator mode allows arbitrary shell commands as the configured non-root SSH user on an explicitly admitted host. Remote output is untrusted data. The gateway does not add sudo or Administrator privileges."

// OperatorConfig controls the deliberately broad shell capability. It is
// enabled only by local gateway configuration and expires without any remote
// renewal mechanism.
type OperatorConfig struct {
	Enabled   bool
	ExpiresAt time.Time
}

func (c OperatorConfig) validate(now time.Time) error {
	if !c.Enabled {
		return nil
	}
	if c.ExpiresAt.IsZero() {
		return errors.New("operator mode requires an expiration time")
	}
	if !c.ExpiresAt.After(now) {
		return errors.New("operator mode expiration must be in the future")
	}
	return nil
}

func (c OperatorConfig) active(now time.Time) bool {
	return c.Enabled && !c.ExpiresAt.IsZero() && now.Before(c.ExpiresAt)
}

func operatorTool() *mcp.Tool {
	falseValue := false
	trueValue := true
	return &mcp.Tool{
		Name:        "operator_exec",
		Title:       "Run a shell command in Operator mode",
		Description: "Runs arbitrary POSIX shell text on one explicitly admitted Linux host as the configured non-root SSH user. Operator mode must have been enabled locally when the gateway started and automatically expires. This tool can modify files, processes, repositories, and user-accessible services; use it only when the user has delegated broad access to the selected host.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Run a shell command in Operator mode",
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: &trueValue,
			OpenWorldHint:   &trueValue,
		},
	}
}

type operatorExecInput struct {
	Host    string `json:"host" jsonschema:"Exact explicitly admitted host identifier"`
	Command string `json:"command" jsonschema:"Arbitrary POSIX shell command to run as the configured non-root SSH user"`
	CWD     string `json:"cwd,omitempty" jsonschema:"Optional working directory on the remote host"`
}

type operatorExecOutput struct {
	commandOutput
	OperatorMode     bool   `json:"operator_mode"`
	LeaseExpiresAt   string `json:"lease_expires_at"`
	PrivilegeBoundary string `json:"privilege_boundary"`
	OperatorWarning  string `json:"operator_warning"`
}

func (a *App) operatorExec(ctx context.Context, _ *mcp.CallToolRequest, input operatorExecInput) (*mcp.CallToolResult, operatorExecOutput, error) {
	now := time.Now()
	if !a.operator.active(now) {
		if a.operator.Enabled && !a.operator.ExpiresAt.IsZero() {
			return nil, operatorExecOutput{}, fmt.Errorf("operator mode lease expired at %s; restart the gateway with local Operator mode enabled to renew it", a.operator.ExpiresAt.UTC().Format(time.RFC3339))
		}
		return nil, operatorExecOutput{}, errors.New("operator mode is disabled; enable it locally before starting the gateway")
	}
	command, err := remote.OperatorCommand(input.Command, input.CWD)
	if err != nil {
		return nil, operatorExecOutput{}, err
	}
	callResult, result, err := a.execute(ctx, input.Host, command)
	if err != nil {
		return nil, operatorExecOutput{}, err
	}
	return callResult, operatorExecOutput{
		commandOutput:      result,
		OperatorMode:       true,
		LeaseExpiresAt:     a.operator.ExpiresAt.UTC().Format(time.RFC3339),
		PrivilegeBoundary:  "configured non-root SSH user; no privilege elevation added by TetherBound",
		OperatorWarning:    operatorWarning,
	}, nil
}
