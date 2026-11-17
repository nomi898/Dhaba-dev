<?php

use App\Http\Controllers\StatsController;
use Illuminate\Support\Facades\Route;

Route::get('/stats', [StatsController::class, 'summary']);
Route::get('/stats/top-dishes', [StatsController::class, 'topDishes']);
