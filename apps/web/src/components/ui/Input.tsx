"use client";

import React from "react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: boolean;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className = "", error = false, ...props }, ref) => {
    return (
      <input
        ref={ref}
        className={`w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-900 border rounded-xl text-sm transition-all focus:outline-none focus:ring-2 disabled:opacity-50 disabled:cursor-not-allowed ${
          error
            ? "border-rose-400 focus:ring-rose-500 focus:bg-white dark:focus:bg-slate-900"
            : "border-slate-200 dark:border-slate-700 focus:ring-indigo-500 focus:bg-white dark:focus:bg-slate-900"
        } ${className}`}
        {...props}
      />
    );
  }
);

Input.displayName = "Input";

