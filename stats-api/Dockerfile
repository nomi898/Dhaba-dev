
# ==========================================
# Stage 1: Common PHP base
# ==========================================
FROM php:8.4-fpm-alpine AS php-base

# DL3018: Alpine drops old package versions, so exact pins break future builds.
# SC2086: $PHPIZE_DEPS is a space-separated package list and must be word-split.
# hadolint ignore=DL3018,SC2086
RUN apk add --no-cache postgresql-libs \
    && apk add --no-cache --virtual .build-deps $PHPIZE_DEPS postgresql-dev \
    && docker-php-ext-install pdo_pgsql opcache \
    && apk del .build-deps

# ==========================================
# Stage 2: Install Composer dependencies
# ==========================================
FROM php-base AS vendor

WORKDIR /app

# Copy Composer executable into this build stage only
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

COPY composer.json composer.lock ./

RUN composer install \
    --no-dev \
    --no-interaction \
    --no-progress \
    --prefer-dist \
    --optimize-autoloader \
    --no-scripts


# ==========================================
# Stage 3: PHP-FPM production runtime
# ==========================================
FROM php-base AS runtime

WORKDIR /var/www/html

# Copy Laravel application
COPY . .

# Copy production dependencies
COPY --from=vendor /app/vendor ./vendor

# Generate Laravel package manifest and prepare permissions
RUN php artisan package:discover --ansi \
    && mkdir -p storage/framework/cache \
        storage/framework/sessions \
        storage/framework/views \
        bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache

EXPOSE 9000

CMD ["php-fpm"]

