package contrib

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

var content50 []byte
var content25 []byte

func TestMain(m *testing.M) {
	var err error
	content50, err = ioutil.ReadFile("test/github50.json")
	if err != nil {
		panic(m)
	}

	content25, err = ioutil.ReadFile("test/github25.json")
	if err != nil {
		panic(m)
	}
	os.Exit(m.Run())
}

func Test_NewUserQuery_github_next_link(t *testing.T) {
	hm := make(map[string][]string)
	var h http.Header = hm
	h.Add("Link", `<https://api.github.com/search/users?q=location%3Abarcelona&sort=repositories&order=desc&type%3Auser=&per_page=20&page=2>; rel="next", <https://api.github.com/search/users?q=location%3Abarcelona&sort=repositories&order=desc&type%3Auser=&per_page=20&page=50>; rel="last"`)
	nextQuery := parseNextQueryUrl(h)
	if nextQuery != `https://api.github.com/search/users?q=location%3Abarcelona&sort=repositories&order=desc&type%3Auser=&per_page=20&page=2` {
		t.Fail()
	}
}

func Test_NewUserQuery_github_rate_limit(t *testing.T) {
	hm := make(map[string][]string)
	var h http.Header = hm
	h.Add("x-ratelimit-remaining", "0")
	h.Add("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(time.Second * 1).Unix(), 10))
	duration := waitForRateLimitToExpire(h)
	if duration == 0 {
		t.Fail()
	}
}

func Test_TopContrib_github_ok(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content50)
	}))
	defer ts.Close()

	objects, err := TopContrib("barcelona", "50", ts.URL, "token")
	if err != nil {
		t.Fatalf("We were expecting some objects, not a error: %s", err)
	}

	if objects == nil || len(objects) != 50 {
		t.Fatal("We were expecting 50 objects")
	}

}

func Test_TopContrib_github_400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

	}))
	defer ts.Close()

	_, err := TopContrib("barcelona", "50", ts.URL, "token")
	if err == nil {
		t.Fatalf("We were expecting an error")
	}
}

func Test_TopContrib_github_multiple_queries(t *testing.T) {

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content25)
		w.WriteHeader(http.StatusOK)

	}))
	defer ts2.Close()

	linkHeader := fmt.Sprintf(`<%s/search/users?q=location%3Abarcelona&sort=repositories&order=desc&type%3Auser=&per_page=20&page=2>; rel="next", <%ssearch/users?q=location%3Abarcelona&sort=repositories&order=desc&type%3Auser=&per_page=20&page=50>; rel="last"`, ts2.URL, ts2.URL)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Link", linkHeader)
		w.Write(content25)

	}))
	defer ts.Close()

	objects, err := TopContrib("barcelona", "50", ts.URL, "token")

	if err != nil {
		t.Fatalf("We were expecting some objects, not a error: %s", err)
	}

	if objects == nil || len(objects) != 50 {
		t.Fatal("We were expecting 50 objects")
	}
}

func Test_TopContrib_github_rate_limit(t *testing.T) {
	callNumber := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callNumber == 0 {
			callNumber++
			w.Header().Add("x-ratelimit-remaining", "0")
			w.Header().Add("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(time.Second * 1).Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.Write(content50)
			w.WriteHeader(http.StatusOK)
		}

	}))
	defer ts.Close()

	objects, err := TopContrib("barcelona", "50", ts.URL, "token")

	if err != nil {
		t.Fatalf("We were expecting some objects, not a error: %s", err)
	}

	if objects == nil || len(objects) != 50 {
		t.Fatal("We were expecting 50 objects")
	}
}
