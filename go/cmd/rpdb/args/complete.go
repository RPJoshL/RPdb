package args

import (
	"fmt"
	"strings"

	"github.com/RPJoshL/RPdb/v4/go/cmd/rpdb/args/completions"
)

type Completion struct {
	// Shell for which the completion should be printed
	Shell string `cli:"--shell,-shell,,1" completion:"GetShells"`
}

func (c *Completion) GetShells(cli *Cli, input string) (rtc []string) {
	return []string{"bash"}
}

func (c *Completion) SetShell(value string) string {
	if strings.ToLower(value) != "bash" {
		return "Only the shell 'Bash' is supported currently"
	}

	c.Shell = value
	return ""
}

func (c *Completion) SetCompletion(cli *Cli) string {
	file, err := completions.Bash.ReadFile("shells/bash.sh")
	if err != nil {
		return err.Error()
	}

	fmt.Println(string(file))
	return ""
}
