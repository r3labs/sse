deps:
	go get github.com/smartystreets/goconvey/convey

install:
	go install .

test:
	go test -v