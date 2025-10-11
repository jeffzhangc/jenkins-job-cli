package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	// 创建自定义的 completion 命令
	var completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for jj.

This command generates completion scripts for various shells. After generating,
you need to source the file or add it to your shell's startup configuration.

Installation Instructions:

Bash (Linux/macOS):
  # Generate and save the completion script
  jj completion bash > ~/.jj-completion.bash
  
  # Add to your bashrc
  echo "source ~/.jj-completion.bash" >> ~/.bashrc
  
  # Reload your current shell
  source ~/.bashrc

  # Alternative: system-wide installation (Linux)
  sudo jj completion bash > /etc/bash_completion.d/jj

Bash (macOS with Homebrew):
  # If installed via Homebrew, completion might be automatically installed
  brew install bash-completion

Zsh:
  # Generate and save the completion script
  jj completion zsh > ~/.jj-completion.zsh
  
  # Add to your zshrc
  echo "source ~/.jj-completion.zsh" >> ~/.zshrc
  
  # Reload your current shell
  source ~/.zshrc

  # Alternative: use the function path
  jj completion zsh > "${fpath[1]}/_jj"

Fish:
  # Generate and save the completion script
  jj completion fish > ~/.config/fish/completions/jj.fish
  
  # Reload your current shell
  exec fish

PowerShell:
  # Generate and execute immediately
  jj completion powershell | Out-String | Invoke-Expression
  
  # To persist across sessions, add to your profile
  jj completion powershell > $PROFILE
  
  # Or create a separate file and source it
  jj completion powershell > ~/.jj-completion.ps1
  Add-Content $PROFILE "~/.jj-completion.ps1"

After installation, restart your shell or source the configuration file.
You can then use tab completion for hosts, groups, and commands.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
		},
	}

	rootCmd.AddCommand(completionCmd)
}
