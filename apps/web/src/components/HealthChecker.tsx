"use client";

import React, { useEffect, useState } from "react";
import { CheckCircle2, AlertTriangle, RefreshCw, Server, Database, Cpu } from "lucide-react";

interface HealthStatus {
  live: {
    status?: string;
    service?: string;
    version?: string;
    timestamp?: string;
    error?: string;
  } | null;
  ready: {
    status?: string;
    service?: string;
    database?: string;
    timestamp?: string;
    error?: string;
  } | null;
  worker: {
    status?: string;
    service?: string;
    worker_id?: string;
    last_heartbeat_at?: string;
    age_seconds?: number;
    timeout_seconds?: number;
    error?: string;
  } | null;
  loading: boolean;
  lastChecked: string | null;
}

interface HealthCheckerProps {
  locale: "en" | "fa";
}

export function HealthChecker({ locale }: HealthCheckerProps) {
  const isRTL = locale === "fa";
  const [health, setHealth] = useState<HealthStatus>({
    live: null,
    ready: null,
    worker: null,
    loading: true,
    lastChecked: null,
  });

  const checkHealth = async () => {
    setHealth((prev) => ({ ...prev, loading: true }));
    const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

    let liveData = null;
    let readyData = null;
    let workerData = null;

    try {
      const liveRes = await fetch(`${apiBase}/health/live`, { cache: "no-store" });
      if (liveRes.ok) {
        liveData = await liveRes.json();
      } else {
        liveData = { error: `HTTP ${liveRes.status}` };
      }
    } catch (err: unknown) {
      liveData = { error: err instanceof Error ? err.message : "Network error" };
    }

    try {
      const readyRes = await fetch(`${apiBase}/health/ready`, { cache: "no-store" });
      if (readyRes.ok) {
        readyData = await readyRes.json();
      } else {
        readyData = await readyRes.json().catch(() => ({ error: `HTTP ${readyRes.status}` }));
      }
    } catch (err: unknown) {
      readyData = { error: err instanceof Error ? err.message : "Network error" };
    }

    try {
      const workerRes = await fetch(`${apiBase}/health/worker`, { cache: "no-store" });
      if (workerRes.ok) {
        workerData = await workerRes.json();
      } else {
        workerData = await workerRes.json().catch(() => ({ error: `HTTP ${workerRes.status}` }));
      }
    } catch (err: unknown) {
      workerData = { error: err instanceof Error ? err.message : "Network error" };
    }

    setHealth({
      live: liveData,
      ready: readyData,
      worker: workerData,
      loading: false,
      lastChecked: new Date().toLocaleTimeString(),
    });
  };

  useEffect(() => {
    checkHealth();
  }, []);

  const isLiveOk = health.live?.status === "ok";
  const isReadyOk = health.ready?.status === "ok" && health.ready?.database === "connected";
  const isWorkerOk = health.worker?.status === "ok";

  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-6 mb-8">
      <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-100">
        <div className="flex items-center gap-2">
          <Server className="w-5 h-5 text-blue-600" />
          <h2 className="text-base font-semibold text-slate-900">
            {isRTL ? "بررسی زنده وضعیت زیرساخت و ارتباطات" : "Live Infrastructure & Health Status"}
          </h2>
        </div>
        <button
          onClick={checkHealth}
          disabled={health.loading}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-700 bg-slate-50 hover:bg-slate-100 border border-slate-200 rounded-md transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${health.loading ? "animate-spin" : ""}`} />
          <span>{isRTL ? "بررسی مجدد" : "Refresh"}</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* API Liveness */}
        <div className="p-4 rounded-lg bg-slate-50 border border-slate-200">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
              API Liveness (/health/live)
            </span>
            {isLiveOk ? (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                <CheckCircle2 className="w-4 h-4" /> Healthy
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-amber-600">
                <AlertTriangle className="w-4 h-4" /> {health.loading ? "Checking..." : "Unavailable"}
              </span>
            )}
          </div>
          <div className="text-sm font-mono text-slate-700 space-y-1">
            <p>Service: {health.live?.service || "clientpulse-api"}</p>
            <p>Version: {health.live?.version || "1.0.0"}</p>
            {health.live?.error && (
              <p className="text-xs text-rose-600 font-sans">Error: {health.live.error}</p>
            )}
          </div>
        </div>

        {/* Database Readiness */}
        <div className="p-4 rounded-lg bg-slate-50 border border-slate-200">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 flex items-center gap-1">
              <Database className="w-3.5 h-3.5" /> DB Readiness (/health/ready)
            </span>
            {isReadyOk ? (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                <CheckCircle2 className="w-4 h-4" /> Connected
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-amber-600">
                <AlertTriangle className="w-4 h-4" /> {health.loading ? "Checking..." : "Disconnected"}
              </span>
            )}
          </div>
          <div className="text-sm font-mono text-slate-700 space-y-1">
            <p>Database: {health.ready?.database || (health.loading ? "checking" : "unreachable")}</p>
            <p>Checked at: {health.lastChecked || "pending"}</p>
            {health.ready?.error && (
              <p className="text-xs text-rose-600 font-sans">Error: {health.ready.error}</p>
            )}
          </div>
        </div>

        {/* Worker Daemon Health */}
        <div className="p-4 rounded-lg bg-slate-50 border border-slate-200">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 flex items-center gap-1">
              <Cpu className="w-3.5 h-3.5" /> Worker Probe (/health/worker)
            </span>
            {isWorkerOk ? (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                <CheckCircle2 className="w-4 h-4" /> Alive & Active
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-xs font-semibold text-rose-600">
                <AlertTriangle className="w-4 h-4" /> {health.loading ? "Checking..." : "Unhealthy"}
              </span>
            )}
          </div>
          <div className="text-sm font-mono text-slate-700 space-y-1">
            <p className="truncate">ID: {health.worker?.worker_id || "none"}</p>
            <p>Age: {health.worker?.age_seconds !== undefined ? `${health.worker.age_seconds}s` : "n/a"}</p>
            {health.worker?.error && (
              <p className="text-xs text-rose-600 font-sans">Error: {health.worker.error}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
