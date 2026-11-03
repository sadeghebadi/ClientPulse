"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Users, ShieldAlert } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { useToast } from "@/components/ui/Toast";

interface ClientResponse {
  client: {
    id: string;
    name: string;
    contact_name?: string;
    contact_email?: string;
    website?: string;
    notes?: string;
  };
}

export default function EditClientPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [name, setName] = useState("");
  const [contactName, setContactName] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [website, setWebsite] = useState("");
  const [notes, setNotes] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<any>(null);

  const canManage = role === "owner" || role === "admin" || role === "member";

  useEffect(() => {
    async function loadClient() {
      try {
        setLoading(true);
        const data = await apiFetch<ClientResponse>(`/api/v1/clients/${resolvedParams.id}`);
        setName(data.client.name || "");
        setContactName(data.client.contact_name || "");
        setContactEmail(data.client.contact_email || "");
        setWebsite(data.client.website || "");
        setNotes(data.client.notes || "");
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    }
    if (user && activeOrg) {
      loadClient();
    }
  }, [user, activeOrg, resolvedParams.id]);

  if (role === "viewer" || !canManage) {
    return (
      <AppLayout>
        <div className="max-w-md mx-auto p-8 text-center bg-white rounded-2xl border border-slate-200 mt-12">
          <ShieldAlert className="w-12 h-12 text-rose-500 mx-auto mb-3" />
          <h2 className="text-base font-bold text-slate-900">{isRTL ? "دسترسی محدود است" : "Access Restricted"}</h2>
          <p className="text-xs text-slate-500 mt-1 mb-4">
            {isRTL
              ? "حساب‌های فقط-خواندنی (Viewer) اجازه ثبت یا ویرایش مشتری را ندارند."
              : "Viewer accounts do not have permission to create or edit clients."}
          </p>
          <Link href={`/clients/${resolvedParams.id}`}>
            <Button variant="outline" size="sm">
              {isRTL ? "بازگشت به مشتری" : "Back to Client"}
            </Button>
          </Link>
        </div>
      </AppLayout>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (saving) return; // Prevent duplicate submission
    if (!name.trim()) {
      setError(new Error(t.clientName + " is required"));
      return;
    }

    try {
      setSaving(true);
      setError(null);
      await apiFetch(`/api/v1/clients/${resolvedParams.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          contact_name: contactName.trim() || undefined,
          contact_email: contactEmail.trim() || undefined,
          website: website.trim() || undefined,
          notes: notes.trim() || undefined,
        }),
      });

      toast.success("Client details updated successfully");
      router.push(`/clients/${resolvedParams.id}`);
    } catch (err) {
      setError(err);
      setSaving(false);
    }
  };

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto">
        <PageHeader
          title={t.editClient}
          description="Update contact information, website, and maintenance notes."
          icon={<Users className="w-5 h-5" />}
          breadcrumbs={[
            { label: t.clients, href: "/clients" },
            { label: name || t.clientDetails, href: `/clients/${resolvedParams.id}` },
            { label: t.editClient },
          ]}
        />

        {loading ? (
          <TableSkeleton rows={4} cols={2} />
        ) : (
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6 sm:p-8">
            <form onSubmit={handleSubmit} className="space-y-6">
              {error && <ErrorState error={error} className="mb-4" />}

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.clientName} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  disabled={saving}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.contactName}
                  </label>
                  <input
                    type="text"
                    disabled={saving}
                    value={contactName}
                    onChange={(e) => setContactName(e.target.value)}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.contactEmail}
                  </label>
                  <input
                    type="email"
                    dir="ltr"
                    disabled={saving}
                    value={contactEmail}
                    onChange={(e) => setContactEmail(e.target.value)}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-start focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.website}
                </label>
                <input
                  type="url"
                  dir="ltr"
                  disabled={saving}
                  value={website}
                  onChange={(e) => setWebsite(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-start focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.notes}
                </label>
                <textarea
                  rows={3}
                  disabled={saving}
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div className="pt-4 flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-end gap-3 border-t border-slate-100">
                <Link href={`/clients/${resolvedParams.id}`} className="w-full sm:w-auto">
                  <Button variant="ghost" size="sm" disabled={saving} className="w-full sm:w-auto">
                    {t.cancel}
                  </Button>
                </Link>
                <Button type="submit" variant="primary" size="md" loading={saving} className="w-full sm:w-auto">
                  {t.saveChanges}
                </Button>
              </div>
            </form>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
