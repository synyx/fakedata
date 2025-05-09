FROM golang:1.24 AS builder

ADD . /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /fakedata .

FROM scratch
COPY --from=builder /fakedata ./
ENTRYPOINT ["./fakedata"]
