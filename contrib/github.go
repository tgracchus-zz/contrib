package contrib

import (
	"context"
	"net/http"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var nextLinkMatch = regexp.MustCompile("^<(.*)>; rel=\"next\", .*")

type Search func(ctx context.Context, query string)

func NewGitHubSearch(token string, out chan []*User, errs chan error) Search {
	return func(ctx context.Context, query string) {
		request, err := http.NewRequest("GET", query, nil)
		if err != nil {
			Cancel(ctx, err, errs)
		}

		request.Header.Add("token", token)
		request = request.WithContext(ctx)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			Cancel(ctx, err, errs)
		}

		if response.StatusCode == http.StatusOK {
			decoder := json.NewDecoder(response.Body)
			var body map[string]interface{}
			err := decoder.Decode(&body)
			if (err != nil) {
				Cancel(ctx, err, errs)
			}

			users := ParseUsers(body)

			newQuery := response.Header.Get("Link")
			links := nextLinkMatch.FindStringSubmatch(newQuery)
			newQuery = links[1]
			newOut := make(chan []*User)

			go NewGitHubSearch(token, newOut, errs)(ctx, newQuery)

			select {
			case <-ctx.Done():
				close(out)
			case err := <-errs:
				Cancel(ctx, err, errs)
			case lastUsers := <-newOut:
				out <- append(users, lastUsers...)
			}

		} else {
			Cancel(ctx, errors.New(fmt.Sprintf("Status Code was: %s", response.Status)), errs)
		}

	}
}

func ParseUsers(body map[string]interface{}) []*User {
	data := body["items"].([]interface{})
	var users []*User
	for _, user := range data {
		users = append(users, ParseUser(user.(map[string]interface{})));
	}

	return users
}

func Cancel(ctx context.Context, err error, errs chan error) {
	errs <- err
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	cancel()
}
