package cmdutil

import (
	"net/http"

	"github.com/abdfnx/gh/context"
	"github.com/abdfnx/gh/core/config"
	"github.com/abdfnx/gh/core/ghrepo"
	"github.com/abdfnx/gh/pkg/iostreams"
)

type Browser interface {
	Browse(string) error
}

type Factory struct {
	IOStreams *iostreams.IOStreams
	Browser   Browser

	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)
	Remotes    func() (context.Remotes, error)
	Config     func() (config.Config, error)
	Branch     func() (string, error)

	Executable string
}
