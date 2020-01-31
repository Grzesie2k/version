FROM golang:onbuild as build

RUN mkdir /app
ADD main.go /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine/git
COPY --from=build /app/main /app
WORKDIR /
ENTRYPOINT ["/app"]
