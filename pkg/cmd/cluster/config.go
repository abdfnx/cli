package config

import (
	"fmt"
	"strings"

	"github.com/abdfnx/gh/core/config"
	cmdGet "github.com/abdfnx/gh/pkg/cmd/cluster/get"
	cmdSet "github.com/abdfnx/gh/pkg/cmd/cluster/set"
	"github.com/abdfnx/gh/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdConfig(f *cmdutil.Factory) *cobra.Command {
	longDoc := strings.Builder{}
	longDoc.WriteString("Display or change configuration settings for tran.\n\n")
	longDoc.WriteString("Current respected settings:\n")
	for _, co := range config.ConfigOptions() {
		longDoc.WriteString(fmt.Sprintf("- %s: %s", co.Key, co.Description))
		if co.DefaultValue != "" {
			longDoc.WriteString(fmt.Sprintf(" (default: %q)", co.DefaultValue))
		}

		longDoc.WriteRune('\n')
	}

	cmd := &cobra.Command{
		Use:   "cluster <command>",
		Short: "Manage configuration of github for tran.",
		Long:  longDoc.String(),
	}

	cmdutil.DisableAuthCheck(cmd)

	cmd.AddCommand(cmdGet.NewCmdConfigGet(f, nil))
	cmd.AddCommand(cmdSet.NewCmdConfigSet(f, nil))

	return cmd
}
