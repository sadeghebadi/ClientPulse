"use client";

import React, { createContext, useContext, useState, useEffect } from "react";

export type Locale = "en" | "fa";

export const translations = {
  en: {
    brandName: "ClientPulse",
    tagline: "Website Monitoring & Agency Care Reports",
    toggleLanguage: "فارسی (FA)",
    login: "Log In",
    signup: "Sign Up",
    logout: "Log Out",
    email: "Email address",
    password: "Password",
    currentPassword: "Current password",
    newPassword: "New password",
    confirmPassword: "Confirm password",
    displayName: "Full Name",
    agencyName: "Agency / Organization Name",
    createAccount: "Create Account",
    alreadyHaveAccount: "Already have an account?",
    dontHaveAccount: "Don't have an account?",
    forgotPassword: "Forgot password?",
    forgotPasswordTitle: "Reset Your Password",
    forgotPasswordDesc: "Enter your registered email address and we'll send you instructions to reset your password.",
    sendResetLink: "Send Reset Link",
    backToLogin: "Back to Log In",
    resetPasswordTitle: "Set New Password",
    resetPasswordDesc: "Enter your new password below (at least 12 characters).",
    resetPasswordBtn: "Update Password",
    verifyEmailTitle: "Email Verification",
    verifyEmailSuccess: "Your email has been successfully verified!",
    verifyEmailBtn: "Go to Dashboard",
    verifyEmailPending: "Verifying token, please wait...",
    resendVerification: "Resend Verification Email",
    onboardingTitle: "Setup Your Agency Workspace",
    onboardingDesc: "Configure your primary organization to start monitoring client websites.",
    orgNameLabel: "Organization Name",
    orgSlugLabel: "URL Slug",
    localeLabel: "Default Language",
    timezoneLabel: "Default Timezone",
    saveAndContinue: "Save & Continue",
    organizations: "Organizations",
    myOrganizations: "My Organizations",
    createOrganization: "Create New Organization",
    activeOrg: "Active Organization",
    switchOrg: "Switch",
    members: "Team Members",
    addMember: "Add Member",
    memberEmail: "Member's Registered Email",
    role: "Role",
    roleOwner: "Owner",
    roleAdmin: "Admin",
    roleMember: "Member",
    roleViewer: "Viewer",
    roleOwnerDesc: "Full administrative control, billing, and member roles.",
    roleAdminDesc: "Can manage monitors, clients, reports, and invite members.",
    roleMemberDesc: "Standard access to view metrics and log maintenance.",
    roleViewerDesc: "Read-only access to dashboards and reports.",
    accountSettings: "Account Settings",
    profile: "Profile",
    security: "Security",
    changePassword: "Change Password",
    changePasswordBtn: "Change Password",
    passwordsDoNotMatch: "Passwords do not match.",
    passwordTooShort: "Password must be at least 12 characters long.",
    loading: "Loading...",
    remove: "Remove",
    confirmRemoveMember: "Are you sure you want to remove this member?",
    noMembers: "No members found.",
    dashboard: "Dashboard",
    welcomeBack: "Welcome back",
    activeWorkspace: "Active Workspace",
    tenantIsolationVerified: "Multi-Tenant Isolation Active",
  },
  fa: {
    brandName: "کلاینت‌پالس",
    tagline: "سامانه پایش وب‌سایت و گزارش‌های دوره‌ای آژانس",
    toggleLanguage: "English (EN)",
    login: "ورود",
    signup: "ثبت‌نام",
    logout: "خروج",
    email: "آدرس ایمیل",
    password: "رمز عبور",
    currentPassword: "رمز عبور فعلی",
    newPassword: "رمز عبور جدید",
    confirmPassword: "تکرار رمز عبور",
    displayName: "نام و نام خانوادگی",
    agencyName: "نام آژانس / سازمان",
    createAccount: "ایجاد حساب کاربری",
    alreadyHaveAccount: "قبلاً ثبت‌نام کرده‌اید؟",
    dontHaveAccount: "حساب کاربری ندارید؟",
    forgotPassword: "فراموشی رمز عبور؟",
    forgotPasswordTitle: "بازیابی رمز عبور",
    forgotPasswordDesc: "ایمیل ثبت‌شده خود را وارد کنید تا لینک بازیابی رمز برای شما ارسال شود.",
    sendResetLink: "ارسال لینک بازیابی",
    backToLogin: "بازگشت به ورود",
    resetPasswordTitle: "تنظیم رمز عبور جدید",
    resetPasswordDesc: "رمز عبور جدید خود را وارد نمایید (حداقل ۱۲ کاراکتر).",
    resetPasswordBtn: "به‌روزرسانی رمز عبور",
    verifyEmailTitle: "تأیید آدرس ایمیل",
    verifyEmailSuccess: "ایمیل شما با موفقیت تأیید شد!",
    verifyEmailBtn: "ورود به پیشخوان",
    verifyEmailPending: "در حال بررسی توکن تأیید...",
    resendVerification: "ارسال مجدد ایمیل تأیید",
    onboardingTitle: "راه‌اندازی فضای کاری آژانس",
    onboardingDesc: "سازمان اصلی خود را برای پایش وب‌سایت‌های مشتریان تنظیم کنید.",
    orgNameLabel: "نام سازمان",
    orgSlugLabel: "شناسه آدرس (Slug)",
    localeLabel: "زبان پیش‌فرض",
    timezoneLabel: "منطقه زمانی پیش‌فرض",
    saveAndContinue: "ذخیره و ادامه",
    organizations: "سازمان‌ها",
    myOrganizations: "سازمان‌های من",
    createOrganization: "ایجاد سازمان جدید",
    activeOrg: "سازمان فعال",
    switchOrg: "انتخاب",
    members: "اعضای تیم",
    addMember: "افزودن عضو",
    memberEmail: "ایمیل عضو ثبت‌نام‌شده",
    role: "نقش",
    roleOwner: "مالک (Owner)",
    roleAdmin: "مدیر (Admin)",
    roleMember: "عضو (Member)",
    roleViewer: "مشاهده‌گر (Viewer)",
    roleOwnerDesc: "دسترسی کامل، تنظیمات سازمان، مالی و تعیین نقش‌ها.",
    roleAdminDesc: "مدیریت مانیتورها، کلاینت‌ها، گزارش‌ها و دعوت اعضا.",
    roleMemberDesc: "دسترسی عادی جهت مشاهده وضعیت و ثبت فعالیت نگهداری.",
    roleViewerDesc: "دسترسی فقط‌خواندنی به پیشخوان و گزارش‌ها.",
    accountSettings: "تنظیمات حساب کاربری",
    profile: "پروفایل",
    security: "امنیت",
    changePassword: "تغییر رمز عبور",
    changePasswordBtn: "تغییر رمز عبور",
    passwordsDoNotMatch: "رمزهای عبور یکسان نیستند.",
    passwordTooShort: "رمز عبور باید حداقل ۱۲ کاراکتر باشد.",
    loading: "در حال بارگذاری...",
    remove: "حذف",
    confirmRemoveMember: "آیا از حذف این عضو از سازمان اطمینان دارید؟",
    noMembers: "عضوی یافت نشد.",
    dashboard: "پیشخوان",
    welcomeBack: "خوش آمدید",
    activeWorkspace: "فضای کاری فعال",
    tenantIsolationVerified: "ایزولاسیون چندمستأجری فعال",
  },
};

interface I18nContextType {
  locale: Locale;
  setLocale: (l: Locale) => void;
  toggleLocale: () => void;
  t: typeof translations["en"];
  isRTL: boolean;
}

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocale] = useState<Locale>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("clientpulse_locale") as Locale;
      if (saved === "en" || saved === "fa") {
        return saved;
      }
    }
    return "en";
  });

  const changeLocale = (newLocale: Locale) => {
    setLocale(newLocale);
    localStorage.setItem("clientpulse_locale", newLocale);
    document.documentElement.lang = newLocale;
    document.documentElement.dir = newLocale === "fa" ? "rtl" : "ltr";
  };

  const toggleLocale = () => {
    changeLocale(locale === "en" ? "fa" : "en");
  };

  const isRTL = locale === "fa";

  return (
    <I18nContext.Provider
      value={{
        locale,
        setLocale: changeLocale,
        toggleLocale,
        t: translations[locale],
        isRTL,
      }}
    >
      <div dir={isRTL ? "rtl" : "ltr"} className={isRTL ? "font-sans" : ""}>
        {children}
      </div>
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return context;
}
