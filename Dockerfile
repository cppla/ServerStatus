# The Dockerfile for build localhost source, not git repo
FROM debian:buster as builder

MAINTAINER cppla https://cpp.la

RUN apt-get update -y && apt-get -y install gcc g++ make libcurl4-openssl-dev

COPY . .

WORKDIR /server

RUN make -j
RUN pwd && ls -a

# glibc env run
FROM nginx:1.19

RUN mkdir -p /ServerStatus/server/ && ln -sf /dev/null /var/log/nginx/access.log && ln -sf /dev/null /var/log/nginx/error.log

COPY --from=builder server /ServerStatus/server/
COPY --from=builder web /usr/share/nginx/html/

# china time 
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

EXPOSE 80 35601
HEALTHCHECK --interval=5s --timeout=3s --retries=3 CMD curl --fail http://localhost:80 || bash -c 'kill -s 15 -1 && (sleep 10; kill -s 9 -1)'
CMD nohup sh -c '/etc/init.d/nginx start && /ServerStatus/server/sergate --config=/ServerStatus/server/config.json --web-dir=/usr/share/nginx/html'
