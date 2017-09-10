package main

import (
	"flag"
	"github.com/gin-gonic/contrib/cache"
	"github.com/gin-gonic/gin"
	"github.com/tgracchus/contrib/contrib"
	"net/http"
	"time"
)

func main() {
	store := cache.NewInMemoryStore(time.Minute)
	gitHubToken := flag.String("token", "", "CONTRIB_GITHUB_TOKEN")
	router := gin.Default()

	// However, this one will match /user/john/ and also /user/john/send
	// If no other routers match /user/john, it will redirect to /user/john/
	router.GET("/topcontrib", cache.CachePage(store, time.Hour, func(c *gin.Context) {
		contributors, err := contrib.TopContrib(c.Query("location"), c.Query("top"), "https://api.github.com", *gitHubToken)
		if err != nil {
			if verr, ok := err.(*contrib.ValidationError); ok {
				c.JSON(http.StatusBadRequest, verr)
			} else {
				c.JSON(http.StatusInternalServerError, newErrorResponse(err))
			}

		} else {
			c.JSON(http.StatusOK, contributors)
		}
	}))

	router.Run(":8080")
}

type ErrorResponse struct {
	Error string
}

func newErrorResponse(msg error) (error *ErrorResponse) {
	return &ErrorResponse{msg.Error()}
}
