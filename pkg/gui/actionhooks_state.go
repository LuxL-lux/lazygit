package gui

import (
	"sync"

	"github.com/jesseduffield/lazygit/pkg/gui/actionhooks"
)

type actionHookState struct {
	mutex     sync.Mutex
	execution *actionhooks.Execution
	pending   int
	completed bool
}

func (gui *Gui) startActionHookExecution(execution *actionhooks.Execution) {
	gui.actionHookState.mutex.Lock()
	defer gui.actionHookState.mutex.Unlock()

	gui.actionHookState.execution = execution
	gui.actionHookState.pending = 0
	gui.actionHookState.completed = execution == nil
}

func (gui *Gui) registerActionHookCompletion() func(success bool) error {
	gui.actionHookState.mutex.Lock()
	defer gui.actionHookState.mutex.Unlock()

	state := &gui.actionHookState
	if state.execution == nil || state.completed {
		return nil
	}

	exec := state.execution
	state.pending++

	return func(success bool) error {
		gui.actionHookState.mutex.Lock()
		defer gui.actionHookState.mutex.Unlock()

		state := &gui.actionHookState
		if state.execution != exec || state.completed {
			return nil
		}

		state.pending--

		if !success {
			if state.pending == 0 {
				state.execution = nil
				state.completed = true
			}
			return nil
		}

		if state.pending == 0 {
			state.execution = nil
			state.completed = true
			return exec.ExecuteAfter()
		}
		return nil
	}
}

func (gui *Gui) finalizeActionHookExecution() error {
	gui.actionHookState.mutex.Lock()
	defer gui.actionHookState.mutex.Unlock()

	state := &gui.actionHookState
	if state.execution == nil || state.completed {
		return nil
	}

	if state.pending > 0 {
		return nil
	}

	exec := state.execution
	state.execution = nil
	state.completed = true

	return exec.ExecuteAfter()
}

func (gui *Gui) abortActionHookExecution() {
	gui.actionHookState.mutex.Lock()
	state := &gui.actionHookState
	state.execution = nil
	state.pending = 0
	state.completed = true
	gui.actionHookState.mutex.Unlock()
}
