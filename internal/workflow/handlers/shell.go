package handlers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow/dag"
)

// ShellHandler executes shell commands
type ShellHandler struct {
	BaseHandler
}

// NewShellHandler creates a new shell handler
func NewShellHandler() *ShellHandler {
	return &ShellHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeShell},
	}
}

// Validate validates the shell step configuration
func (h *ShellHandler) Validate(config map[string]interface{}) error {
	command := GetStringConfig(config, "command", "")
	script := GetStringConfig(config, "script", "")

	if command == "" && script == "" {
		return fmt.Errorf("shell step requires either 'command' or 'script' configuration")
	}

	return nil
}

// Execute runs the shell command
func (h *ShellHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	// Get command or script
	command := GetStringConfig(config, "command", "")
	script := GetStringConfig(config, "script", "")
	shell := GetStringConfig(config, "shell", "")
	workDir := GetStringConfig(config, "workdir", "")
	envVars := GetMapConfig(config, "env")

	// Determine what to run
	var cmdStr string
	if script != "" {
		cmdStr = script
	} else {
		cmdStr = command
	}

	// Determine shell
	var shellCmd string
	var shellArgs []string

	if shell != "" {
		shellCmd = shell
		shellArgs = []string{"-c", cmdStr}
	} else if runtime.GOOS == "windows" {
		shellCmd = "cmd"
		shellArgs = []string{"/C", cmdStr}
	} else {
		shellCmd = "/bin/sh"
		shellArgs = []string{"-c", cmdStr}
	}

	// Create command
	cmd := exec.CommandContext(ctx, shellCmd, shellArgs...)

	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, val := range envVars {
		if strVal, ok := val.(string); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, strVal))
		}
	}

	// Add step env if provided in execution
	for key, val := range step.Inputs {
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", envKey, val))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log start
	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Executing command: %s", cmdStr),
	})

	// Run command
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// Capture output
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	result.Outputs["stdout"] = stdoutStr
	result.Outputs["stderr"] = stderrStr
	result.Outputs["duration_ms"] = duration.Milliseconds()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	result.Outputs["exitCode"] = exitCode

	// Log output
	if stdoutStr != "" {
		// Truncate if too long
		logStdout := stdoutStr
		if len(logStdout) > 1000 {
			logStdout = logStdout[:1000] + "... (truncated)"
		}
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("stdout: %s", logStdout),
		})
	}

	if stderrStr != "" {
		logStderr := stderrStr
		if len(logStderr) > 1000 {
			logStderr = logStderr[:1000] + "... (truncated)"
		}
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: fmt.Sprintf("stderr: %s", logStderr),
		})
	}

	// Log completion
	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Command completed with exit code %d in %v", exitCode, duration),
	})

	// Set status based on exit code
	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.Error = err
		result.ErrorMsg = fmt.Sprintf("command failed with exit code %d: %s", exitCode, stderrStr)
		return result, err
	}

	result.Status = workflow.StepStatusCompleted
	return result, nil
}
