FROM php:8.2-cli-alpine

# Install extensions
RUN apk add --no-cache postgresql-dev \
    && docker-php-ext-install pdo pdo_pgsql \
    && pecl install redis \
    && docker-php-ext-enable redis

# Install composer
COPY --from=composer:latest /usr/bin/composer /usr/bin/composer

WORKDIR /app

# Copy composer files
COPY composer.json composer.lock* ./

# Install dependencies
RUN composer install --no-dev --optimize-autoloader

# Copy application
COPY . .

# Create non-root user
RUN addgroup -g 1001 -S php && \
    adduser -S php -u 1001 && \
    chown -R php:php /app

USER php

EXPOSE ${PORT:-8080}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD php -r "file_get_contents('http://localhost:${PORT:-8080}/health');" || exit 1

CMD ["php", "-S", "0.0.0.0:${PORT:-8080}", "-t", "."]