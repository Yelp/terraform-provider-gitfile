FROM docker-dev.yelpcorp.com/trusty_yelp
MAINTAINER Evan Krall <krall@yelp.com>

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
    wget \
    git \
    build-essential \
    ruby1.9.1 rubygems1.9.1 \
    libopenssl-ruby1.9.1 \
    ruby1.9.1-dev \
    --no-install-recommends

RUN wget https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz && tar xzf go1.6.linux-amd64.tar.gz && mv go /usr/local
ENV PATH /usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin:/usr/local/sbin:/usr/local/go/bin:/go/bin
ENV GOPATH /go
RUN mkdir /go
RUN gem install json  -v 1.8.3
RUN gem install fpm
RUN git clone https://github.com/hashicorp/terraform.git /go/src/github.com/hashicorp/terraform && \
    cd /go/src/github.com/hashicorp/terraform && \
        git checkout e822a79165dbc06bbf8271ee349fe256867d53dc
#WORKDIR /go/src/github.com/Yelp/terraform-provider-gitfile
