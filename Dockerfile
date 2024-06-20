FROM alpine:3.4

RUN apk update
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN apk add libpcap=1.7.4-r0 libpcap-dev=1.7.4-r0
WORKDIR /usr/lib
RUN ln -s libpcap.so.1.7.4 libpcap.so.0.8

RUN apk add -U tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && apk del tzdata

WORKDIR /usr/local/bin
COPY traffic-statistics .

WORKDIR /usr/local/app

ENTRYPOINT ["traffic-statistics"]
