import { useVirtualizer } from '@tanstack/react-virtual';
import { forwardRef, useImperativeHandle, useRef, type ReactElement, type ReactNode, type Ref } from 'react';

export interface VirtualListRef {
  scrollToEnd(): void;
}

interface VirtualListProps<T> {
  data: T[];
  keyExtractor: (item: T) => string;
  renderItem: (item: T) => ReactNode;
  /** Rough row height before it's actually measured; rows resize themselves (e.g. expanded log lines). */
  estimateSize?: () => number;
}

/** Scrollable, virtualized row list -- the web equivalent of @shopify/flash-list. */
function VirtualListInner<T>(
  { data, keyExtractor, renderItem, estimateSize = () => 44 }: VirtualListProps<T>,
  ref: Ref<VirtualListRef>,
) {
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize,
    getItemKey: (index) => keyExtractor(data[index]),
    overscan: 8,
  });

  useImperativeHandle(
    ref,
    () => ({
      scrollToEnd: () => {
        if (data.length > 0) virtualizer.scrollToIndex(data.length - 1, { align: 'end' });
      },
    }),
    [virtualizer, data.length],
  );

  return (
    <div ref={parentRef} style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
      <div style={{ height: virtualizer.getTotalSize(), width: '100%', position: 'relative' }}>
        {virtualizer.getVirtualItems().map((vi) => (
          <div
            key={vi.key}
            data-index={vi.index}
            ref={virtualizer.measureElement}
            style={{ position: 'absolute', top: 0, left: 0, width: '100%', transform: `translateY(${vi.start}px)` }}>
            {renderItem(data[vi.index])}
          </div>
        ))}
      </div>
    </div>
  );
}

export const VirtualList = forwardRef(VirtualListInner) as unknown as <T>(
  props: VirtualListProps<T> & { ref?: Ref<VirtualListRef> },
) => ReactElement;
