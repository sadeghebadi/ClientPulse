"use client";

import React, { useState, useEffect, Suspense } from "react";
import Link from "next/link";
import { useSearchParams, useRouter } from "next/navigation";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { MailCheck, CheckCircle2, AlertCircle } from "lucide-react";

function VerifyEmailContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") || "";

  const { user, refreshUser } = useAuth();
  const { t } = useI18n();

  const [status, setStatus] = useState<"pending" | "success" | "error">(() => (token ? "pending" : "error"));
  const [errorMessage, setErrorMessage] = useState<string | null>(() => (token ? null : "No verification token provided in URL."));
  const [resending, setResending] = useState(false);
  const [resendSuccess, setResendSuccess] = useState(false);

  useEffect(() => {
    if (!token) {
      return;
    }

    let isMounted = true;
    apiFetch("/api/v1/auth/verify-email", {
      method: "POST",
      body: JSON.stringify({ token }),
    })
      .then(() => {
        if (isMounted) {
          setStatus("success");
          refreshUser();
        }
      })
      .catch((err: unknown) => {
        if (isMounted) {
          setStatus("error");
          setErrorMessage(err instanceof Error ? err.message : "Verification failed");
        }
      });

    return () => {
      isMounted = false;
    };
  }, [token, refreshUser]);

  const handleResend = async () => {
    setResending(true);
    setResendSuccess(false);
    try {
      await apiFetch("/api/v1/auth/resend-verification", { method: "POST" });
      setResendSuccess(true);
    } catch (err: unknown) {
      setErrorMessage(err instanceof Error ? err.message : "Failed to resend email");
    } finally {
      setResending(false);
    }
  };

  return (
    <div className="text-center py-4">
      {status === "pending" && (
        <div>
          <div className="w-12 h-12 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center mx-auto mb-4 animate-pulse">
            <MailCheck className="w-6 h-6" />
          </div>
          <p className="text-sm text-slate-700">{t.verifyEmailPending}</p>
        </div>
      )}

      {status === "success" && (
        <div>
          <div className="w-12 h-12 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <h3 className="text-base font-bold text-slate-900 mb-2">{t.verifyEmailSuccess}</h3>
          <p className="text-xs text-slate-600 mb-6">Your email address has been verified.</p>
          <button
            onClick={() => router.push("/")}
            className="w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold text-sm rounded-lg shadow-sm transition-colors"
          >
            {t.verifyEmailBtn}
          </button>
        </div>
      )}

      {status === "error" && (
        <div>
          <div className="w-12 h-12 bg-red-100 text-red-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h3 className="text-base font-bold text-slate-900 mb-2">Verification Failed</h3>
          <p className="text-xs text-red-600 mb-6">{errorMessage || "Invalid or expired link."}</p>

          {user && (
            <div className="space-y-3">
              <button
                onClick={handleResend}
                disabled={resending}
                className="w-full py-2.5 px-4 bg-slate-100 hover:bg-slate-200 text-slate-800 font-semibold text-xs rounded-lg transition-colors"
              >
                {resending ? t.loading : t.resendVerification}
              </button>
              {resendSuccess && (
                <p className="text-xs text-emerald-600">Verification email sent! Check your inbox.</p>
              )}
            </div>
          )}

          <div className="mt-4">
            <Link href="/" className="text-xs font-semibold text-blue-600 hover:underline">
              {t.verifyEmailBtn}
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}

export default function VerifyEmailPage() {
  const { t } = useI18n();

  return (
    <div className="min-h-screen flex flex-col bg-slate-50">
      <Header />
      <div className="flex-1 flex items-center justify-center p-4">
        <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl p-8 shadow-sm">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-10 h-10 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center">
              <MailCheck className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-slate-900">{t.verifyEmailTitle}</h2>
              <p className="text-xs text-slate-500">{t.tagline}</p>
            </div>
          </div>

          <Suspense fallback={<div className="py-8 text-center text-xs text-slate-500">{t.loading}</div>}>
            <VerifyEmailContent />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
