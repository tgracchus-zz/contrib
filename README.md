# contrib

## To compile, go  to server directory and execute
``` go build ```
--------------------------

## To run
``` go run server.go --token={GITHUB_TOKEN} ```

or if compiled
``` ./server --token={GITHUB_TOKEN} ```

The server includes a simply cache for queries

--------------------------
## To query the server
http://localhost:8080/topcontrib?location=barcelona&top=50
