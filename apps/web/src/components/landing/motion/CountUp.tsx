"use client";

import React, { useEffect, useRef, useState } from "react";
import { useReducedMotion } from "./useReducedMotion";

interface CountUpProps {
  end: number;
  start?: number;
  duration?: number; // ms, default 1600
  prefix?: string;
  suffix?: string;
  decimals?: number;
  className?: string;
}

export function CountUp({
  end,
  start = 0,
  duration = 1600,
  prefix = "",
  suffix = "",
  decimals = 0,
  className = "",
}: CountUpProps) {
  const [value, setValue] = useState(start);
  const elementRef = useRef<HTMLSpanElement>(null);
  const hasAnimated = useRef(false);
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (reducedMotion) {
      setValue(end);
      return;
    }

    const currentEl = elementRef.current;
    if (!currentEl) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry.isIntersecting && !hasAnimated.current) {
          hasAnimated.current = true;
          const startTime = performance.now();

          const step = (now: number) => {
            const elapsed = now - startTime;
            const progress = Math.min(elapsed / duration, 1);
            // Ease-out cubic formula
            const easeOutProgress = 1 - Math.pow(1 - progress, 3);
            const currentVal = start + (end - start) * easeOutProgress;

            setValue(currentVal);

            if (progress < 1) {
              requestAnimationFrame(step);
            } else {
              setValue(end);
            }
          };

          requestAnimationFrame(step);
        }
      },
      { threshold: 0.2 }
    );

    observer.observe(currentEl);

    return () => {
      observer.disconnect();
    };
  }, [end, start, duration, reducedMotion]);

  const formatted =
    decimals > 0
      ? value.toFixed(decimals)
      : Math.round(value).toLocaleString();

  return (
    <span ref={elementRef} className={`tabular-nums ${className}`}>
      {prefix}
      {formatted}
      {suffix}
    </span>
  );
}

