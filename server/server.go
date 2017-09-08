package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/tgracchus/contrib/contrib"
	"flag"
	"context"
)

func main() {

	gitHubToken := flag.String("token", "", "CONTRIB_GITHUB_TOKEN")
	router := gin.Default()

	// However, this one will match /user/john/ and also /user/john/send
	// If no other routers match /user/john, it will redirect to /user/john/
	router.GET("/topcontrib", func(c *gin.Context) {
		query := contrib.NewQuery(c.Query("location"), c.Query("top"))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		contributors, err := contrib.SearchContrib(ctx, query, "https://api.github.com", *gitHubToken)
		if (err != nil) {
			c.JSON(http.StatusBadRequest, newErrorResponse(err))
		} else {
			c.JSON(http.StatusOK, contributors)
		}
	})

	router.Run(":8080")
}

type ErrorResponse struct {
	Error string
}

func newErrorResponse(msg error) (error *ErrorResponse) {
	return &ErrorResponse{msg.Error()}
}