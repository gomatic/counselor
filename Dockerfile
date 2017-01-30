FROM scratch

ENV TMP=/
ENV TEMP=/

WORKDIR /

ENV HOME=/
ENV PWD=/
ENV PATH=/
COPY counselor-linux-amd64 /counselor

ENTRYPOINT ["counselor"]
CMD ["run", "--", "counselor", "test"]
