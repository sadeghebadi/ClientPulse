"use client";

import React from "react";

export function LoadingSkeleton({ className = "" }: { className?: string }) {
  return (
    <div
      className={`animate-pulse bg-slate-200/80 rounded-lg ${className}`}
      aria-hidden="true"
    />
  );
}

export function TableSkeleton({ rows = 5, cols = 5 }: { rows?: number; cols?: number }) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-4 space-y-4">
      <div className="flex items-center justify-between pb-3 border-b border-slate-100">
        <LoadingSkeleton className="h-5 w-32" />
        <LoadingSkeleton className="h-8 w-24" />
      </div>
      <div className="space-y-3">
        {Array.from({ length: rows }).map((_, r) => (
          <div key={r} className="flex items-center gap-4 py-2 border-b border-slate-50 last:border-0">
            {Array.from({ length: cols }).map((_, c) => (
              <LoadingSkeleton
                key={c}
                className={`h-4 ${c === 0 ? "w-1/3" : c === cols - 1 ? "w-16 ms-auto" : "w-1/4"}`}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export function MetricSkeleton() {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 sm:gap-6">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-3">
          <div className="flex items-center justify-between">
            <LoadingSkeleton className="h-3 w-20" />
            <LoadingSkeleton className="h-6 w-6 rounded-lg" />
          </div>
          <LoadingSkeleton className="h-8 w-16" />
          <LoadingSkeleton className="h-3 w-24" />
        </div>
      ))}
    </div>
  );
}

