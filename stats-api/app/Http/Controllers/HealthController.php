<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Stats\StatsStore;
use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\Log;
use Throwable;

final class HealthController
{
    public function __invoke(StatsStore $store): JsonResponse
    {
        try {
            $store->ping();
            $redis = 'ok';
        } catch (Throwable $e) {
            Log::warning('health check: redis unavailable', ['error' => $e->getMessage()]);
            $redis = 'error: '.class_basename($e);
        }
        $healthy = $redis === 'ok';

        return response()->json([
            'status' => $healthy ? 'ok' : 'degraded',
            'service' => 'stats-api',
            'version' => (string) config('app.version'),
            'checks' => ['redis' => $redis],
        ], $healthy ? 200 : 503);
    }
}
