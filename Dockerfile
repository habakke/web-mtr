FROM scratch
MAINTAINER Havard Bakke <habakke@matrise.net>

ADD build/web-mtr /usr/bin/web-mtr
COPY web /opt/web

EXPOSE 80
ENTRYPOINT ["/usr/bin/web-mtr"]
