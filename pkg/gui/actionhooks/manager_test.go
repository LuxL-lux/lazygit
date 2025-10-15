package actionhooks

import (
	"errors"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
)

func TestExecuteBeforeAndAfter(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", After: "echo after"}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool {
		if cmdObj.ShouldLog() {
			t.Fatalf("before hook should not log output")
		}
		if !strings.Contains(cmdObj.ToString(), "echo before") {
			t.Fatalf("unexpected command: %s", cmdObj.ToString())
		}
		assertEnvContains(t, cmdObj.GetEnvVars(), "LAZYGIT_ACTION_PHASE=before")
		return true
	}, "", nil)
	runner.ExpectFunc("after hook", func(cmdObj *oscommands.CmdObj) bool {
		if cmdObj.ShouldLog() {
			t.Fatalf("after hook should not log output")
		}
		if !strings.Contains(cmdObj.ToString(), "echo after") {
			t.Fatalf("unexpected command: %s", cmdObj.ToString())
		}
		assertEnvContains(t, cmdObj.GetEnvVars(), "LAZYGIT_ACTION_PHASE=after")
		return true
	}, "", nil)

	manager := newManagerWithRunner(t, hooks, runner)
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec == nil {
		t.Fatalf("expected execution handle")
	}

	if err := exec.ExecuteAfter(); err != nil {
		t.Fatalf("unexpected ExecuteAfter error: %v", err)
	}

	runner.CheckForMissingCalls()
}

func TestExecuteBeforeNoMatch(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before"}}
	manager := newManagerWithRunner(t, hooks, oscommands.NewFakeRunner(t))

	exec, err := manager.ExecuteBefore("branches", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec != nil {
		t.Fatalf("expected no execution when hooks do not match")
	}
}

func TestExecuteBeforePropagatesError(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before"}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool { return true }, "", errors.New("boom"))

	manager := newManagerWithRunner(t, hooks, runner)
	exec, err := manager.ExecuteBefore("files", "c")
	if err == nil {
		t.Fatalf("expected error")
	}
	if exec != nil {
		t.Fatalf("expected nil execution on error")
	}

	runner.CheckForMissingCalls()
}

func TestExecuteBeforeAbortOnSuccess(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", AbortOnSuccess: true, AbortMessage: "stop it"}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool { return true }, "", nil)

	manager := newManagerWithRunner(t, hooks, runner)
	exec, err := manager.ExecuteBefore("files", "c")
	if err == nil {
		t.Fatalf("expected abort error")
	}
	abortErr, ok := err.(AbortError)
	if !ok {
		t.Fatalf("expected AbortError, got %T", err)
	}
	if abortErr.Message != "stop it" {
		t.Fatalf("unexpected abort message: %s", abortErr.Message)
	}
	if exec != nil {
		t.Fatalf("expected nil execution on abort")
	}
}

func TestExecuteBeforeAbortOnSuccessDefaultMessage(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", AbortOnSuccess: true}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool { return true }, "", nil)

	manager := newManagerWithRunner(t, hooks, runner)
	_, err := manager.ExecuteBefore("files", "c")
	if err == nil {
		t.Fatalf("expected abort error")
	}
	abortErr, ok := err.(AbortError)
	if !ok {
		t.Fatalf("expected AbortError, got %T", err)
	}
	if abortErr.Message != DefaultAbortMessage {
		t.Fatalf("unexpected abort message: %s", abortErr.Message)
	}
}

func TestExecuteAfterAbortOnSuccess(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", After: "echo after", AbortOnSuccess: true}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("after hook", func(cmdObj *oscommands.CmdObj) bool {
		return strings.Contains(cmdObj.ToString(), "echo after")
	}, "", nil)

	manager := newManagerWithRunner(t, hooks, runner)
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = exec.ExecuteAfter()
	if err == nil {
		t.Fatalf("expected abort error")
	}
	abortErr, ok := err.(AbortError)
	if !ok {
		t.Fatalf("expected AbortError, got %T", err)
	}
	if abortErr.Message != DefaultAbortMessage {
		t.Fatalf("unexpected abort message: %s", abortErr.Message)
	}

	runner.CheckForMissingCalls()
}

func TestExecuteBeforeLogOutput(t *testing.T) {
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", LogOutput: true}}
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool {
		if !cmdObj.ShouldLog() {
			t.Fatalf("expected command to log output")
		}
		return true
	}, "", nil)

	manager := newManagerWithRunner(t, hooks, runner)
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec == nil {
		t.Fatalf("expected execution handle")
	}

	runner.CheckForMissingCalls()
}

func newManagerWithRunner(t *testing.T, hooks []config.ActionHook, runner *oscommands.FakeCmdObjRunner) *Manager {
	osCmd := oscommands.NewDummyOSCommandWithRunner(runner)
	return NewManager(func() *config.UserConfig {
		return &config.UserConfig{ActionHooks: hooks}
	}, osCmd)
}

func assertEnvContains(t *testing.T, env []string, expected string) {
	t.Helper()
	for _, entry := range env {
		if entry == expected {
			return
		}
	}
	t.Fatalf("expected env to contain %s, got %v", expected, env)
}
