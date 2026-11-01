"use client";

import React, { useState, Suspense } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Lock, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") || "";

  const { t } = useI18n();

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<any>(null);
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;
    setError(null);

    if (password.length < 12) {
      setError(new Error(t.passwordTooShort));
      return;
    }

    if (password !== confirmPassword) {
      setError(new Error(t.passwordsDoNotMatch));
      return;
    }

    setLoading(true);
    try {
      await apiFetch("/api/v1/auth/reset-password", {
        method: "POST",
        body: JSON.stringify({ token, password }),
      });
      setSuccess(true);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  if (!token) {
    return (
      <div className="text-center py-4 space-y-4">
        <p className="text-xs text-rose-600">
          Invalid or missing reset token. Please request a new link.
        </p>
        <Link href="/forgot-password">
          <Button variant="outline" size="sm">
            {t.forgotPasswordTitle}
          </Button>
        </Link>
      </div>
    );
  }

  if (success) {
    return (
      <div className="text-center py-4 space-y-4">
        <div className="w-12 h-12 bg-emerald-100 text-emerald-600 rounded-2xl flex items-center justify-center mx-auto mb-2">
          <CheckCircle2 className="w-6 h-6" />
        </div>
        <h3 className="text-base font-bold text-slate-900">Password Reset Successful</h3>
        <p className="text-xs text-slate-600 mb-6 max-w-xs mx-auto">
          Your password has been updated. Please log in with your new credentials.
        </p>
        <Button
          variant="primary"
          size="lg"
          onClick={() => router.push("/login")}
          className="w-full"
        >
          {t.login}
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <ErrorState error={error} className="mb-4" />}

      <div>
        <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
          {t.newPassword}
        </label>
        <input
          type="password"
          required
          minLength={12}
          disabled={loading}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
          placeholder="••••••••••••"
        />
        <p className="text-[11px] text-slate-500 mt-1">{t.passwordTooShort}</p>
      </div>

      <div>
        <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
          {t.confirmPassword}
        </label>
        <input
          type="password"
          required
          minLength={12}
          disabled={loading}
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
          placeholder="••••••••••••"
        />
      </div>

      <Button
        type="submit"
        variant="primary"
        size="lg"
        loading={loading}
        className="w-full mt-4"
      >
        {t.resetPasswordBtn}
      </Button>
    </form>
  );
}

export default function ResetPasswordPage() {
  const { t } = useI18n();

  return (
    <AppLayout>
      <div className="flex-1 flex items-center justify-center py-12 px-4">
        <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl p-8 sm:p-10 shadow-lg">
          <div className="flex items-center gap-3.5 mb-8">
            <div className="w-11 h-11 rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center shadow-inner">
              <Lock className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-extrabold text-slate-900 tracking-tight">{t.resetPasswordTitle}</h2>
              <p className="text-xs text-slate-500 font-medium">{t.resetPasswordDesc}</p>
            </div>
          </div>

          <Suspense fallback={<div className="py-8 text-center text-xs text-slate-500">{t.loading}</div>}>
            <ResetPasswordForm />
          </Suspense>
        </div>
      </div>
    </AppLayout>
  );
}
