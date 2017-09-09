package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tgracchus/contrib/stream"
	"net/http"
	"regexp"
	"strconv"
	"time"
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

func ParseUser(user map[string]interface{}) *User {
	id := user["id"].(float64)
	url := user["url"].(string)
	userType := user["type"].(string)
	score := user["score"].(float64)
	return &User{id, url, userType, score}
}

func NewQuery(location string, limit string) *query {
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

type HttpGetQuery func(ctx context.Context, token string, query string) error

type HandleResponse func(ctx context.Context, response *http.Response) error

type HandleResponseFactory func(stream stream.Stream) HandleResponse

type query struct {
	location string
	top      string
}

func SearchContrib(query *query, host string, token string) ([]*stream.Object, error) {
	userQuery := NewUserQuery(query, host, token)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return stream.Subscribe(ctx, stream.NewStream(ctx, userQuery))
}

func NewUserQuery(query *query, host string, token string) stream.Source {
	return func(ctx context.Context, s stream.Stream) error {
		err := query.validate()
		if err != nil {
			return err
		}
		queryUrl := host + fmt.Sprintf("/search/users?q=location:%s", query.location)
		rhf := UserHandleResponseFactory(s)
		gerr := NewHttpGetFactory(rhf)(ctx, token, queryUrl)
		if gerr != nil {
			return gerr
		}
		return nil
	}

}

func NewHttpGetFactory(hr HandleResponse) HttpGetQuery {
	var GetQuery HttpGetQuery
	GetQuery = func(ctx context.Context, token string, query string) error {
		request, err := http.NewRequest("GET", query, nil)
		if err != nil {
			return err
		}

		request.Header.Add("token", token)
		request = request.WithContext(ctx)
		response, cerr := http.DefaultClient.Do(request)
		if cerr != nil {
			return cerr
		}

		if response.StatusCode == http.StatusOK {
			hr(ctx, response)
		} else {
			return errors.New(fmt.Sprintf("Status Code was: %s", response.Status))
		}

		rateLimit := response.Header.Get("X-Ratelimit-Remaining")
		if rateLimit == "0" {
			waitUntil := response.Header.Get("X-Ratelimit-Reset")
			unixTime, err := strconv.ParseInt(waitUntil, 10, 64)
			if err != nil {
				return err
			}
			waitTime := time.Unix(unixTime, 0)
			sleep := time.Until(waitTime)
			time.Sleep(sleep)
		}

		newQuery := response.Header.Get("Link")
		links := nextLinkMatch.FindStringSubmatch(newQuery)

		if len(links) == 2 {
			nextQuery := links[1]
			return GetQuery(ctx, token, nextQuery)
		} else {
			return nil
		}
	}
	return GetQuery
}

func UserHandleResponseFactory(s stream.Stream) HandleResponse {
	return func(ctx context.Context, response *http.Response) error {
		decoder := json.NewDecoder(response.Body)
		var body map[string]interface{}
		err := decoder.Decode(&body)
		if err != nil {
			return err
		}

		for _, user := range parseUsers(body) {
			s.Push(user)
		}

		return nil
	}

}

func parseUsers(body map[string]interface{}) []*stream.Object {
	data := body["items"].([]interface{})
	var users []*stream.Object
	for _, user := range data {
		object := &stream.Object{user.(map[string]interface{}), "user"}
		users = append(users, object)
	}

	return users
}
