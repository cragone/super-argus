import { useEffect } from 'react';

/** Sets `document.title` to `"<title> — Argus"` for the lifetime of the calling screen. */
export function useDocumentTitle(title: string) {
  useEffect(() => {
    const prev = document.title;
    document.title = title ? `${title} — Argus` : 'Argus';
    return () => {
      document.title = prev;
    };
  }, [title]);
}
