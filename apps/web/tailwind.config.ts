import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        brand: {
          50: "var(--brand-50, #eff6ff)",
          100: "var(--brand-100, #dbeafe)",
          500: "var(--brand-500, #3b82f6)",
          600: "var(--brand-600, #2563eb)",
          700: "var(--brand-700, #1d4ed8)",
          800: "var(--brand-800, #1e40af)",
          900: "var(--brand-900, #1e3a8a)",
        },
        surface: {
          background: "var(--surface-background, #f8fafc)",
          card: "var(--surface-card, #ffffff)",
          sidebar: "var(--surface-sidebar, #0f172a)",
          border: "var(--surface-border, #e2e8f0)",
        },
        token: {
          text: {
            primary: "var(--text-primary, #0f172a)",
            secondary: "var(--text-secondary, #475569)",
            muted: "var(--text-muted, #64748b)",
            inverse: "var(--text-inverse, #ffffff)",
            disabled: "var(--text-disabled, #94a3b8)",
          },
          status: {
            operational: "var(--status-operational, #10b981)",
            degraded: "var(--status-degraded, #f59e0b)",
            down: "var(--status-down, #ef4444)",
            paused: "var(--status-paused, #64748b)",
            missing: "var(--status-missing, #8b5cf6)",
          },
        },
      },
      fontFamily: {
        sans: ["var(--font-inter)", "-apple-system", "BlinkMacSystemFont", "Segoe UI", "Roboto", "sans-serif"],
        persian: ["var(--font-vazirmatn)", "Tahoma", "Segoe UI", "Arial", "sans-serif"],
      },
    },
  },
  plugins: [],
};

export default config;
