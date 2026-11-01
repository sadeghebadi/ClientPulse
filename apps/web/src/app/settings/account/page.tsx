"use client";

import React, { useState } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { User, ShieldCheck, KeyRound, CheckCircle2, Mail } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { useToast } from "@/components/ui/Toast";

export default function AccountSettingsPage() {
  const { user, loading } = useAuth();
  const { t } = useI18n();
  const toast = useToast();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState<any>(null);
  const [passwordLoading, setPasswordLoading] = useState(false);

  const [resendingEmail, setResendingEmail] = useState(false);

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwordLoading) return;
    setPasswordError(null);

    if (newPassword.length < 12) {
      setPasswordError(new Error(t.passwordTooShort));
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordError(new Error(t.passwordsDoNotMatch));
      return;
    }

    setPasswordLoading(true);
    try {
      await apiFetch("/api/v1/auth/change-password", {
        method: "POST",
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      toast.success("Password changed successfully");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err) {
      setPasswordError(err);
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleResendVerification = async () => {
    setResendingEmail(true);
    try {
      await apiFetch("/api/v1/auth/resend-verification", { method: "POST" });
      toast.success("Verification email dispatched. Please check your inbox.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to resend verification email");
    } finally {
      setResendingEmail(false);
    }
  };

  const isEmailVerified = Boolean(user?.email_verified_at);

  return (
    <AppLayout>
      <PageHeader
        title={t.accountSettings}
        description="Manage personal credentials, email verification and password security."
        icon={<User className="w-5 h-5" />}
      />

      {loading ? (
        <TableSkeleton rows={3} cols={2} />
      ) : user ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Profile Card */}
          <div className="bg-white border border-slate-200 rounded-2xl p-6 sm:p-8 shadow-sm space-y-5">
            <div className="flex items-center gap-3 pb-4 border-b border-slate-100">
              <div className="w-10 h-10 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center">
                <User className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-base font-bold text-slate-900">{t.profile}</h2>
                <p className="text-xs text-slate-500">Your registered operator identity</p>
              </div>
            </div>

            <div>
              <label className="block text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-1">
                {t.displayName}
              </label>
              <p className="text-sm font-bold text-slate-900">{user.display_name}</p>
            </div>

            <div>
              <label className="block text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-1">
                {t.email}
              </label>
              <div className="flex items-center justify-between">
                <p className="text-sm font-bold text-slate-900 font-mono">{user.email}</p>
                {isEmailVerified ? (
                  <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2.5 py-0.5 rounded-full border border-emerald-200">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    Verified
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-700 bg-amber-50 px-2.5 py-0.5 rounded-full border border-amber-200">
                    Unverified
                  </span>
                )}
              </div>

              {!isEmailVerified && (
                <div className="mt-4 p-4 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-800">
                  <p className="mb-3">
                    Your email address has not been verified yet. Mutating sensitive operations require verification.
                  </p>
                  <Button
                    size="sm"
                    variant="outline"
                    loading={resendingEmail}
                    onClick={handleResendVerification}
                    icon={<Mail className="w-3.5 h-3.5" />}
                  >
                    {t.resendVerification}
                  </Button>
                </div>
              )}
            </div>
          </div>

          {/* Change Password Card */}
          <div className="bg-white border border-slate-200 rounded-2xl p-6 sm:p-8 shadow-sm">
            <div className="flex items-center gap-3 pb-4 border-b border-slate-100 mb-5">
              <div className="w-10 h-10 rounded-xl bg-indigo-50 text-indigo-600 flex items-center justify-center">
                <KeyRound className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-base font-bold text-slate-900">{t.changePassword}</h2>
                <p className="text-xs text-slate-500">Must be at least 12 characters long</p>
              </div>
            </div>

            {passwordError && <ErrorState error={passwordError} className="mb-4" />}

            <form onSubmit={handleChangePassword} className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.currentPassword}
                </label>
                <input
                  type="password"
                  required
                  disabled={passwordLoading}
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  placeholder="••••••••••••"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.newPassword}
                </label>
                <input
                  type="password"
                  required
                  minLength={12}
                  disabled={passwordLoading}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  placeholder="••••••••••••"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                  {t.confirmPassword}
                </label>
                <input
                  type="password"
                  required
                  minLength={12}
                  disabled={passwordLoading}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                  placeholder="••••••••••••"
                />
              </div>

              <div className="pt-2">
                <Button
                  type="submit"
                  variant="primary"
                  size="md"
                  loading={passwordLoading}
                  className="w-full"
                >
                  {t.changePasswordBtn}
                </Button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </AppLayout>
  );
}
