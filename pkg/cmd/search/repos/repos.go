package repos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/pkg/cmd/search/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/export"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/search"
	"github.com/cli/cli/v2/pkg/text"
	"github.com/cli/cli/v2/utils"
	"github.com/spf13/cobra"
)

type ReposOptions struct {
	Browser      cmdutil.Browser
	Config       func() (config.Config, error)
	Exporter     cmdutil.Exporter
	GoTemplate   string
	HttpClient   func() (*http.Client, error)
	IO           *iostreams.IOStreams
	JqExpression string
	Query        search.Query
	WebMode      bool
}

func NewCmdRepos(f *cmdutil.Factory, runF func(*ReposOptions) error) *cobra.Command {
	opts := &ReposOptions{
		Browser:    f.Browser,
		Config:     f.Config,
		HttpClient: f.HttpClient,
		IO:         f.IOStreams,
		Query:      NewSearchQuery(),
	}

	cmd := &cobra.Command{
		Use:   "repos <query>",
		Short: "",
		Long:  "",
		RunE: func(c *cobra.Command, args []string) error {
			opts.Query.Keywords = args
			err := cmdutil.MutuallyExclusive("expected exactly one of `--jq`, `--template`, or `--web`",
				opts.GoTemplate != "",
				opts.JqExpression != "",
				opts.WebMode)
			if err != nil {
				return err
			}
			if opts.Query.Limit < 1 || opts.Query.Limit > 1000 {
				return cmdutil.FlagErrorf("`--limit` must be between 1 and 1000")
			}
			if runF != nil {
				return runF(opts)
			}
			return reposRun(opts)
		},
	}

	// Output flags
	cmd.Flags().StringVarP(&opts.GoTemplate, "template", "t", "", "Format JSON output using a Go template")
	cmd.Flags().StringVarP(&opts.JqExpression, "jq", "q", "", "Filter JSON output using a jq `expression`")
	cmd.Flags().BoolVarP(&opts.WebMode, "web", "w", false, "Open the query in the web browser")

	// Query parameter flags
	cmd.Flags().IntVarP(&opts.Query.Limit, "limit", "L", 30, "Maximum number of repositories to fetch")
	cmd.Flags().Var(opts.Query.Order, "order", "Order of repositories returned, ignored unless `--sort` is specified")
	cmd.Flags().Var(opts.Query.Sort, "sort", "Sorts the repositories by stars, forks, help-wanted-issues, or updated")

	// TODO: Negative qualifier flags
	// Query qualifier flags
	cmd.Flags().Var(opts.Query.Qualifiers["Archived"], "archived", "archived")
	cmd.Flags().Var(opts.Query.Qualifiers["Created"], "created", "created at")
	cmd.Flags().Var(opts.Query.Qualifiers["Followers"], "followers", "followers")
	cmd.Flags().Var(opts.Query.Qualifiers["Fork"], "fork", "forks")
	cmd.Flags().Var(opts.Query.Qualifiers["Forks"], "forks", "forks")
	cmd.Flags().Var(opts.Query.Qualifiers["GoodFirstIssues"], "good-first-issues", "good-first-issues")
	cmd.Flags().Var(opts.Query.Qualifiers["HelpWantedIssues"], "help-wanted-issues", "help-wanted-issues")
	cmd.Flags().Var(opts.Query.Qualifiers["In"], "in", "in")
	cmd.Flags().Var(opts.Query.Qualifiers["Language"], "language", "language")
	cmd.Flags().Var(opts.Query.Qualifiers["License"], "license", "license")
	cmd.Flags().Var(opts.Query.Qualifiers["Mirror"], "mirror", "mirror")
	cmd.Flags().Var(opts.Query.Qualifiers["Org"], "org", "org")
	cmd.Flags().Var(opts.Query.Qualifiers["Pushed"], "pushed", "pushed at")
	cmd.Flags().Var(opts.Query.Qualifiers["Repo"], "repo", "repo")
	cmd.Flags().Var(opts.Query.Qualifiers["Size"], "size", "size")
	cmd.Flags().Var(opts.Query.Qualifiers["Stars"], "stars", "stars")
	cmd.Flags().Var(opts.Query.Qualifiers["Topic"], "topic", "topic")
	cmd.Flags().Var(opts.Query.Qualifiers["Topics"], "topics", "number topics")
	cmd.Flags().Var(opts.Query.Qualifiers["User"], "user", "user")
	cmd.Flags().Var(opts.Query.Qualifiers["Visibility"], "is", "is")

	return cmd
}

