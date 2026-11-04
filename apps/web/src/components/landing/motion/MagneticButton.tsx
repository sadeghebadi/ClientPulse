"use client";

import React, { useRef, useState, useEffect } from "react";
import { useReducedMotion } from "./useReducedMotion";

interface MagneticProps {
  children: React.ReactNode;
  className?: string;
  strength?: number; // 0 to 1, default 0.25
}

export function MagneticWrapper({
  children,
  className = "",
  strength = 0.28,
}: MagneticProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const reducedMotion = useReducedMotion();
  const [canHover, setCanHover] = useState(true);

  useEffect(() => {
    if (typeof window !== "undefined") {
      setCanHover(window.matchMedia("(hover: hover)").matches);
    }
  }, []);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (reducedMotion || !canHover || !ref.current) return;
    const { left, top, width, height } = ref.current.getBoundingClientRect();
    const centerX = left + width / 2;
    const centerY = top + height / 2;
    const deltaX = (e.clientX - centerX) * strength;
    const deltaY = (e.clientY - centerY) * strength;
    setPosition({ x: deltaX, y: deltaY });
  };

  const handleMouseLeave = () => {
    setPosition({ x: 0, y: 0 });
  };

  return (
    <div
      ref={ref}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={`inline-block transition-transform duration-200 ease-out ${className}`}
      style={{
        transform:
          !reducedMotion && canHover
            ? `translate3d(${position.x}px, ${position.y}px, 0)`
            : undefined,
      }}
    >
      {children}
    </div>
  );
}

