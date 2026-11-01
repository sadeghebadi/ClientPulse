"use client";

import React, { useState, useEffect, Suspense } from "react";
import Link from "next/link";
import { useSearchParams, useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { MailCheck, CheckCircle2, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/Button";

function VerifyEmailContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") || "";

  const { user, refreshUser } = useAuth();
  const { t } = useI18n();

  const [status, setStatus] = useState<"pending" | "success" | "error">(() =>
    token ? "pending" : "error"
  );
  const [errorMessage, setErrorMessage] = useState<string | null>(() =>
    token ? null : "No verification token provided in URL."
  );
  const [resending, setResending] = useState(false);
  const [resendSuccess, setResendSuccess] = useState(false);

  useEffect(() => {
    if (!token) return;

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
    <div className="text-center py-4 space-y-4">
      {status === "pending" && (
        <div>
          <div className="w-12 h-12 bg-blue-100 text-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-4 animate-pulse">
            <MailCheck className="w-6 h-6" />
          </div>
          <p className="text-sm font-semibold text-slate-700">{t.verifyEmailPending}</p>
        </div>
      )}

      {status === "success" && (
        <div>
          <div className="w-12 h-12 bg-emerald-100 text-emerald-600 rounded-2xl flex items-center justify-center mx-auto mb-4">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <h3 className="text-base font-bold text-slate-900 mb-1">{t.verifyEmailSuccess}</h3>
          <p className="text-xs text-slate-600 mb-6">Your email address has been verified.</p>
          <Button
            variant="primary"
            size="lg"
            onClick={() => router.push("/")}
            className="w-full"
          >
            {t.verifyEmailBtn}
          </Button>
        </div>
      )}

      {status === "error" && (
        <div>
          <div className="w-12 h-12 bg-rose-100 text-rose-600 rounded-2xl flex items-center justify-center mx-auto mb-4">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h3 className="text-base font-bold text-slate-900 mb-1">Verification Failed</h3>
          <p className="text-xs text-rose-600 mb-6">{errorMessage || "Invalid or expired link."}</p>

          {user && (
            <div className="space-y-3">
              <Button
                variant="outline"
                size="md"
                onClick={handleResend}
                loading={resending}
                className="w-full"
              >
                {t.resendVerification}
              </Button>
              {resendSuccess && (
                <p className="text-xs text-emerald-600 font-semibold">
                  Verification email sent! Check your inbox.
                </p>
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
    <AppLayout>
      <div className="flex-1 flex items-center justify-center py-12 px-4">
        <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl p-8 sm:p-10 shadow-lg">
          <div className="flex items-center gap-3.5 mb-6">
            <div className="w-11 h-11 rounded-2xl bg-blue-50 text-blue-600 flex items-center justify-center shadow-inner">
              <MailCheck className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-extrabold text-slate-900 tracking-tight">{t.verifyEmailTitle}</h2>
              <p className="text-xs text-slate-500 font-medium">{t.tagline}</p>
            </div>
          </div>

          <Suspense fallback={<div className="py-8 text-center text-xs text-slate-500">{t.loading}</div>}>
            <VerifyEmailContent />
          </Suspense>
        </div>
      </div>
    </AppLayout>
  );
}
