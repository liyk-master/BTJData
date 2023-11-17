FROM alpine
ENV TZ=Asia/Shanghai LANG=C.UTF-8
RUN echo 'http://mirrors.ustc.edu.cn/alpine/v3.5/main' > /etc/apk/repositories \
&& echo 'http://mirrors.ustc.edu.cn/alpine/v3.5/community' >>/etc/apk/repositories \
&& apk update && apk add tzdata \
&& ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
&& echo "Asia/Shanghai" > /etc/timezone
COPY build/qhData /go/src/qhData
COPY config /go/src/config
RUN chmod a+x /go/src/qhData
WORKDIR /go/src/
ENTRYPOINT ["./qhData"]