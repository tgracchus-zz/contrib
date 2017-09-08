package contrib

import (
	"context"
	"net/http"
	"errors"
	"fmt"
	"regexp"
	"encoding/json"
)

var nextLinkMatch = regexp.MustCompile("^<(.*)>; rel=\"next\", .*")

const limitValues = "top50|top100|top150"

var limits = regexp.MustCompile("^(" + limitValues + ")$")

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

func NewQuery(location string, limit string) (*query) {
	return &query{location: location, top: limit}
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

type HttpGetQuery func(ctx context.Context, token string, query string)

type Object struct {
	Data       map[string]interface{}
	objectType string
}

type HandleResponse func(ctx context.Context, response *http.Response)

type HandleResponseFactory func(out chan *Object, eh ErrorHandler) HandleResponse

type ErrorHandlerFactory func(errs chan *error) ErrorHandler

type ErrorHandler func(ctx context.Context, err error)

type query struct {
	location string
	top      string
}

func SearchContrib(ctx context.Context, query *query, host string, token string) ([]*Object, error) {
	err := query.validate()
	if err != nil {
		return nil, err
	}

	out := make(chan *Object)
	errs := make(chan *error)
	queryUrl := host + fmt.Sprintf("/search/users?q=location:%s", query.location)

	ehf := CancelErrorHandlerFactory(errs)
	rhf := UserHandleResponseFactory(out, ehf)
	go NewHttpGetFactory(rhf, ehf)(ctx, token, queryUrl)
	var objects []*Object

	for {
		select {
		case <-ctx.Done():
			return objects, ctx.Err()
		case object := <-out:
			objects = append(objects, object)
		case err := <-errs:
			return nil, *err
		}
	}
}

func NewHttpGetFactory(hr HandleResponse, eh ErrorHandler) HttpGetQuery {
	var GetQuery HttpGetQuery
	GetQuery = func(ctx context.Context, token string, query string) {
		request, err := http.NewRequest("GET", query, nil)
		if err != nil {
			eh(ctx, err)
		}

		request.Header.Add("token", token)
		request = request.WithContext(ctx)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			eh(ctx, err)
		}

		if response.StatusCode == http.StatusOK {
			hr(ctx, response)
		} else {
			eh(ctx, errors.New(fmt.Sprintf("Status Code was: %s", response.Status)))
		}

		newQuery := response.Header.Get("Link")
		links := nextLinkMatch.FindStringSubmatch(newQuery)

		if len(links) == 2 {
			nextQuery := links[1]
			GetQuery(ctx, token, nextQuery)
		}

	}

	return GetQuery
}

func UserHandleResponseFactory(out chan *Object, eh ErrorHandler) HandleResponse {
	return func(ctx context.Context, response *http.Response) {
		decoder := json.NewDecoder(response.Body)
		var body map[string]interface{}
		err := decoder.Decode(&body)
		if (err != nil) {
			eh(ctx, err)
		}

		for _, user := range parseUsers(body) {
			out <- user
		}
	}

}

func parseUsers(body map[string]interface{}) []*Object {
	data := body["items"].([]interface{})
	var users []*Object
	for _, user := range data {
		object := &Object{user.(map[string]interface{}), "user"}
		users = append(users, object);
	}

	return users
}

func CancelErrorHandlerFactory(errs chan *error) ErrorHandler {
	return func(ctx context.Context, err error) {
		errs <- &err
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		cancel()
	}
}
