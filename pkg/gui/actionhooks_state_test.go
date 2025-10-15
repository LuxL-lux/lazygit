package gui

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/actionhooks"
)

func setupActionHookManager(t *testing.T, hooks []config.ActionHook, afterRan *bool) (*actionhooks.Manager, *oscommands.FakeCmdObjRunner) {
	runner := oscommands.NewFakeRunner(t)
	for _, hook := range hooks {
		hook := hook
		if strings.TrimSpace(hook.Before) != "" {
			expectedBefore := strings.TrimSpace(hook.Before)
			runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool {
				return strings.Contains(cmdObj.ToString(), expectedBefore)
			}, "", nil)
		}
		if strings.TrimSpace(hook.After) != "" {
			expectedAfter := strings.TrimSpace(hook.After)
			runner.ExpectFunc("after hook", func(cmdObj *oscommands.CmdObj) bool {
				if !strings.Contains(cmdObj.ToString(), expectedAfter) {
					return false
				}
				if afterRan != nil {
					*afterRan = true
				}
				return true
			}, "", nil)
		}
	}

	osCmd := oscommands.NewDummyOSCommandWithRunner(runner)
	manager := actionhooks.NewManager(func() *config.UserConfig {
		return &config.UserConfig{ActionHooks: hooks}
	}, osCmd)

	return manager, runner
}

func TestActionHookImmediateCompletion(t *testing.T) {
	afterRan := false
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", After: "echo after"}}
	manager, runner := setupActionHookManager(t, hooks, &afterRan)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	if err := gui.finalizeActionHookExecution(); err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}

	runner.CheckForMissingCalls()

	if !afterRan {
		t.Fatalf("expected after hook to run")
	}
}

func TestActionHookDeferredCompletion(t *testing.T) {
	afterRan := false
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", After: "echo after"}}
	manager, runner := setupActionHookManager(t, hooks, &afterRan)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	completion := gui.registerActionHookCompletion()
	if completion == nil {
		t.Fatalf("expected completion callback")
	}

	if err := gui.finalizeActionHookExecution(); err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}

	if afterRan {
		t.Fatalf("after hook should not run before completion")
	}

	if err := completion(true); err != nil {
		t.Fatalf("unexpected completion error: %v", err)
	}

	runner.CheckForMissingCalls()

	if !afterRan {
		t.Fatalf("expected after hook to run after completion")
	}
}

func TestActionHookMultipleDeferredCompletion(t *testing.T) {
	afterRan := false
	hooks := []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", After: "echo after"}}
	manager, runner := setupActionHookManager(t, hooks, &afterRan)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	completion1 := gui.registerActionHookCompletion()
	completion2 := gui.registerActionHookCompletion()
	if completion1 == nil || completion2 == nil {
		t.Fatalf("expected completion callbacks")
	}

	if err := gui.finalizeActionHookExecution(); err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}

	if err := completion1(true); err != nil {
		t.Fatalf("unexpected completion1 error: %v", err)
	}

	if afterRan {
		t.Fatalf("after hook should not run until all completions finished")
	}

	if err := completion2(true); err != nil {
		t.Fatalf("unexpected completion2 error: %v", err)
	}

	runner.CheckForMissingCalls()

	if !afterRan {
		t.Fatalf("expected after hook to run after all completions")
	}
}

func TestActionHookAbortSkipsAfter(t *testing.T) {
	beforeRan := false
	afterRan := false
	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook", func(cmdObj *oscommands.CmdObj) bool {
		beforeRan = true
		return strings.Contains(cmdObj.ToString(), "echo before")
	}, "", nil)
	runner.ExpectFunc("after hook", func(cmdObj *oscommands.CmdObj) bool {
		afterRan = true
		return strings.Contains(cmdObj.ToString(), "echo after")
	}, "", nil)

	osCmd := oscommands.NewDummyOSCommandWithRunner(runner)
	manager := actionhooks.NewManager(func() *config.UserConfig {
		return &config.UserConfig{ActionHooks: []config.ActionHook{{Context: "files", Key: "c", Before: "echo before", After: "echo after"}}}
	}, osCmd)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	completion := gui.registerActionHookCompletion()
	if completion == nil {
		t.Fatalf("expected completion callback")
	}

	gui.abortActionHookExecution()

	if err := completion(true); err != nil {
		t.Fatalf("unexpected completion error: %v", err)
	}

	if !beforeRan {
		t.Fatalf("expected before hook to run")
	}

	if afterRan {
		t.Fatalf("after hook should not run after abort")
	}
}

func TestActionHookAfterAbortOnSuccess(t *testing.T) {
	afterRan := false
	hooks := []config.ActionHook{{Context: "files", Key: "c", After: "echo after", AbortOnSuccess: true}}
	manager, runner := setupActionHookManager(t, hooks, &afterRan)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	err = gui.finalizeActionHookExecution()
	if err == nil {
		t.Fatalf("expected abort error")
	}
	abortErr, ok := err.(actionhooks.AbortError)
	if !ok {
		t.Fatalf("expected AbortError, got %T", err)
	}
	if abortErr.Message != actionhooks.DefaultAbortMessage {
		t.Fatalf("unexpected abort message: %s", abortErr.Message)
	}

	if !afterRan {
		t.Fatalf("expected after hook to run before aborting")
	}

	runner.CheckForMissingCalls()
}

func TestActionHookMultipleHooks(t *testing.T) {
	afterCount := 0
	hooks := []config.ActionHook{
		{Context: "files", Key: "c", Before: "echo before1", After: "echo after1"},
		{Context: "files", Key: "c", Before: "echo before2", After: "echo after2"},
	}

	runner := oscommands.NewFakeRunner(t)
	runner.ExpectFunc("before hook 1", func(cmdObj *oscommands.CmdObj) bool {
		return strings.Contains(cmdObj.ToString(), "echo before1")
	}, "", nil)
	runner.ExpectFunc("after hook 1", func(cmdObj *oscommands.CmdObj) bool {
		if strings.Contains(cmdObj.ToString(), "echo after1") {
			afterCount++
			return true
		}
		return false
	}, "", nil)
	runner.ExpectFunc("before hook 2", func(cmdObj *oscommands.CmdObj) bool {
		return strings.Contains(cmdObj.ToString(), "echo before2")
	}, "", nil)
	runner.ExpectFunc("after hook 2", func(cmdObj *oscommands.CmdObj) bool {
		if strings.Contains(cmdObj.ToString(), "echo after2") {
			afterCount++
			return true
		}
		return false
	}, "", nil)

	osCmd := oscommands.NewDummyOSCommandWithRunner(runner)
	manager := actionhooks.NewManager(func() *config.UserConfig {
		return &config.UserConfig{ActionHooks: hooks}
	}, osCmd)

	gui := &Gui{ActionHookManager: manager}
	exec, err := manager.ExecuteBefore("files", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gui.startActionHookExecution(exec)

	if err := gui.finalizeActionHookExecution(); err != nil {
		t.Fatalf("unexpected finalize error: %v", err)
	}

	runner.CheckForMissingCalls()

	if afterCount != 2 {
		t.Fatalf("expected both after hooks to run, got %d", afterCount)
	}
}
