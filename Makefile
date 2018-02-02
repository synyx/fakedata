.PHONY : go-build docker-build docker-push

go-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fakedata .

docker-build:
	docker build -t rjayasinghe/fakedata .

docker-push:
	docker push rjayasinghe/fakedata

publish: go-build docker-build docker-push
