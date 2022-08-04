FROM alpine:3.16 
RUN apk update && \
    apk --no-cache add tzdata && \
    apk --no-cache add ca-certificates && rm -rf /var/cache/apk/*

COPY dist/venom.linux-amd64 /usr/local/venom

VOLUME /workdir/results
VOLUME /workdir/tests
WORKDIR /workdir

ENTRYPOINT ["/usr/local/venom"]

ENV VENOM_OUTPUT_DIR=/workdir/results
ENV VENOM_LIB_DIR=/workdir/tests/lib
ENV VENOM_VERBOSE=1

CMD [ "run", "./tests/*.y*ml"]