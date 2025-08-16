# Official nself Docker image
FROM alpine:3.18

# Install dependencies
RUN apk add --no-cache \
    bash \
    curl \
    git \
    docker \
    docker-compose

# Create nself directory
WORKDIR /opt/nself

# Copy source files
COPY src/ ./src/
COPY bin/ ./bin/

# Make scripts executable
RUN chmod +x bin/nself src/cli/*.sh

# Add nself to PATH
ENV PATH="/opt/nself/bin:${PATH}"

# Metadata
LABEL org.opencontainers.image.title="nself" \
      org.opencontainers.image.description="Deploy feature-complete backend infrastructure in seconds" \
      org.opencontainers.image.url="https://nself.org" \
      org.opencontainers.image.source="https://github.com/acamarata/nself" \
      org.opencontainers.image.version="0.3.7" \
      org.opencontainers.image.vendor="nself" \
      org.opencontainers.image.licenses="Source-Available"

# Default command
CMD ["nself", "help"]