covertest:
	go test  -coverprofile=coverage.out
	go tool cover -html=coverage.out

coveralls:
	go test  -coverprofile=coverage.out `go list ./... | grep -v test`
	goveralls -coverprofile=coverage.out -reponame=go-wirenet -repotoken=${COVERALLS_CHANNEL} -service=local
