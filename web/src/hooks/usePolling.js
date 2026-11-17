import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Calls `fetcher(signal)` immediately and then every `intervalMs`, pausing while the tab is hidden.
 * Keeps the last good data visible if a refresh fails.
 */
export function usePolling(fetcher, intervalMs, deps = []) {
  const [state, setState] = useState({ data: null, error: null, loading: true, updatedAt: null });
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  const [tick, setTick] = useState(0);

  const refresh = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    let cancelled = false;
    let timer;
    let controller;

    const run = async () => {
      controller = new AbortController();
      try {
        const data = await fetcherRef.current(controller.signal);
        if (!cancelled) setState({ data, error: null, loading: false, updatedAt: new Date() });
      } catch (error) {
        if (!cancelled && !controller.signal.aborted) setState((s) => ({ ...s, error, loading: false }));
      }
      if (!cancelled && intervalMs > 0) timer = setTimeout(schedule, intervalMs);
    };
    const schedule = () => {
      if (document.hidden) {
        timer = setTimeout(schedule, intervalMs);
        return;
      }
      run();
    };
    run();
    return () => {
      cancelled = true;
      clearTimeout(timer);
      controller?.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, tick, ...deps]);

  return { ...state, refresh };
}
