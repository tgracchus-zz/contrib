package contrib

import (
	"context"
	"errors"
	"fmt"
	"github.com/tgracchus/contrib/stream"
	"regexp"
	"strconv"
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
	return stream.Subscribe(ctx, stream.NewStream(ctx, userQuery))
}

func validate(location string, top string) (int, error) {
	match := limits.MatchString(top)
	if !match {
		return 0, errors.New(fmt.Sprintf("top value: %s, is not valid, please use one of this values %s", top, limitValues))
	}
	if location == "" {
		return 0, errors.New("location can not be empty")
	}

	limit, err := strconv.Atoi(top)
	if err != nil {
		return 0, err
	}
	return limit, nil
}
