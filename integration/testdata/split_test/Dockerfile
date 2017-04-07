FROM golang:1.7

ARG APP_VERSION=0.0.0
ARG APP_REVISION=unknown
ENV SOUS_RUN_IMAGE_SPEC=/runspec.json

COPY manifest.json ${SOUS_BUILD_MANIFEST}
COPY . /go/src/github.com/opentable/sous-demo
RUN \
  cd src/github.com/opentable/sous-demo; \
  go build .; \
  mkdir /built; \
  cat <<-SPEC > $SOUS_RUN_IMAGE_SPEC \
  {"image": {"type": "Docker", "from": "scratch"}, \
   "files":[{"source": {"dir": "/built"}, "dest": {"dir": "/"}}], \
   "exec":["/sous-demo"]} \
  SPEC\
  cp sous-demo /built
