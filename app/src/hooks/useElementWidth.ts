import { useLayoutEffect, useRef, useState } from 'react';

/**
 * Tracks an element's content width. Reads synchronously on first paint (matching RN's
 * synchronous `onLayout`) via `useLayoutEffect`, then `ResizeObserver` for later resizes.
 */
export function useElementWidth<T extends HTMLElement>(): [React.RefObject<T | null>, number] {
  const ref = useRef<T | null>(null);
  const [width, setWidth] = useState(0);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    setWidth(Math.floor(el.getBoundingClientRect().width));
    const ro = new ResizeObserver(([entry]) => setWidth(Math.floor(entry.contentRect.width)));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return [ref, width];
}
