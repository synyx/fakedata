FROM golang:1.13 AS builder

WORKDIR /src

RUN git clone  https://github.com/synyx/fakedata

WORKDIR /src/fakedata
RUN GOPROXY=https://proxy.golang.org CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /fakedata .

FROM scratch
COPY --from=builder /fakedata ./
ENTRYPOINT ["./fakedata"]