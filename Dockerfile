FROM golang:alpine as build

WORKDIR /docso

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN go build

FROM alpine
WORKDIR /docso
COPY --from=build /docso/dr-docso /bin/dr-docso

ENTRYPOINT [ "/bin/dr-docso" ]