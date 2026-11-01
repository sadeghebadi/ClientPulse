"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { UserPlus, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";

export default function SignupPage() {
  const router = useRouter();
  const { refreshUser } = useAuth();
  const { t, isRTL } = useI18n();

  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [orgName, setOrgName] = useState("");
  const [error, setError] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading) return;
    setError(null);

    if (password.length < 12) {
      setError(new Error(t.passwordTooShort));
      return;
    }

    setLoading(true);
    try {
      await apiFetch("/api/v1/auth/signup", {
        method: "POST",
        body: JSON.stringify({
          display_name: displayName.trim(),
          email: email.trim(),
          password,
          organization_name: orgName.trim() || undefined,
        }),
      });

      await refreshUser();
      router.push("/");
    } catch (err) {
      setError(err);
      setLoading(false);
    }
  };

  const isPasswordValid = password.length >= 12;

  return (
    <AppLayout>
      <div className="flex-1 flex items-center justify-center py-12 px-4">
        <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl p-8 sm:p-10 shadow-lg">
          <div className="flex items-center gap-3.5 mb-8">
            <div className="w-11 h-11 rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center shadow-inner">
              <UserPlus className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-extrabold text-slate-900 tracking-tight">{t.createAccount}</h2>
              <p className="text-xs text-slate-500 font-medium">{t.tagline}</p>
            </div>
          </div>

          {error && <ErrorState error={error} className="mb-6" />}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                {t.displayName}
              </label>
              <input
                type="text"
                name="display_name"
                required
                disabled={loading}
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                placeholder={isRTL ? "مثال: علی رضایی" : "e.g. Jane Doe"}
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
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
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                {t.agencyName}
              </label>
              <input
                type="text"
                name="org_name"
                disabled={loading}
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                placeholder={isRTL ? "مثال: آژانس دیجیتال سپهر" : "e.g. Acme Web Agency"}
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5">
                {t.password}
              </label>
              <input
                type="password"
                name="password"
                required
                minLength={12}
                disabled={loading}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                placeholder="••••••••••••"
              />
              <div className="flex items-center gap-1.5 text-[11px] mt-1.5">
                <CheckCircle2
                  className={`w-3.5 h-3.5 ${
                    isPasswordValid ? "text-emerald-600" : "text-slate-300"
                  }`}
                />
                <span className={isPasswordValid ? "text-emerald-700 font-semibold" : "text-slate-500"}>
                  {t.passwordTooShort}
                </span>
              </div>
            </div>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={loading}
              className="w-full mt-4"
            >
              {t.createAccount}
            </Button>
          </form>

          <div className="mt-8 pt-6 text-center text-xs text-slate-600 border-t border-slate-100">
            <span>{t.alreadyHaveAccount} </span>
            <Link href="/login" className="text-blue-600 hover:underline font-bold">
              {t.login}
            </Link>
          </div>
        </div>
      </div>
    </AppLayout>
  );
}