func reposRun(opts *ReposOptions) error {
	io := opts.IO
	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	host, err := cfg.DefaultHost()
	if err != nil {
		return err
	}
	client, err := opts.HttpClient()
	if err != nil {
		return err
	}
	searcher := shared.NewSearcher(host, client)
	if opts.WebMode {
		url := searcher.URL(opts.Query)
		if io.IsStdoutTTY() {
			fmt.Fprintf(io.ErrOut, "Opening %s in your browser.\n", utils.DisplayURL(url))
		}
		return opts.Browser.Browse(url)
	}
	opts.IO.StartProgressIndicator()
	result, err := searcher.Search(opts.Query)
	opts.IO.StopProgressIndicator()
	if err != nil {
		return err
	}
	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	} else {
		fmt.Fprintf(opts.IO.ErrOut, "failed to start pager: %v\n", err)
	}
	if opts.JqExpression != "" {
		j, err := json.Marshal(result.Items)
		if err != nil {
			return err
		}
		err = export.FilterJSON(io.Out, bytes.NewReader(j), opts.JqExpression)
		if err != nil {
			return err
		}
	} else if opts.GoTemplate != "" {
		t := export.NewTemplate(opts.IO, opts.GoTemplate)
		j, err := json.Marshal(result.Items)
		if err != nil {
			return err
		}
		err = t.Execute(bytes.NewReader(j))
		if err != nil {
			return err
		}
	} else {
		err := displayResults(opts.IO, result)
		if err != nil {
			return err
		}
	}
	return nil
}

func displayResults(io *iostreams.IOStreams, results search.Result) error {
	cs := io.ColorScheme()
	tp := utils.NewTablePrinter(io)
	for _, repo := range results.Items {
		var tags []string
		private, _ := repo["private"].(bool)
		fork, _ := repo["fork"].(bool)
		archived, _ := repo["archived"].(bool)
		if private {
			tags = append(tags, "private")
		} else {
			tags = append(tags, "public")
		}
		if fork {
			tags = append(tags, "fork")
		}
		if archived {
			tags = append(tags, "archived")
		}
		info := strings.Join(tags, ", ")
		infoColor := cs.Gray
		if private {
			infoColor = cs.Yellow
		}

		updatedAt, _ := repo["updated_at"].(string)
		tp.AddField(repo["full_name"].(string), nil, cs.Bold)
		description, _ := repo["description"].(string)
		tp.AddField(text.ReplaceExcessiveWhitespace(description), nil, nil)
		tp.AddField(info, nil, infoColor)
		if tp.IsTTY() {
			t, _ := time.Parse(time.RFC3339, updatedAt)
			tp.AddField(t.Format(time.RFC822), nil, cs.Gray)
		} else {
			tp.AddField(updatedAt, nil, nil)
		}
		tp.EndRow()
	}

	if io.IsStdoutTTY() {
		header := "No repositories matched your search\n"
		if len(results.Items) > 0 {
			header = fmt.Sprintf("Showing %d of %d repositories\n\n", len(results.Items), results.TotalCount)
		}
		fmt.Fprintf(io.Out, "\n%s", header)
	}

	return tp.Render()
}

func NewSearchQuery() search.Query {
	return search.Query{
		Kind:  "repositories",
		Order: shared.NewParameter("order", "string", "desc", shared.OptsValidator([]string{"asc", "desc"})),
		Sort:  shared.NewParameter("sort", "string", "best match", shared.OptsValidator([]string{"forks", "help-wanted-issues", "stars", "updated"})),
		Qualifiers: search.Qualifiers{
			"Archived":         shared.NewQualifier("archived", "bool", "", shared.BoolValidator()),
			"Created":          shared.NewQualifier("created", "string", "", shared.DateValidator()),
			"Followers":        shared.NewQualifier("followers", "string", "", shared.RangeValidator()),
			"Fork":             shared.NewQualifier("fork", "string", "false", shared.OptsValidator([]string{"false", "true", "only"})),
			"Forks":            shared.NewQualifier("forks", "string", "", shared.RangeValidator()),
			"GoodFirstIssues":  shared.NewQualifier("good-first-issues", "string", "", shared.RangeValidator()),
			"HelpWantedIssues": shared.NewQualifier("help-wanted-issues", "string", "", shared.RangeValidator()),
			"In":               shared.NewQualifier("in", "string", "name,descripton", shared.MultiOptsValidator([]string{"name", "description", "readme"})),
			"Language":         shared.NewQualifier("language", "string", "", nil),
			"License":          shared.NewQualifier("license", "string", "", nil),
			"Mirror":           shared.NewQualifier("mirror", "bool", "", shared.BoolValidator()),
			"Org":              shared.NewQualifier("org", "string", "", nil),
			"Pushed":           shared.NewQualifier("pushed", "string", "", shared.DateValidator()),
			"Repo":             shared.NewQualifier("repo", "string", "", nil),
			"Size":             shared.NewQualifier("size", "string", "", shared.RangeValidator()),
			"Stars":            shared.NewQualifier("stars", "string", "", shared.RangeValidator()),
			"Topic":            shared.NewQualifier("topic", "string", "", nil),
			"Topics":           shared.NewQualifier("topics", "string", "", shared.RangeValidator()),
			"User":             shared.NewQualifier("user", "string", "", nil),
			"Visibility":       shared.NewQualifier("is", "string", "public", shared.OptsValidator([]string{"public", "private"})),
		},
	}
}
