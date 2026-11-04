"use client";

import React, { useState, useEffect, useRef } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, API_BASE_URL, getCookie, humanizeApiError } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import {
  Palette,
  Upload,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  Eye,
  Building,
  Save,
  Globe,
  Mail,
  FileText,
} from "lucide-react";

interface Branding {
  organization_id: string;
  display_name: string;
  logo_storage_key?: string;
  primary_color: string;
  accent_color: string;
  report_footer_text: string;
  contact_email: string;
  website_url: string;
  default_report_locale: string;
}

export default function BrandingSettingsPage() {
  const { role, activeOrg } = useAuth();
  const { t, isLocaleEnabled } = useI18n();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [branding, setBranding] = useState<Branding>({
    organization_id: "",
    display_name: "",
    primary_color: "#0ea5e9",
    accent_color: "#6366f1",
    report_footer_text: "",
    contact_email: "",
    website_url: "",
    default_report_locale: "en",
  });

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploadingLogo, setUploadingLogo] = useState(false);
  const [hasLogo, setHasLogo] = useState(false);
  const [logoPreviewUrl, setLogoPreviewUrl] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [contrastRatio, setContrastRatio] = useState<number>(3.0);
  const [isAccessible, setIsAccessible] = useState<boolean>(true);

  const canEdit = role === "owner" || role === "admin";

  useEffect(() => {
    let isMounted = true;
    apiFetch<{
      branding: Branding;
      has_logo: boolean;
      primary_contrast_ratio?: number;
      is_accessible?: boolean;
    }>("/api/v1/settings/branding")
      .then((data) => {
        if (!isMounted) return;
        if (data.branding) {
          setBranding(data.branding);
          setHasLogo(data.has_logo || false);
          if (data.has_logo) {
            setLogoPreviewUrl(`${API_BASE_URL}/api/v1/settings/branding/logo-file?t=${Date.now()}`);
          }
        }
        if (data.primary_contrast_ratio !== undefined) {
          setContrastRatio(data.primary_contrast_ratio);
          setIsAccessible(data.is_accessible ?? true);
        }
        setLoading(false);
      })
      .catch(() => {
        if (isMounted) {
          setMessage({ type: "error", text: "Failed to load branding settings" });
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canEdit) return;

    try {
      setSaving(true);
      setMessage(null);

      const data = await apiFetch<{
        branding: Branding;
        primary_contrast_ratio?: number;
        is_accessible?: boolean;
      }>("/api/v1/settings/branding", {
        method: "PATCH",
        body: JSON.stringify(branding),
      });

      if (data.primary_contrast_ratio !== undefined) {
        setContrastRatio(data.primary_contrast_ratio);
        setIsAccessible(data.is_accessible ?? true);
      }

      setMessage({ type: "success", text: "Branding settings saved successfully!" });
    } catch (err: unknown) {
      setMessage({ type: "error", text: humanizeApiError(err, false) });
    } finally {
      setSaving(false);
    }
  };

  const handleLogoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Reject SVG client-side
    if (file.name.toLowerCase().endsWith(".svg") || file.type.includes("svg")) {
      setMessage({ type: "error", text: "SVG format is disabled for security reasons. Please upload PNG, JPEG or WebP." });
      return;
    }

    if (file.size > 2 * 1024 * 1024) {
      setMessage({ type: "error", text: "Logo file exceeds 2MB limit." });
      return;
    }

    try {
      setUploadingLogo(true);
      setMessage(null);

      const fd = new FormData();
      fd.append("logo", file);

      const headers = new Headers();
      const csrf = getCookie("clientpulse_csrf");
      if (csrf) headers.set("X-CSRF-Token", csrf);

      const res = await fetch(`${API_BASE_URL}/api/v1/settings/branding/logo`, {
        method: "POST",
        headers,
        body: fd,
        credentials: "include",
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error?.message || "Failed to upload logo");
      }

      setHasLogo(true);
      setLogoPreviewUrl(`${API_BASE_URL}/api/v1/settings/branding/logo-file?t=${Date.now()}`);
      setMessage({ type: "success", text: "Logo uploaded successfully!" });
    } catch (err: unknown) {
      setMessage({ type: "error", text: humanizeApiError(err, false) });
    } finally {
      setUploadingLogo(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleDeleteLogo = async () => {
    if (!confirm("Are you sure you want to remove the agency logo?")) return;
    try {
      setUploadingLogo(true);
      await apiFetch("/api/v1/settings/branding/logo", {
        method: "DELETE",
      });
      setHasLogo(false);
      setLogoPreviewUrl(null);
      setMessage({ type: "success", text: "Logo removed successfully." });
    } catch (err: unknown) {
      setMessage({ type: "error", text: humanizeApiError(err, false) });
    } finally {
      setUploadingLogo(false);
    }
  };

  return (
    <AppLayout>
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Palette className="w-6 h-6 text-sky-600" />
            {t.agencyBranding}
          </h1>
          <p className="text-slate-600 text-sm mt-1">{t.brandingDesc}</p>
        </div>

        {message && (
          <div
            className={`p-4 rounded-lg text-sm flex items-center gap-2 ${
              message.type === "success"
                ? "bg-emerald-50 text-emerald-800 border border-emerald-200"
                : "bg-red-50 text-red-800 border border-red-200"
            }`}
          >
            {message.type === "success" ? (
              <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
            ) : (
              <AlertTriangle className="w-5 h-5 flex-shrink-0" />
            )}
            <span>{message.text}</span>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Main Branding Form */}
          <div className="md:col-span-2 bg-white p-6 rounded-xl border border-slate-200 shadow-sm space-y-6">
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.agencyDisplayName} *
                </label>
                <Input
                  type="text"
                  required
                  disabled={!canEdit}
                  placeholder="e.g. Apex Digital Operations"
                  value={branding.display_name}
                  onChange={(e) => setBranding({ ...branding, display_name: e.target.value })}
                />
              </div>

              {/* Logo Uploader */}
              <div className="border border-slate-200 rounded-lg p-4 bg-slate-50 space-y-3">
                <label className="block text-xs font-semibold text-slate-700 uppercase">
                  {t.logoUpload}
                </label>
                <div className="flex items-center gap-4">
                  {hasLogo && logoPreviewUrl ? (
                    <div className="relative border border-slate-200 bg-white p-2 rounded-lg">
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={logoPreviewUrl}
                        alt="Agency Logo"
                        className="h-12 max-w-[160px] object-contain"
                      />
                    </div>
                  ) : (
                    <div className="h-12 w-32 border-2 border-dashed border-slate-300 rounded-lg flex items-center justify-center text-xs text-slate-400">
                      No Logo
                    </div>
                  )}

                  {canEdit && (
                    <div className="flex items-center gap-2">
                      <input
                        type="file"
                        ref={fileInputRef}
                        accept="image/png,image/jpeg,image/webp"
                        onChange={handleLogoUpload}
                        className="hidden"
                      />
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={uploadingLogo}
                        onClick={() => fileInputRef.current?.click()}
                        className="flex items-center gap-1.5"
                      >
                        <Upload className="w-3.5 h-3.5" />
                        {uploadingLogo ? t.loading : t.logoUpload}
                      </Button>
                      {hasLogo && (
                        <Button
                          type="button"
                          size="sm"
                          variant="danger"
                          disabled={uploadingLogo}
                          onClick={handleDeleteLogo}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </Button>
                      )}
                    </div>
                  )}
                </div>
                <p className="text-xs text-slate-500">{t.logoUploadHelp}</p>
              </div>

              {/* Colors */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.primaryColor} *
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      disabled={!canEdit}
                      value={branding.primary_color || "#0ea5e9"}
                      onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })}
                      className="w-10 h-10 border border-slate-300 rounded-lg cursor-pointer p-0.5"
                    />
                    <Input
                      type="text"
                      disabled={!canEdit}
                      value={branding.primary_color}
                      onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })}
                      className="font-mono text-sm uppercase"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.accentColor} *
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      disabled={!canEdit}
                      value={branding.accent_color || "#6366f1"}
                      onChange={(e) => setBranding({ ...branding, accent_color: e.target.value })}
                      className="w-10 h-10 border border-slate-300 rounded-lg cursor-pointer p-0.5"
                    />
                    <Input
                      type="text"
                      disabled={!canEdit}
                      value={branding.accent_color}
                      onChange={(e) => setBranding({ ...branding, accent_color: e.target.value })}
                      className="font-mono text-sm uppercase"
                    />
                  </div>
                </div>
              </div>

              {/* Contrast Indicator */}
              <div className="p-3 bg-slate-50 border border-slate-200 rounded-lg flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-slate-700">{t.contrastCheck}:</span>
                  <span className="font-mono font-bold text-slate-900">{contrastRatio.toFixed(2)}:1</span>
                </div>
                {isAccessible ? (
                  <Badge variant="active" className="flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" />
                    {t.contrastPassed}
                  </Badge>
                ) : (
                  <Badge variant="archived" className="flex items-center gap-1">
                    <AlertTriangle className="w-3 h-3" />
                    {t.contrastFailed}
                  </Badge>
                )}
              </div>

              {/* Footer text, Email, URL */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.reportFooterText}
                </label>
                <textarea
                  rows={2}
                  disabled={!canEdit}
                  placeholder="e.g. Confidential client report prepared by Apex Digital Agency. All rights reserved."
                  value={branding.report_footer_text}
                  onChange={(e) => setBranding({ ...branding, report_footer_text: e.target.value })}
                  className="w-full border border-slate-300 rounded-lg p-2 text-sm focus:outline-none focus:ring-2 focus:ring-sky-500"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.contactEmail}
                  </label>
                  <Input
                    type="email"
                    disabled={!canEdit}
                    placeholder="support@agency.example"
                    value={branding.contact_email}
                    onChange={(e) => setBranding({ ...branding, contact_email: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.websiteURL}
                  </label>
                  <Input
                    type="url"
                    disabled={!canEdit}
                    placeholder="https://agency.example"
                    value={branding.website_url}
                    onChange={(e) => setBranding({ ...branding, website_url: e.target.value })}
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.defaultReportLocale}
                </label>
                <select
                  disabled={!canEdit}
                  value={branding.default_report_locale}
                  onChange={(e) => setBranding({ ...branding, default_report_locale: e.target.value })}
                  className="w-full border border-slate-300 rounded-lg p-2 text-sm bg-white"
                >
                  {isLocaleEnabled("en") && <option value="en">English (LTR)</option>}
                  {isLocaleEnabled("fa") && <option value="fa">فارسی (Persian RTL)</option>}
                </select>
              </div>

              {canEdit && (
                <div className="pt-2">
                  <Button type="submit" disabled={saving} className="flex items-center gap-2">
                    <Save className="w-4 h-4" />
                    {saving ? t.loading : "Save Branding"}
                  </Button>
                </div>
              )}
            </form>
          </div>

          {/* Live Report Preview Widget */}
          <div className="space-y-4">
            <div className="bg-white p-5 rounded-xl border border-slate-200 shadow-sm space-y-4">
              <h3 className="text-sm font-bold text-slate-900 flex items-center gap-1.5 border-b pb-2">
                <Eye className="w-4 h-4 text-sky-600" />
                Live Report Header Preview
              </h3>

              <div
                className="border rounded-lg p-4 bg-white shadow-inner space-y-3"
                style={{ borderTop: `4px solid ${branding.primary_color || "#0ea5e9"}` }}
              >
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <h4
                      className="font-bold text-base line-clamp-1"
                      style={{ color: branding.primary_color || "#0ea5e9" }}
                    >
                      {branding.display_name || "Apex Operations"}
                    </h4>
                    <p className="text-[10px] text-slate-500">Monthly Website Care Report</p>
                  </div>
                  {hasLogo && logoPreviewUrl && (
                    /* eslint-disable-next-line @next/next/no-img-element */
                    <img
                      src={logoPreviewUrl}
                      alt="Logo Preview"
                      className="h-8 max-w-[80px] object-contain"
                    />
                  )}
                </div>

                <div className="bg-slate-50 border border-slate-200 rounded p-2 text-[10px] space-y-1 text-slate-600">
                  <div className="flex justify-between">
                    <span className="font-semibold">Client:</span>
                    <span>Acme Corp</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-semibold">Period:</span>
                    <span>October 2026</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-semibold">Language:</span>
                    <span>{branding.default_report_locale === "fa" ? "فارسی (RTL)" : "English (LTR)"}</span>
                  </div>
                </div>

                <div
                  className="rounded p-2 text-[10px] text-white text-center font-semibold"
                  style={{ backgroundColor: branding.accent_color || "#6366f1" }}
                >
                  99.9% Uptime Verified
                </div>

                <div className="text-[9px] text-slate-400 text-center border-t pt-1 line-clamp-1">
                  {branding.report_footer_text || "ClientPulse Care Report"}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </AppLayout>
  );
}
