import type { Metadata } from "next";
import "./globals.css";

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
      <body className="antialiased flex flex-col min-h-screen">
        {children}
      </body>
    </html>
  );
}

