"use client";

import React, { useState } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Building, Plus, Check, Shield } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { useToast } from "@/components/ui/Toast";

export default function OrganizationsPage() {
  const { user, activeOrg, memberships, switchOrganization, refreshUser, loading } = useAuth();
  const { t } = useI18n();
  const toast = useToast();

  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgSlug, setNewOrgSlug] = useState("");
  const [createLoading, setCreateLoading] = useState(false);
  const [switchingID, setSwitchingID] = useState<string | null>(null);
  const [error, setError] = useState<any>(null);

  const handleCreateOrg = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newOrgName.trim() || createLoading) return;

    setCreateLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/organizations", {
        method: "POST",
        body: JSON.stringify({
          name: newOrgName.trim(),
          slug: newOrgSlug.trim() || undefined,
        }),
      });

      toast.success(`Organization "${newOrgName}" created successfully`);
      setNewOrgName("");
      setNewOrgSlug("");
      await refreshUser();
    } catch (err) {
      setError(err);
    } finally {
      setCreateLoading(false);
    }
  };

  const handleSwitch = async (orgID: string, name: string) => {
    try {
      setSwitchingID(orgID);
      await switchOrganization(orgID);
      toast.success(`Switched active workspace to "${name}"`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to switch organization");
    } finally {
      setSwitchingID(null);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={t.myOrganizations}
        description="Switch between active agency workspaces or provision a new isolated organization."
        icon={<Building className="w-5 h-5" />}
      />

      {error && <ErrorState error={error} className="mb-6" />}

      {loading ? (
        <TableSkeleton rows={3} cols={3} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Organization List */}
          <div className="md:col-span-2 space-y-4">
            {memberships.map((m) => {
              const isActive = activeOrg?.id === m.organization_id;
              const isSwitching = switchingID === m.organization_id;
              return (
                <div
                  key={m.id}
                  className={`bg-white border rounded-2xl p-5 sm:p-6 shadow-sm transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-4 ${
                    isActive
                      ? "border-blue-500 ring-2 ring-blue-500/20"
                      : "border-slate-200 hover:border-slate-300"
                  }`}
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`w-11 h-11 rounded-xl flex items-center justify-center font-bold text-sm shrink-0 ${
                        isActive ? "bg-blue-600 text-white shadow-md" : "bg-slate-100 text-slate-600"
                      }`}
                    >
                      <Building className="w-5 h-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-base font-bold text-slate-900">
                          {m.org_name || "Agency Workspace"}
                        </h3>
                        {isActive && (
                          <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                            <Check className="w-3 h-3" />
                            {t.activeOrg}
                          </span>
                        )}
                      </div>
                      <div className="flex items-center gap-2.5 mt-1 text-xs">
                        <span className="text-slate-500 font-mono">slug: {m.org_slug}</span>
                        <span className="text-slate-300">•</span>
                        <Badge variant={m.role} />
                      </div>
                    </div>
                  </div>

                  <div>
                    {!isActive ? (
                      <Button
                        size="sm"
                        variant="outline"
                        loading={isSwitching}
                        onClick={() => handleSwitch(m.organization_id, m.org_name || "")}
                      >
                        {t.switchOrg}
                      </Button>
                    ) : (
                      <span className="text-xs font-bold text-blue-600 px-3 py-1 bg-blue-50 rounded-lg">
                        Active Workspace
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Create New Org Card */}
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm h-fit">
            <div className="flex items-center gap-2 mb-4">
              <Plus className="w-4 h-4 text-blue-600" />
              <h2 className="text-sm font-bold text-slate-900">{t.createOrganization}</h2>
            </div>

            <form onSubmit={handleCreateOrg} className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.orgNameLabel}
                </label>
                <input
                  type="text"
                  name="name"
                  required
                  disabled={createLoading}
                  value={newOrgName}
                  onChange={(e) => {
                    setNewOrgName(e.target.value);
                    if (!newOrgSlug) {
                      setNewOrgSlug(
                        e.target.value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "")
                      );
                    }
                  }}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  placeholder="e.g. Apex Agency"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.orgSlugLabel}
                </label>
                <input
                  type="text"
                  name="slug"
                  disabled={createLoading}
                  value={newOrgSlug}
                  onChange={(e) => setNewOrgSlug(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
                  placeholder="apex-agency"
                />
              </div>

              <Button
                type="submit"
                variant="primary"
                size="md"
                loading={createLoading}
                className="w-full"
              >
                {t.createOrganization}
              </Button>
            </form>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
