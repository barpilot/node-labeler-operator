FROM golang:1.27.1 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOMAXPROCS=2 go build -p=2 -trimpath -ldflags="-s -w" -o /node-labeler-operator .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /node-labeler-operator /node-labeler-operator
ENTRYPOINT ["/node-labeler-operator"]
