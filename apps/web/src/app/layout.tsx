import type { Metadata } from "next";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n";
import { AuthProvider } from "@/lib/auth-context";
import { ToastProvider } from "@/components/ui/Toast";
import { inter, vazirmatn } from "@/lib/fonts";

export const metadata: Metadata = {
  title: "ClientPulse — Website Monitoring & Client Care Reports",
  description: "Production-oriented SaaS for web agencies to monitor websites and deliver branded care reports.",
};

const dirScript = `(function() {
  try {
    var saved = localStorage.getItem('clientpulse_locale');
    if (saved === 'fa') {
      document.documentElement.lang = 'fa';
      document.documentElement.dir = 'rtl';
    } else {
      document.documentElement.lang = 'en';
      document.documentElement.dir = 'ltr';
    }
  } catch(e) {}
})();`;

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      dir="ltr"
      suppressHydrationWarning
      className={`${inter.variable} ${vazirmatn.variable}`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: dirScript }} />
      </head>
      <body className="antialiased flex flex-col min-h-screen bg-slate-50 text-slate-900">
        <I18nProvider>
          <AuthProvider>
            <ToastProvider>
              {children}
            </ToastProvider>
          </AuthProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
