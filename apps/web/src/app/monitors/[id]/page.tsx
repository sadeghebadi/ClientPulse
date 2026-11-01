"use client";

import React, { useEffect, useState, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Activity,
  Play,
  Pause,
  ExternalLink,
  Edit,
  Shield,
  ShieldAlert,
  ShieldCheck,
  CheckCircle,
  XCircle,
  Clock,
  PauseCircle,
  ArrowLeft,
  ArrowRight,
  AlertCircle,
  Zap,
  RotateCcw,
  Globe,
  Building,
} from "lucide-react";

interface MonitorDetail {
  id: string;
  asset_name: string;
  asset_url: string;
  client_name: string;
  client_id: string;
  effective_url: string;
  method: string;
  interval_seconds: number;
  timeout_seconds: number;
  expected_status_min: number;
  expected_status_max: number;
  follow_redirects: boolean;
  max_redirects: number;
  expected_text?: string;
  json_pointer?: string;
  json_expected_value?: string;
  status: "up" | "down" | "pending" | "paused";
  is_active: boolean;
  next_check_at: string;
  last_checked_at?: string;
}

interface CheckResultItem {
  id: string;
  scheduled_at: string;
  completed_at: string;
  latency_ms: number;
  result: "success" | "failure" | "timeout" | "blocked" | "error";
  http_status_code?: number;
  error_category?: string;
  error_message?: string;
  tls_version?: string;
  tls_cipher?: string;
  tls_cert_issuer?: string;
  tls_cert_subject?: string;
  tls_days_remaining?: number;
  tls_is_valid?: boolean;
  execution_type: "scheduled" | "manual";
}

