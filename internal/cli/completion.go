package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for Platform Foundry CLI.

To load completions:

Bash:
  $ pf completion bash > /etc/bash_completion.d/pf
  # or for local user
  $ pf completion bash > ~/.bash_completion

Zsh:
  $ pf completion zsh > ~/.zsh/completion/_pf
  # or add to .zshrc:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

Fish:
  $ pf completion fish > ~/.config/fish/completions/pf.fish

PowerShell:
  PS> pf completion powershell | Out-String | Invoke-Expression
  # or add to profile:
  PS> pf completion powershell >> $PROFILE
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE:                  runCompletion,
}

func init() {
	// completion command has no additional flags
}

func runCompletion(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	}

	return nil
}
