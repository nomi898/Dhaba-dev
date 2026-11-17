<?php

declare(strict_types=1);

namespace App\Stats;

use Carbon\CarbonImmutable;

/**
 * Builds statistics responses from the raw counters in the store.
 */
final class StatsService
{
    public const SERIES_MINUTES = 15;

    public const RECENT_LIMIT = 10;

    public const DEFAULT_TOP = 5;

    public function __construct(private readonly StatsStore $store) {}

    /** @return array<string, mixed> */
    public function summary(): array
    {
        $now = CarbonImmutable::now('UTC');

        return [
            'totalOrders' => $this->store->totalOrders(),
            'totalRevenueCents' => $this->store->totalRevenueCents(),
            'ordersPerMinute' => $this->ordersPerMinute($now),
            'statusCounts' => $this->store->statusCounts(),
            'topDishes' => $this->store->topDishes(self::DEFAULT_TOP),
            'recentOrders' => $this->store->recentOrders(self::RECENT_LIMIT),
            'worker' => $this->worker($now),
            'queue' => $this->store->queueDepth(),
            'generatedAt' => $now->toIso8601ZuluString(),
        ];
    }

    /** @return list<array{menuItemId: int, name: string, quantity: int, revenueCents: int}> */
    public function topDishes(int $limit): array
    {
        return $this->store->topDishes($limit);
    }

    /**
     * Orders per minute: the current (partial) minute, the last complete minute, rolling averages over complete
     * minutes, and a per-minute series (oldest first) for charting.
     *
     * @return array<string, mixed>
     */
    public function ordersPerMinute(CarbonImmutable $now): array
    {
        $current = $now->startOfMinute();
        $buckets = [];
        for ($i = self::SERIES_MINUTES; $i >= 0; $i--) {
            $buckets[] = $current->subMinutes($i)->format('YmdHi');
        }
        $counts = $this->store->ordersPerMinuteBuckets($buckets);

        $series = [];
        foreach ($buckets as $i => $bucket) {
            $series[] = [
                'minute' => $current->subMinutes(self::SERIES_MINUTES - $i)->toIso8601ZuluString(),
                'count' => $counts[$bucket] ?? 0,
            ];
        }
        $complete = array_column(array_slice($series, 0, -1), 'count');
        $last5 = array_slice($complete, -5);

        return [
            'current' => $series[count($series) - 1]['count'],
            'lastMinute' => $complete[count($complete) - 1] ?? 0,
            'average5m' => round(array_sum($last5) / max(count($last5), 1), 2),
            'average15m' => round(array_sum($complete) / max(count($complete), 1), 2),
            'series' => $series,
        ];
    }

    /** @return array<string, mixed> */
    private function worker(CarbonImmutable $now): array
    {
        $hb = $this->store->workerHeartbeat();
        if ($hb === null) {
            return ['alive' => false, 'workerId' => null, 'lastHeartbeat' => null, 'ageSeconds' => null];
        }
        $timestamp = is_string($hb['timestamp'] ?? null) ? $hb['timestamp'] : null;
        $age = $timestamp !== null ? max(0, $now->getTimestamp() - CarbonImmutable::parse($timestamp)->getTimestamp())
            : null;

        return [
            'alive' => true,
            'workerId' => $hb['workerId'] ?? null,
            'lastHeartbeat' => $timestamp,
            'ageSeconds' => $age,
            'processed' => $hb['processed'] ?? null,
            'deadLettered' => $hb['deadLettered'] ?? null,
        ];
    }
}