export default function MonitorDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();

  const [monitor, setMonitor] = useState<MonitorDetail | null>(null);
  const [history, setHistory] = useState<CheckResultItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<CheckResultItem | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadMonitorData = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      apiFetch<{ monitor: MonitorDetail }>(`/api/v1/monitors/${resolvedParams.id}`),
      apiFetch<{ results: CheckResultItem[] }>(`/api/v1/monitors/${resolvedParams.id}/history?limit=20`),
    ])
      .then(([monRes, histRes]) => {
        setMonitor(monRes.monitor);
        setHistory(histRes.results || []);
      })
      .catch((err) => {
        setError(err?.message || "Failed to load monitor details");
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<{ monitor: MonitorDetail }>(`/api/v1/monitors/${resolvedParams.id}`),
      apiFetch<{ results: CheckResultItem[] }>(`/api/v1/monitors/${resolvedParams.id}/history?limit=20`),
    ])
      .then(([monRes, histRes]) => {
        if (isMounted) {
          setMonitor(monRes.monitor);
          setHistory(histRes.results || []);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err?.message || "Failed to load monitor details");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, resolvedParams.id]);

  const handleTestNow = async () => {
    try {
      setTesting(true);
      setTestResult(null);
      const res = await apiFetch<{ result: CheckResultItem }>(`/api/v1/monitors/${resolvedParams.id}/test`, {
        method: "POST",
      });
      setTestResult(res.result);
      // Refresh history
      const hist = await apiFetch<{ results: CheckResultItem[] }>(`/api/v1/monitors/${resolvedParams.id}/history?limit=20`);
      setHistory(hist.results || []);
    } catch (err: any) {
      alert(err?.message || "Test check failed to execute");
    } finally {
      setTesting(false);
    }
  };

  const handleTogglePause = async () => {
    if (!monitor) return;
    try {
      setActionLoading(true);
      const endpoint = monitor.status === "paused"
        ? `/api/v1/monitors/${monitor.id}/resume`
        : `/api/v1/monitors/${monitor.id}/pause`;
      await apiFetch(endpoint, { method: "POST" });
      reloadMonitorData();
    } catch (err: any) {
      alert(err?.message || "Action failed");
    } finally {
      setActionLoading(false);
    }
  };

  // Find latest SSL info if available from latest check
  const latestSSL = history.find((h) => h.tls_cert_issuer || h.tls_cert_subject);

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        {/* Breadcrumb */}
        <div className="mb-6 flex items-center justify-between">
          <Link
            href="/monitors"
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{t.monitors}</span>
          </Link>

          {monitor && (
            <div className="flex items-center gap-2">
              <button
                onClick={handleTestNow}
                disabled={testing}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
              >
                <Zap className="w-3.5 h-3.5 text-amber-300" />
                <span>{testing ? t.testing : t.testNow}</span>
              </button>

              <button
                onClick={handleTogglePause}
                disabled={actionLoading}
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 rounded-lg text-xs font-semibold transition-colors shadow-sm"
              >
                {monitor.status === "paused" ? (
                  <>
                    <Play className="w-3.5 h-3.5 text-emerald-600" />
                    <span>{t.resumeMonitor}</span>
                  </>
                ) : (
                  <>
                    <Pause className="w-3.5 h-3.5 text-slate-500" />
                    <span>{t.pauseMonitor}</span>
                  </>
                )}
              </button>

              <Link
                href={`/monitors/${monitor.id}/edit`}
                className="inline-flex items-center gap-1 px-3 py-1.5 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 rounded-lg text-xs font-semibold transition-colors shadow-sm"
              >
                <Edit className="w-3.5 h-3.5" />
                <span>{t.editMonitor}</span>
              </Link>
            </div>
          )}
        </div>

        {error && (
          <div className="p-4 mb-6 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Test Result Toast/Banner */}
        {testResult && (
          <div
            className={`p-4 mb-6 rounded-xl border flex items-center justify-between gap-4 ${
              testResult.result === "success"
                ? "bg-emerald-50 border-emerald-200 text-emerald-800"
                : "bg-rose-50 border-rose-200 text-rose-800"
            }`}
          >
            <div className="flex items-center gap-3">
              {testResult.result === "success" ? (
                <CheckCircle className="w-5 h-5 text-emerald-600 flex-shrink-0" />
              ) : (
                <XCircle className="w-5 h-5 text-rose-600 flex-shrink-0" />
              )}
              <div>
                <div className="text-xs font-bold uppercase tracking-wider">
                  {testResult.result === "success" ? t.testSuccess : t.testFailed}
                </div>
                <div className="text-xs mt-0.5 opacity-90">
                  HTTP {testResult.http_status_code || "—"} • {testResult.latency_ms} ms
                  {testResult.error_message && ` • ${testResult.error_message}`}
                </div>
              </div>
            </div>
            <button
              onClick={() => setTestResult(null)}
              className="text-xs font-semibold px-2 py-1 hover:bg-black/5 rounded"
            >
              Dismiss
            </button>
          </div>
        )}

        {loading ? (
          <div className="p-12 text-center text-slate-500 text-sm bg-white rounded-xl border border-slate-200">
            {t.loading}
          </div>
        ) : monitor ? (
          <div className="space-y-8">
            {/* Monitor Summary Card */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 pb-6 border-b border-slate-100">
                <div>
                  <div className="flex items-center gap-3">
                    <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{monitor.asset_name}</h1>
                    <span
                      className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold ${
                        monitor.status === "up"
                          ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                          : monitor.status === "down"
                          ? "bg-rose-50 text-rose-700 border border-rose-200"
                          : monitor.status === "paused"
                          ? "bg-slate-100 text-slate-700 border border-slate-200"
                          : "bg-amber-50 text-amber-700 border border-amber-200"
                      }`}
                    >
                      {monitor.status === "up" && <CheckCircle className="w-3.5 h-3.5" />}
                      {monitor.status === "down" && <XCircle className="w-3.5 h-3.5" />}
                      {monitor.status === "paused" && <PauseCircle className="w-3.5 h-3.5" />}
                      {monitor.status === "pending" && <Clock className="w-3.5 h-3.5" />}
                      <span>
                        {monitor.status === "up" && t.statusUp}
                        {monitor.status === "down" && t.statusDown}
                        {monitor.status === "paused" && t.statusPaused}
                        {monitor.status === "pending" && t.statusPending}
                      </span>
                    </span>
                  </div>

                  <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-slate-600">
                    <span className="flex items-center gap-1">
                      <Building className="w-3.5 h-3.5 text-slate-400" />
                      <span>Client: </span>
                      <Link href={`/clients/${monitor.client_id}`} className="font-semibold text-blue-600 hover:underline">
                        {monitor.client_name}
                      </Link>
                    </span>

                    <a
                      href={monitor.effective_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                    >
                      <Globe className="w-3.5 h-3.5" />
                      <span>{monitor.effective_url}</span>
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  </div>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 text-center">
                  <div className="bg-slate-50 p-3 rounded-xl border border-slate-100">
                    <div className="text-[10px] uppercase font-bold text-slate-400">Interval</div>
                    <div className="text-sm font-bold text-slate-900 mt-0.5">{monitor.interval_seconds}s</div>
                  </div>
                  <div className="bg-slate-50 p-3 rounded-xl border border-slate-100">
                    <div className="text-[10px] uppercase font-bold text-slate-400">Timeout</div>
                    <div className="text-sm font-bold text-slate-900 mt-0.5">{monitor.timeout_seconds}s</div>
                  </div>
                  <div className="bg-slate-50 p-3 rounded-xl border border-slate-100 col-span-2 sm:col-span-1">
                    <div className="text-[10px] uppercase font-bold text-slate-400">Method</div>
                    <div className="text-sm font-bold text-slate-900 mt-0.5">{monitor.method}</div>
                  </div>
                </div>
              </div>

              {/* Extra check configuration details */}
              <div className="pt-6 grid grid-cols-1 sm:grid-cols-3 gap-6 text-xs text-slate-600">
                <div>
                  <span className="font-semibold text-slate-800">Expected Status: </span>
                  <span>
                    {monitor.expected_status_min}–{monitor.expected_status_max}
                  </span>
                </div>
                <div>
                  <span className="font-semibold text-slate-800">Follow Redirects: </span>
                  <span>{monitor.follow_redirects ? `Yes (max ${monitor.max_redirects})` : "No"}</span>
                </div>
                {monitor.expected_text && (
                  <div>
                    <span className="font-semibold text-slate-800">Expected Text: </span>
                    <span className="font-mono bg-slate-100 px-1 py-0.5 rounded">{monitor.expected_text}</span>
                  </div>
                )}
                {monitor.json_pointer && (
                  <div>
                    <span className="font-semibold text-slate-800">JSON Pointer: </span>
                    <span className="font-mono bg-slate-100 px-1 py-0.5 rounded">{monitor.json_pointer}</span>
                  </div>
                )}
              </div>
            </div>

            {/* SSL / TLS Certificate Card (If HTTPS) */}
            {latestSSL && (
              <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
                <div className="flex items-center justify-between pb-6 border-b border-slate-100">
                  <div className="flex items-center gap-2.5">
                    <ShieldCheck className="w-6 h-6 text-emerald-600" />
                    <div>
                      <h2 className="text-base font-bold text-slate-900 tracking-tight">{t.sslDetails}</h2>
                      <p className="text-xs text-slate-500 mt-0.5">Verified TLS security handshake and certificate expiration.</p>
                    </div>
                  </div>

                  {latestSSL.tls_days_remaining !== undefined && latestSSL.tls_days_remaining < 30 ? (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-amber-50 border border-amber-200 text-amber-800 rounded-full text-xs font-semibold">
                      <ShieldAlert className="w-3.5 h-3.5 text-amber-600" />
                      <span>{t.sslExpiryWarning}</span>
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-full text-xs font-semibold">
                      <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
                      <span>{t.sslValid}</span>
                    </span>
                  )}
                </div>

                <div className="pt-6 grid grid-cols-1 sm:grid-cols-4 gap-6 text-xs">
                  <div>
                    <div className="text-slate-400 font-semibold uppercase">{t.sslSubject}</div>
                    <div className="text-slate-800 font-bold mt-1">{latestSSL.tls_cert_subject || "—"}</div>
                  </div>
                  <div>
                    <div className="text-slate-400 font-semibold uppercase">{t.sslIssuer}</div>
                    <div className="text-slate-800 font-bold mt-1">{latestSSL.tls_cert_issuer || "—"}</div>
                  </div>
                  <div>
                    <div className="text-slate-400 font-semibold uppercase">{t.sslDaysRemaining}</div>
                    <div className="text-slate-800 font-bold mt-1 text-sm">
                      {latestSSL.tls_days_remaining !== undefined ? `${latestSSL.tls_days_remaining} days` : "—"}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-400 font-semibold uppercase">Protocol & Cipher</div>
                    <div className="text-slate-800 font-bold mt-1 truncate" title={latestSSL.tls_cipher}>
                      {latestSSL.tls_version} • {latestSSL.tls_cipher}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Check Execution History Table */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                    <Activity className="w-5 h-5 text-blue-600" />
                    <span>{t.checkHistory}</span>
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Recent checks executed by the background monitoring service.
                  </p>
                </div>

                <button
                  onClick={reloadMonitorData}
                  className="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
                >
                  <RotateCcw className="w-3.5 h-3.5" />
                  <span>Refresh</span>
                </button>
              </div>

              {history.length === 0 ? (
                <div className="p-12 text-center text-slate-500 text-xs">
                  {t.noHistory}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                      <tr>
                        <th className="px-6 py-3.5">{t.status}</th>
                        <th className="px-6 py-3.5">{t.latestCheck}</th>
                        <th className="px-6 py-3.5">{t.latency}</th>
                        <th className="px-6 py-3.5">Status Code</th>
                        <th className="px-6 py-3.5">{t.executionType}</th>
                        <th className="px-6 py-3.5">Error Details</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 text-xs">
                      {history.map((h) => (
                        <tr key={h.id} className="hover:bg-slate-50/70 transition-colors">
                          <td className="px-6 py-3.5">
                            {h.result === "success" && (
                              <span className="inline-flex items-center gap-1 text-emerald-600 font-semibold">
                                <CheckCircle className="w-3.5 h-3.5" />
                                <span>Success</span>
                              </span>
                            )}
                            {h.result === "failure" && (
                              <span className="inline-flex items-center gap-1 text-rose-600 font-semibold">
                                <XCircle className="w-3.5 h-3.5" />
                                <span>Failure</span>
                              </span>
                            )}
                            {h.result === "timeout" && (
                              <span className="inline-flex items-center gap-1 text-amber-600 font-semibold">
                                <Clock className="w-3.5 h-3.5" />
                                <span>Timeout</span>
                              </span>
                            )}
                            {h.result === "blocked" && (
                              <span className="inline-flex items-center gap-1 text-purple-600 font-semibold">
                                <Shield className="w-3.5 h-3.5" />
                                <span>Blocked</span>
                              </span>
                            )}
                            {h.result === "error" && (
                              <span className="inline-flex items-center gap-1 text-slate-600 font-semibold">
                                <AlertCircle className="w-3.5 h-3.5" />
                                <span>Error</span>
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-3.5 text-slate-600">
                            {new Date(h.completed_at).toLocaleString()}
                          </td>
                          <td className="px-6 py-3.5 font-mono font-medium text-slate-800">
                            {h.latency_ms} ms
                          </td>
                          <td className="px-6 py-3.5 font-mono font-bold text-slate-800">
                            {h.http_status_code || "—"}
                          </td>
                          <td className="px-6 py-3.5">
                            <span
                              className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded ${
                                h.execution_type === "manual"
                                  ? "bg-purple-100 text-purple-800"
                                  : "bg-slate-100 text-slate-700"
                              }`}
                            >
                              {h.execution_type}
                            </span>
                          </td>
                          <td className="px-6 py-3.5 text-slate-500 max-w-xs truncate">
                            {h.error_message || "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}
