FROM debian:buster-slim

RUN apt-get update && \
    apt-get install -y curl unixodbc && \
    rm -rf /var/lib/apt/lists/*

RUN LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/venom/releases | grep tag_name | head -n 1 | cut -d '"' -f 4) && \
    todl=$(curl -s https://api.github.com/repos/ovh/venom/releases | grep ${LAST_RELEASE} | grep browser_download_url | grep -E 'venom.linux-amd64' | cut -d '"' -f 4) && \
    curl -s $todl -L -o /opt/venom && \
    chmod +x /opt/venom

VOLUME /outputs

#Default volume for tests suite
VOLUME /testsuite
WORKDIR /testsuite

ENTRYPOINT ["/opt/venom" ]

CMD [ "run", "--output-dir", "/outputs"]
