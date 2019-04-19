FROM golang:1.12 as golang-builder

ENV CGO_LDFLAGS "-lmecab -lstdc++"
ENV LC_ALL C.UTF-8
ENV LANG C.UTF-8

COPY . /

RUN apt-get update && apt-get -y install mecab libmecab-dev mecab-ipadic-utf8 \
 && apt clean && rm -rf /var/lib/apt/lists/* \
 && go build -o /main /main.go

FROM debian:9-slim as neologd-builder

ENV LC_ALL C.UTF-8
ENV LANG C.UTF-8

RUN apt-get update &&  apt -y install mecab libmecab-dev mecab-ipadic-utf8 git make curl xz-utils file \
 && git clone --depth 1 https://github.com/neologd/mecab-ipadic-neologd.git \
 && cd mecab-ipadic-neologd \
 && ./bin/install-mecab-ipadic-neologd -n -a -y -u \
 && mkdir /mecab-ipadic-neologd-dic \
 && mv `mecab-config --dicdir`/mecab-ipadic-neologd/* /mecab-ipadic-neologd-dic

FROM debian:9-slim

ENV CGO_LDFLAGS "-lmecab -lstdc++"
ENV LC_ALL C.UTF-8
ENV LANG C.UTF-8

COPY . /
COPY --from=golang-builder /main /
COPY --from=neologd-builder /mecab-ipadic-neologd-dic /mecab-ipadic-neologd-dic

RUN apt update && apt -y install mecab libmecab-dev mecab-ipadic-utf8 \
 && apt clean && rm -rf /var/lib/apt/lists/* \
 && echo "dicdir = /mecab-ipadic-neologd-dic" > `mecab-config --sysconfdir`/mecabrc

ENTRYPOINT /main
