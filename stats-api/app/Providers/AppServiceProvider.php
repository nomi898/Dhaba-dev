<?php

declare(strict_types=1);

namespace App\Providers;

use App\Stats\RedisStatsStore;
use App\Stats\StatsStore;
use Illuminate\Support\Facades\Redis;
use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->app->bind(StatsStore::class, static fn (): StatsStore => new RedisStatsStore(Redis::connection()));
    }

    public function boot(): void {}
}
