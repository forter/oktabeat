FROM golang:stretch AS builder
MAINTAINER Forter RnD

WORKDIR /go/src/github.com/forter/oktabeat
RUN mkdir -p /config
RUN apt-get update && \
    apt-get install -y \
    git gcc g++ binutils make
RUN mkdir -p ${GOPATH}/src/github.com/elastic && git clone https://github.com/elastic/beats ${GOPATH}/src/github.com/elastic/beats
COPY . /go/src/github.com/forter/oktabeat/
RUN make
RUN chmod +x oktabeat
# ---

FROM ubuntu:latest
COPY --from=builder /go/src/github.com/forter/oktabeat/oktabeat /bin/oktabeat
RUN apt-get -y update \
 && apt-get -y install ca-certificates dumb-init curl \
 && update-ca-certificates
VOLUME  /config/beat.yml
ENTRYPOINT [ "/bin/oktabeat" ]
CMD [ "-e -c /config/beat.yml" ]
