FROM alpine:3.21

RUN apk update && \
    apk --no-cache add tzdata ca-certificates && \
    rm -rf /var/cache/apk/*

COPY dist/venom.linux-amd64 /usr/local/bin/venom
RUN chmod +x /usr/local/bin/venom

VOLUME /workdir/results
VOLUME /workdir/tests
WORKDIR /workdir

ENV VENOM_OUTPUT_DIR=/workdir/results \
    VENOM_LIB_DIR=/workdir/tests/lib \
    VENOM_VERBOSE=1

ENTRYPOINT ["/usr/local/bin/venom"]
CMD ["run", "./tests/*.y*ml"]