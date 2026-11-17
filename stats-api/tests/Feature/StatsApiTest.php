<?php

declare(strict_types=1);

namespace Tests\Feature;

use App\Stats\StatsStore;
use Tests\Support\FakeStatsStore;
use Tests\TestCase;

final class StatsApiTest extends TestCase
{
    private FakeStatsStore $store;

    protected function setUp(): void
    {
        parent::setUp();
        $this->store = new FakeStatsStore;
        $this->store->total = 3;
        $this->store->dishes = [
            ['menuItemId' => 12, 'name' => 'Chicken Biryani', 'quantity' => 9, 'revenueCents' => 540000],
            ['menuItemId' => 17, 'name' => 'Doodh Patti Chai', 'quantity' => 5, 'revenueCents' => 50000],
        ];
        $this->store->recent = [['id' => 3, 'customerName' => 'Ali', 'totalCents' => 60000, 'status' => 'COOKING']];
        $this->app->instance(StatsStore::class, $this->store);
    }

    public function test_stats_summary(): void
    {
        $this->getJson('/api/stats')
            ->assertOk()
            ->assertJsonPath('data.totalOrders', 3)
            ->assertJsonPath('data.topDishes.0.name', 'Chicken Biryani')
            ->assertJsonPath('data.recentOrders.0.status', 'COOKING')
            ->assertJsonStructure(['data' => [
                'ordersPerMinute' => ['current', 'lastMinute', 'average5m', 'series'],
                'statusCounts' => ['NEW', 'COOKING', 'READY'],
                'worker' => ['alive'],
                'queue' => ['pending', 'delayed', 'deadLettered'],
                'generatedAt',
            ]]);
    }

    public function test_top_dishes_default_and_custom_limit(): void
    {
        $this->getJson('/api/stats/top-dishes')->assertOk()->assertJsonCount(2, 'data');
        $this->assertSame(5, $this->store->lastTopLimit);

        $this->getJson('/api/stats/top-dishes?limit=1')
            ->assertOk()
            ->assertJsonCount(1, 'data')
            ->assertJsonPath('data.0.quantity', 9);
    }

    public function test_top_dishes_rejects_invalid_limit(): void
    {
        foreach (['0', '51', 'abc'] as $bad) {
            $this->getJson('/api/stats/top-dishes?limit='.$bad)
                ->assertStatus(422)
                ->assertJsonPath('error.code', 'validation_error');
        }
    }

    public function test_health_ok_and_degraded(): void
    {
        $this->getJson('/health')->assertOk()->assertJson(['status' => 'ok', 'checks' => ['redis' => 'ok']]);

        $this->store->down = true;
        $this->getJson('/health')->assertStatus(503)->assertJsonPath('status', 'degraded');
    }

    public function test_unknown_route_uses_error_envelope(): void
    {
        $this->getJson('/api/nope')->assertNotFound()->assertJsonPath('error.code', 'not_found');
        $this->postJson('/api/stats')->assertStatus(405)->assertJsonPath('error.code', 'method_not_allowed');
    }
}
