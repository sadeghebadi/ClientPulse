"use client";

import React from "react";
import { AlertCircle, RotateCcw, ShieldAlert } from "lucide-react";
import { Button } from "./Button";
import { useI18n } from "@/lib/i18n";
import { humanizeApiError } from "@/lib/api";

interface ErrorStateProps {
  error: unknown;
  onRetry?: () => void;
  className?: string;
  title?: string;
}

export function ErrorState({ error, onRetry, className = "", title }: ErrorStateProps) {
  const { t, isRTL } = useI18n();

  if (!error) return null;

  const humanized = humanizeApiError(error, isRTL);
  const isSSRF =
    String(error).includes("SSRF") ||
    String(error).includes("restricted internal or private ip") ||
    String(error).includes("loopback");

  return (
    <div
      role="alert"
      className={`p-4 sm:p-5 rounded-2xl border text-xs transition-all ${
        isSSRF
          ? "bg-amber-50/70 border-amber-200 text-amber-900"
          : "bg-rose-50/70 border-rose-200 text-rose-900"
      } ${className}`}
    >
      <div className="flex items-start gap-3">
        <div className="shrink-0 mt-0.5">
          {isSSRF ? (
            <ShieldAlert className="w-5 h-5 text-amber-600" />
          ) : (
            <AlertCircle className="w-5 h-5 text-rose-600" />
          )}
        </div>
        <div className="flex-1">
          <h4 className="font-bold text-slate-900 mb-1">
            {title || (isSSRF ? t.ssrfErrorTitle : t.errorResult)}
          </h4>
          <p className="leading-relaxed text-slate-700">{humanized}</p>
          {onRetry && (
            <div className="mt-3">
              <Button
                variant="outline"
                size="sm"
                onClick={onRetry}
                icon={<RotateCcw className="w-3.5 h-3.5" />}
              >
                {t.refresh}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

