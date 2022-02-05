package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/pkg/search"
)

type searcher struct {
	host   string
	client *http.Client
}

func NewSearcher(host string, client *http.Client) *searcher {
	return &searcher{
		host:   host,
		client: client,
	}
}

func (s *searcher) Search(query search.Query) (search.Result, error) {
	result := search.Result{}
	path := fmt.Sprintf("https://api.%s/search/%s", s.host, query.Kind)
	queryString := url.Values{}
	q := strings.Builder{}
	q.WriteString(strings.Join(query.Keywords, " "))
	for k, v := range query.Qualifiers.ListSet() {
		q.WriteString(fmt.Sprintf(" %s:%s", k, v))
	}
	queryString.Set("q", q.String())
	if query.Order.IsSet() {
		queryString.Set(query.Order.Key(), query.Order.String())
	}
	if query.Sort.IsSet() {
		queryString.Set(query.Sort.Key(), query.Sort.String())
	}
	if query.Limit > 100 {
		queryString.Set("per_page", "100")
	} else {
		queryString.Set("per_page", strconv.Itoa(query.Limit))
	}
	pages := (query.Limit / 100) + 1
	for i := 1; i <= pages; i++ {
		queryString.Set("page", strconv.Itoa(i))
		url := fmt.Sprintf("%s?%s", path, queryString.Encode())
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return result, err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		resp, err := s.client.Do(req)
		if err != nil {
			return result, err
		}
		defer resp.Body.Close()
		success := resp.StatusCode >= 200 && resp.StatusCode < 300
		if !success {
			//TODO: Have specialized error handling code
			//TODO: Handle search failures due to too long of query
			//TODO: Handle too many results
			//TODO: Handle validation failure
			return result, api.HandleHTTPError(resp)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, err
		}
		pageResult := search.Result{}
		err = json.Unmarshal(b, &pageResult)
		if err != nil {
			return result, err
		}
		result.IncompleteResults = pageResult.IncompleteResults
		result.TotalCount = pageResult.TotalCount
		result.Items = append(result.Items, pageResult.Items...)
	}
	return result, nil
}

func (s *searcher) URL(query search.Query) string {
	path := fmt.Sprintf("https://%s/search", s.host)
	queryString := url.Values{}
	queryString.Set("type", query.Kind)
	if query.Order.IsSet() {
		queryString.Set(query.Order.Key(), query.Order.String())
	}
	if query.Sort.IsSet() {
		queryString.Set(query.Sort.Key(), query.Sort.String())
	}
	q := strings.Builder{}
	q.WriteString(strings.Join(query.Keywords, " "))
	for k, v := range query.Qualifiers.ListSet() {
		q.WriteString(fmt.Sprintf(" %s:%s", k, v))
	}
	queryString.Add("q", q.String())
	url := fmt.Sprintf("%s?%s", path, queryString.Encode())
	return url
}
