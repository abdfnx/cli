package search

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"

	searchReposCmd "github.com/cli/cli/v2/pkg/cmd/search/repos"
)

func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <command>",
		Short: "search",
		Long:  `search.`,
		Example: heredoc.Doc(`
			$ gh search repos
		`),
		Annotations: map[string]string{
			"IsCore": "true",
		},
	}

	cmd.AddCommand(searchReposCmd.NewCmdRepos(f, nil))

	return cmd
}
