"use client";

import React from "react";
import { Loader2 } from "lucide-react";

export type ButtonVariant =
  | "primary"
  | "secondary"
  | "outline"
  | "ghost"
  | "danger"
  | "destructive"
  | "link";

export type ButtonSize = "sm" | "md" | "lg";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  icon?: React.ReactNode;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      children,
      variant = "primary",
      size = "md",
      loading = false,
      disabled = false,
      icon,
      className = "",
      ...props
    },
    ref
  ) => {
    // Normalise destructive to danger
    const effectiveVariant = variant === "destructive" ? "danger" : variant;

    const baseClasses =
      "inline-flex items-center justify-center font-medium rounded-lg transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed select-none active:scale-[0.98]";

    const sizeClasses: Record<ButtonSize, string> = {
      sm: "text-xs px-3 py-1.5 min-h-[32px] gap-1.5",
      md: "text-xs sm:text-sm px-4 py-2 min-h-[38px] gap-2",
      lg: "text-sm sm:text-base px-5 py-2.5 min-h-[44px] gap-2.5",
    };

    const variantClasses: Record<string, string> = {
      primary:
        "bg-brand-600 hover:bg-brand-700 text-white shadow-sm border border-brand-600 focus-visible:ring-brand-500",
      secondary:
        "bg-slate-100 hover:bg-slate-200 text-slate-800 border border-slate-200 focus-visible:ring-slate-400",
      outline:
        "bg-white hover:bg-slate-50 text-slate-700 border border-slate-300 shadow-sm focus-visible:ring-brand-500",
      danger:
        "bg-rose-600 hover:bg-rose-700 text-white shadow-sm border border-rose-600 focus-visible:ring-rose-500",
      ghost:
        "bg-transparent hover:bg-slate-100 text-slate-700 focus-visible:ring-slate-400",
      link:
        "bg-transparent text-brand-600 hover:text-brand-700 underline-offset-4 hover:underline p-0 min-h-0 focus-visible:ring-brand-500 shadow-none border-0",
    };

    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={`${baseClasses} ${sizeClasses[size] || sizeClasses.md} ${variantClasses[effectiveVariant] || variantClasses.primary} ${className}`}
        {...props}
      >
        {loading ? (
          <Loader2 className="w-4 h-4 animate-spin shrink-0" />
        ) : (
          icon && <span className="shrink-0 flex items-center">{icon}</span>
        )}
        {children && <span>{children}</span>}
      </button>
    );
  }
);

Button.displayName = "Button";
