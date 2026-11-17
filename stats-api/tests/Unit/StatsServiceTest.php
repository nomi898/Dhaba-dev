<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Stats\StatsService;
use Carbon\CarbonImmutable;
use PHPUnit\Framework\TestCase;
use Tests\Support\FakeStatsStore;

final class StatsServiceTest extends TestCase
{
    protected function tearDown(): void
    {
        CarbonImmutable::setTestNow();
        parent::tearDown();
    }

    public function test_orders_per_minute_uses_utc_minute_buckets(): void
    {
        $store = new FakeStatsStore;
        $store->minutes = [
            '202611171030' => 4,  // current, partial minute
            '202611171029' => 6,  // last complete minute
            '202611171028' => 2,
            '202611171020' => 3,
            '202611171000' => 99, // outside the 15-minute window
        ];
        $now = CarbonImmutable::parse('2026-11-17T10:30:42Z');

        $opm = (new StatsService($store))->ordersPerMinute($now);

        $this->assertSame(4, $opm['current']);
        $this->assertSame(6, $opm['lastMinute']);
        $this->assertSame(1.6, $opm['average5m']);       // (0+0+0+2+6)/5
        $this->assertSame(0.73, $opm['average15m']);     // 11/15
        $this->assertCount(16, $opm['series']);
        $this->assertSame('2026-11-17T10:15:00Z', $opm['series'][0]['minute']);
        $this->assertSame('2026-11-17T10:30:00Z', $opm['series'][15]['minute']);
    }

    public function test_summary_reports_worker_alive_with_age(): void
    {
        CarbonImmutable::setTestNow(CarbonImmutable::parse('2026-11-17T10:00:12Z'));
        $store = new FakeStatsStore;
        $store->total = 12;
        $store->revenue = 345000;
        $store->heartbeat = ['workerId' => 'kw-1', 'timestamp' => '2026-11-17T10:00:05Z', 'processed' => 40];

        $summary = (new StatsService($store))->summary();

        $this->assertSame(12, $summary['totalOrders']);
        $this->assertSame(345000, $summary['totalRevenueCents']);
        $this->assertTrue($summary['worker']['alive']);
        $this->assertSame(7, $summary['worker']['ageSeconds']);
        $this->assertSame('kw-1', $summary['worker']['workerId']);
        $this->assertSame(['pending' => 2, 'delayed' => 1, 'deadLettered' => 0], $summary['queue']);
    }

    public function test_summary_reports_worker_down_without_heartbeat(): void
    {
        $summary = (new StatsService(new FakeStatsStore))->summary();

        $this->assertFalse($summary['worker']['alive']);
        $this->assertSame(0, $summary['ordersPerMinute']['current']);
    }
}
