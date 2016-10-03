FROM golang:1.7

# Install dumb-init
RUN wget -O /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.1.3/dumb-init_1.1.3_amd64
RUN chmod +x /usr/local/bin/dumb-init
ENTRYPOINT ["/usr/local/bin/dumb-init", "--"]

# Install sous
RUN mkdir -p /go/src/github.com/opentable/sous
WORKDIR /go/src/github.com/opentable/sous
COPY . /go/src/github.com/opentable/sous
RUN go install -v

# Run sous server
CMD /go/bin/sous server -listen :$PORT0

