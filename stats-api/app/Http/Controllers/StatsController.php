<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Stats\StatsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

final class StatsController
{
    public function __construct(private readonly StatsService $stats) {}

    public function summary(): JsonResponse
    {
        return response()->json(['data' => $this->stats->summary()]);
    }

    public function topDishes(Request $request): JsonResponse
    {
        /** @var array{limit?: int|string} $validated */
        $validated = $request->validate(['limit' => ['sometimes', 'integer', 'between:1,50']]);
        $limit = (int) ($validated['limit'] ?? StatsService::DEFAULT_TOP);

        return response()->json(['data' => $this->stats->topDishes($limit)]);
    }
}
