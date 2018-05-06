FROM golang:alpine AS build

RUN apk update && \
    apk add --nocache git && \
    go get -u github.com/golang/dep/cmd/dep

RUN mkdir -p $GOPATH/src/github.com/imduffy15/minikube-assign-external-ip

COPY . $GOPATH/src/github.com/imduffy15/minikube-assign-external-ip

RUN cd $GOPATH/src/github.com/imduffy15/minikube-assign-external-ip && \
    dep ensure && \
    go build -o /minikube-assign-external-ip main.go

# Create final image
FROM alpine

RUN apk update && apk add git 
COPY --from=build /minikube-assign-external-ip /usr/local/bin/minikube-assign-external-ip

ENTRYPOINT ["minikube-assign-external-ip"]

