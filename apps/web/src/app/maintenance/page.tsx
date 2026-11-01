"use client";

import React, { useState, useEffect, useCallback } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
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
  const { t } = useI18n();

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
      setDeleteId(null);
      fetchActivities();
    } catch (err: unknown) {
      alert(humanizeApiError(err, false));
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
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
              <Wrench className="w-6 h-6 text-sky-600" />
              {t.maintenance}
            </h1>
            <p className="text-slate-600 text-sm mt-1">{t.maintenanceDesc}</p>
          </div>
          {canModify && (
            <Button onClick={() => openCreateModal()} className="flex items-center gap-2">
              <Plus className="w-4 h-4" />
              {t.addActivity}
            </Button>
          )}
        </div>

        {/* Filters */}
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex flex-col md:flex-row gap-4 items-center justify-between">
          <div className="relative flex-1 w-full">
            <Search className="w-4 h-4 absolute left-3 top-3 text-slate-400" />
            <input
              type="text"
              placeholder="Search by title, description or client..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-4 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-sky-500"
            />
          </div>

          <div className="flex items-center gap-3 w-full md:w-auto">
            <Filter className="w-4 h-4 text-slate-400 hidden sm:block" />
            <select
              value={selectedClient}
              onChange={(e) => setSelectedClient(e.target.value)}
              className="border border-slate-300 rounded-lg text-sm px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-sky-500"
            >
              <option value="">All Clients</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>

            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="border border-slate-300 rounded-lg text-sm px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-sky-500 capitalize"
            >
              <option value="">All Categories</option>
              {CATEGORIES.map((cat) => (
                <option key={cat} value={cat}>
                  {cat.replace("_", " ")}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm flex items-center gap-2">
            <ShieldAlert className="w-5 h-5 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Activity List Table */}
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          {loading ? (
            <div className="p-12 text-center text-slate-500 text-sm">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-slate-200 border-t-sky-600 mb-3" />
              <p>{t.loading}</p>
            </div>
          ) : displayedActivities.length === 0 ? (
            <div className="p-12 text-center text-slate-500">
              <Wrench className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <p className="font-semibold text-slate-700">{t.noActivitiesFound}</p>
              <p className="text-xs text-slate-400 mt-1">
                Activities logged here are automatically included in Monthly Care Reports.
              </p>
              {canModify && (
                <Button onClick={() => openCreateModal()} variant="outline" className="mt-4">
                  <Plus className="w-4 h-4 mr-1" />
                  {t.addActivity}
                </Button>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-slate-50 border-b border-slate-200 text-xs uppercase font-semibold text-slate-500">
                    <th className="py-3.5 px-4">{t.activityTitle}</th>
                    <th className="py-3.5 px-4">{t.clientDetails}</th>
                    <th className="py-3.5 px-4">{t.activityCategory}</th>
                    <th className="py-3.5 px-4">{t.activityPerformedAt}</th>
                    <th className="py-3.5 px-4">{t.activityDuration}</th>
                    <th className="py-3.5 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 text-sm text-slate-700">
                  {displayedActivities.map((act) => (
                    <tr key={act.id} className="hover:bg-slate-50 transition-colors">
                      <td className="py-3.5 px-4">
                        <div className="font-semibold text-slate-900">{act.title}</div>
                        <div className="text-xs text-slate-500 line-clamp-1 max-w-md">
                          {act.description}
                        </div>
                      </td>
                      <td className="py-3.5 px-4">
                        <Link
                          href={`/clients/${act.client_id}`}
                          className="font-medium text-sky-600 hover:text-sky-800 flex items-center gap-1"
                        >
                          {act.client_name}
                          <ExternalLink className="w-3 h-3" />
                        </Link>
                        {act.asset_name && (
                          <div className="text-xs text-slate-400">{act.asset_name}</div>
                        )}
                      </td>
                      <td className="py-3.5 px-4">
                        <Badge variant="neutral" className="capitalize">
                          {act.category.replace("_", " ")}
                        </Badge>
                      </td>
                      <td className="py-3.5 px-4 text-xs text-slate-600">
                        <div className="flex items-center gap-1.5">
                          <Calendar className="w-3.5 h-3.5 text-slate-400" />
                          {new Date(act.performed_at).toLocaleDateString()}
                        </div>
                        <div className="text-slate-400 pl-5">
                          {new Date(act.performed_at).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </div>
                      </td>
                      <td className="py-3.5 px-4 text-xs text-slate-600">
                        {act.duration_minutes ? (
                          <div className="flex items-center gap-1">
                            <Clock className="w-3.5 h-3.5 text-slate-400" />
                            {act.duration_minutes} min
                          </div>
                        ) : (
                          <span className="text-slate-400">—</span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-right">
                        {canModify && (
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => openEditModal(act)}
                              className="p-1.5 text-slate-500 hover:text-sky-600 hover:bg-slate-100 rounded transition-colors"
                              title={t.editActivity}
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => setDeleteId(act.id)}
                              className="p-1.5 text-slate-500 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
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
          )}
        </div>
      </div>

      {/* Create/Edit Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl max-w-lg w-full p-6 space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between border-b pb-3">
              <h3 className="font-bold text-slate-900 text-lg flex items-center gap-2">
                <Wrench className="w-5 h-5 text-sky-600" />
                {editingId ? t.editActivity : t.addActivity}
              </h3>
              <button
                onClick={() => setModalOpen(false)}
                className="text-slate-400 hover:text-slate-600 p-1"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {formError && (
              <div className="p-3 bg-red-50 border border-red-200 text-red-700 rounded text-sm">
                {formError}
              </div>
            )}

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  Client *
                </label>
                <select
                  disabled={!!editingId}
                  value={formData.client_id}
                  onChange={(e) => setFormData({ ...formData, client_id: e.target.value, asset_id: "" })}
                  className="w-full border border-slate-300 rounded-lg p-2 text-sm bg-white"
                >
                  <option value="">Select Client...</option>
                  {clients.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </div>

              {filteredAssets.length > 0 && (
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.activityAsset} (Optional)
                  </label>
                  <select
                    value={formData.asset_id}
                    onChange={(e) => setFormData({ ...formData, asset_id: e.target.value })}
                    className="w-full border border-slate-300 rounded-lg p-2 text-sm bg-white"
                  >
                    <option value="">None (General Client Maintenance)</option>
                    {filteredAssets.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.activityCategory} *
                </label>
                <select
                  value={formData.category}
                  onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                  className="w-full border border-slate-300 rounded-lg p-2 text-sm bg-white capitalize"
                >
                  {CATEGORIES.map((cat) => (
                    <option key={cat} value={cat}>
                      {cat.replace("_", " ")}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.activityTitle} *
                </label>
                <Input
                  type="text"
                  required
                  maxLength={200}
                  placeholder="e.g. WordPress core updates & PHP 8.3 testing"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                  {t.activityDescription} *
                </label>
                <textarea
                  required
                  rows={3}
                  className="w-full border border-slate-300 rounded-lg p-2 text-sm focus:outline-none focus:ring-2 focus:ring-sky-500"
                  placeholder="Explain what was accomplished, patches applied, or issues resolved..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.activityPerformedAt} *
                  </label>
                  <Input
                    type="datetime-local"
                    required
                    value={formData.performed_at}
                    onChange={(e) => setFormData({ ...formData, performed_at: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-1">
                    {t.activityDuration}
                  </label>
                  <Input
                    type="number"
                    min="0"
                    max="14400"
                    placeholder="30"
                    value={formData.duration_minutes}
                    onChange={(e) => setFormData({ ...formData, duration_minutes: e.target.value })}
                  />
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t">
                <Button type="button" variant="outline" onClick={() => setModalOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={saving}>
                  {saving ? t.loading : editingId ? "Save Changes" : t.addActivity}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl max-w-sm w-full p-6 space-y-4">
            <h3 className="font-bold text-slate-900 text-lg flex items-center gap-2 text-red-600">
              <Trash2 className="w-5 h-5" />
              {t.deleteActivity}
            </h3>
            <p className="text-sm text-slate-600">
              Are you sure you want to delete this maintenance activity? If it was already included
              in an issued monthly care report, an audit log will be created.
            </p>
            <div className="flex items-center justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDeleteId(null)}>
                Cancel
              </Button>
              <Button
                variant="danger"
                disabled={deleting}
                onClick={handleDelete}
              >
                {deleting ? t.loading : "Delete Activity"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
