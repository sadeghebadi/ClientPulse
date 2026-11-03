import React from "react";

export function DashboardSkeleton() {
  return (
    <div className="space-y-6 animate-pulse" aria-busy="true" aria-label="Loading dashboard">
      {/* Header Skeleton */}
      <div className="bg-slate-100 rounded-2xl p-6 sm:p-8 h-32 border border-slate-200" />

      {/* Health Bar Skeleton */}
      <div className="bg-slate-100 rounded-2xl h-20 border border-slate-200" />

      {/* KPI Cards Skeleton */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="bg-white p-5 sm:p-6 rounded-2xl border border-slate-200 h-36"
          >
            <div className="h-4 bg-slate-100 rounded w-1/2 mb-3" />
            <div className="h-8 bg-slate-100 rounded w-1/3 mb-4" />
            <div className="h-3 bg-slate-100 rounded w-2/3" />
          </div>
        ))}
      </div>

      {/* Split View Skeleton */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Main Operational (65% / col-span-8) */}
        <div className="lg:col-span-8 space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-56" />
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-64" />
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-64" />
        </div>

        {/* Supporting Column (35% / col-span-4) */}
        <div className="lg:col-span-4 space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-48" />
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-64" />
          <div className="bg-white rounded-2xl border border-slate-200 p-6 h-64" />
        </div>
      </div>
    </div>
  );
}

