package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gregjones/httpcache"
	"github.com/tgracchus/contrib/stream"
	"net/http"
	"strconv"
	"time"
)

type HttpGetQuery func(ctx context.Context, token string, query string) (err error, nextQueryUrl string, elems int)

type HandleResponse func(ctx context.Context, response *http.Response) (err error, elems int)

type HandleResponseFactory func(stream stream.Stream) HandleResponse

func NewUserQuery(location string, limit int, host string, token string) stream.Source {
	return func(ctx context.Context, s *stream.Stream) error {
		queryUrl := host + fmt.Sprintf("/search/users?q=location:%s&sort=repositories&order=asc&type:user", location)
		rhf := UserHandleResponseFactory(s)
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
			nextQueryUrl := ""
			links := nextLinkMatch.FindStringSubmatch(response.Header.Get("Link"))
			if len(links) == 2 {
				nextQueryUrl = links[1]
			}

			herr, find := hr(ctx, response)
			return herr, nextQueryUrl, find
		} else {
			return errors.New(fmt.Sprintf("Status Code was: %s", response.Status)), "", 0
		}
	}
}

func rateLimit(header http.Header) {
	rateLimit := header.Get("X-Ratelimit-Remaining")
	if rateLimit == "0" {
		unixTime, _ := strconv.ParseInt(header.Get("X-Ratelimit-Reset"), 10, 64)
		waitTime := time.Unix(unixTime, 0)
		sleep := time.Until(waitTime)
		time.Sleep(sleep)
	}
}

func UserHandleResponseFactory(s *stream.Stream) HandleResponse {
	return func(ctx context.Context, response *http.Response) (err error, elems int) {
		decoder := json.NewDecoder(response.Body)
		var body map[string]interface{}
		err = decoder.Decode(&body)
		if err != nil {
			return err, 0
		}
		users := parseUsers(body)
		for _, user := range users {
			s.Push(user)
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
