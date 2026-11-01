"use client";

import React, { useState, useEffect, useCallback } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Users, UserPlus, Shield, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";

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
  const toast = useToast();

  const [members, setMembers] = useState<MemberItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  const [newEmail, setNewEmail] = useState("");
  const [newRole, setNewRole] = useState("member");
  const [addLoading, setAddLoading] = useState(false);

  // Remove Modal
  const [targetMember, setTargetMember] = useState<MemberItem | null>(null);
  const [removeModalOpen, setRemoveModalOpen] = useState(false);
  const [removeLoading, setRemoveLoading] = useState(false);

  const loadMembers = useCallback(async () => {
    if (!activeOrg) return;
    try {
      const data = await apiFetch<{ members: MemberItem[] }>(
        "/api/v1/organizations/current/members",
        {
          headers: { "X-Organization-ID": activeOrg.id },
        }
      );
      setMembers(data.members || []);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [activeOrg]);

  useEffect(() => {
    if (!activeOrg) return;
    let isMounted = true;
    apiFetch<{ members: MemberItem[] }>(
      "/api/v1/organizations/current/members",
      {
        headers: { "X-Organization-ID": activeOrg.id },
      }
    )
      .then((data) => {
        if (isMounted) {
          setMembers(data.members || []);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err);
        }
      })
      .finally(() => {
        if (isMounted) {
          setLoading(false);
        }
      });
    return () => {
      isMounted = false;
    };
  }, [activeOrg]);

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newEmail.trim() || !activeOrg || addLoading) return;

    setAddLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/organizations/current/members", {
        method: "POST",
        headers: { "X-Organization-ID": activeOrg.id },
        body: JSON.stringify({ email: newEmail.trim(), role: newRole }),
      });
      toast.success(`Member invited: ${newEmail.trim()}`);
      setNewEmail("");
      await loadMembers();
    } catch (err) {
      setError(err);
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
      toast.success("Member role updated");
      await loadMembers();
    } catch (err) {
      setError(err);
    }
  };

  const handleConfirmRemove = async () => {
    if (!activeOrg || !targetMember) return;
    try {
      setRemoveLoading(true);
      await apiFetch(`/api/v1/organizations/current/members/${targetMember.user_id}`, {
        method: "DELETE",
        headers: { "X-Organization-ID": activeOrg.id },
      });
      toast.success("Member removed from organization");
      setRemoveModalOpen(false);
      setTargetMember(null);
      await loadMembers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove member");
    } finally {
      setRemoveLoading(false);
    }
  };

  const canManageMembers = role === "owner" || role === "admin";
  const isOwner = role === "owner";

  return (
    <AppLayout>
      <PageHeader
        title={t.members}
        description={`Manage team members and authorization roles for ${activeOrg?.name || "your workspace"}.`}
        icon={<Users className="w-5 h-5" />}
      />

      {error && <ErrorState error={error} className="mb-6" />}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        {/* Members List */}
        <div className="md:col-span-2 space-y-4">
          {loading ? (
            <TableSkeleton rows={4} cols={3} />
          ) : members.length === 0 ? (
            <div className="p-8 text-center bg-white rounded-2xl border border-slate-200 text-xs text-slate-500">
              {t.noMembers}
            </div>
          ) : (
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden divide-y divide-slate-100">
              {members.map((m) => {
                const isSelf = m.user_id === user?.id;
                return (
                  <div key={m.id} className="p-4 sm:p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-sm text-slate-900">
                          {m.user_display_name || "Team Member"}
                        </span>
                        {isSelf && (
                          <span className="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-blue-100 text-blue-800">
                            You
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5">{m.user_email}</div>
                    </div>

                    <div className="flex items-center gap-3">
                      {isOwner && !isSelf ? (
                        <select
                          value={m.role}
                          onChange={(e) => handleRoleChange(m.user_id, e.target.value)}
                          className="px-3 py-1.5 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          <option value="owner">{t.roleOwner}</option>
                          <option value="admin">{t.roleAdmin}</option>
                          <option value="member">{t.roleMember}</option>
                          <option value="viewer">{t.roleViewer}</option>
                        </select>
                      ) : (
                        <Badge variant={m.role} />
                      )}

                      {canManageMembers && !isSelf && (
                        <button
                          onClick={() => {
                            setTargetMember(m);
                            setRemoveModalOpen(true)}
                          }
                          className="p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-50 rounded-lg transition-colors border border-slate-200"
                          title={t.remove}
                          aria-label={t.remove}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Add Member Form (Admin/Owner only) */}
        {canManageMembers && (
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm h-fit">
            <div className="flex items-center gap-2 mb-4">
              <UserPlus className="w-4 h-4 text-blue-600" />
              <h2 className="text-sm font-bold text-slate-900">{t.addMember}</h2>
            </div>

            <form onSubmit={handleAddMember} className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.memberEmail}
                </label>
                <input
                  type="email"
                  required
                  disabled={addLoading}
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  placeholder="colleague@agency.com"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.role}
                </label>
                <select
                  disabled={addLoading}
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  <option value="admin">{t.roleAdmin} (Manage monitors, clients, reports)</option>
                  <option value="member">{t.roleMember} (Standard view & maintenance)</option>
                  <option value="viewer">{t.roleViewer} (Read-only dashboard access)</option>
                </select>
              </div>

              <Button
                type="submit"
                variant="primary"
                size="md"
                loading={addLoading}
                className="w-full"
              >
                {t.addMember}
              </Button>
            </form>
          </div>
        )}
      </div>

      {/* Remove Confirmation Modal */}
      <ConfirmModal
        isOpen={removeModalOpen}
        onClose={() => {
          setRemoveModalOpen(false);
          setTargetMember(null);
        }}
        onConfirm={handleConfirmRemove}
        title={t.remove}
        message={t.confirmRemoveMember}
        confirmLabel={t.remove}
        variant="danger"
        loading={removeLoading}
      />
    </AppLayout>
  );
}
