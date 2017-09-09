package contrib

import (
	"context"
	"errors"
	"fmt"
	"github.com/tgracchus/contrib/stream"
	"regexp"
)

var nextLinkMatch = regexp.MustCompile("^<(.*)>; rel=\"next\", .*")

const limitValues = "top50|top100|top150"

var limits = regexp.MustCompile("^(" + limitValues + ")$")

func TopContrib(location string, top string, host string, token string) ([]*stream.Object, error) {
	err := validate(location, top)
	if err != nil {
		return nil, err
	}
	userQuery := NewUserQuery(location, 50, host, token)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return stream.Subscribe(ctx, stream.NewStream(ctx, userQuery))
}

func validate(location string, top string) error {
	match := limits.MatchString(top)
	if !match {
		return errors.New(fmt.Sprintf("top value: %s, is not valid, please use one of this values %s", top, limitValues))
	}
	if location == "" {
		return errors.New("location can not be empty")
	}

	return nil
}
