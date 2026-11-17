<?php

declare(strict_types=1);

namespace App\Stats;

use Illuminate\Redis\Connections\Connection;

/**
 * Redis implementation. Key names are the contract documented in the README and used by kitchen-worker.
 */
final class RedisStatsStore implements StatsStore
{
    public const TOTAL = 'stats:orders:total';

    public const REVENUE = 'stats:revenue:total_cents';

    public const MINUTE_PREFIX = 'stats:orders:minute:';

    public const DISH_QTY = 'stats:dishes:qty';

    public const DISH_REVENUE = 'stats:dishes:revenue';

    public const DISH_NAMES = 'stats:dishes:names';

    public const RECENT = 'stats:recent';

    public const STATUS_COUNTS = 'stats:status_counts';

    public const HEARTBEAT = 'kitchen-worker:heartbeat';

    public const QUEUE = 'orders:queue';

    public const DELAYED = 'orders:delayed';

    public const DEAD = 'orders:dead';

    public function __construct(private readonly Connection $redis) {}

    public function totalOrders(): int
    {
        return (int) $this->redis->command('get', [self::TOTAL]);
    }

    public function totalRevenueCents(): int
    {
        return (int) $this->redis->command('get', [self::REVENUE]);
    }

    public function ordersPerMinuteBuckets(array $minuteBuckets): array
    {
        if ($minuteBuckets === []) {
            return [];
        }
        $keys = array_map(static fn (string $b): string => self::MINUTE_PREFIX.$b, $minuteBuckets);
        /** @var list<string|null> $values */
        $values = $this->redis->command('mget', [$keys]);
        $out = [];
        foreach ($minuteBuckets as $i => $bucket) {
            $out[$bucket] = (int) ($values[$i] ?? 0);
        }

        return $out;
    }

    public function statusCounts(): array
    {
        /** @var array<string, string> $raw */
        $raw = $this->redis->command('hgetall', [self::STATUS_COUNTS]);
        $counts = ['NEW' => 0, 'COOKING' => 0, 'READY' => 0];
        foreach ($raw as $status => $count) {
            $counts[$status] = max(0, (int) $count);
        }

        return $counts;
    }

    public function topDishes(int $limit): array
    {
        /** @var array<string, string|float> $scores */
        $scores = $this->redis->command('zrevrange', [self::DISH_QTY, 0, $limit - 1, ['withscores' => true]]);
        if ($scores === []) {
            return [];
        }
        $ids = array_map('strval', array_keys($scores));
        /** @var list<string|null> $names */
        $names = $this->redis->command('hmget', [self::DISH_NAMES, $ids]);
        $dishes = [];
        foreach ($ids as $i => $id) {
            $revenue = $this->redis->command('zscore', [self::DISH_REVENUE, $id]);
            $dishes[] = [
                'menuItemId' => (int) $id,
                'name' => (string) ($names[$i] ?? 'Item #'.$id),
                'quantity' => (int) $scores[$id],
                'revenueCents' => (int) $revenue,
            ];
        }

        return $dishes;
    }

    public function recentOrders(int $limit): array
    {
        /** @var list<string> $rows */
        $rows = $this->redis->command('lrange', [self::RECENT, 0, $limit - 1]);
        $orders = [];
        foreach ($rows as $row) {
            $decoded = json_decode($row, true);
            if (is_array($decoded) && isset($decoded['id'])) {
                $orders[] = $decoded;
            }
        }
        if ($orders === []) {
            return [];
        }
        $statusKeys = array_map(static fn (array $o): string => 'stats:order:'.$o['id'].':status', $orders);
        /** @var list<string|null> $statuses */
        $statuses = $this->redis->command('mget', [$statusKeys]);
        foreach ($orders as $i => $order) {
            $orders[$i]['status'] = $statuses[$i] ?? null;
        }

        /** @var list<array<string, mixed>> $orders */
        return $orders;
    }

    public function workerHeartbeat(): ?array
    {
        $raw = $this->redis->command('get', [self::HEARTBEAT]);
        if (! is_string($raw)) {
            return null;
        }
        $decoded = json_decode($raw, true);

        /** @var array<string, mixed>|null */
        return is_array($decoded) ? $decoded : null;
    }

    public function queueDepth(): array
    {
        return [
            'pending' => (int) $this->redis->command('llen', [self::QUEUE]),
            'delayed' => (int) $this->redis->command('zcard', [self::DELAYED]),
            'deadLettered' => (int) $this->redis->command('llen', [self::DEAD]),
        ];
    }

    public function ping(): void
    {
        $this->redis->command('ping');
    }
}
