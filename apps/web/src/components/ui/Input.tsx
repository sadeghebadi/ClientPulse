"use client";

import React from "react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  helperText?: string;
  errorMessage?: string;
  error?: boolean;
  isRequired?: boolean;
  containerClassName?: string;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  (
    {
      id,
      label,
      helperText,
      errorMessage,
      error = false,
      isRequired = false,
      className = "",
      containerClassName = "",
      disabled = false,
      type = "text",
      ...props
    },
    ref
  ) => {
    const inputId = id || (label ? `input-${label.toLowerCase().replace(/\s+/g, "-")}` : undefined);
    const hasError = error || Boolean(errorMessage);

    // Numeric and technical input types should remain LTR for consistent entry even in RTL
    const isTechnical = ["email", "url", "password", "number", "tel"].includes(type);

    return (
      <div className={`w-full flex flex-col gap-1.5 ${containerClassName}`}>
        {label && (
          <label
            htmlFor={inputId}
            className="text-xs font-semibold text-slate-700 select-none flex items-center justify-between"
          >
            <span>
              {label}
              {isRequired && <span className="text-rose-500 ms-1" aria-hidden="true">*</span>}
            </span>
          </label>
        )}

        <input
          ref={ref}
          id={inputId}
          type={type}
          disabled={disabled}
          aria-invalid={hasError ? "true" : undefined}
          aria-describedby={
            errorMessage && inputId
              ? `${inputId}-error`
              : helperText && inputId
              ? `${inputId}-helper`
              : undefined
          }
          className={`w-full px-3.5 py-2 min-h-[40px] bg-white border rounded-lg text-sm transition-colors duration-150 focus:outline-none focus:ring-2 disabled:opacity-50 disabled:bg-slate-50 disabled:cursor-not-allowed ${
            isTechnical ? "ltr-text" : ""
          } ${
            hasError
              ? "border-rose-400 focus:border-rose-500 focus:ring-rose-500/20 text-slate-900"
              : "border-slate-300 focus:border-brand-500 focus:ring-brand-500/20 text-slate-900"
          } ${className}`}
          {...props}
        />

        {errorMessage ? (
          <p
            id={inputId ? `${inputId}-error` : undefined}
            className="text-xs text-rose-600 font-medium"
            role="alert"
          >
            {errorMessage}
          </p>
        ) : helperText ? (
          <p
            id={inputId ? `${inputId}-helper` : undefined}
            className="text-xs text-slate-500"
          >
            {helperText}
          </p>
        ) : null}
      </div>
    );
  }
);

Input.displayName = "Input";
