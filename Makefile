.PHONY : go-build docker-build docker-push

go-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fakedata .

docker-build:
	docker build -t synyx/fakedata .

docker-push:
	docker push synyx/fakedata

publish: go-build docker-build docker-push
