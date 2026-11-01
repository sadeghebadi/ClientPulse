"use client";

import React, { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { useToast } from "@/components/ui/Toast";
import {
  Radio,
  Check,
  Copy,
  AlertTriangle,
  Terminal,
  ShieldCheck,
  Info,
} from "lucide-react";

interface ClientOption {
  id: string;
  name: string;
}

interface AssetOption {
  id: string;
  name: string;
  url: string;
}

export default function NewHeartbeatPage() {
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [clients, setClients] = useState<ClientOption[]>([]);
  const [assets, setAssets] = useState<AssetOption[]>([]);
  const [selectedClient, setSelectedClient] = useState("");
  const [selectedAsset, setSelectedAsset] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [intervalPreset, setIntervalPreset] = useState("3600");
  const [customInterval, setCustomInterval] = useState("");
  const [customGrace, setCustomGrace] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Secret Modal State
  const [createdData, setCreatedData] = useState<{
    id: string;
    token: string;
    pingURL: string;
    bashSnippet: string;
    powerShellSnippet: string;
  } | null>(null);

  const [copiedToken, setCopiedToken] = useState(false);
  const [copiedBash, setCopiedBash] = useState(false);
  const [copiedPS, setCopiedPS] = useState(false);

  useEffect(() => {
    if (!user || !activeOrg) return;
    apiFetch<{ clients: ClientOption[] }>("/api/v1/clients")
      .then((data) => setClients(data.clients || []))
      .catch(() => {});
  }, [user, activeOrg]);

  useEffect(() => {
    if (!selectedClient) return;
    let isMounted = true;
    apiFetch<{ assets: AssetOption[] }>(`/api/v1/assets?client_id=${selectedClient}`)
      .then((data) => {
        if (isMounted) setAssets(data.assets || []);
      })
      .catch(() => {
        if (isMounted) setAssets([]);
      });
    return () => {
      isMounted = false;
    };
  }, [selectedClient]);

  const effectiveInterval =
    intervalPreset === "custom"
      ? parseInt(customInterval, 10) || 3600
      : parseInt(intervalPreset, 10);

  const defaultGrace = Math.max(60, Math.round(effectiveInterval * 0.2));
  const effectiveGrace = customGrace ? parseInt(customGrace, 10) || defaultGrace : defaultGrace;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedClient) {
      setError(isRTL ? "انتخاب مشتری الزامی است" : "Client is required");
      return;
    }
    if (!name.trim()) {
      setError(isRTL ? "نام هارت‌بیت الزامی است" : "Name is required");
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const resp = await apiFetch<{
        heartbeat: { id: string };
        token: string;
        ping_url: string;
        bash_snippet: string;
        powershell_snippet: string;
      }>("/api/v1/heartbeats", {
        method: "POST",
        body: JSON.stringify({
          client_id: selectedClient,
          asset_id: selectedAsset || undefined,
          name: name.trim(),
          description: description.trim() || undefined,
          expected_interval_seconds: effectiveInterval,
          grace_period_seconds: effectiveGrace,
        }),
      });

      setCreatedData({
        id: resp.heartbeat.id,
        token: resp.token,
        pingURL: resp.ping_url,
        bashSnippet: resp.bash_snippet,
        powerShellSnippet: resp.powershell_snippet,
      });
      toast.success(isRTL ? "مانیتور هارت‌بیت با موفقیت ساخته شد" : "Heartbeat created successfully");
    } catch (err: any) {
      setError(humanizeApiError(err, isRTL));
    } finally {
      setSubmitting(false);
    }
  };

  const copyToClipboard = (text: string, type: "token" | "bash" | "ps") => {
    navigator.clipboard.writeText(text);
    if (type === "token") {
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    } else if (type === "bash") {
      setCopiedBash(true);
      setTimeout(() => setCopiedBash(false), 2000);
    } else {
      setCopiedPS(true);
      setTimeout(() => setCopiedPS(false), 2000);
    }
    toast.success(t.copiedToClipboard);
  };

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto space-y-6">
        <PageHeader
          title={t.createHeartbeatTitle}
          description={t.createHeartbeatDesc}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
            { label: t.heartbeats, href: "/heartbeats" },
            { label: t.createHeartbeat },
          ]}
        />

        {error && (
          <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm space-y-5">
          {/* Client Selection */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              {t.client} <span className="text-rose-500">*</span>
            </label>
            <select
              name="client_id"
              value={selectedClient}
              onChange={(e) => {
                setSelectedClient(e.target.value);
                setSelectedAsset("");
                if (!e.target.value) setAssets([]);
              }}
              required
              className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-slate-900/10"
            >
              <option value="">{isRTL ? "انتخاب مشتری..." : "Select Client..."}</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          {/* Asset Selection (Optional) */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              {t.asset} ({isRTL ? "اختیاری" : "Optional"})
            </label>
            <select
              value={selectedAsset}
              onChange={(e) => setSelectedAsset(e.target.value)}
              disabled={!selectedClient}
              className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-slate-900/10 disabled:bg-slate-50 disabled:text-slate-400"
            >
              <option value="">{isRTL ? "بدون انتساب به وب‌سایت خاص" : "Not tied to a specific website"}</option>
              {assets.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name} ({a.url})
                </option>
              ))}
            </select>
          </div>

          {/* Name */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              {t.heartbeatName} <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              name="name"
              required
              placeholder={isRTL ? "مثلاً: اسکریپت پشتیبان‌گیری شبانه دیتابیس" : "e.g. Nightly Database Dump"}
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1.5">
              {t.heartbeatDesc} ({isRTL ? "اختیاری" : "Optional"})
            </label>
            <textarea
              rows={2}
              name="description"
              placeholder={isRTL ? "توضیحات کوتاه در مورد کرون‌جاب یا سرور اجراکننده..." : "Optional notes on the job runner or server..."}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
            />
          </div>

          {/* Expected Interval */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t.expectedInterval}
              </label>
              <select
                value={intervalPreset}
                onChange={(e) => setIntervalPreset(e.target.value)}
                className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-slate-900/10"
              >
                <option value="300">5 {isRTL ? "دقیقه" : "minutes"}</option>
                <option value="900">15 {isRTL ? "دقیقه" : "minutes"}</option>
                <option value="1800">30 {isRTL ? "دقیقه" : "minutes"}</option>
                <option value="3600">1 {isRTL ? "ساعت" : "hour"}</option>
                <option value="21600">6 {isRTL ? "ساعت" : "hours"}</option>
                <option value="43200">12 {isRTL ? "ساعت" : "hours"}</option>
                <option value="86400">24 {isRTL ? "ساعت (روزانه)" : "hours (daily)"}</option>
                <option value="604800">7 {isRTL ? "روز (هفتگی)" : "days (weekly)"}</option>
                <option value="custom">{isRTL ? "تنظیم دستی (ثانیه)" : "Custom (seconds)"}</option>
              </select>
              {intervalPreset === "custom" && (
                <input
                  type="number"
                  min="60"
                  max="2592000"
                  placeholder="3600"
                  value={customInterval}
                  onChange={(e) => setCustomInterval(e.target.value)}
                  className="mt-2 w-full text-xs px-3 py-2 rounded-lg border border-slate-200"
                />
              )}
            </div>

            {/* Grace Period */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t.gracePeriod}
              </label>
              <input
                type="number"
                min="60"
                max="86400"
                placeholder={defaultGrace.toString()}
                value={customGrace}
                onChange={(e) => setCustomGrace(e.target.value)}
                className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                {isRTL
                  ? `پیش‌فرض: ۲۰٪ دوره تناوب (حداقل ۶۰ ثانیه) = ${defaultGrace} ثانیه`
                  : `Default: 20% of interval (min 60s) = ${defaultGrace}s`}
              </p>
            </div>
          </div>

          <div className="pt-4 border-t border-slate-200 flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => router.push("/heartbeats")}
            >
              {t.cancel}
            </Button>
            <Button
              type="submit"
              loading={submitting}
              icon={<Radio className="w-4 h-4" />}
            >
              {t.createHeartbeat}
            </Button>
          </div>
        </form>

        {/* Reveal Secret Token Modal */}
        {createdData && (
          <div className="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl max-w-xl w-full p-6 shadow-2xl border border-slate-200 space-y-5 animate-in fade-in duration-200">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-emerald-100 flex items-center justify-center text-emerald-600 shrink-0">
                  <ShieldCheck className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-slate-900">
                    {isRTL ? "توکن محرمانه هارت‌بیت تولید شد" : "Secret Heartbeat Token Generated"}
                  </h3>
                  <p className="text-xs text-slate-500">
                    {t.tokenWarning}
                  </p>
                </div>
              </div>

              <div className="p-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-900 flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                <span>
                  {isRTL
                    ? "این توکن فقط همین یک‌بار نمایش داده می‌شود و هرگز در پایگاه‌داده به صورت متن خام ذخیره نمی‌گردد."
                    : "This secret token is displayed only once. It will never be shown again and is hashed in PostgreSQL."}
                </span>
              </div>

              {/* Token Copy Box */}
              <div>
                <label className="block text-[11px] font-bold text-slate-500 uppercase tracking-wider mb-1">
                  {isRTL ? "توکن امنیتی" : "Secret Token"}
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={createdData.token}
                    className="w-full text-xs font-mono bg-slate-50 border border-slate-200 px-3 py-2 rounded-lg text-slate-900 select-all"
                  />
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={copiedToken ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                    onClick={() => copyToClipboard(createdData.token, "token")}
                  >
                    {copiedToken ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}
                  </Button>
                </div>
              </div>

              {/* Ready-to-copy Command Snippets */}
              <div className="space-y-3 pt-2 border-t border-slate-100">
                <h4 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                  <Terminal className="w-3.5 h-3.5 text-slate-600" />
                  {t.commandSnippets}
                </h4>

                {/* Bash Snippet */}
                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.bashCronSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(createdData.bashSnippet, "bash")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1"
                    >
                      {copiedBash ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedBash ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-lg overflow-x-auto text-left" dir="ltr">
                    <code>{createdData.bashSnippet}</code>
                  </pre>
                </div>

                {/* PowerShell Snippet */}
                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.powerShellSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(createdData.powerShellSnippet, "ps")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1"
                    >
                      {copiedPS ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedPS ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-lg overflow-x-auto text-left" dir="ltr">
                    <code>{createdData.powerShellSnippet}</code>
                  </pre>
                </div>
              </div>

              <div className="pt-3 border-t border-slate-100 flex justify-end">
                <Button
                  onClick={() => router.push(`/heartbeats/${createdData.id}`)}
                >
                  {isRTL ? "متوجه شدم و دستور را ذخیره کردم" : "I have saved the token & snippet"}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
