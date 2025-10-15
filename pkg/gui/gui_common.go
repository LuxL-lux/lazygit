package gui

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// hacking this by including the gui struct for now until we split more things out
type guiCommon struct {
	gui *Gui
	types.IPopupHandler
}

var _ types.IGuiCommon = &guiCommon{}

func (self *guiCommon) LogAction(msg string) {
	self.gui.LogAction(msg)
}

func (self *guiCommon) LogCommand(cmdStr string, isCommandLine bool) {
	self.gui.LogCommand(cmdStr, isCommandLine)
}

func (self *guiCommon) Refresh(opts types.RefreshOptions) {
	self.gui.helpers.Refresh.Refresh(opts)
}

func (self *guiCommon) PostRefreshUpdate(context types.Context) {
	self.gui.postRefreshUpdate(context)
}

func (self *guiCommon) RunSubprocessAndRefresh(cmdObj *oscommands.CmdObj) error {
	completion := self.gui.registerActionHookCompletion()
	err := self.gui.runSubprocessWithSuspenseAndRefresh(cmdObj)
	if completion != nil {
		if err != nil {
			_ = completion(false)
		} else if compErr := completion(true); compErr != nil {
			return compErr
		}
	}
	return err
}

func (self *guiCommon) RunSubprocess(cmdObj *oscommands.CmdObj) (bool, error) {
	completion := self.gui.registerActionHookCompletion()
	success, err := self.gui.runSubprocessWithSuspense(cmdObj)
	if completion != nil {
		if err != nil || !success {
			_ = completion(false)
		} else if compErr := completion(true); compErr != nil {
			return success, compErr
		}
	}
	return success, err
}

func (self *guiCommon) Suspend() error {
	return self.gui.suspend()
}

func (self *guiCommon) Resume() error {
	return self.gui.resume()
}

func (self *guiCommon) Context() types.IContextMgr {
	return self.gui.State.ContextMgr
}

func (self *guiCommon) ContextForKey(key types.ContextKey) types.Context {
	return self.gui.State.ContextMgr.ContextForKey(key)
}

func (self *guiCommon) GetAppState() *config.AppState {
	return self.gui.Config.GetAppState()
}

func (self *guiCommon) SaveAppState() error {
	return self.gui.Config.SaveAppState()
}

func (self *guiCommon) SaveAppStateAndLogError() {
	if err := self.gui.Config.SaveAppState(); err != nil {
		self.gui.Log.Errorf("error when saving app state: %v", err)
	}
}

func (self *guiCommon) GetConfig() config.AppConfigurer {
	return self.gui.Config
}

func (self *guiCommon) ResetViewOrigin(view *gocui.View) {
	self.gui.resetViewOrigin(view)
}

func (self *guiCommon) SetViewContent(view *gocui.View, content string) {
	self.gui.setViewContent(view, content)
}

func (self *guiCommon) Render() {
	self.gui.render()
}

func (self *guiCommon) Views() types.Views {
	return self.gui.Views
}

func (self *guiCommon) Git() *commands.GitCommand {
	return self.gui.git
}

func (self *guiCommon) OS() *oscommands.OSCommand {
	return self.gui.os
}

func (self *guiCommon) Modes() *types.Modes {
	return self.gui.State.Modes
}

func (self *guiCommon) Model() *types.Model {
	return self.gui.State.Model
}

func (self *guiCommon) Mutexes() *types.Mutexes {
	return &self.gui.Mutexes
}

func (self *guiCommon) GocuiGui() *gocui.Gui {
	return self.gui.g
}

func (self *guiCommon) OnUIThread(f func() error) {
	self.gui.onUIThread(f)
}

func (self *guiCommon) OnWorker(f func(gocui.Task) error) {
	self.gui.onWorker(f)
}

func (self *guiCommon) RenderToMainViews(opts types.RefreshMainOpts) {
	self.gui.refreshMainViews(opts)
}

func (self *guiCommon) MainViewPairs() types.MainViewPairs {
	return types.MainViewPairs{
		Normal:         self.gui.normalMainContextPair(),
		Staging:        self.gui.stagingMainContextPair(),
		PatchBuilding:  self.gui.patchBuildingMainContextPair(),
		MergeConflicts: self.gui.mergingMainContextPair(),
	}
}

func (self *guiCommon) GetViewBufferManagerForView(view *gocui.View) *tasks.ViewBufferManager {
	return self.gui.getViewBufferManagerForView(view)
}

func (self *guiCommon) State() types.IStateAccessor {
	return self.gui.stateAccessor
}

func (self *guiCommon) KeybindingsOpts() types.KeybindingsOpts {
	return self.gui.keybindingOpts()
}

func (self *guiCommon) CallKeybindingHandler(binding *types.Binding) error {
	return self.gui.callKeybindingHandler(binding)
}

func (self *guiCommon) ResetKeybindings() error {
	return self.gui.resetKeybindings()
}

func (self *guiCommon) IsAnyModeActive() bool {
	return self.gui.helpers.Mode.IsAnyModeActive()
}

func (self *guiCommon) GetInitialKeybindingsWithCustomCommands() ([]*types.Binding, []*gocui.ViewMouseBinding) {
	return self.gui.GetInitialKeybindingsWithCustomCommands()
}

func (self *guiCommon) AfterLayout(f func() error) {
	self.gui.afterLayout(f)
}

func (self *guiCommon) RunningIntegrationTest() bool {
	return self.gui.integrationTest != nil
}

func (self *guiCommon) InDemo() bool {
	return self.gui.integrationTest != nil && self.gui.integrationTest.IsDemo()
}

func (self *guiCommon) WithInlineStatus(item types.HasUrn, operation types.ItemOperation, contextKey types.ContextKey, f func(gocui.Task) error) error {
	completion := self.gui.registerActionHookCompletion()
	wrapped := f
	if completion != nil {
		wrapped = func(task gocui.Task) error {
			err := f(task)
			if err != nil {
				_ = completion(false)
				return err
			}

			if err := completion(true); err != nil {
				return err
			}

			return nil
		}
	}

	self.gui.helpers.InlineStatus.WithInlineStatus(helpers.InlineStatusOpts{Item: item, Operation: operation, ContextKey: contextKey}, wrapped)
	return nil
}

func (self *guiCommon) RegisterActionHookCompletion() func(success bool) error {
	return self.gui.registerActionHookCompletion()
}
