import { useEffect, useRef, useState } from "react";

type Props<T> = {
  placeholder?: string;
  value: string;
  onValueChange: (v: string) => void;
  search: (q: string) => Promise<T[]>;
  renderOption: (item: T) => React.ReactNode;
  onSelect: (item: T) => void;
  keyOf: (item: T) => string;
  debounceMs?: number;
};

export function Autocomplete<T>({
  placeholder, value, onValueChange,
  search, renderOption, onSelect, keyOf,
  debounceMs = 200,
}: Props<T>) {
  const [results, setResults] = useState<T[]>([]);
  const [open, setOpen] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const boxRef = useRef<HTMLDivElement>(null);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(async () => {
      try {
        const r = await search(value);
        setResults(r);
        setHighlighted(0);
      } catch {
        setResults([]);
      }
    }, debounceMs);
    return () => {
      if (timer.current) window.clearTimeout(timer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  function onKeyDown(e: React.KeyboardEvent) {
    if (!open) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlighted((h) => Math.min(h + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlighted((h) => Math.max(h - 1, 0));
    } else if (e.key === "Enter" && results[highlighted]) {
      e.preventDefault();
      onSelect(results[highlighted]);
      setOpen(false);
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  }

  return (
    <div className="relative" ref={boxRef}>
      <input
        className="input"
        placeholder={placeholder}
        value={value}
        onChange={(e) => {
          onValueChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={onKeyDown}
      />
      {open && results.length > 0 && (
        <div className="absolute z-30 mt-1 w-full bg-white border border-slate-200 rounded-lg shadow-lg max-h-64 overflow-auto">
          {results.map((item, i) => (
            <button
              key={keyOf(item)}
              type="button"
              className={`block w-full text-left px-3 py-2 text-sm hover:bg-brand-50 ${
                i === highlighted ? "bg-brand-50" : ""
              }`}
              onMouseEnter={() => setHighlighted(i)}
              onClick={() => {
                onSelect(item);
                setOpen(false);
              }}
            >
              {renderOption(item)}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
