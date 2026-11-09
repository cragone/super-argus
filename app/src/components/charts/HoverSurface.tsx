export interface HoverSurfaceProps {
  width: number;
  height: number;
  /** x in px relative to the surface's left edge, or null when the pointer leaves. */
  onMove(x: number | null): void;
}

/** Transparent layer that reports the pointer x position: real mouse hover, plus touch. */
export function HoverSurface({ width, height, onMove }: HoverSurfaceProps) {
  const rel = (e: { currentTarget: HTMLElement; clientX: number }) => e.clientX - e.currentTarget.getBoundingClientRect().left;
  return (
    <div
      role="presentation"
      style={{ position: 'absolute', left: 0, top: 0, width, height, cursor: 'crosshair', touchAction: 'pan-y' }}
      onMouseMove={(e) => onMove(rel(e))}
      onMouseLeave={() => onMove(null)}
      onTouchStart={(e) => onMove(e.touches[0].clientX - e.currentTarget.getBoundingClientRect().left)}
      onTouchMove={(e) => onMove(e.touches[0].clientX - e.currentTarget.getBoundingClientRect().left)}
      onTouchEnd={() => onMove(null)}
    />
  );
}
