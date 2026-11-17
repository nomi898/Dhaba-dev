
# ==========================================
# Stage 1: Common PHP base
# ==========================================
FROM php:8.4-fpm-alpine AS php-base

RUN docker-php-ext-install opcache


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

