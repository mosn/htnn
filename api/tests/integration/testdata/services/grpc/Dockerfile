FROM docker.io/library/golang:1.22-bullseye

WORKDIR /app
COPY . /app
RUN go build
CMD ["./grpc"]
