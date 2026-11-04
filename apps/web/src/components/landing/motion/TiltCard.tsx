"use client";

import React, { useRef, useState, useEffect } from "react";
import { useReducedMotion } from "./useReducedMotion";

interface TiltCardProps {
  children: React.ReactNode;
  className?: string;
  maxTilt?: number; // max tilt angle in degrees, default 6
  sheen?: boolean;
}

export function TiltCard({
  children,
  className = "",
  maxTilt = 6,
  sheen = true,
}: TiltCardProps) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [transform, setTransform] = useState("");
  const [sheenPos, setSheenPos] = useState({ x: 50, y: 50, opacity: 0 });
  const reducedMotion = useReducedMotion();
  const [canHover, setCanHover] = useState(true);

  useEffect(() => {
    if (typeof window !== "undefined") {
      setCanHover(window.matchMedia("(hover: hover)").matches);
    }
  }, []);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (reducedMotion || !canHover || !cardRef.current) return;
    const rect = cardRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const width = rect.width;
    const height = rect.height;

    const xRatio = (x / width - 0.5) * 2; // -1 to 1
    const yRatio = (y / height - 0.5) * 2; // -1 to 1

    const rotateX = -yRatio * maxTilt;
    const rotateY = xRatio * maxTilt;

    setTransform(
      `perspective(1000px) rotateX(${rotateX.toFixed(2)}deg) rotateY(${rotateY.toFixed(2)}deg) translateZ(4px)`
    );

    if (sheen) {
      setSheenPos({
        x: (x / width) * 100,
        y: (y / height) * 100,
        opacity: 0.15,
      });
    }
  };

  const handleMouseLeave = () => {
    setTransform("perspective(1000px) rotateX(0deg) rotateY(0deg) translateZ(0px)");
    setSheenPos((prev) => ({ ...prev, opacity: 0 }));
  };

  return (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={`relative transition-transform duration-300 ease-out will-change-transform ${className}`}
      style={{
        transform: !reducedMotion && canHover ? transform : undefined,
      }}
    >
      {sheen && !reducedMotion && canHover && (
        <div
          className="pointer-events-none absolute inset-0 rounded-[inherit] transition-opacity duration-300 z-20"
          style={{
            opacity: sheenPos.opacity,
            background: `radial-gradient(circle at ${sheenPos.x}% ${sheenPos.y}%, rgba(99, 102, 241, 0.45), transparent 70%)`,
          }}
        />
      )}
      {children}
    </div>
  );
}

