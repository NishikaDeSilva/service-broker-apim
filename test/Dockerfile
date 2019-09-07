FROM golang:1.12.1 AS builder

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

COPY Gopkg.toml Gopkg.lock $GOPATH/src/github.com/wso2/service-broker-apim/
COPY cmd/ $GOPATH/src/github.com/wso2/service-broker-apim/cmd/
COPY pkg/ $GOPATH/src/github.com/wso2/service-broker-apim/pkg/

WORKDIR $GOPATH/src/github.com/wso2/service-broker-apim
RUN dep ensure

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /servicebroker-linux github.com/wso2/service-broker-apim/cmd/servicebroker

FROM amd64/alpine:3.10.2
COPY --from=builder /servicebroker-linux ./
COPY ./test/init-scripts/wait-for-deps.sh ./wait-for-deps.sh
RUN chmod +x ./wait-for-deps.sh
ENTRYPOINT ["./wait-for-deps.sh"]