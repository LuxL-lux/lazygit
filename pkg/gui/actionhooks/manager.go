package actionhooks

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
)

const (
	envContextKey = "LAZYGIT_ACTION_CONTEXT"
	envKeyLabel   = "LAZYGIT_ACTION_KEY"
	envPhase      = "LAZYGIT_ACTION_PHASE"

	phaseBefore = "before"
	phaseAfter  = "after"
)

// Manager coordinates execution of user-defined action hooks.
type Manager struct {
	cfgProvider func() *config.UserConfig
	osCommand   *oscommands.OSCommand
}

func NewManager(cfgProvider func() *config.UserConfig, osCommand *oscommands.OSCommand) *Manager {
	return &Manager{cfgProvider: cfgProvider, osCommand: osCommand}
}

// Execution represents a set of hooks that have already had their "before"
// commands run and may later run their "after" commands.
type Execution struct {
	manager    *Manager
	hooks      []*config.ActionHook
	contextKey string
	keyLabel   string
}

// ExecuteBefore runs matching before-hooks and returns an Execution that can be
// used to trigger post-hooks once the action completes.
func (m *Manager) ExecuteBefore(contextKey, keyLabel string) (*Execution, error) {
	hooks := m.matchHooks(contextKey, keyLabel)
	if len(hooks) == 0 {
		return nil, nil
	}

	if err := m.runCommands(hooks, phaseBefore, contextKey, keyLabel); err != nil {
		return nil, err
	}

	return &Execution{manager: m, hooks: hooks, contextKey: contextKey, keyLabel: keyLabel}, nil
}

// ExecuteAfter runs all "after" hooks associated with this execution.
func (e *Execution) ExecuteAfter() error {
	if e == nil {
		return nil
	}

	return e.manager.runCommands(e.hooks, phaseAfter, e.contextKey, e.keyLabel)
}

func (m *Manager) matchHooks(contextKey, keyLabel string) []*config.ActionHook {
	cfg := m.cfgProvider()
	if cfg == nil || len(cfg.ActionHooks) == 0 {
		return nil
	}

	keyLabel = strings.TrimSpace(strings.ToLower(keyLabel))
	if keyLabel == "" {
		return nil
	}

	matches := []*config.ActionHook{}

	for i := range cfg.ActionHooks {
		hook := &cfg.ActionHooks[i]
		hookKey := strings.TrimSpace(strings.ToLower(hook.Key))
		if hookKey == "" || hookKey != keyLabel {
			continue
		}

		hookContext := strings.TrimSpace(strings.ToLower(hook.Context))
		if hookContext != "" && !strings.EqualFold(hookContext, contextKey) {
			continue
		}

		if strings.TrimSpace(hook.Before) == "" && strings.TrimSpace(hook.After) == "" {
			continue
		}

		matches = append(matches, hook)
	}

	return matches
}

func (m *Manager) runCommands(hooks []*config.ActionHook, phase, contextKey, keyLabel string) error {
	shellFunctionsFile := ""
	cfg := m.cfgProvider()
	if cfg != nil {
		shellFunctionsFile = cfg.OS.ShellFunctionsFile
	}

	for _, hook := range hooks {
		command := ""
		switch phase {
		case phaseBefore:
			command = hook.Before
		case phaseAfter:
			command = hook.After
		}

		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}

		cmdObj := m.osCommand.Cmd.NewShell(command, shellFunctionsFile)
		if !hook.LogOutput {
			cmdObj.DontLog()
		}
		envs := []string{
			fmt.Sprintf("%s=%s", envContextKey, contextKey),
			fmt.Sprintf("%s=%s", envKeyLabel, keyLabel),
			fmt.Sprintf("%s=%s", envPhase, phase),
		}
		cmdObj.AddEnvVars(envs...)

		output, err := cmdObj.RunWithOutput()
		if err != nil {
			trimmed := strings.TrimSpace(output)
			if trimmed != "" {
				return fmt.Errorf("action hook (%s) failed: %s", phase, trimmed)
			}
			return fmt.Errorf("action hook (%s) failed: %w", phase, err)
		}

		if hook.AbortOnSuccess {
			message := strings.TrimSpace(hook.AbortMessage)
			if message == "" {
				message = DefaultAbortMessage
			}
			return AbortError{Message: message}
		}
	}

	return nil
}

const DefaultAbortMessage = "Action aborted by hook"

type AbortError struct {
	Message string
}

func (e AbortError) Error() string {
	return e.Message
}
