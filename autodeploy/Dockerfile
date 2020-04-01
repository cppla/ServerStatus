FROM debian:latest as builder

MAINTAINER cppla https://cpp.la

RUN apt-get update
RUN apt-get -y install gcc g++ make git
RUN git clone https://github.com/cppla/ServerStatus

WORKDIR /ServerStatus/server

RUN make
RUN pwd && ls -a

# glibc env run 
FROM nginx:latest

RUN mkdir -p /ServerStatus/server/

COPY --from=builder /ServerStatus/server /ServerStatus/server/
COPY --from=builder /ServerStatus/web /usr/share/nginx/html/

EXPOSE 80 35601

CMD nohup sh -c '/etc/init.d/nginx start && /ServerStatus/server/sergate --config=/ServerStatus/server/config.json --web-dir=/usr/share/nginx/html'
