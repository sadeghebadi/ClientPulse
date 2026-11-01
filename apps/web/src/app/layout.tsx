import type { Metadata } from "next";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n";
import { AuthProvider } from "@/lib/auth-context";

export const metadata: Metadata = {
  title: "ClientPulse — Website Monitoring & Client Care Reports",
  description: "Production-oriented SaaS for web agencies to monitor websites and deliver branded care reports.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" dir="ltr">
      <body className="antialiased flex flex-col min-h-screen bg-slate-50 text-slate-900">
        <I18nProvider>
          <AuthProvider>
            {children}
          </AuthProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
