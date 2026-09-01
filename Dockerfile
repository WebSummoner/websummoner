FROM --platform=$BUILDPLATFORM alpine:3

RUN apk add -U ca-certificates tzdata mailcap && rm -Rf /var/cache/apk/*

ARG TARGETARCH
COPY dist/websummoner_linux_$TARGETARCH /usr/bin/websummoner

EXPOSE 4444
ENTRYPOINT ["/usr/bin/websummoner", "-listen", ":4444", "-conf", "/etc/websummoner/browsers.json", "-video-output-dir", "/opt/websummoner/video/"]
