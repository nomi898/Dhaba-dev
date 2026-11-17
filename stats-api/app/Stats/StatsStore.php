<?php

declare(strict_types=1);

namespace App\Stats;

/**
 * Read access to the live statistics that kitchen-worker writes to Redis.
 */
interface StatsStore
{
    public function totalOrders(): int;

    public function totalRevenueCents(): int;

    /**
     * @param  list<string>  $minuteBuckets  bucket ids in yyyyMMddHHmm (UTC)
     * @return array<string, int> bucket id => order count
     */
    public function ordersPerMinuteBuckets(array $minuteBuckets): array;

    /** @return array<string, int> status => count */
    public function statusCounts(): array;

    /** @return list<array{menuItemId: int, name: string, quantity: int, revenueCents: int}> */
    public function topDishes(int $limit): array;

    /** @return list<array<string, mixed>> newest first, each including a live "status" */
    public function recentOrders(int $limit): array;

    /** @return array<string, mixed>|null decoded heartbeat, or null when no worker is alive */
    public function workerHeartbeat(): ?array;

    /** @return array{pending: int, delayed: int, deadLettered: int} */
    public function queueDepth(): array;

    public function ping(): void;
}
