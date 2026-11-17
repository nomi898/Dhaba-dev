<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Stats\StatsStore;
use RuntimeException;

final class FakeStatsStore implements StatsStore
{
    public int $total = 0;

    public int $revenue = 0;

    /** @var array<int|string, int> minute bucket => count (PHP casts numeric-string keys to int) */
    public array $minutes = [];

    /** @var array<string, int> */
    public array $statuses = ['NEW' => 0, 'COOKING' => 0, 'READY' => 0];

    /** @var list<array{menuItemId: int, name: string, quantity: int, revenueCents: int}> */
    public array $dishes = [];

    /** @var list<array<string, mixed>> */
    public array $recent = [];

    /** @var array<string, mixed>|null */
    public ?array $heartbeat = null;

    public bool $down = false;

    public ?int $lastTopLimit = null;

    public function totalOrders(): int
    {
        return $this->total;
    }

    public function totalRevenueCents(): int
    {
        return $this->revenue;
    }

    public function ordersPerMinuteBuckets(array $minuteBuckets): array
    {
        $out = [];
        foreach ($minuteBuckets as $b) {
            $out[$b] = $this->minutes[$b] ?? 0;
        }

        return $out;
    }

    public function statusCounts(): array
    {
        return $this->statuses;
    }

    public function topDishes(int $limit): array
    {
        $this->lastTopLimit = $limit;

        return array_slice($this->dishes, 0, $limit);
    }

    public function recentOrders(int $limit): array
    {
        return array_slice($this->recent, 0, $limit);
    }

    public function workerHeartbeat(): ?array
    {
        return $this->heartbeat;
    }

    public function queueDepth(): array
    {
        return ['pending' => 2, 'delayed' => 1, 'deadLettered' => 0];
    }

    public function ping(): void
    {
        if ($this->down) {
            throw new RuntimeException('Connection refused');
        }
    }
}
