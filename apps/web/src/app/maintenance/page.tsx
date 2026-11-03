"use client";

import React, { useState, useEffect, useCallback } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { PageHeader } from "@/components/ui/PageHeader";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";
import {
  Wrench,
  Plus,
  Search,
  Filter,
  Edit2,
  Trash2,
  Calendar,
  Clock,
  ExternalLink,
  ShieldAlert,
  X,
  FileText,
  User,
} from "lucide-react";
import Link from "next/link";

interface MaintenanceActivity {
  id: string;
  client_id: string;
  client_name: string;
  asset_id?: string;
  asset_name?: string;
  title: string;
  description: string;
  category: string;
  performed_at: string;
  duration_minutes?: number;
  created_by: string;
  created_by_name?: string;
  created_at: string;
}

interface ClientOption {
  id: string;
  name: string;
}

interface AssetOption {
  id: string;
  client_id: string;
  name: string;
}

const CATEGORIES = [
  "update",
  "backup",
  "bug_fix",
  "performance",
  "security_review",
  "content_change",
  "configuration",
  "other",
];

export default function MaintenancePage() {
  const { user, role, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [activities, setActivities] = useState<MaintenanceActivity[]>([]);
  const [clients, setClients] = useState<ClientOption[]>([]);
  const [assets, setAssets] = useState<AssetOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [selectedClient, setSelectedClient] = useState<string>("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState<string>("");

  // Create/Edit Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    client_id: "",
    asset_id: "",
    title: "",
    description: "",
    category: "update",
    performed_at: new Date().toISOString().slice(0, 16),
    duration_minutes: "30",
  });
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // Delete State
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const canModify = role === "owner" || role === "admin" || role === "member";

  const getCategoryLabel = (category: string) => {
    switch (category) {
      case "update":
        return t.catUpdate || "Update";
      case "backup":
        return t.catBackup || "Backup";
      case "bug_fix":
        return t.catBugFix || "Bug Fix";
      case "performance":
        return t.catPerformance || "Performance";
      case "security_review":
        return t.catSecurityReview || "Security Review";
      case "content_change":
        return t.catContentChange || "Content Change";
      case "configuration":
        return t.catConfiguration || "Configuration";
      default:
        return t.catOther || "Other";
    }
  };

  const fetchActivities = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      let url = "/api/v1/maintenance-activities?limit=100";
      if (selectedClient) url += `&client_id=${selectedClient}`;
      if (selectedCategory) url += `&category=${selectedCategory}`;

      const data = await apiFetch<{ activities: MaintenanceActivity[] }>(url);
      setActivities(data.activities || []);
    } catch (err: unknown) {
      setError(humanizeApiError(err, false));
    } finally {
      setLoading(false);
    }
  }, [selectedClient, selectedCategory]);

  useEffect(() => {
    let isMounted = true;
    let url = "/api/v1/maintenance-activities?limit=100";
    if (selectedClient) url += `&client_id=${selectedClient}`;
    if (selectedCategory) url += `&category=${selectedCategory}`;

    apiFetch<{ activities: MaintenanceActivity[] }>(url)
      .then((data) => {
        if (isMounted) {
          setActivities(data.activities || []);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (isMounted) {
          setError(humanizeApiError(err, false));
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [selectedClient, selectedCategory]);

  // Fetch client & asset dropdown options
  useEffect(() => {
    let isMounted = true;
    Promise.all([
      apiFetch<{ clients: ClientOption[] }>("/api/v1/clients?limit=100"),
      apiFetch<{ assets: AssetOption[] }>("/api/v1/assets?limit=100"),
    ])
      .then(([cData, aData]) => {
        if (isMounted) {
          setClients(cData.clients || []);
          setAssets(aData.assets || []);
        }
      })
      .catch(() => {
        // silent fallback
      });

    return () => {
      isMounted = false;
    };
  }, []);

  // Handle modal keyboard listeners
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && modalOpen && !saving) {
        setModalOpen(false);
      }
    };
    if (modalOpen) {
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [modalOpen, saving]);

  const openCreateModal = (clientId?: string) => {
    setEditingId(null);
    setFormData({
      client_id: clientId || (clients[0]?.id || ""),
      asset_id: "",
      title: "",
      description: "",
      category: "update",
      performed_at: new Date().toISOString().slice(0, 16),
      duration_minutes: "30",
    });
    setFormError(null);
    setModalOpen(true);
  };

  const openEditModal = (act: MaintenanceActivity) => {
    setEditingId(act.id);
    setFormData({
      client_id: act.client_id,
      asset_id: act.asset_id || "",
      title: act.title,
      description: act.description,
      category: act.category,
      performed_at: new Date(act.performed_at).toISOString().slice(0, 16),
      duration_minutes: act.duration_minutes !== undefined && act.duration_minutes !== null ? String(act.duration_minutes) : "",
    });
    setFormError(null);
    setModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.title.trim()) {
      setFormError("Title is required");
      return;
    }
    if (!formData.client_id) {
      setFormError("Please select a client");
      return;
    }

    try {
      setSaving(true);
      setFormError(null);

      const payload = {
        client_id: formData.client_id,
        asset_id: formData.asset_id ? formData.asset_id : undefined,
        title: formData.title.trim(),
        description: formData.description.trim(),
        category: formData.category,
        performed_at: new Date(formData.performed_at).toISOString(),
        duration_minutes: formData.duration_minutes ? parseInt(formData.duration_minutes, 10) : undefined,
      };

      const url = editingId
        ? `/api/v1/maintenance-activities/${editingId}`
        : "/api/v1/maintenance-activities";
      const method = editingId ? "PATCH" : "POST";

      await apiFetch(url, {
        method,
        body: JSON.stringify(payload),
      });

      toast.success(t.maintenanceSaved);
      setModalOpen(false);
      fetchActivities();
    } catch (err: unknown) {
      setFormError(humanizeApiError(err, false));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteId) return;
    try {
      setDeleting(true);
      await apiFetch(`/api/v1/maintenance-activities/${deleteId}`, {
        method: "DELETE",
      });
      toast.success(t.maintenanceDeleted);
      setDeleteId(null);
      fetchActivities();
    } catch (err: unknown) {
      toast.error(humanizeApiError(err, false));
    } finally {
      setDeleting(false);
    }
  };

  const filteredAssets = assets.filter((a) => a.client_id === formData.client_id);

  const displayedActivities = activities.filter((act) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      act.title.toLowerCase().includes(q) ||
      act.description.toLowerCase().includes(q) ||
      (act.client_name && act.client_name.toLowerCase().includes(q))
    );
  });

  return (
    <AppLayout>
      <div className="space-y-6">
        {/* Header */}
        <PageHeader
          title={t.maintenance}
          description={t.maintenanceDesc}
          icon={<Wrench className="w-5 h-5 text-sky-600" />}
          actions={
            canModify && (
              <Button onClick={() => openCreateModal()} className="flex items-center gap-2 w-full sm:w-auto justify-center">
                <Plus className="w-4 h-4" />
                {t.addActivity}
              </Button>
            )
          }
        />

        {/* Timezone Info Pill */}
        <div className="flex items-center gap-2 px-3.5 py-2 bg-slate-50 border border-slate-200/80 rounded-xl text-xs text-slate-500">
          <Clock className="w-3.5 h-3.5 text-slate-400 shrink-0" />
          <span>{t.timezoneNotice.replace("{tz}", activeOrg?.timezone || "UTC")}</span>
        </div>

        {/* Filters */}
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex flex-col md:flex-row gap-3 items-stretch md:items-center justify-between">
          <div className="relative flex-1 w-full">
            <Search className={`w-4 h-4 absolute ${isRTL ? "right-3" : "left-3"} top-3 text-slate-400`} />
            <input
              type="text"
              placeholder={t.searchMaintenance}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className={`w-full ${isRTL ? "pr-9 pl-4" : "pl-9 pr-4"} py-2 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-sky-500 bg-slate-50/50`}
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 w-full md:w-auto">
            <select
              value={selectedClient}
              onChange={(e) => setSelectedClient(e.target.value)}
              className="border border-slate-200 rounded-xl text-sm px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-sky-500 text-slate-700"
            >
              <option value="">{t.allClients}</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>

            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="border border-slate-200 rounded-xl text-sm px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-sky-500 text-slate-700 capitalize"
            >
              <option value="">{t.allCategories}</option>
              {CATEGORIES.map((cat) => (
                <option key={cat} value={cat}>
                  {getCategoryLabel(cat)}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm flex items-center gap-2">
            <ShieldAlert className="w-5 h-5 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Activity List Container */}
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          {loading ? (
            <div className="p-12 text-center text-slate-500 text-sm">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-slate-200 border-t-sky-600 mb-3" />
              <p>{t.loading}</p>
            </div>
          ) : displayedActivities.length === 0 ? (
            <div className="p-12 text-center text-slate-500">
              <Wrench className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <p className="font-semibold text-slate-700">{t.noActivitiesFound}</p>
              <p className="text-xs text-slate-400 mt-1 max-w-md mx-auto">
                {t.recentMaintenanceSub}
              </p>
              {canModify && (
                <Button onClick={() => openCreateModal()} variant="outline" className="mt-4">
                  <Plus className="w-4 h-4 mr-1" />
                  {t.addActivity}
                </Button>
              )}
            </div>
          ) : (
            <>
              {/* Mobile Card View (< 768px) */}
              <div className="md:hidden divide-y divide-slate-100">
                {displayedActivities.map((act) => (
                  <div key={act.id} className="p-4 space-y-3 hover:bg-slate-50/60 transition-colors">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <h4 className="font-semibold text-sm text-slate-900 break-words">{act.title}</h4>
                        <div className="flex items-center gap-2 mt-1">
                          <Link
                            href={`/clients/${act.client_id}`}
                            className="font-medium text-xs text-sky-600 hover:text-sky-800 inline-flex items-center gap-1"
                          >
                            <span>{act.client_name}</span>
                            <ExternalLink className="w-3 h-3" />
                          </Link>
                          {act.asset_name && (
                            <span className="text-xs text-slate-400 truncate">
                              • {act.asset_name}
                            </span>
                          )}
                        </div>
                      </div>
                      <Badge variant="neutral" className="text-xs shrink-0">
                        {getCategoryLabel(act.category)}
                      </Badge>
                    </div>

                    {act.description && (
                      <p className="text-xs text-slate-600 line-clamp-2 leading-relaxed bg-slate-50 p-2 rounded-lg">
                        {act.description}
                      </p>
                    )}

                    <div className="flex items-center justify-between text-xs text-slate-500 pt-1">
                      <div className="flex items-center gap-3">
                        <div className="flex items-center gap-1" dir="ltr">
                          <Calendar className="w-3.5 h-3.5 text-slate-400" />
                          <span>{new Date(act.performed_at).toLocaleDateString()}</span>
                        </div>
                        {act.duration_minutes ? (
                          <div className="flex items-center gap-1" dir="ltr">
                            <Clock className="w-3.5 h-3.5 text-slate-400" />
                            <span>{act.duration_minutes} min</span>
                          </div>
                        ) : null}
                      </div>

                      {canModify && (
                        <div className="flex items-center gap-1">
                          <button
                            onClick={() => openEditModal(act)}
                            className="p-1.5 text-slate-500 hover:text-sky-600 hover:bg-slate-100 rounded-lg transition-colors"
                            aria-label={t.editActivity}
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => setDeleteId(act.id)}
                            className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                            aria-label={t.deleteActivity}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>

              {/* Desktop Table View (>= 768px) */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full text-start border-collapse">
                  <thead>
                    <tr className="bg-slate-50/80 border-b border-slate-200 text-xs uppercase font-semibold text-slate-500">
                      <th className="py-3.5 px-4 text-start">{t.activityTitle}</th>
                      <th className="py-3.5 px-4 text-start">{t.clientDetails}</th>
                      <th className="py-3.5 px-4 text-start">{t.activityCategory}</th>
                      <th className="py-3.5 px-4 text-start">{t.activityPerformedAt}</th>
                      <th className="py-3.5 px-4 text-start">{t.activityDuration}</th>
                      <th className="py-3.5 px-4 text-end">{t.actions}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 text-sm text-slate-700">
                    {displayedActivities.map((act) => (
                      <tr key={act.id} className="hover:bg-slate-50/60 transition-colors">
                        <td className="py-3.5 px-4">
                          <div className="font-semibold text-slate-900">{act.title}</div>
                          <div className="text-xs text-slate-500 line-clamp-1 max-w-md mt-0.5">
                            {act.description}
                          </div>
                        </td>
                        <td className="py-3.5 px-4">
                          <Link
                            href={`/clients/${act.client_id}`}
                            className="font-medium text-sky-600 hover:text-sky-800 inline-flex items-center gap-1"
                          >
                            <span>{act.client_name}</span>
                            <ExternalLink className="w-3 h-3" />
                          </Link>
                          {act.asset_name && (
                            <div className="text-xs text-slate-400">{act.asset_name}</div>
                          )}
                        </td>
                        <td className="py-3.5 px-4">
                          <Badge variant="neutral">
                            {getCategoryLabel(act.category)}
                          </Badge>
                        </td>
                        <td className="py-3.5 px-4 text-xs text-slate-600">
                          <div className="flex items-center gap-1.5" dir="ltr">
                            <Calendar className="w-3.5 h-3.5 text-slate-400" />
                            <span>{new Date(act.performed_at).toLocaleDateString()}</span>
                          </div>
                          <div className="text-slate-400 ps-5" dir="ltr">
                            <span>
                              {new Date(act.performed_at).toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                              })}
                            </span>
                          </div>
                        </td>
                        <td className="py-3.5 px-4 text-xs text-slate-600">
                          {act.duration_minutes ? (
                            <div className="flex items-center gap-1" dir="ltr">
                              <Clock className="w-3.5 h-3.5 text-slate-400" />
                              <span>{act.duration_minutes} min</span>
                            </div>
                          ) : (
                            <span className="text-slate-400">—</span>
                          )}
                        </td>
                        <td className="py-3.5 px-4 text-end">
                          {canModify && (
                            <div className="flex items-center justify-end gap-1.5">
                              <button
                                onClick={() => openEditModal(act)}
                                className="p-1.5 text-slate-500 hover:text-sky-600 hover:bg-slate-100 rounded-lg transition-colors"
                                title={t.editActivity}
                              >
                                <Edit2 className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => setDeleteId(act.id)}
                                className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                                title={t.deleteActivity}
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Create/Edit Modal */}
      {modalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4"
          onClick={(e) => {
            if (e.target === e.currentTarget && !saving) setModalOpen(false);
          }}
        >
          <div className="bg-white rounded-2xl shadow-xl max-w-lg w-full p-6 space-y-4 max-h-[90vh] overflow-y-auto border border-slate-100">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <h3 className="font-bold text-slate-900 text-lg flex items-center gap-2">
                <Wrench className="w-5 h-5 text-sky-600" />
                {editingId ? t.editActivity : t.addActivity}
              </h3>
              <button
                onClick={() => setModalOpen(false)}
                className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-100"
                aria-label={t.cancel}
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {formError && (
              <div className="p-3 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm">
                {formError}
              </div>
            )}

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                  {t.clientDetails} *
                </label>
                <select
                  disabled={!!editingId}
                  value={formData.client_id}
                  onChange={(e) => setFormData({ ...formData, client_id: e.target.value, asset_id: "" })}
                  className="w-full border border-slate-200 rounded-xl p-2.5 text-sm bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:opacity-50"
                >
                  <option value="">{isRTL ? "انتخاب مشتری..." : "Select Client..."}</option>
                  {clients.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>

              {filteredAssets.length > 0 && (
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                    {t.activityAsset} ({isRTL ? "اختیاری" : "Optional"})
                  </label>
                  <select
                    value={formData.asset_id}
                    onChange={(e) => setFormData({ ...formData, asset_id: e.target.value })}
                    className="w-full border border-slate-200 rounded-xl p-2.5 text-sm bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-sky-500"
                  >
                    <option value="">{t.generalClientMaintenance}</option>
                    {filteredAssets.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                  {t.activityCategory} *
                </label>
                <select
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                  className="w-full border border-slate-200 rounded-xl p-2.5 text-sm bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-sky-500 capitalize"
                >
                  {CATEGORIES.map((cat) => (
                    <option key={cat} value={cat}>
                      {getCategoryLabel(cat)}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                  {t.activityTitle} *
                </label>
                <Input
                  type="text"
                  required
                  maxLength={200}
                  placeholder={isRTL ? "مثال: به‌روزرسانی هسته وردپرس و بررسی PHP 8.3" : "e.g. WordPress core updates & PHP 8.3 testing"}
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                  {t.activityDescription} *
                </label>
                <textarea
                  required
                  rows={3}
                  className="w-full border border-slate-200 rounded-xl p-2.5 text-sm bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-sky-500"
                  placeholder={isRTL ? "شرح کارهای انجام‌شده، پچ‌های اعمال‌شده یا مشکلات برطرف‌شده..." : "Explain what was accomplished, patches applied, or issues resolved..."}
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                    {t.activityPerformedAt} *
                  </label>
                  <Input
                    type="datetime-local"
                    dir="ltr"
                    required
                    value={formData.performed_at}
                    onChange={(e) => setFormData({ ...formData, performed_at: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1.5">
                    {t.activityDuration}
                  </label>
                  <Input
                    type="number"
                    dir="ltr"
                    min="0"
                    max="14400"
                    placeholder="30"
                    value={formData.duration_minutes}
                    onChange={(e) => setFormData({ ...formData, duration_minutes: e.target.value })}
                  />
                </div>
              </div>

              <div className="flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-end gap-2.5 pt-3 border-t border-slate-100">
                <Button type="button" variant="outline" onClick={() => setModalOpen(false)}>
                  {t.cancel}
                </Button>
                <Button type="submit" disabled={saving}>
                  {saving ? t.loading : editingId ? t.saveChanges : t.addActivity}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteId}
        onClose={() => setDeleteId(null)}
        onConfirm={handleDelete}
        title={t.deleteMaintenanceTitle}
        message={t.deleteMaintenanceConfirm}
        confirmLabel={t.deleteActivity}
        cancelLabel={t.cancel}
        variant="danger"
        loading={deleting}
      />
    </AppLayout>
  );
}
