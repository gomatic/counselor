FROM scratch

MAINTAINER nicerobot "https://github.com/gomatic/counselor"

ENV TMP=/
ENV TEMP=/

WORKDIR /

ENV HOME=/
ENV PWD=/
ENV PATH=/
COPY counselor-linux-amd64 /counselor

ENTRYPOINT ["counselor", "run", "--debug", "--verbose", "--"]
CMD ["counselor", "test"]
