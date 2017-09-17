package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gregjones/httpcache"
	"github.com/tgracchus/contrib/stream"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var nextLinkMatch = regexp.MustCompile("^<(.*)>; rel=\"next\", .*")

const limitValues = "50|100|150"

var limits = regexp.MustCompile("^(" + limitValues + ")$")

func TopContrib(location string, top string, host string, token string) ([]*stream.Object, error) {
	limit, err := validate(location, top)
	if err != nil {
		return nil, err
	}
	userQuery := NewUserQuery(location, limit, host, token)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return stream.NewStream(ctx, userQuery).Map(trimUser).Subscribe()
}

type ValidationError struct {
	msg string // description of error
}

func (e *ValidationError) Error() string { return e.msg }

func validate(location string, top string) (int, error) {
	match := limits.MatchString(top)
	if !match {
		return 0, &ValidationError{fmt.Sprintf("top value: %s, is not valid, please use one of this values %s", top, limitValues)}
	}
	if location == "" {
		return 0, &ValidationError{"location can not be empty"}
	}

	limit, err := strconv.Atoi(top)
	if err != nil {
		return 0, err
	}
	return limit, nil
}

type HttpGetQuery func(ctx context.Context, token string, query string) (err error, nextQueryUrl string, elems int)

type HandleResponse func(ctx context.Context, response *http.Response) (err error, elems int)

type HandleResponseFactory func(stream stream.Stream) HandleResponse

func NewUserQuery(location string, limit int, host string, token string) stream.Source {
	return func(ctx context.Context, s *stream.Stream) error {
		perPage := elemsPerPage(limit)
		queryUrl := host + fmt.Sprintf("/search/users?q=location:%s&sort=repositories&order=asc&type:user&per_page=%d", location, perPage)
		rhf := UserHandleResponseFactory(s, limit)
		httpGf := NewHttpGetFactory(rhf)

		elems := 0
		var err error
		var newElems int
		for elems < limit {
			err, queryUrl, newElems = httpGf(ctx, token, queryUrl)
			elems += newElems
			if err != nil {
				return err
			}
		}

		return nil
	}
}
func elemsPerPage(i int) int {
	if i >= 100 {
		return 100
	} else {
		return i
	}
}

var tp *httpcache.Transport = httpcache.NewMemoryCacheTransport()
var client = http.Client{Transport: tp}

func NewHttpGetFactory(hr HandleResponse) HttpGetQuery {
	return func(ctx context.Context, token string, query string) (error, string, int) {
		request, err := http.NewRequest("GET", query, nil)
		if err != nil {
			return err, "", 0
		}

		request.Header.Add("token", token)
		request = request.WithContext(ctx)
		response, cerr := client.Do(request)
		if cerr != nil {
			return cerr, "", 0
		}

		rateLimit(response.Header)

		if response.StatusCode == http.StatusOK {
			nextQueryUrl := parseNextQueryUrl(response.Header)
			herr, find := hr(ctx, response)
			return herr, nextQueryUrl, find
		} else {
			return errors.New(fmt.Sprintf("Status Code was: %s", response.Status)), "", 0
		}
	}
}
func parseNextQueryUrl(header http.Header) (nextQueryUrl string) {
	links := nextLinkMatch.FindStringSubmatch(header.Get("Link"))
	if len(links) == 2 {
		nextQueryUrl = links[1]
	}
	return
}

func rateLimit(header http.Header) time.Duration {
	rateLimit := header.Get("X-Ratelimit-Remaining")
	if rateLimit == "0" {
		unixTime, _ := strconv.ParseInt(header.Get("X-Ratelimit-Reset"), 10, 64)
		waitTime := time.Unix(unixTime, 0)
		sleep := time.Until(waitTime)
		time.Sleep(sleep)
		return sleep
	}

	return 0
}

func UserHandleResponseFactory(s *stream.Stream, limit int) HandleResponse {
	total := 0
	return func(ctx context.Context, response *http.Response) (err error, elems int) {
		decoder := json.NewDecoder(response.Body)
		var body map[string]interface{}
		err = decoder.Decode(&body)
		if err != nil {
			return err, 0
		}
		users := parseUsers(body)
		for _, user := range users {
			if total < limit {
				s.Push(user)
				total++
			} else {
				return nil, len(users)
			}
		}
		return nil, len(users)
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

func trimUser(ctx context.Context, object *stream.Object) (*stream.Object, error) {
	trimmedUser := make(map[string]interface{})
	if id, ok := object.Data["id"]; ok {
		trimmedUser["id"] = id
	}
	if url, ok := object.Data["url"]; ok {
		trimmedUser["url"] = url
	}
	if typeVal, ok := object.Data["type"]; ok {
		trimmedUser["type"] = typeVal
	}
	if id, ok := object.Data["score"]; ok {
		trimmedUser["score"] = id
	}
	return &stream.Object{trimmedUser, "user"}, nil
}
