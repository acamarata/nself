# nself Docker Image
# Version: 0.3.9

FROM alpine:3.18

LABEL maintainer="nself.org"
LABEL version="0.3.9"
LABEL description="nself - Self-hosted backend platform"

# Install dependencies
RUN apk add --no-cache \
    bash \
    curl \
    git \
    jq \
    docker-cli \
    docker-cli-compose \
    openssl \
    ca-certificates \
    wget \
    ncurses

# Install mkcert
RUN wget -O /usr/local/bin/mkcert https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-linux-amd64 && \
    chmod +x /usr/local/bin/mkcert

# Copy nself from tarball
COPY nself-v0.3.9.tar.gz /tmp/
RUN mkdir -p /usr/local/nself && \
    cd /usr/local/nself && \
    tar -xzf /tmp/nself-v0.3.9.tar.gz && \
    rm /tmp/nself-v0.3.9.tar.gz

# Set up environment
ENV NSELF_HOME=/usr/local/nself
ENV PATH="/usr/local/nself/bin:${PATH}"

# Make scripts executable
RUN chmod +x /usr/local/nself/bin/nself && \
    chmod -R 755 /usr/local/nself

# Create working directory
WORKDIR /workspace

# Entry point
ENTRYPOINT ["nself"]
CMD ["help"]