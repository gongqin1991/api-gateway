FROM alpine:latest
WORKDIR /root
COPY ./Shanghai /etc/localtime
COPY ./main .
COPY ./cfg.toml .
CMD ./main --cfg=./cfg.toml
#ENTRYPOINT ["top", "-b"]