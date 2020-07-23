FROM golang:1.13 AS builder

ADD . /src
WORKDIR /src
RUN GOPROXY=https://proxy.golang.org go mod download
RUN GOPROXY=https://proxy.golang.org CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /fakedata .

FROM scratch
COPY --from=builder /fakedata ./
ENTRYPOINT ["./fakedata"]