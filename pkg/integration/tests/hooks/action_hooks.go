package hooks

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ActionHooks = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Verify action hooks run before and after committing",
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().ActionHooks = []config.ActionHook{
			{
				Context: "files",
				Key:     cfg.GetUserConfig().Keybinding.Files.CommitChanges,
				Before:  "mkdir -p .git/lazygit-actionhooks && echo \"[TEST HOOK] commit starting\" >> .git/lazygit-actionhooks/action-hook.log",
				After:   "mkdir -p .git/lazygit-actionhooks && echo \"[TEST HOOK] commit finished\" >> .git/lazygit-actionhooks/action-hook.log",
			},
		}
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFile("file.txt", "hello world\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Lines(
				Contains("?? file.txt").IsSelected(),
			).
			PressPrimaryAction().
			Lines(
				Contains("A  file.txt").IsSelected(),
			)

		t.Views().Files().Press(keys.Files.CommitChanges)

		commitMessage := "initial commit"
		t.ExpectPopup().CommitMessagePanel().Type(commitMessage).Confirm()

		t.Views().Commits().Lines(Contains(commitMessage))

		t.FileSystem().FileContent(".git/lazygit-actionhooks/action-hook.log", Equals("[TEST HOOK] commit starting\n[TEST HOOK] commit finished\n"))
	},
})
