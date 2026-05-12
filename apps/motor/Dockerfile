FROM php:8.2-cli-alpine

RUN apk add --no-cache \
        libxml2-dev \
        libzip-dev \
        zip \
        unzip \
        git \
        oniguruma-dev \
        ca-certificates \
        tzdata \
    && docker-php-ext-install soap zip dom \
    && cp /usr/share/zoneinfo/America/Lima /etc/localtime \
    && echo "America/Lima" > /etc/timezone

# Composer
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /app

COPY composer.json ./
COPY composer.lock* ./
RUN composer install --no-dev --no-scripts --no-interaction --prefer-dist || \
    composer install --no-dev --no-interaction --prefer-dist

COPY . .

# Carpetas mutables (logs/cache). El motor es stateless: no persiste nada.
RUN mkdir -p storage/logs storage/cache && chmod -R 0775 storage

EXPOSE 8000
ENV MOTOR_PORT=8000

# php -S sirve para devs; en producción puede ponerse detrás de un PHP-FPM + nginx.
# Para el MVP, el built-in server es suficiente (lo expone solo al network interno).
CMD ["sh", "-c", "php -S 0.0.0.0:${MOTOR_PORT} -t public public/index.php"]
