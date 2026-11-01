"use client";

import React, { useState, useEffect, useCallback } from "react";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Users, UserPlus, Shield, Trash2, AlertCircle } from "lucide-react";

interface MemberItem {
  id: string;
  organization_id: string;
  user_id: string;
  role: string;
  user_email?: string;
  user_display_name?: string;
}

export default function MembersSettingsPage() {
  const { user, activeOrg, role } = useAuth();
  const { t } = useI18n();

  const [members, setMembers] = useState<MemberItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [newEmail, setNewEmail] = useState("");
  const [newRole, setNewRole] = useState("member");
  const [addLoading, setAddLoading] = useState(false);

  const fetchMembers = useCallback(async () => {
    if (!activeOrg) return;
    try {
      const data = await apiFetch<{ members: MemberItem[] }>("/api/v1/organizations/current/members", {
        headers: { "X-Organization-ID": activeOrg.id },
      });
      setMembers(data.members || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load members");
    } finally {
      setLoading(false);
    }
  }, [activeOrg]);

  useEffect(() => {
    if (!activeOrg) return;
    let isMounted = true;
    apiFetch<{ members: MemberItem[] }>("/api/v1/organizations/current/members", {
      headers: { "X-Organization-ID": activeOrg.id },
    })
      .then((data) => {
        if (isMounted) {
          setMembers(data.members || []);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (isMounted) {
          setError(err instanceof Error ? err.message : "Failed to load members");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [activeOrg]);

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newEmail.trim() || !activeOrg) return;

    setAddLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/organizations/current/members", {
        method: "POST",
        headers: { "X-Organization-ID": activeOrg.id },
        body: JSON.stringify({ email: newEmail, role: newRole }),
      });
      setNewEmail("");
      await fetchMembers();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to add member");
    } finally {
      setAddLoading(false);
    }
  };

  const handleRoleChange = async (targetUserID: string, roleValue: string) => {
    if (!activeOrg) return;
    setError(null);
    try {
      await apiFetch(`/api/v1/organizations/current/members/${targetUserID}`, {
        method: "PATCH",
        headers: { "X-Organization-ID": activeOrg.id },
        body: JSON.stringify({ role: roleValue }),
      });
      await fetchMembers();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to change role");
    }
  };

  const handleRemoveMember = async (targetUserID: string) => {
    if (!activeOrg) return;
    if (!confirm(t.confirmRemoveMember)) return;

    setError(null);
    try {
      await apiFetch(`/api/v1/organizations/current/members/${targetUserID}`, {
        method: "DELETE",
        headers: { "X-Organization-ID": activeOrg.id },
      });
      await fetchMembers();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to remove member");
    }
  };

  const canManageMembers = role === "owner" || role === "admin";
  const isOwner = role === "owner";

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />
      <main className="max-w-4xl mx-auto px-4 sm:px-6 py-8 w-full flex-1">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{t.members}</h1>
            <p className="text-xs text-slate-500 mt-1">
              Manage organization members and role permissions for {activeOrg?.name || "your workspace"}.
            </p>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2.5 text-xs text-red-700">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-red-500" />
            <span>{error}</span>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {/* Members List */}
          <div className="md:col-span-2 space-y-3">
            {loading ? (
              <p className="text-xs text-slate-500 py-4">{t.loading}</p>
            ) : members.length === 0 ? (
              <p className="text-xs text-slate-500 py-4">{t.noMembers}</p>
            ) : (
              members.map((m) => (
                <div
                  key={m.id}
                  className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm flex items-center justify-between"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-full bg-slate-100 text-slate-700 font-bold text-xs flex items-center justify-center">
                      {(m.user_display_name || m.user_email || "?").charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold text-slate-900">{m.user_display_name}</span>
                        {m.user_id === user?.id && (
                          <span className="text-[10px] text-slate-400 font-medium">(You)</span>
                        )}
                      </div>
                      <p className="text-[11px] text-slate-500">{m.user_email}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    {isOwner && m.user_id !== user?.id ? (
                      <select
                        value={m.role}
                        onChange={(e) => handleRoleChange(m.user_id, e.target.value)}
                        className="text-xs border border-slate-300 rounded-md px-2 py-1 bg-white font-medium text-slate-700"
                      >
                        <option value="owner">{t.roleOwner}</option>
                        <option value="admin">{t.roleAdmin}</option>
                        <option value="member">{t.roleMember}</option>
                        <option value="viewer">{t.roleViewer}</option>
                      </select>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-slate-700 bg-slate-100 px-2 py-0.5 rounded border border-slate-200 capitalize">
                        <Shield className="w-3 h-3 text-slate-400" />
                        {m.role}
                      </span>
                    )}

                    {canManageMembers && (isOwner || m.role !== "owner") && m.user_id !== user?.id && (
                      <button
                        onClick={() => handleRemoveMember(m.user_id)}
                        className="p-1 text-slate-400 hover:text-red-600 transition-colors"
                        title={t.remove}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Add Member Card */}
          {canManageMembers && (
            <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm h-fit">
              <div className="flex items-center gap-2 mb-4">
                <UserPlus className="w-4 h-4 text-blue-600" />
                <h2 className="text-sm font-bold text-slate-900">{t.addMember}</h2>
              </div>

              <form onSubmit={handleAddMember} className="space-y-3.5">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    {t.memberEmail}
                  </label>
                  <input
                    type="email"
                    required
                    value={newEmail}
                    onChange={(e) => setNewEmail(e.target.value)}
                    className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="teammate@agency.com"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    {t.role}
                  </label>
                  <select
                    value={newRole}
                    onChange={(e) => setNewRole(e.target.value)}
                    className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                  >
                    {isOwner && <option value="owner">{t.roleOwner}</option>}
                    <option value="admin">{t.roleAdmin}</option>
                    <option value="member">{t.roleMember}</option>
                    <option value="viewer">{t.roleViewer}</option>
                  </select>
                </div>

                <button
                  type="submit"
                  disabled={addLoading}
                  className="w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs rounded-lg shadow-sm transition-colors disabled:opacity-50"
                >
                  {addLoading ? t.loading : t.addMember}
                </button>
              </form>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
