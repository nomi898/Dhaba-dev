# stats-api

Laravel 12 service that serves live order statistics from Redis. See the root README for the full API and the Redis key contract.

```bash
composer install
cp .env.example .env          # set REDIS_HOST=127.0.0.1 for local runs outside Docker
php artisan test              # PHPUnit
composer analyse              # PHPStan (Larastan, level 8)
vendor/bin/pint --test        # code style
```
