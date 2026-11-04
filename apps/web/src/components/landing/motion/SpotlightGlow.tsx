"use client";

import React, { useRef, useState, useEffect } from "react";
import { useReducedMotion } from "./useReducedMotion";

interface SpotlightGlowProps {
  className?: string;
  size?: number; // radius in px, default 400
  color?: string; // default electric blue / violet
}

export function SpotlightGlow({
  className = "",
  size = 420,
  color = "rgba(59, 130, 246, 0.12)",
}: SpotlightGlowProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [coords, setCoords] = useState({ x: -1000, y: -1000 });
  const [opacity, setOpacity] = useState(0);
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (reducedMotion) return;

    const handlePointerMove = (e: PointerEvent) => {
      if (!containerRef.current) return;
      const rect = containerRef.current.parentElement?.getBoundingClientRect();
      if (!rect) return;

      if (
        e.clientX >= rect.left &&
        e.clientX <= rect.right &&
        e.clientY >= rect.top &&
        e.clientY <= rect.bottom
      ) {
        setCoords({
          x: e.clientX - rect.left,
          y: e.clientY - rect.top,
        });
        setOpacity(1);
      } else {
        setOpacity(0);
      }
    };

    window.addEventListener("pointermove", handlePointerMove, { passive: true });
    return () => window.removeEventListener("pointermove", handlePointerMove);
  }, [reducedMotion]);

  if (reducedMotion) return null;

  return (
    <div
      ref={containerRef}
      className={`pointer-events-none absolute inset-0 z-0 overflow-hidden transition-opacity duration-500 ease-out ${className}`}
      style={{ opacity }}
    >
      <div
        className="absolute rounded-full blur-3xl transform -translate-x-1/2 -translate-y-1/2 will-change-transform"
        style={{
          left: `${coords.x}px`,
          top: `${coords.y}px`,
          width: `${size * 2}px`,
          height: `${size * 2}px`,
          background: `radial-gradient(circle, ${color} 0%, rgba(139, 92, 246, 0.06) 40%, transparent 70%)`,
        }}
      />
    </div>
  );
}

