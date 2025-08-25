FROM ruby:3.2-slim

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Gemfile
COPY Gemfile Gemfile.lock* ./

# Install gems
RUN bundle install --without development test

# Copy application
COPY . .

# Create non-root user
RUN useradd -m -u 1001 ruby && \
    chown -R ruby:ruby /app

USER ruby

EXPOSE ${PORT:-4567}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${PORT:-4567}/health || exit 1

CMD ["bundle", "exec", "puma", "-p", "${PORT:-4567}", "-e", "${ENV:-development}"]