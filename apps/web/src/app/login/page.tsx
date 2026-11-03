"use client";

import React, { useState, Suspense } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { LogIn, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const sessionExpired = searchParams.get("session_expired") === "true";
  const { refreshUser } = useAuth();
  const { t, isRTL } = useI18n();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;
    setError(null);
    setLoading(true);

    try {
      await apiFetch("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email: email.trim(), password }),
      });

      await refreshUser();
      router.push("/dashboard");
    } catch (err) {
      setError(err);
      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl p-8 sm:p-10 shadow-lg">
      <div className="flex items-center gap-3.5 mb-8">
        <div className="w-11 h-11 rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center shadow-inner">
          <LogIn className="w-5 h-5" />
        </div>
        <div>
          <h2 className="text-xl font-extrabold text-slate-900 tracking-tight">{t.login}</h2>
          <p className="text-xs text-slate-500 font-medium">{t.tagline}</p>
        </div>
      </div>

      {sessionExpired && !error && (
        <div className="mb-6 p-3.5 bg-amber-50 border border-amber-200 rounded-xl text-amber-800 text-xs flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600" />
          <span>
            {isRTL
              ? "نشست قبلی شما منقضی یا نامعتبر شده است. لطفاً دوباره وارد شوید."
              : "Your previous session has expired or was invalidated. Please sign in again."}
          </span>
        </div>
      )}

      {error && <ErrorState error={error} className="mb-6" />}

      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
            {t.email}
          </label>
          <input
            type="email"
            name="email"
            required
            disabled={loading}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
            placeholder="name@agency.com"
          />
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider">
              {t.password}
            </label>
            <Link
              href="/forgot-password"
              className="text-xs text-blue-600 hover:underline font-semibold"
            >
              {t.forgotPassword}
            </Link>
          </div>
          <input
            type="password"
            name="password"
            required
            disabled={loading}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
            placeholder="••••••••••••"
          />
        </div>

        <Button
          type="submit"
          variant="primary"
          size="lg"
          loading={loading}
          className="w-full mt-2"
        >
          {t.login}
        </Button>
      </form>

      <div className="mt-8 pt-6 text-center text-xs text-slate-600 border-t border-slate-100">
        <span>{t.dontHaveAccount} </span>
        <Link href="/signup" className="text-blue-600 hover:underline font-bold">
          {t.signup}
        </Link>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <AppLayout>
      <div className="flex-1 flex items-center justify-center py-12 px-4">
        <Suspense fallback={<div className="p-8 text-center text-slate-400">Loading...</div>}>
          <LoginForm />
        </Suspense>
      </div>
    </AppLayout>
  );
}
