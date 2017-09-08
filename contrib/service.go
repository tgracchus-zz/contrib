package contrib

import (
	"regexp"
	"errors"
	"fmt"
	"context"
)

const limitValues = "top50|top100|top150"

var limits = regexp.MustCompile("^(" + limitValues + ")$")

func NewQuery(location string, limit string) (*query) {
	return &query{location: location, top: limit}
}

type query struct {
	location string
	top      string
}

func (c query) validate() error {
	match := limits.MatchString(c.top)
	if !match {
		return errors.New(fmt.Sprintf("top value: %s, is not valid, please use one of this values %s", c.top, limitValues))
	}
	if c.location == "" {
		return errors.New("location can not be empty")
	}

	return nil
}

type User struct {
	Id       float64
	Url      string
	UserType string
	Score    float64
}

func ParseUser(user map[string]interface{}) (*User) {
	id := user["id"].(float64)
	url := user["url"].(string)
	userType := user["type"].(string)
	score := user["score"].(float64)
	return &User{id, url, userType, score}

}

func SearchContrib(ctx context.Context, query *query, host string, token string) ([]*User, error) {
	err := query.validate()
	if err != nil {
		return nil, err
	}

	out := make(chan []*User)
	errs := make(chan error)
	url := host + fmt.Sprintf("/search/users?q=location:%s", query.location)
	search := NewGitHubSearch(token, out, errs)
	go search(ctx, url)

	select {

	case <-ctx.Done():
		return nil, ctx.Err()
	case contributor := <-out:
		return contributor, nil
	case err := <-errs:
		return nil, err

	}

}
