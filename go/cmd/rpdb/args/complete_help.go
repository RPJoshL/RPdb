package args

func (c *Completion) Help() string {
	return `
completion {bash}

Output shell completion code for the specified shell (only bash is supported at the moment). The shell code must be evaluated
to provide interactive completion of RPdb commands. This can be done by sourcing it from the .bash_profile.

Examples:

# Installing bash completion on Linux
## If bash-completion is not installed on Linux, install the 'bash-completion' package
## via your distribution's package manager.
## Load the completion code for bash into the current shell
  source <(RPdb-go completion bash)
## Write bash completion code to a file and source it from .bash_profile
  RPdb-go completion bash > ~/.config/RPJosh/RPdb-go/completion.bash.inc
  printf "
  # RPdb shell completion
  source '$HOME/.config/RPJosh/RPdb-go/completion.bash.inc'
  " >> $HOME/.bashrc
## Or load it every time dynamically on shell startup (this could be slow!)
  echo -e '\nsource <(RPdb-go completion bash)' >> ~/.bashrc
`
}
