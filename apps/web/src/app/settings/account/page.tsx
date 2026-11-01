"use client";

import React, { useState } from "react";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { User, ShieldCheck, KeyRound, AlertCircle, CheckCircle2, Mail } from "lucide-react";

export default function AccountSettingsPage() {
  const { user, loading } = useAuth();
  const { t } = useI18n();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);

  const [resendingEmail, setResendingEmail] = useState(false);
  const [resendEmailSuccess, setResendEmailSuccess] = useState(false);

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError(null);
    setPasswordSuccess(false);

    if (newPassword.length < 12) {
      setPasswordError(t.passwordTooShort);
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordError(t.passwordsDoNotMatch);
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
      setPasswordSuccess(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err: unknown) {
      setPasswordError(err instanceof Error ? err.message : "Failed to change password");
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleResendVerification = async () => {
    setResendingEmail(true);
    setResendEmailSuccess(false);
    try {
      await apiFetch("/api/v1/auth/resend-verification", { method: "POST" });
      setResendEmailSuccess(true);
    } catch (err: unknown) {
      setPasswordError(err instanceof Error ? err.message : "Failed to resend verification email");
    } finally {
      setResendingEmail(false);
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
            <p className="text-sm text-slate-700 mb-4">Please log in to manage your account.</p>
            <a href="/login" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-xs font-semibold">
              {t.login}
            </a>
          </div>
        </div>
      </div>
    );
  }

  const isEmailVerified = Boolean(user.email_verified_at);

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />
      <main className="max-w-4xl mx-auto px-4 sm:px-6 py-8 w-full flex-1">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{t.accountSettings}</h1>
          <p className="text-xs text-slate-500 mt-1">Manage personal profile, email verification and password security.</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Profile Card */}
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm space-y-4">
            <div className="flex items-center gap-3 mb-2">
              <div className="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center">
                <User className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-slate-900">{t.profile}</h2>
                <p className="text-xs text-slate-500">Your personal details</p>
              </div>
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-1">
                {t.displayName}
              </label>
              <p className="text-sm font-medium text-slate-900">{user.display_name}</p>
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-1">
                {t.email}
              </label>
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-slate-900">{user.email}</p>
                {isEmailVerified ? (
                  <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                    <CheckCircle2 className="w-3 h-3" />
                    Verified
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-700 bg-amber-50 px-2 py-0.5 rounded-full border border-amber-200">
                    Unverified
                  </span>
                )}
              </div>

              {!isEmailVerified && (
                <div className="mt-3 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
                  <p className="mb-2">Your email has not been verified yet.</p>
                  <button
                    onClick={handleResendVerification}
                    disabled={resendingEmail}
                    className="px-3 py-1 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded text-[11px] transition-colors"
                  >
                    {resendingEmail ? t.loading : t.resendVerification}
                  </button>
                  {resendEmailSuccess && (
                    <p className="mt-2 text-emerald-700 font-medium">Verification link sent!</p>
                  )}
                </div>
              )}
            </div>

            <div className="pt-2 border-t border-slate-100 flex items-center gap-2 text-xs text-slate-500">
              <ShieldCheck className="w-4 h-4 text-emerald-600" />
              <span>{t.tenantIsolationVerified}</span>
            </div>
          </div>

          {/* Change Password Card */}
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center">
                <KeyRound className="w-5 h-5" />
              </div>
              <div>
                <h2 className="text-sm font-bold text-slate-900">{t.changePassword}</h2>
                <p className="text-xs text-slate-500">Argon2id password hashing</p>
              </div>
            </div>

            {passwordError && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2 text-xs text-red-700">
                <AlertCircle className="w-4 h-4 shrink-0 mt-0.5 text-red-500" />
                <span>{passwordError}</span>
              </div>
            )}

            {passwordSuccess && (
              <div className="mb-4 p-3 bg-emerald-50 border border-emerald-200 rounded-lg flex items-start gap-2 text-xs text-emerald-700">
                <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5 text-emerald-500" />
                <span>Password changed successfully. All previous sessions revoked.</span>
              </div>
            )}

            <form onSubmit={handleChangePassword} className="space-y-3.5">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t.currentPassword}
                </label>
                <input
                  type="password"
                  required
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t.newPassword}
                </label>
                <input
                  type="password"
                  required
                  minLength={12}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="••••••••••••"
                />
                <p className="text-[10px] text-slate-500 mt-0.5">{t.passwordTooShort}</p>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t.confirmPassword}
                </label>
                <input
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full px-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="••••••••••••"
                />
              </div>

              <button
                type="submit"
                disabled={passwordLoading}
                className="w-full py-2 px-3 bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs rounded-lg shadow-sm transition-colors disabled:opacity-50"
              >
                {passwordLoading ? t.loading : t.changePasswordBtn}
              </button>
            </form>
          </div>
        </div>
      </main>
    </div>
  );
}
