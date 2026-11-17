<?php

use App\Http\Controllers\HealthController;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Route;
use Illuminate\Validation\ValidationException;
use Symfony\Component\HttpKernel\Exception\HttpExceptionInterface;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        api: __DIR__.'/../routes/api.php',
        commands: __DIR__.'/../routes/console.php',
        then: function (): void {
            Route::get('/health', HealthController::class);
        },
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->trustProxies(at: '*');
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        $exceptions->shouldRenderJsonWhen(fn (): bool => true);

        $exceptions->render(function (ValidationException $e) {
            return response()->json(['error' => [
                'code' => 'validation_error',
                'message' => $e->getMessage(),
                'fields' => array_map(fn (array $m) => $m[0], $e->errors()),
            ]], 422);
        });

        $exceptions->render(function (Throwable $e, Request $request) {
            if ($e instanceof ValidationException) {
                return null;
            }
            $status = $e instanceof HttpExceptionInterface ? $e->getStatusCode() : 500;
            $code = match (true) {
                $status === 404 => 'not_found',
                $status === 405 => 'method_not_allowed',
                $status >= 500 => 'internal_error',
                default => 'http_error',
            };
            $message = $status >= 500 && ! config('app.debug') ? 'internal server error'
                : ($e->getMessage() !== '' ? $e->getMessage() : 'request failed');

            return response()->json(['error' => ['code' => $code, 'message' => $message]], $status);
        });
    })->create();
