"use client";

import React, { useState } from "react";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Building, Plus, Check, Shield } from "lucide-react";

export default function OrganizationsPage() {
  const { user, activeOrg, memberships, switchOrganization, refreshUser, loading } = useAuth();
  const { t, isRTL } = useI18n();

  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgSlug, setNewOrgSlug] = useState("");
  const [createLoading, setCreateLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCreateOrg = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newOrgName.trim()) return;

    setCreateLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/organizations", {
        method: "POST",
        body: JSON.stringify({
          name: newOrgName,
          slug: newOrgSlug || undefined,
        }),
      });

      setNewOrgName("");
      setNewOrgSlug("");
      await refreshUser();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create organization");
    } finally {
      setCreateLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col">
        <Header />
        <div className="flex-1 flex items-center justify-center p-4">
          <p className="text-xs text-slate-500">{t.loading}</p>
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col">
        <Header />
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="bg-white border border-slate-200 rounded-xl p-8 max-w-sm text-center">
            <p className="text-sm text-slate-700 mb-4">Please log in to manage your organizations.</p>
            <a href="/login" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-xs font-semibold">
              {t.login}
            </a>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />
      <main className="max-w-4xl mx-auto px-4 sm:px-6 py-8 w-full flex-1">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{t.myOrganizations}</h1>
            <p className="text-xs text-slate-500 mt-1">
              Switch between workspaces or provision a new agency organization.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Organization List */}
          <div className="md:col-span-2 space-y-4">
            {memberships.map((m) => {
              const isActive = activeOrg?.id === m.organization_id;
              return (
                <div
                  key={m.id}
                  className={`bg-white border rounded-xl p-5 shadow-sm transition-all flex items-center justify-between ${
                    isActive ? "border-blue-500 ring-1 ring-blue-500" : "border-slate-200 hover:border-slate-300"
                  }`}
                >
                  <div className="flex items-center gap-3.5">
                    <div
                      className={`w-10 h-10 rounded-lg flex items-center justify-center font-bold text-sm ${
                        isActive ? "bg-blue-600 text-white" : "bg-slate-100 text-slate-600"
                      }`}
                    >
                      <Building className="w-5 h-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-sm font-bold text-slate-900">{m.org_name || "Agency Workspace"}</h3>
                        {isActive && (
                          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                            <Check className="w-3 h-3" />
                            {t.activeOrg}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-[11px] text-slate-500 font-mono">slug: {m.org_slug}</span>
                        <span className="text-slate-300">•</span>
                        <span className="text-[11px] font-semibold text-slate-600 flex items-center gap-1">
                          <Shield className="w-3 h-3 text-slate-400" />
                          {m.role}
                        </span>
                      </div>
                    </div>
                  </div>

                  <div>
                    {!isActive ? (
                      <button
                        onClick={() => switchOrganization(m.organization_id)}
                        className="px-3 py-1.5 bg-slate-100 hover:bg-blue-50 hover:text-blue-600 text-slate-700 text-xs font-semibold rounded-lg transition-colors border border-slate-200"
                      >
                        {t.switchOrg}
                      </button>
                    ) : (
                      <span className="text-xs font-medium text-slate-400">Current</span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Create New Org Card */}
          <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm h-fit">
            <div className="flex items-center gap-2 mb-4">
              <Plus className="w-4 h-4 text-blue-600" />
              <h2 className="text-sm font-bold text-slate-900">{t.createOrganization}</h2>
            </div>

            {error && (
              <div className="mb-3 p-2.5 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">
                {error}
              </div>
            )}

            <form onSubmit={handleCreateOrg} className="space-y-3.5">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t.orgNameLabel}
                </label>
                <input
                  type="text"
                  required
                  value={newOrgName}
                  onChange={(e) => setNewOrgName(e.target.value)}
                  className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder={isRTL ? "مثال: استودیو دیزاین نوین" : "e.g. Modern Web Studio"}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t.orgSlugLabel} (Optional)
                </label>
                <input
                  type="text"
                  value={newOrgSlug}
                  onChange={(e) => setNewOrgSlug(e.target.value)}
                  className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                  placeholder="modern-web-studio"
                />
              </div>

              <button
                type="submit"
                disabled={createLoading}
                className="w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs rounded-lg shadow-sm transition-colors disabled:opacity-50"
              >
                {createLoading ? t.loading : t.createOrganization}
              </button>
            </form>
          </div>
        </div>
      </main>
    </div>
  );
}

