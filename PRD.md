# سند نیازمندی‌های محصول (PRD) — ClientPulse

**نسخه:** 1.0  
**تاریخ:** 2026-10-31  
**وضعیت:** مشخصات مبنا برای توسعه نسخه اول  
**نوع محصول:** Hosted Multi-Tenant SaaS  
**مخاطب اصلی:** آژانس‌های کوچک طراحی، توسعه و نگهداری وب‌سایت  
**زبان سند:** فارسی؛ اصطلاحات فنی و نام موجودیت‌ها به انگلیسی حفظ شده‌اند.

---

## 0. دستور اصلی برای تیم پیاده‌سازی / Gemini

این سند مرجع اصلی محصول است. وظیفه تو صرفاً پیشنهاد معماری، تولید Mockup یا نوشتن چند فایل نمونه نیست؛ باید بر اساس این PRD یک محصول واقعی، یکپارچه، قابل اجرا، قابل تست و قابل استقرار بسازی.

### قواعد غیرقابل مذاکره

1. **ابتدا Repository را بررسی کن.** اگر کد موجود است، ساختار، تکنولوژی، تغییرات محلی و فایل‌های مهم را بررسی کن و بدون دلیل چیزی را حذف یا بازنویسی نکن. اگر Repository خالی است، ساختار پیشنهادی همین سند را ایجاد کن.
2. **قبل از تغییرات بزرگ، یک برنامه اجرایی کوتاه و قابل پیگیری بساز.** برنامه باید به فازها و کارهای قابل تست تقسیم شود. سپس اجرای واقعی را شروع کن؛ در مرحله برنامه‌ریزی متوقف نشو.
3. **کار را به Vertical Slice تقسیم کن.** هر Slice باید از UI تا API، منطق تجاری، دیتابیس و تست مربوطه کامل باشد. از تولید تعداد زیادی فایل غیرمتصل خودداری کن.
4. **قابلیت‌های P0 را کامل کن.** اگر محدودیت زمان یا محیط مانع تکمیل چیزی شد، آن را شفاف گزارش کن؛ قابلیت ناقص را به‌عنوان تکمیل‌شده معرفی نکن.
5. **Mock فقط برای توسعه و تست مجاز است.** برای قابلیت‌های اصلی محصول مانند Monitor، Incident، Report و Tenant Isolation منطق واقعی پیاده‌سازی کن. برای سرویس خارجی فاقد Credential، Adapter واقعی و قابل تنظیم بساز و وضعیت غیرفعال بودن آن را صادقانه در UI نشان بده.
6. **هیچ Secret واقعی اختراع نکن.** تمام رمزها، API Tokenها، کلیدهای امضا و اطلاعات اتصال فقط از Environment Variables یا Secret Manager دریافت شوند. فایل `.env.example` باید مقادیر نمونه غیرواقعی داشته باشد.
7. **دامنه را Hard-code نکن.** فعلاً دامنه نهایی انتخاب نشده است. URL اصلی باید از `APP_BASE_URL` خوانده شود. نام محصول در UI برابر `ClientPulse` است. از دامنه‌های آزمایشی یا دامنه‌های شخصی به‌عنوان دامنه قطعی محصول استفاده نکن.
8. **پرداخت واقعی را جعل نکن.** Billing، Plan و Subscription را کامل طراحی و پیاده‌سازی کن؛ اما Gateway واقعی را فقط وقتی فعال کن که Provider قانونی و Credential واقعی در اختیار پروژه باشد. تا آن زمان Manual Payment/Admin Activation یا Development Adapter مجاز است و باید آشکارا با برچسب Test/Manual نمایش داده شود.
9. **امنیت Multi-Tenant حیاتی است.** هیچ درخواست API، Query، Job، Report، File یا عملیات پس‌زمینه نباید امکان دسترسی به Tenant دیگر را داشته باشد. تمام مسیرها باید در Backend کنترل شوند؛ مخفی کردن داده در Frontend کافی نیست.
10. **تست‌ها بخشی از قابلیت هستند.** برای هر قابلیت مهم Unit، Integration و در مسیرهای کلیدی E2E Test بنویس. مخصوصاً Tenant Isolation، SSRF، Heartbeat Token، Report Sharing و Permissionها.
11. **مهاجرت دیتابیس باید Versioned باشد.** از تغییر دستی Schema در محیط اجرا خودداری کن. Migration و Seed قابل تکرار بساز.
12. **هیچ داده نمایشی را به‌عنوان داده واقعی نشان نده.** Demo Data باید صریحاً Demo باشد. Uptime، تعداد تعمیرات، زمان آخرین Backup یا هر شاخص دیگری نباید جعل شود.
13. **قابلیت‌های خارج از MVP را بدون تأیید اضافه نکن.** به‌ویژه نصب Plugin، تغییر خودکار سایت مشتری، اجرای خودکار تعمیرات، اسکن آسیب‌پذیری و دسترسی SSH در نسخه اول ممنوع است.
14. **هر فاز را اجرا و تأیید کن.** پس از هر فاز، Build، Lint، Test و در صورت امکان اجرای واقعی را انجام بده و نتیجه را در `task-report/` ثبت کن.
15. **در پایان، دستورالعمل Run محلی و Deploy را واقعاً آزمایش کن.** README صرفاً یک متن تشریفاتی نباشد.

### خروجی‌های اجباری توسعه

- کد کامل Frontend و Backend
- Worker واقعی برای Monitoring و Report Scheduling
- PostgreSQL Schema و Migrationها
- Docker Compose برای اجرای محلی و استقرار اولیه
- `.env.example` کامل و بدون Secret واقعی
- OpenAPI Specification برای API
- تست‌های خودکار و دستور اجرای آن‌ها
- Seed توسعه و حساب Demo امن
- README فارسی یا انگلیسی با راه‌اندازی، معماری، Backup و Deploy
- فایل‌های اجرایی/گزارشی در `task/` و `task-report/`؛ در هر گزارش، کار انجام‌شده، تست اجراشده، نتیجه، محدودیت و کار باقی‌مانده ذکر شود.
- فهرست صریح قابلیت‌های تکمیل‌شده، ناقص و خارج از محدوده

---

# 1. خلاصه محصول

## 1.1 نام و جایگاه

نام محصول: **ClientPulse**

دسته محصول: ابزار SaaS برای آژانس‌هایی که وب‌سایت مشتریان را طراحی، میزبانی، پشتیبانی یا نگهداری می‌کنند.

ClientPulse به آژانس کمک می‌کند سلامت وب‌سایت‌ها و سرویس‌های مشتریان را پایش کند، قطعی‌ها و رویدادهای مهم را ثبت کند، فعالیت‌های نگهداری را مستند کند و گزارش ماهانه‌ای با برند خود آژانس برای مشتری بسازد.

**وعده اصلی محصول:**

> ClientPulse helps web agencies monitor client websites, document maintenance work, and deliver clear, branded care reports with less manual effort.

## 1.2 مسئله‌ای که حل می‌کنیم

آژانس‌های کوچک ممکن است 5 تا 100 وب‌سایت مشتری را نگهداری کنند. اطلاعات مربوط به وضعیت سایت‌ها، خطاها، SSL، Backup، وظایف زمان‌بندی‌شده و کارهای انجام‌شده معمولاً بین ابزارهای مختلف، پیام‌رسان‌ها، فایل‌ها و حافظه اعضای تیم پراکنده است.

این پراکندگی باعث می‌شود:

- آژانس برای فهمیدن وضعیت همه سایت‌ها زمان صرف کند.
- قطعی یا منقضی شدن SSL دیر تشخیص داده شود.
- مشخص نباشد Backup یا Cron Job آخرین بار چه زمانی موفق بوده است.
- کارهایی که تیم برای مشتری انجام داده، ثبت و قابل اثبات نباشند.
- تهیه گزارش ماهانه دستی و تکراری شود.
- مشتری ارزش خدمات نگهداری را به‌خوبی نبیند.

ClientPulse این اطلاعات را در یک محل جمع می‌کند و گزارش قابل فهم و برندشده می‌سازد.

## 1.3 تمایز محصول

ClientPulse قرار نیست صرفاً یک Uptime Monitor عمومی باشد. ارزش اصلی آن ترکیب این چهار بخش است:

1. **Health Monitoring:** مشاهده وضعیت واقعی سایت و API.
2. **Operational Signals:** پایش Heartbeat برای Cron و Backup.
3. **Maintenance Evidence:** ثبت کارهای واقعی انجام‌شده توسط تیم.
4. **Client Reporting:** تبدیل شواهد بالا به گزارش ماهانه برندشده.

گزارش باید روشن کند کدام داده به‌صورت خودکار اندازه‌گیری شده و کدام فعالیت را یک عضو تیم ثبت کرده است.

## 1.4 مدل ارائه

- Hosted SaaS؛ مشتری روی سایت ClientPulse ثبت‌نام می‌کند.
- هر آژانس یک Tenant مستقل است.
- هر Tenant می‌تواند چند User، Client و Website/Asset داشته باشد.
- آژانس می‌تواند گزارش را PDF کند یا با لینک امن برای مشتری بفرستد.
- در نسخه اول نیازی نیست مشتری نهایی حساب کاربری بسازد.

## 1.5 بازار و زبان

محصول برای بازار جهانی و فارسی‌زبان طراحی می‌شود.

الزامات:

- زبان انگلیسی و فارسی از نسخه اول.
- پشتیبانی واقعی RTL برای فارسی، نه صرفاً ترجمه متن.
- امکان تعیین زبان جداگانه برای هر Organization.
- امکان تعیین Timezone برای هر Organization.
- نمایش تاریخ، عدد و واحد مطابق Locale.
- زبان گزارش می‌تواند بر اساس تنظیمات Organization انتخاب شود.
- زبان پیش‌فرض سیستم انگلیسی باشد؛ کاربر در Onboarding بتواند فارسی را انتخاب کند.

## 1.6 دامنه و برند

- نام محصول در رابط کاربری: ClientPulse.
- دامنه نهایی هنوز انتخاب نشده است.
- تمام URLهای مطلق باید از `APP_BASE_URL` ساخته شوند.
- نام دامنه در Email، لینک Report و Telegram Deep Link نباید Hard-code شود.
- انتخاب نهایی دامنه و بررسی Trademark/Availability خارج از این PRD است.

---

# 2. اهداف، معیار موفقیت و Non-Goals

## 2.1 اهداف تجاری

هدف اولیه رسیدن به حدود 200 دلار درآمد ناخالص تکرارشونده ماهانه (MRR) است؛ این عدد هدف برنامه‌ریزی است، نه تضمین.

سناریوی هدف اولیه:

- 9 مشتری روی پلن Studio با قیمت آزمایشی 24 دلار در ماه = 216 دلار MRR.
- قیمت‌ها باید از Admin قابل تغییر باشند.
- محصول نباید به یک Provider پرداخت یا کشور خاص وابسته شود.

## 2.2 اهداف محصولی

- کاربر بتواند در کمتر از 10 دقیقه اولین Organization و Website Monitor خود را ایجاد کند.
- کاربر بتواند Client و Websiteها را گروه‌بندی کند.
- کاربر از Dashboard وضعیت سلامت دارایی‌های خود را بفهمد.
- کاربر بتواند قطعی و بازیابی را دریافت کند.
- کاربر بتواند Cron/Backup Heartbeat تعریف کند.
- کاربر بتواند Maintenance Activity ثبت کند.
- کاربر بتواند برای یک Client گزارش دقیق تولید و PDF دریافت کند.
- گزارش قابل استفاده بدون ایجاد حساب برای مشتری نهایی باشد.
- سیستم با حداقل دخالت دستی قابل نگهداری باشد.

## 2.3 معیارهای موفقیت MVP

معیارهای قابل اندازه‌گیری:

- حداقل 95% تست‌های خودکار CI موفق باشند؛ برای Merge/Release هدف 100% تست‌های Required است.
- هیچ Tenant Isolation Failure شناخته‌شده‌ای وجود نداشته باشد.
- حداقل 99% Jobهای مانیتورینگ در حالت بار هدف، در بازه مجاز اجرا شوند؛ این SLO داخلی است و باید با Load Test بررسی شود.
- در تشخیص قطعی با آستانه دو Failure متوالی، Incident به‌درستی ساخته شود.
- Recovery Incident با موفقیت متوالی ثبت شود.
- PDF برای Client دارای داده و Client بدون داده به‌درستی تولید شود.
- لینک عمومی Report قابل لغو و دارای Token غیرقابل حدس باشد.
- کاربر بتواند بدون تنظیم سرویس ایمیل یا تلگرام، Core Monitoring و PDF را آزمایش کند.

## 2.4 Non-Goals نسخه اول

موارد زیر در MVP ساخته نمی‌شوند:

- ورود SSH به سرور مشتری.
- نصب Agent روی سرور مشتری.
- نصب یا مدیریت WordPress Plugin.
- آپدیت خودکار WordPress، Theme یا Plugin.
- تعمیر خودکار سایت یا اجرای Command دلخواه روی مشتری.
- Vulnerability Scanner یا Penetration Testing.
- Malware Removal.
- SEO Rank Tracking یا Analytics کامل.
- مانیتورینگ چند منطقه جغرافیایی.
- اپلیکیشن Native موبایل.
- Customer Portal با حساب جداگانه.
- AI Chatbot یا تولید آزادانه گزارش به کمک AI.
- پردازش یا نگهداری محتوای حساس سایت مشتری.
- تضمین صحت فایل Backup؛ Heartbeat صرفاً اجرای موفق اسکریپت را گزارش می‌کند.
- پایش انقضای ثبت دامنه از طریق WHOIS در MVP.

---

# 3. کاربران و نقش‌ها

## 3.1 Persona A — Agency Owner

- مالک یا مدیر آژانس کوچک.
- 5 تا 50 مشتری/سایت دارد.
- می‌خواهد وضعیت کل سرویس‌ها و گزارش ماهانه را ببیند.
- به Branding، Billing و مدیریت اعضا نیاز دارد.

نیاز اصلی: صرفه‌جویی در زمان و ارائه گزارش حرفه‌ای به مشتری.

## 3.2 Persona B — Technical Manager

- مسئول نگهداری و عملیات فنی.
- نیازمند مشاهده Monitor، Incident، Heartbeat و کارهای عقب‌افتاده است.
- می‌خواهد خطاها سریع و بدون Notification تکراری اعلام شوند.

نیاز اصلی: تشخیص و پیگیری مشکل.

## 3.3 Persona C — Developer / Maintenance Staff

- وظایف نگهداری را انجام می‌دهد.
- باید فعالیت واقعی را با زمان و توضیح ثبت کند.
- ممکن است اجازه مدیریت Billing یا حذف Organization نداشته باشد.

نیاز اصلی: ثبت ساده و سریع کارهای انجام‌شده.

## 3.4 Persona D — Agency Client (گیرنده گزارش)

- صاحب کسب‌وکار یا مدیر سایت.
- لزوماً دانش فنی ندارد.
- فقط گزارش قابل فهم و وضعیت سایت را دریافت می‌کند.
- در MVP حساب کاربری مستقل ندارد؛ Report از طریق لینک امن یا PDF ارائه می‌شود.

## 3.5 نقش‌های داخلی Tenant

### Owner

- مالک Organization.
- دسترسی کامل به همه داده‌ها و تنظیمات.
- مدیریت کاربران، پلن و Subscription.
- تغییر یا حذف Organization.
- تنها نقش مجاز برای انتقال مالکیت.

### Admin

- مدیریت Client، Asset، Monitor، Report، Branding و Notification.
- دعوت اعضا.
- مشاهده Billing؛ تغییر مالکیت یا حذف Organization مجاز نیست.

### Member

- مشاهده Dashboard و داده‌های عملیاتی.
- ایجاد و ویرایش Maintenance Log.
- ایجاد و مدیریت Monitor طبق Permission تعریف‌شده.
- دسترسی به Billing و حذف Organization ندارد.

### Viewer (در صورت فعال بودن در پلن)

- فقط مشاهده Dashboard، Client، Monitor، Incident و Report.
- هیچ عملیات تغییر داده انجام نمی‌دهد.

**قاعده:** Permission همیشه در Backend بررسی شود. Frontend فقط کنترل نمایشی مکمل است.

## 3.6 نقش Platform Admin

کاربر داخلی پلتفرم که Tenantها، Planها و وضعیت سرویس را مدیریت می‌کند.

- از User عادی Tenant کاملاً جدا است.
- مسیرهای Admin با Authorization مستقل محافظت شوند.
- دسترسی Platform Admin به داده محتوایی Tenant فقط برای پشتیبانی ضروری و با ثبت Audit Log مجاز است.
- ایجاد اولین Platform Admin فقط از طریق Seed/Environment امن؛ هرگز از Signup عمومی.

---

# 4. محدوده نسخه‌ها و اولویت‌بندی

## 4.1 اولویت‌ها

- **P0 — Required for MVP:** بدون آن محصول اولیه کامل نیست.
- **P1 — Important:** بعد از تکمیل End-to-End قابلیت‌های P0.
- **P2 — Later:** خارج از Release اول.

## 4.2 P0 — MVP

1. Authentication و Organization Onboarding.
2. Multi-Tenant Data Isolation.
3. Client Management.
4. Website/HTTP(S) Monitor.
5. SSL Certificate Expiry Check.
6. DNS Resolution Check پایه.
7. Incident Detection و Recovery.
8. Cron/Backup Heartbeat.
9. Telegram Notification Adapter (در صورت وجود Bot Token) و Email Adapter قابل تنظیم.
10. Maintenance Activity Log.
11. Dashboard و Client Detail.
12. Report Generation و PDF.
13. White Label پایه.
14. Secure Public Report Link.
15. Subscription/Plan/Usage Limit و Trial.
16. Platform Admin پایه برای مدیریت Tenant و Plan.
17. فارسی/انگلیسی و RTL.
18. Docker Compose، Migrations، Tests و Documentation.

## 4.3 P1

- JSON Response Assertions کامل‌تر.
- Scheduled Monthly Reports با Email Delivery.
- Report Preview قبل از نهایی‌سازی.
- Invitation Flow کامل‌تر و Resend Invitation.
- Team Activity/Audit Log قابل مشاهده برای Owner.
- Domain Expiration Check در صورت وجود منبع قابل اتکا.
- Report Templates قابل سفارشی‌سازی.
- Maintenance Categories و فیلترهای پیشرفته.
- Public Status Page محدود و اختیاری.

## 4.4 P2

- Customer Portal و حساب مستقل مشتری نهایی.
- WordPress Plugin.
- Server Agent و CPU/RAM/Disk Monitoring.
- Multi-Region Monitoring.
- AI-Assisted Report Summary با تأیید کاربر.
- Jira/Trello/Slack Integrations.
- Automatic Maintenance Actions.
- White-label Custom Domain.
- Mobile App.

Gemini می‌تواند P1 را پس از تکمیل و تأیید P0 شروع کند؛ P2 بدون درخواست صریح اجرا نشود.

---

# 5. جریان‌های اصلی کاربر

## 5.1 Signup تا اولین Monitor

1. کاربر صفحه Landing را باز می‌کند.
2. Sign Up با Email و Password.
3. Email Verification (در حالت Dev امکان مشاهده لینک Verification در Mailpit یا لاگ توسعه، نه خاموش کردن منطق).
4. ساخت Organization شامل نام آژانس، Locale و Timezone.
5. ایجاد Trial پیش‌فرض.
6. Onboarding Wizard از کاربر می‌خواهد اولین Client را بسازد.
7. کاربر Client را ایجاد می‌کند.
8. اولین Website URL را ثبت می‌کند.
9. سیستم URL را اعتبارسنجی و SSRF Guard را اجرا می‌کند.
10. اولین Check فوری زمان‌بندی می‌شود.
11. پس از دریافت نتیجه واقعی، Dashboard وضعیت سایت را نشان می‌دهد.
12. UI باید تفاوت `Pending`, `Up`, `Down` و `Unknown` را واضح نشان دهد.

## 5.2 تعریف Client و Asset

1. کاربر از Clients گزینه Add Client را انتخاب می‌کند.
2. نام Client، ایمیل اختیاری و Notes اختیاری را وارد می‌کند.
3. Asset از نوع Website یا API به Client متصل می‌کند.
4. Monitor Type و Interval و Timeout را انتخاب می‌کند.
5. سیستم اولین Check را اجرا می‌کند.
6. Asset در صفحه Client نمایش داده می‌شود.

## 5.3 Incident

1. Worker در Check به خطای قطعی می‌رسد.
2. سیستم Failure Count را افزایش می‌دهد.
3. تا قبل از رسیدن به Failure Threshold، وضعیت `Suspected` یا `Degraded` است؛ Incident رسمی ساخته یا Notification ارسال نشود.
4. پس از دو Failure متوالی، Incident باز می‌شود.
5. Notification بر اساس تنظیمات ارسال می‌شود.
6. هر Check جدید به Incident متصل می‌شود.
7. پس از یک Check موفق (Recovery Threshold پیش‌فرض 1)، Incident بسته می‌شود.
8. Recovery Notification ارسال می‌شود.
9. مدت قطعی بر اساس زمان تشخیص و زمان Recovery محاسبه و در گزارش توضیح داده می‌شود.

## 5.4 Cron/Backup Heartbeat

1. کاربر از Client یا Asset گزینه Add Heartbeat Monitor را انتخاب می‌کند.
2. نام Job، Expected Interval و Grace Period را مشخص می‌کند.
3. سیستم Token تصادفی قوی تولید می‌کند.
4. Secret فقط یک بار به کاربر نشان داده می‌شود.
5. کاربر دستور نمونه `curl` یا Script را در Cron خود قرار می‌دهد.
6. Endpoint عمومی فقط Token معتبر را دریافت می‌کند و زمان Ping را ثبت می‌کند.
7. اگر Heartbeat تا پایان Grace Period نرسد، وضعیت Stale/Down می‌شود.
8. Token در UI دوباره به شکل Plaintext نمایش داده نمی‌شود؛ فقط امکان Rotate/Revoke وجود دارد.

## 5.5 ثبت فعالیت نگهداری

1. کاربر Client یا Asset را انتخاب می‌کند.
2. Add Maintenance Activity را می‌زند.
3. Category، Title، Description، زمان انجام و مدت صرف‌شده را وارد می‌کند.
4. Actor به‌صورت خودکار از Session مشخص می‌شود.
5. فعالیت در Timeline ثبت می‌شود.
6. Report Generator فقط فعالیت‌های ثبت‌شده و معتبر را وارد گزارش می‌کند.

## 5.6 تولید گزارش

1. کاربر Client و بازه زمانی را انتخاب می‌کند؛ پیش‌فرض ماه تقویمی قبلی در Timezone سازمان است.
2. سیستم Preview/summary داده‌های موجود را می‌سازد.
3. Uptime، Incidentها، Heartbeatها، SSL و Maintenance Activities جمع‌بندی می‌شوند.
4. Coverage و Missing Data به‌وضوح نشان داده می‌شوند.
5. کاربر Branding و زبان گزارش را می‌بیند.
6. Generate PDF انجام می‌شود.
7. کاربر PDF را دانلود یا Share Link امن ایجاد می‌کند.
8. Share Link قابل Revoke و دارای Expiry است.

## 5.7 دعوت عضو تیم

1. Owner/Admin ایمیل عضو و Role را وارد می‌کند.
2. دعوت با Token امن و زمان انقضا ایجاد می‌شود.
3. دعوت‌شونده با Signup/Login دعوت را می‌پذیرد.
4. Membership به Organization مربوط متصل می‌شود.
5. Token یک‌بارمصرف و باطل می‌شود.

---

# 6. الزامات عملکردی (Functional Requirements)

## 6.1 Authentication & Account

### FR-AUTH-01 — Signup

- Signup با Email و Password.
- Email باید Normalize و Unique باشد.
- Password حداقل 12 کاراکتر یا سیاست امنیتی معادل مبتنی بر طول و بررسی Passwordهای رایج داشته باشد.
- Password با Argon2id ترجیحاً؛ یا الگوریتم استاندارد قوی با پارامتر مناسب Hash شود.
- Password هرگز Log یا Plaintext ذخیره نشود.
- تأیید Email قبل از قابلیت‌های حساس الزامی باشد.

### FR-AUTH-02 — Login/Logout

- Session امن با Cookieهای `HttpOnly`, `Secure` در Production و `SameSite=Lax` یا سیاست مناسب.
- از نگهداری Access Token بلندمدت در LocalStorage خودداری شود.
- Logout باید Session را سمت سرور باطل کند.
- Login Rate Limit و Brute Force Protection داشته باشد.

### FR-AUTH-03 — Password Reset

- Token تصادفی، Hash‌شده در DB و یک‌بارمصرف.
- Expiry حداکثر 30 دقیقه.
- پس از Reset، Sessionهای قبلی باطل شوند.
- وجود یا عدم وجود Email در پاسخ عمومی افشا نشود.

### FR-AUTH-04 — Organization Onboarding

- Signup عادی یک Organization اولیه ایجاد کند.
- سازنده به‌صورت Owner ثبت شود.
- Locale و Timezone انتخابی ذخیره شوند.
- Slug یکتا و قابل تغییر با کنترل Collision ایجاد شود.

### FR-AUTH-05 — Session Security

- Session Expiry قابل تنظیم.
- تغییر Password و Revoke All Sessions.
- CSRF Protection برای عملیات Cookie-authenticated.
- امنیت Headerهای پایه و CORS محدود به Origin مجاز.

## 6.2 Organization / Tenant

### FR-TEN-01

تمام داده‌های Tenant-scoped باید `organization_id` داشته باشند یا از طریق Relationship غیرقابل دور زدن به Organization برسند.

### FR-TEN-02

تمام Queryهای Tenant-scoped باید Tenant Context را از Session/Token معتبر Backend بگیرند؛ هرگز `organization_id` ارسالی از Client را به‌عنوان مرجع اعتماد نپذیرند.

### FR-TEN-03

کاربر می‌تواند بین Organizationهایی که عضو آن‌هاست جابه‌جا شود. Context هر Request باید از Membership معتبر تعیین شود.

### FR-TEN-04

حذف Organization باید Soft Delete/Grace Period داشته باشد و قبل از حذف دائمی، Confirmation و Export/Backup مناسب ارائه شود.

## 6.3 Client Management

### FR-CLIENT-01

CRUD برای Client شامل:

- `name` (Required)
- `contact_name` (Optional)
- `contact_email` (Optional)
- `website` (Optional)
- `notes` (Optional)
- `status` (`active`, `archived`)
- `created_at`, `updated_at`

### FR-CLIENT-02

- Clientها در Organization خود فهرست و Search/Filter شوند.
- Client دارای Assetهای مربوط، Incidentهای اخیر و گزارش‌های قبلی باشد.
- حذف Client دارای تأیید و رفتار مشخص برای Assetها و Reportها باشد؛ پیشنهاد: ابتدا Archive و حذف دائمی فقط توسط Owner/Admin با هشدار.

### FR-CLIENT-03

- Client Email فقط برای دریافت گزارش/اطلاع‌رسانی استفاده می‌شود.
- ارسال Email باید بر اساس Consent/تنظیمات آژانس باشد.
- هیچ Email خودکاری به Client نهایی بدون اقدام یا تنظیم صریح آژانس ارسال نشود.

## 6.4 Asset Management

Asset موجودیتی است که به Client تعلق دارد.

### انواع MVP

- `website`: سایت عمومی HTTP/HTTPS.
- `api`: Endpoint عمومی HTTP/HTTPS.

Heartbeat Monitor می‌تواند به Client یا Asset متصل شود؛ برای مدل‌سازی ساده، در ابتدا Heartbeat Monitor نوع مستقلی از Monitor است ولی Client اجباری و Asset اختیاری دارد.

### FR-ASSET-01

فیلدهای Website/API:

- `name`
- `client_id`
- `url`
- `type`
- `environment` (`production`, `staging`, `development`؛ در نسخه Hosted مانیتور کردن Development خصوصی ممنوع)
- `description`
- `is_active`
- `created_at`, `updated_at`

### FR-ASSET-02

- تغییر URL باید Revalidation و SSRF Guard مجدد داشته باشد.
- URLهای تکراری در یک Tenant می‌توانند مجاز باشند، ولی UI باید هشدار Duplicate بدهد.
- Archive کردن Asset باید Jobهای آن را متوقف کند.

### FR-ASSET-03

- Asset Detail باید وضعیت فعلی، Uptime، Latency، Incident History و SSL Summary را نشان دهد.
- وضعیت `Unknown` با `Down` اشتباه نشود.

## 6.5 Website / HTTP Monitor

### FR-MON-01 — Monitor Creation

تنظیمات اولیه:

- Asset
- HTTP Method: در MVP فقط `GET` و `HEAD`؛ پیش‌فرض GET.
- URL از Asset یا URL Override.
- Check Interval: پیش‌فرض 5 دقیقه.
- Timeout: پیش‌فرض 10 ثانیه، بازه مجاز 3 تا 30 ثانیه.
- Expected Status: پیش‌فرض 200–399.
- Follow Redirects: پیش‌فرض فعال؛ حداکثر 5 Redirect.
- Expected Text یا JSON Assertion اختیاری.
- Active/Paused.

### FR-MON-02 — Check Execution

هر Check باید حداقل این اطلاعات را ثبت کند:

- Scheduled Time
- Started Time
- Completed Time
- Duration/Latency ms
- Result (`success`, `failure`, `timeout`, `blocked`, `error`)
- HTTP Status Code در صورت وجود
- Error Category/Code امن و قابل فهم
- TLS Metadata در صورت HTTPS
- Resolved Public IP (فقط در صورت نیاز عملیاتی و با سیاست Retention؛ از نمایش عمومی IPهای حساس خودداری شود)
- Check Region/Worker Identifier

بدنه کامل Response به‌صورت پیش‌فرض ذخیره نشود.

### FR-MON-03 — Success Criteria

پیش‌فرض موفقیت:

- Request با Timeout تعیین‌شده پاسخ دهد.
- TLS Verification معتبر باشد.
- Status نهایی بین 200 و 399 باشد.
- اگر Expected Text تعریف شده، متن مورد انتظار در محدوده مجاز Body پیدا شود.
- اگر JSON Assertion تعریف شده، مقدار JSON Pointer با مقدار مورد انتظار مطابق باشد.

4xx و 5xx در حالت پیش‌فرض Failure هستند، مگر اینکه کاربر Expected Status را تغییر دهد.

### FR-MON-04 — Body Safety

- حداکثر Body بررسی‌شده 256 KB.
- Response Body فقط در حافظه پردازش شود و به‌صورت خام Log نشود.
- Content-Type و Encoding بررسی شوند.
- پاسخ‌های غیرمتنی برای Expected Text پذیرفته نشوند.

### FR-MON-05 — SSL Check

برای HTTPS:

- تاریخ Not Before و Not After گواهی خوانده شود.
- TLS Verification فعال باشد.
- Expiry Warning آستانه‌های 30، 14 و 7 روز داشته باشد.
- گواهی Expired یا Invalid در وضعیت Monitor اثر بگذارد.
- اطلاع‌رسانی SSL Warning از Incident قطعی سایت تفکیک شود.

### FR-MON-06 — DNS Check

- DNS Resolve برای A و AAAA در صورت وجود انجام شود.
- DNS Failure در Check Result ثبت شود.
- در صورت DNS Failure، Request HTTP انجام نشود.
- اطلاعات لازم برای Debug ذخیره شود؛ نه اینکه DNS record عمومی غیرضروری به Report نشت کند.

### FR-MON-07 — Interval و Scheduling

- پیش‌فرض 5 دقیقه.
- Interval باید در سطح Plan محدود شود.
- زمان‌بندی باید از Drift جلوگیری کند؛ مبنا زمان اجرای قبلی نباشد، بلکه `next_check_at` مدیریت‌شده و قابل بازیابی باشد.
- اجرای هم‌زمان تکراری برای یک Monitor ممنوع است.
- Jobهای عقب‌افتاده نباید پس از رفع قطعی ناگهان به‌صورت Burst اجرا شوند؛ سیاست Skip/Misfire صریح داشته باشند.

### FR-MON-08 — Pause/Resume

- Pause مانیتور، Check جدید را متوقف کند.
- نتایج قبلی حفظ شوند.
- Resume یک Check فوری یا زمان‌بندی‌شده نزدیک ایجاد کند.

### FR-MON-09 — Manual Test

کاربر بتواند `Test Now` را اجرا کند. این Test باید Rate Limit داشته باشد و در History با `manual` مشخص شود؛ در محاسبه Uptime عادی وارد نشود مگر سیاست صریح تعریف شده باشد.

## 6.6 SSRF و ایمنی Network Requests — اجباری

ClientPulse به URL دلخواه Request می‌زند؛ بنابراین SSRF یک ریسک اصلی است.

### FR-SSRF-01

فقط `http` و `https` مجاز است. Schemeهای دیگر رد شوند.

### FR-SSRF-02

موارد زیر مسدود شوند:

- `localhost` و هر Subdomain آن.
- IPv4 Private، Loopback، Link-local، Multicast، Reserved و Unspecified.
- IPv6 Loopback، Link-local، Unique Local، Multicast و Reserved.
- Cloud Metadata IP/Hostnameها از جمله آدرس‌های رایج Metadata.
- Hostnameهای داخلی مانند `.local`, `.internal` و نام‌های بدون دامنه عمومی.
- Userinfo داخل URL.
- URLهای دارای Port غیرمجاز؛ در MVP فقط 80 و 443 مجاز باشند، مگر با تنظیم Platform Admin.

### FR-SSRF-03 — DNS Rebinding

- Hostname در لحظه Check Resolve شود.
- تمام IPهای A/AAAA بررسی شوند؛ اگر هر IP ممنوع است، Request رد شود.
- اتصال باید به IP اعتبارسنجی‌شده Pin شود یا معادل امنی داشته باشد تا DNS Rebinding بین Resolve و Connect ممکن نشود.
- Redirectها باید مجدداً از ابتدا Validate شوند.
- Redirect به IP/Hostname ممنوع رد شود.
- TLS SNI و Host Header باید با Hostname معتبر باقی بماند.

### FR-SSRF-04

- از Proxy Environment ناخواسته برای دور زدن Guard استفاده نشود.
- Redirect Count محدود باشد.
- Response Size و Duration محدود باشند.
- Outbound Concurrency و Per-Host Rate Limit اعمال شود.
- Check URLها نباید امکان جعل Request Headerهای حساس مانند `Authorization` را بدون نیاز واقعی فراهم کنند.

### FR-SSRF-05

تست‌های امنیتی با URLهای زیر و معادل IPv6/DNS Rebinding نوشته شوند: localhost، 127.0.0.1، 10.0.0.0/8، 172.16.0.0/12، 192.168.0.0/16، 169.254.169.254، ::1، fc00::/7، fe80::/10، Redirect به IP داخلی، DNS که ابتدا Public و سپس Private برمی‌گرداند.

## 6.7 Incident Management

### FR-INC-01 — State Machine

وضعیت Monitor:

- `pending`: هنوز Check معتبر انجام نشده.
- `up`: آخرین وضعیت تأییدشده سالم.
- `suspected_down`: یک Failure ثبت شده، Threshold هنوز کامل نیست.
- `down`: Incident باز است.
- `paused`: مانیتور عمداً غیرفعال است.
- `unknown`: داده کافی در بازه مورد نظر وجود ندارد.

Incident:

- `open`
- `resolved`
- `acknowledged` به‌عنوان Flag یا State کمکی؛ Acknowledge نباید Incident را Resolve کند.

### FR-INC-02 — Threshold

- Failure Threshold پیش‌فرض: 2 Failure متوالی.
- Recovery Threshold پیش‌فرض: 1 Success متوالی.
- مقادیر باید در طراحی قابل تغییر باشند؛ UI پیشرفته تنظیم آن می‌تواند P1 باشد.
- Failureهای پشت سر هم باید به یک Incident متصل شوند، نه اینکه Incidentهای تکراری تولید شود.

### FR-INC-03 — Incident Fields

- Monitor ID
- Opened At (زمان اولین Failure تشخیص‌داده‌شده و زمان Confirm جداگانه قابل نگهداری باشد)
- Confirmed At
- Resolved At
- Duration
- Failure Category
- First Failure Result
- Recovery Result
- Notification State
- Acknowledged By/At

### FR-INC-04 — Flapping

- اگر Monitor مکرراً بین Up/Down جابه‌جا شود، سیستم Notification را Throttle کند.
- Incidentهای متوالی نزدیک به هم نباید Spam ایجاد کنند.
- سیاست Dedup و Cooldown مستند باشد.

### FR-INC-05 — Uptime Calculation

- Uptime بر اساس بازه‌های Check مورد انتظار و Incidentهای تشخیص‌داده‌شده محاسبه شود.
- زمان آغاز قطعی در Report به‌عنوان Detection Time/Estimated Outage Start توصیف شود؛ ادعا نشود لحظه دقیق شروع قطعی معلوم است.
- Monitorهایی که در میانه Period ساخته شده‌اند، Coverage جداگانه داشته باشند.
- بازه Pause شده از Denominator محاسبه حذف شود.
- فقدان داده نباید به‌عنوان Up تفسیر شود.
- روش محاسبه و محدودیت آن در Documentation و Tooltip شرح داده شود.

## 6.8 Heartbeat Monitoring

### FR-HB-01 — Creation

فیلدها:

- Client (Required)
- Asset (Optional)
- Name (Required)
- Description (Optional)
- Expected Interval (مثلاً 1h، 24h؛ مقدار ثانیه ذخیره شود)
- Grace Period (پیش‌فرض 20% Interval یا مقدار حداقل معقول؛ منطق دقیق باید یکسان و مستند باشد)
- Active/Paused
- Token Hash
- Last Received At
- Status

### FR-HB-02 — Endpoint

Endpoint پیشنهادی:

`POST /api/v1/heartbeat-pings/{token}`

- فقط HTTPS در Production.
- Token با حداقل 32 Bytes Random تولید شود.
- فقط Hash Token ذخیره شود.
- مقایسه Token در زمان اجرا امن و Constant-Time باشد.
- Token در Access Log، Error Log، Analytics و Trace ثبت نشود.
- پاسخ موفق سبک و بدون اطلاعات اضافه باشد.
- Rate Limit و Abuse Protection اعمال شود.
- Endpoint نباید امکان ارسال Payload دلخواه حجیم داشته باشد.

### FR-HB-03 — Rotation/Revoke

- Secret فقط هنگام Creation/Rotation یک‌بار نمایش داده شود.
- Rotate، Token قبلی را فوراً باطل کند.
- Revoke، Heartbeat را از کار بیندازد.

### FR-HB-04 — Late/Missed Heartbeat

- اگر Last Received از Expected Interval + Grace عبور کرد، Monitor به `down` یا `stale` برود.
- Incident ساخته شود.
- پس از دریافت Heartbeat معتبر، Incident Resolve شود.
- سیاست برای Jobی که چند بار در یک Interval اجرا می‌شود باید مستند باشد.

### FR-HB-05 — Command Snippets

UI نمونه‌های Bash/Cron و PowerShell را ارائه دهد. نمونه‌ها باید از URL واقعی ساخته شوند و Token فقط در همان صفحه Secret نمایش داده شود.

نمونه Bash:

```bash
curl --fail --silent --show-error --max-time 10 \
  -X POST "${CLIENTPULSE_HEARTBEAT_URL}"
```

در مثال عمومی مستندات، Token واقعی قرار نده.

### FR-HB-06 — Semantic Limitation

UI و Report باید صریحاً بنویسند: «Heartbeat received» یعنی اسکریپت/Job تا مرحله ارسال درخواست اجرا شده است؛ این به‌تنهایی صحت محتوا، کامل بودن یا قابلیت Restore شدن Backup را ثابت نمی‌کند.

## 6.9 Maintenance Activity Log

### FR-ACT-01

هر Activity شامل:

- Client ID (Required)
- Asset ID (Optional)
- Title (Required، محدودیت طول)
- Description (Required یا حداقل اختیاری با Category)
- Category: `update`, `backup`, `bug_fix`, `performance`, `security_review`, `content_change`, `configuration`, `other`
- Performed At
- Duration Minutes (Optional، عدد غیرمنفی با سقف معقول)
- Created By
- Created At / Updated At
- `is_billable` در MVP اختیاری یا حذف شود؛ اگر اضافه شد در UI واضح باشد.

### FR-ACT-02

- Activity قابل Create/Read/Update/Delete با Permission مناسب باشد.
- تغییر Activity بعد از استفاده در Report باید Audit شود.
- داده جعلی یا Activity خودکار بدون منبع معتبر ایجاد نشود.
- UI امکان ثبت سریع از Client Detail داشته باشد.

### FR-ACT-03

Report باید Activityهای داخل Period را با تاریخ، Category، Title، Description و در صورت وجود مدت‌زمان نشان دهد.

## 6.10 Notifications

### FR-NOT-01 — Notification Preferences

کاربر بتواند برای هر Organization مشخص کند:

- Incident Opened
- Incident Resolved
- SSL Expiry Warning
- Heartbeat Missed
- Monthly Report Ready

کانال‌ها:

- Email
- Telegram

### FR-NOT-02 — Telegram Connection

- Bot Token از Environment Variable `TELEGRAM_BOT_TOKEN` تأمین شود.
- اگر Token تنظیم نشده، UI وضعیت `Not configured by platform` نشان دهد.
- اتصال User/Organization با Deep Link و One-Time Code انجام شود.
- Code تصادفی، Hash‌شده، یک‌بارمصرف و دارای Expiry کوتاه باشد.
- Chat ID در DB ذخیره شود؛ Bot Token هرگز به Frontend ارسال نشود.
- امکان Disconnect وجود داشته باشد.
- پیام شامل نام Client/Monitor، نوع رخداد، زمان و لینک Dashboard باشد.
- Secret، Token Heartbeat یا داده حساس در پیام Telegram قرار نگیرد.

### FR-NOT-03 — Email Provider

- Email از طریق Interface/Adapter ارسال شود.
- حداقل SMTP Adapter برای نصب‌های عمومی یا Provider Adapter مانند Resend قابل پیکربندی باشد.
- `EMAIL_PROVIDER`, Host/Port/User/Password یا Provider Key از Env دریافت شوند.
- بدون Credential، Email ارسال نشود و UI/Logs وضعیت پیکربندی را واضح نشان دهند.
- Mailpit برای Development پیشنهاد می‌شود.

### FR-NOT-04 — Outbox/Retry

- Notificationها از Request اصلی جدا و در Outbox ذخیره شوند.
- Retry با Exponential Backoff و سقف مشخص.
- پیام تکراری برای یک Event/Channel ارسال نشود؛ Idempotency Key داشته باشد.
- Failure نهایی قابل مشاهده برای Admin باشد.

### FR-NOT-05 — Rate Limit / Dedup

- برای هر Incident یک Open و یک Recovery Notification؛ با رعایت Cooldown.
- SSL Warning برای هر Threshold فقط یک‌بار در هر دوره Certificate ارسال شود، مگر Expiry Date تغییر کند.
- Notificationهای fail شده در صف نامحدود نمانند.

## 6.11 Branding / White Label

### FR-BRAND-01

Organization بتواند موارد زیر را تنظیم کند:

- Agency Display Name
- Logo (PNG/JPEG/WebP، اندازه محدود)
- Primary Color با Contrast مناسب
- Accent Color اختیاری
- Report Footer Text
- Contact Email
- Website URL
- Default Report Locale

### FR-BRAND-02

- فایل Logo با MIME واقعی بررسی شود، نه صرفاً Extension.
- SVG Upload در MVP غیرفعال باشد یا با Sanitizer امن پردازش شود.
- فایل‌ها در Storage خصوصی/کنترل‌شده ذخیره شوند.
- نام فایل تصادفی باشد؛ مسیر اصلی کاربر استفاده نشود.

### FR-BRAND-03

- Branding فقط در Report همان Organization نمایش داده شود.
- اگر Logo موجود نیست، Agency Name با Typography مناسب نشان داده شود.
- ClientPulse به‌عنوان Powered by در پلن‌های پایین می‌تواند نمایش داده شود؛ قابلیت حذف آن با Entitlement کنترل شود.

## 6.12 Reports

### FR-REP-01 — Report Types

نسخه اول حداقل یک نوع Report داشته باشد: `Monthly Website Care Report`.

### FR-REP-02 — Report Period

- دوره پیش‌فرض: ماه تقویمی قبلی بر اساس Timezone Organization.
- کاربر بتواند Date Range دلخواه انتخاب کند.
- شروع/پایان بازه باید به شکل واضح در Report نمایش داده شود.
- همه Queryها از یک Period Boundary یکسان استفاده کنند.

### FR-REP-03 — Report Sections

Report شامل:

1. Agency Branding و Client Name.
2. Report Period و Generated At.
3. Executive Summary بدون ادعاهای ساختگی.
4. Website/Asset Overview.
5. Uptime Percentage و Coverage.
6. Incidentها و مدت آن‌ها.
7. Average/Median Response Time در صورت داده کافی.
8. SSL Certificate Status و Expiry Date.
9. Heartbeat/Backup Job Status و Last Received.
10. Maintenance Activities ثبت‌شده.
11. Open Issues/Recommendations مبتنی بر داده واقعی.
12. توضیح روش محاسبه و محدودیت‌ها.

### FR-REP-04 — Data Completeness

- اگر Monitor در تمام ماه فعال نبوده، Coverage Period نمایش داده شود.
- اگر داده کافی نیست، مقدار `Insufficient data` نمایش داده شود؛ 100% Uptime فرض نشود.
- اگر هیچ Activity ثبت نشده، عبارت «No activities were recorded in this period» نمایش داده شود؛ از تولید Activity خودداری شود.
- اگر Heartbeat بررسی نمی‌شود، Section آن حذف یا `Not monitored` نمایش داده شود.

### FR-REP-05 — PDF

- PDF باید متن Selectable داشته باشد.
- فونت فارسی خوانا و Embed شده باشد.
- RTL، جدول‌ها، عددها و تاریخ‌ها تست شوند.
- PDF روی Desktop و Mobile خوانا باشد.
- PDF با Template نسخه‌دار تولید شود تا Reportهای قدیمی بعد از تغییر Template آسیب نبینند.

### FR-REP-06 — Report Persistence

- Report Snapshot باید داده/مقادیر محاسبه‌شده و Template Version را ثبت کند.
- تولید مجدد Report باید مشخص باشد که Snapshot قبلی جایگزین می‌شود یا Version جدید می‌سازد.
- پیشنهاد: هر Generate یک Report Version جدید بسازد؛ نسخه قبلی قابل مشاهده بماند.

### FR-REP-07 — Sharing

- Token اشتراک‌گذاری حداقل 32 Bytes Random.
- فقط Hash Token در DB ذخیره شود.
- امکان تعیین Expiry و Revoke.
- لینک عمومی نباید ID ترتیبی یا اطلاعات Tenant دیگر را آشکار کند.
- Rate Limit روی Public Report Endpoint.
- Report فقط Read-Only باشد.
- در صورت Expired/Revoked پیام عمومی بدون افشای اطلاعات داخلی نمایش داده شود.

### FR-REP-08 — Scheduled Reports

- Scheduler ماهانه بتواند Report بسازد و برای Agency User ارسال کند.
- ارسال مستقیم به Client Contact فقط با فعال‌سازی صریح و تأیید ایمیل Client مجاز باشد.
- Timezone Organization رعایت شود.
- Jobها Idempotent باشند تا Report یا Email تکراری ایجاد نشود.

## 6.13 Dashboard

### FR-DASH-01

Dashboard حداقل شامل:

- Total Clients
- Total Active Monitors
- Current Up / Down / Unknown
- Open Incidents
- SSL Expiring Soon
- Missed Heartbeats
- Recent Maintenance Activities
- Recent Incidents
- Quick Actions: Add Client، Add Website، Add Activity، Generate Report

### FR-DASH-02

- فیلتر بر اساس Client، وضعیت و بازه زمانی.
- Tenant فقط داده Organization جاری را ببیند.
- Dashboard در حالت Empty، راهنمای Onboarding نشان دهد.
- Loading/Error/Empty Stateها طراحی شوند.
- اعداد Summary از Query واقعی بیایند و Hard-code نباشند.

## 6.14 Search / Filtering

- Client List بر اساس Name جست‌وجو شود.
- Monitorها بر اساس Status، Client و Type فیلتر شوند.
- Incidentها بر اساس Status، Date و Client.
- Reportها بر اساس Client و Period.
- Pagination Server-side.

## 6.15 Billing / Plans / Trial

### FR-BILL-01 — Plan Configuration

Planها باید از Platform Admin قابل تنظیم باشند؛ Limitها در کد Hard-code نشوند.

Planهای اولیه پیشنهادی:

| Plan | قیمت آزمایشی ماهانه | Website/Monitor Limit | Seat Limit | توضیح |
|---|---:|---:|---:|---|
| Solo | $12 | 10 | 1 | گزارش پایه |
| Studio | $24 | 30 | 3 | White Label و Heartbeat |
| Agency | $49 | 100 | 5 | امکانات تیمی و Reports بیشتر |

این قیمت‌ها فرضیه اولیه هستند و باید قابل تغییر باشند.

### FR-BILL-02 — Trial

- Signup یک Trial چهارده‌روزه ایجاد کند.
- Trial پیش‌فرض: حداکثر 3 Website Monitors و 1 User.
- هیچ Credit Card اجباری در Onboarding نباشد.
- پایان Trial به‌صورت واضح اعلام شود.
- سیاست Grace Period و Read-Only باید مشخص باشد؛ پیشنهاد: پس از اتمام Trial، ساخت Monitor جدید متوقف شود ولی مشاهده و Export داده‌ها برای مدتی محدود باقی بماند.

### FR-BILL-03 — Entitlements

- قبل از ایجاد Monitor، User، Report یا Share Link محدودیت Plan بررسی شود.
- API پاسخ مشخص و قابل ترجمه برای Limit Exceeded بدهد.
- UI Usage Bar برای Monitor/Seat نمایش دهد.

### FR-BILL-04 — Subscription State

حداقل Stateها:

- `trialing`
- `active`
- `past_due`
- `canceled`
- `expired`
- `manual_comped`

### FR-BILL-05 — Payment Gateway Abstraction

- Interface برای Payment Provider تعریف شود.
- هیچ Payment Card Data در ClientPulse ذخیره نشود.
- Gateway واقعی با Feature Flag غیرفعال باشد تا Credential و امکان استفاده قانونی مشخص شود.
- Manual Activation توسط Platform Admin در MVP قابل استفاده باشد.
- تمام تغییرات Subscription در Audit Log ثبت شود.
- Test Adapter نباید در Production به‌عنوان پرداخت موفق واقعی نمایش داده شود.

## 6.16 Platform Admin

P0 Admin Console شامل:

- Login امن با Role Platform Admin.
- List/Search Organizationها.
- مشاهده Plan، Subscription State، Usage و Trial Expiry.
- تغییر Plan/Entitlement با Audit Log.
- Suspend/Unsuspend Organization با دلیل.
- مشاهده سلامت Outbox/Worker و Job Queue.
- مدیریت Planها و Feature Flags.
- مشاهده Aggregate Metrics غیرحساس.

Platform Admin نباید بدون Audit به داده‌های خصوصی مشتریان دسترسی گسترده داشته باشد.

---

# 7. قوانین محاسباتی و داده

## 7.1 Uptime

فرمول پایه:

`Uptime % = (Observed Period - Estimated Downtime) / Observed Period * 100`

اما:

- Observed Period باید فقط بازه‌ای را دربر گیرد که Monitor فعال و داده کافی دارد.
- زمان Pause از محاسبه حذف شود.
- Detection Delay باید در Documentation شرح داده شود.
- Unknown/Missing Data معادل Up نیست.
- اگر داده Check ناقص است، Coverage نمایش داده شود.
- تمام محاسبات در Timezone UTC ذخیره و در UI بر اساس Organization Timezone نمایش داده شوند.

## 7.2 Latency

- زمان از شروع Request تا دریافت/اتمام پاسخ اندازه‌گیری شود.
- Timeout برابر Timeout تنظیم‌شده یا Duration واقعی تا قطع باشد و در Aggregate مشخص شود.
- Average و Median محاسبه شوند؛ P95 فقط وقتی Sample کافی وجود دارد.
- Manual Check از Uptime Periodic جدا باشد.

## 7.3 SSL Expiry

- Days Remaining از UTC و تاریخ معتبر محاسبه شود.
- Certificate Chain و Hostname Verification بررسی شود.
- خطای Certificate از خطای HTTP تفکیک شود.
- Expiry Warning Thresholdها قابل تنظیم در Backend باشند.

## 7.4 Report Snapshot

- هر Report باید Snapshot زمان تولید داشته باشد.
- تغییر Log یا Checkهای بعدی نباید محتوای PDF قدیمی را بی‌صدا تغییر دهد.
- Rebuild باید Version جدید ثبت کند.

---

# 8. مدل داده پیشنهادی

نام جدول‌ها و Fieldها می‌توانند در پیاده‌سازی بهینه شوند، اما معنا و Constraintها باید حفظ شوند.

## 8.1 جدول‌های اصلی

### `users`

- `id` UUID PK
- `email` normalized unique
- `password_hash`
- `email_verified_at` nullable
- `display_name`
- `status`
- `created_at`, `updated_at`

### `organizations`

- `id` UUID PK
- `name`
- `slug` unique
- `locale` (`en`, `fa`)
- `timezone`
- `status`
- `created_at`, `updated_at`, `deleted_at`

### `organization_memberships`

- `id` UUID PK
- `organization_id` FK
- `user_id` FK
- `role`
- Unique(`organization_id`, `user_id`)

### `clients`

- `id` UUID PK
- `organization_id` FK
- `name`, `contact_name`, `contact_email`, `website`, `notes`, `status`
- timestamps

### `assets`

- `id` UUID PK
- `organization_id` FK
- `client_id` FK
- `name`, `type`, `url`, `environment`, `description`, `is_active`
- timestamps

### `monitors`

- `id` UUID PK
- `organization_id` FK
- `asset_id` FK nullable where valid
- `client_id` FK
- `type` (`http`, `heartbeat`)
- `name`
- `config_json` validated against explicit schema/version
- `interval_seconds`
- `timeout_seconds`
- `status`
- `next_check_at`
- `last_check_at`
- `consecutive_failures`
- `consecutive_successes`
- timestamps

پیشنهاد مهم: داده‌های اصلی و Queryable را بی‌دلیل داخل JSON قرار نده. JSON فقط برای تنظیمات نوع‌محور Monitor باشد و Schema Version داشته باشد.

### `check_results`

- `id` UUID PK
- `organization_id` FK
- `monitor_id` FK
- `scheduled_at`, `started_at`, `completed_at`
- `result`, `status_code`, `latency_ms`
- `error_code`, `safe_error_message`
- `tls_not_before`, `tls_not_after`
- `check_source` (`scheduled`, `manual`)
- `created_at`

Index مهم: (`monitor_id`, `completed_at DESC`)، (`organization_id`, `completed_at DESC`).

### `incidents`

- `id` UUID PK
- `organization_id` FK
- `monitor_id` FK
- `status`
- `first_failure_at`
- `confirmed_at`
- `resolved_at`
- `duration_seconds`
- `failure_category`
- `acknowledged_by`, `acknowledged_at`
- timestamps

Constraint: برای هر Monitor حداکثر یک Incident باز (Partial Unique Index).

### `heartbeat_secrets`

- `id` UUID PK
- `organization_id` FK
- `monitor_id` FK
- `token_hash` unique
- `last_received_at`
- `expires_at` nullable
- `revoked_at` nullable
- timestamps

### `maintenance_activities`

- `id` UUID PK
- `organization_id` FK
- `client_id` FK
- `asset_id` FK nullable
- `created_by` FK
- `title`, `description`, `category`
- `performed_at`
- `duration_minutes` nullable
- timestamps

### `reports`

- `id` UUID PK
- `organization_id` FK
- `client_id` FK
- `period_start`, `period_end`
- `locale`
- `template_version`
- `version_number`
- `snapshot_json` یا جداول Snapshot نرمال‌شده
- `pdf_storage_key`
- `generated_by`
- `generated_at`

### `report_share_tokens`

- `id` UUID PK
- `organization_id` FK
- `report_id` FK
- `token_hash` unique
- `expires_at`
- `revoked_at`
- `created_by`

### `organization_branding`

- `organization_id` PK/FK
- `display_name`
- `logo_storage_key`
- `primary_color`, `accent_color`
- `footer_text`, `contact_email`, `website_url`

### `notification_channels`

- `id` UUID PK
- `organization_id` FK
- `type` (`email`, `telegram`)
- `destination` (Email/Chat ID؛ داده حساس با سیاست مناسب محافظت شود)
- `status`
- `created_at`

### `notification_preferences`

- `id` UUID PK
- `organization_id` FK
- `event_type`
- `channel_id` FK
- `enabled`

### `notification_outbox`

- `id` UUID PK
- `organization_id` FK nullable for platform event
- `event_key` unique/idempotency
- `channel_id`
- `payload_json` حداقل داده لازم
- `status`, `attempt_count`, `next_attempt_at`, `last_error_code`
- timestamps

### `invitations`

- `id` UUID PK
- `organization_id` FK
- `email`, `role`, `token_hash`, `expires_at`, `accepted_at`

### `plans`

- `id` UUID PK
- `code` unique
- `name`
- `price_amount`, `currency`, `billing_interval`
- `limits_json` نسخه‌دار یا فیلدهای مشخص
- `features_json`
- `is_active`

### `subscriptions`

- `id` UUID PK
- `organization_id` FK
- `plan_id` FK
- `status`
- `trial_ends_at`, `current_period_start`, `current_period_end`
- `provider`, `provider_subscription_id` nullable
- timestamps

### `audit_logs`

- `id` UUID PK
- `organization_id` nullable
- `actor_user_id` nullable
- `action`, `entity_type`, `entity_id`
- `metadata_json` با حذف Secret/PII غیرضروری
- `ip_hash` یا IP با Retention و Privacy Policy مشخص
- `created_at`

### `scheduled_jobs` یا معادل داخلی Queue

- `id` UUID PK
- `job_type`
- `payload_json`
- `run_at`
- `status`
- `locked_at`, `locked_by`
- `attempt_count`, `last_error_code`
- `idempotency_key` unique
- timestamps

## 8.2 Database Constraints

- Foreign Keyهای لازم و Referential Integrity.
- Unique Index برای Email، Slug، Token Hash و Idempotency Key.
- Partial Unique Index برای Incident باز هر Monitor.
- `organization_id` در Foreign Keyهای ترکیبی یا Query Policy برای جلوگیری از Cross-Tenant Relationship.
- `CHECK` برای مقدارهای منفی/نامعتبر.
- زمان‌ها با UTC ذخیره شوند.

## 8.3 Retention اولیه

مقادیر باید قابل تنظیم باشند:

- Raw Check Results: 90 روز.
- Aggregate Uptime/Latency: حداقل 13 ماه.
- Incidentها: 24 ماه یا طبق Subscription/Policy.
- Report Snapshotها: تا زمان حذف Organization یا سیاست Retention.
- Notification Logs: 90 روز.
- Audit Logs: حداقل 12 ماه.

پاک‌سازی باید Job کنترل‌شده و قابل گزارش باشد؛ حذف ناگهانی داده بدون Policy ممنوع است.

---

# 9. معماری فنی

## 9.1 تصمیم معماری

**Modular Monolith + Separate Worker Process**

در MVP از Microservices استفاده نکن. Backend و Worker می‌توانند از یک Go Codebase با Entry Point/Mode جدا اجرا شوند.

### اجزای اصلی

1. **Frontend:** Next.js + TypeScript.
2. **Backend API:** Go.
3. **Worker:** Go، برای Monitoring، Heartbeat Evaluation، Notification، Report Scheduling و PDF Generation.
4. **Database:** PostgreSQL.
5. **Storage:** S3-Compatible Object Storage در Production؛ Local Storage فقط Development.
6. **Reverse Proxy:** Caddy یا Nginx.
7. **Email:** Adapter قابل انتخاب.
8. **Telegram:** Telegram Bot API.

## 9.2 چرا Redis در MVP الزامی نیست؟

Job Scheduling اولیه می‌تواند با PostgreSQL و `FOR UPDATE SKIP LOCKED` پیاده‌سازی شود. این روش هزینه و پیچیدگی زیرساخت را پایین نگه می‌دارد.

اگر Load Test نشان دهد Queue دیتابیس به گلوگاه تبدیل شده است، در فاز بعد Redis/Asynq یا Queue مناسب اضافه شود. از ابتدا Abstraction داخلی Job Queue را طوری طراحی کن که Backend به یک پیاده‌سازی خاص وابسته نباشد.

## 9.3 ساختار Repository پیشنهادی

```text
clientpulse/
├── apps/
│   └── web/                    # Next.js Frontend
├── services/
│   └── api/                    # Go API + Worker modules
├── packages/
│   └── shared/                 # Shared schemas/types if useful
├── deploy/
│   ├── docker/
│   ├── caddy/
│   └── scripts/
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── security.md
│   └── operations.md
├── migrations/
├── task/
├── task-report/
├── docker-compose.yml
├── .env.example
├── README.md
└── PRD.md
```

ساختار قابل اصلاح است، اما مرزبندی Frontend، API، Worker، Deploy و Documentation حفظ شود.

## 9.4 Go Backend Modules

ماژول‌های منطقی پیشنهادی:

- `auth`
- `tenancy`
- `users`
- `organizations`
- `clients`
- `assets`
- `monitors`
- `checks`
- `incidents`
- `heartbeats`
- `maintenance`
- `reports`
- `notifications`
- `billing`
- `admin`
- `audit`
- `jobs`

Business Logic نباید داخل HTTP Handlerها انباشته شود. Handler فقط Authentication/Validation/Mapping/Response را انجام دهد.

## 9.5 Frontend Modules

- `auth`
- `onboarding`
- `dashboard`
- `clients`
- `assets`
- `monitors`
- `incidents`
- `heartbeats`
- `maintenance`
- `reports`
- `settings`
- `team`
- `billing`
- `admin` (با Route/Guard مستقل)

## 9.6 API Design

- REST JSON API با Prefix `/api/v1`.
- OpenAPI 3.x تولید و نگهداری شود.
- JSON Error Format یکپارچه.
- Pagination استاندارد.
- DateTime در API به ISO 8601 UTC.
- `request_id` برای Trace درخواست.
- Idempotency برای عملیات حساس و Job-triggerها.
- Versioning قرارداد API.

فرمت Error پیشنهادی:

```json
{
  "error": {
    "code": "MONITOR_LIMIT_REACHED",
    "message": "Your plan limit has been reached.",
    "request_id": "...",
    "details": {}
  }
}
```

`message` قابل ترجمه است؛ Frontend بهتر است بر اساس `code` متن Locale را انتخاب کند.

## 9.7 API Endpointهای اصلی پیشنهادی

### Auth

- `POST /api/v1/auth/signup`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/verify-email`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### Organization / Team

- `GET /api/v1/organizations/current`
- `PATCH /api/v1/organizations/current`
- `GET /api/v1/organizations/current/members`
- `POST /api/v1/organizations/current/invitations`
- `POST /api/v1/invitations/accept`

### Clients / Assets

- `GET|POST /api/v1/clients`
- `GET|PATCH|DELETE /api/v1/clients/{clientID}`
- `GET|POST /api/v1/clients/{clientID}/assets`
- `GET|PATCH|DELETE /api/v1/assets/{assetID}`

### Monitors

- `GET|POST /api/v1/monitors`
- `GET|PATCH|DELETE /api/v1/monitors/{monitorID}`
- `POST /api/v1/monitors/{monitorID}/test`
- `POST /api/v1/monitors/{monitorID}/pause`
- `POST /api/v1/monitors/{monitorID}/resume`
- `GET /api/v1/monitors/{monitorID}/results`

### Heartbeats

- `POST /api/v1/heartbeats`
- `GET|PATCH /api/v1/heartbeats/{monitorID}`
- `POST /api/v1/heartbeats/{monitorID}/rotate-token`
- `POST /api/v1/heartbeats/{monitorID}/revoke`
- `POST /api/v1/heartbeat-pings/{token}` — Public Token Endpoint؛ بدون Session ولی دارای محدودیت و محافظت اختصاصی. این مسیر از Routeهای مدیریت Heartbeat جداست تا ابهام Router ایجاد نشود.

### Incidents

- `GET /api/v1/incidents`
- `GET /api/v1/incidents/{incidentID}`
- `POST /api/v1/incidents/{incidentID}/acknowledge`
- `POST /api/v1/incidents/{incidentID}/resolve` — فقط در صورت فراهم بودن Manual Resolve Policy؛ سیستم نباید وضعیت مانیتور را به‌اشتباه سالم کند.

### Maintenance

- `GET|POST /api/v1/maintenance-activities`
- `GET|PATCH|DELETE /api/v1/maintenance-activities/{activityID}`

### Reports

- `GET|POST /api/v1/reports`
- `GET /api/v1/reports/{reportID}`
- `POST /api/v1/reports/{reportID}/generate-pdf`
- `POST /api/v1/reports/{reportID}/share`
- `POST /api/v1/reports/{reportID}/revoke-share`
- `GET /public/reports/{shareToken}` — بدون افشای شناسه Tenant.

### Branding / Notifications

- `GET|PATCH /api/v1/settings/branding`
- `POST /api/v1/settings/branding/logo`
- `GET|POST /api/v1/notification-channels`
- `DELETE /api/v1/notification-channels/{channelID}`
- `GET|PATCH /api/v1/notification-preferences`
- `POST /api/v1/telegram/connect/start`
- `GET /api/v1/telegram/connect/status`

### Billing

- `GET /api/v1/billing/plan`
- `GET /api/v1/billing/usage`
- `GET /api/v1/billing/subscription`
- `POST /api/v1/billing/checkout` فقط در صورت فعال بودن Provider واقعی.
- `POST /api/v1/billing/portal` فقط در صورت فعال بودن Provider واقعی.

### Admin

- `GET /api/v1/admin/organizations`
- `GET /api/v1/admin/organizations/{id}`
- `PATCH /api/v1/admin/organizations/{id}/subscription`
- `POST /api/v1/admin/organizations/{id}/suspend`
- `POST /api/v1/admin/organizations/{id}/unsuspend`
- `GET|POST|PATCH /api/v1/admin/plans`
- `GET /api/v1/admin/jobs/health`
- `GET /api/v1/admin/outbox`

Endpointها نمونه قرارداد هستند؛ هنگام پیاده‌سازی نام‌گذاری را یکدست کن و همه را در OpenAPI مستند کن.

## 9.8 Worker Scheduling

- Worker می‌تواند با چند Replica اجرا شود.
- Claim کردن Job با Transaction و `FOR UPDATE SKIP LOCKED` باشد.
- Lock Timeout و Recovery برای Worker Crash تعریف شود.
- Job دارای Idempotency Key باشد.
- Jobهای قدیمی قابل مانیتور باشند.
- Concurrency قابل تنظیم باشد.
- Per-Tenant Fairness تا حد امکان رعایت شود تا یک Tenant پرمصرف منابع همه را نگیرد.
- Checkهای یک Monitor به‌شکل هم‌زمان اجرا نشوند.
- Retry برای خطاهای گذرا و Permanent Failure جدا باشد.

## 9.9 PDF Generation

- PDF به‌صورت Async Job تولید شود.
- UI وضعیت `queued`, `generating`, `ready`, `failed` را نشان دهد.
- تولید PDF باید با Font فارسی، RTL و Page Break درست تست شود.
- اگر Playwright/Chromium استفاده می‌شود، در Container جدا یا با Resource Limit اجرا شود.
- PDF Storage خصوصی باشد؛ Download فقط بعد از Authorization یا Share Token معتبر.

---

# 10. UI / UX و صفحات

## 10.1 Design Principles

- SaaS حرفه‌ای، تمیز، خلوت و سریع.
- اولویت با خوانایی اطلاعات عملیاتی.
- Status Colorها با Icon و متن همراه باشند؛ رنگ تنها شاخص معنی نباشد.
- Dashboard برای استفاده روزمره آژانس طراحی شود.
- طراحی Responsive برای Desktop، Tablet و Mobile.
- Loading، Empty، Error، Permission Denied و Offline States کامل باشند.
- فرم‌ها Validation سمت Client و Server داشته باشند.
- عملیات Delete/Rotate/Revoke تأیید مناسب بخواهند.

## 10.2 Navigation اصلی

Sidebar پیشنهادی:

- Overview
- Clients
- Monitors
- Incidents
- Maintenance
- Reports
- Team
- Settings
- Billing

Organization Switcher در Header/Sidebar.

## 10.3 فهرست صفحات P0

### Public

1. Landing Page
2. Pricing Page
3. Login
4. Signup
5. Verify Email
6. Forgot/Reset Password
7. Terms/Privacy Placeholderهای قابل تکمیل (متن حقوقی جعلی یا ادعای انطباق تولید نشود)

### Authenticated App

1. Onboarding Wizard
2. Overview Dashboard
3. Client List
4. Client Create/Edit
5. Client Detail
6. Asset Create/Edit
7. Monitor List
8. Monitor Detail
9. Add/Edit Monitor
10. Incident List/Detail
11. Heartbeat Setup/Detail
12. Maintenance Activity List/Create/Edit
13. Reports List
14. Generate Report Wizard
15. Report Preview/Detail
16. Branding Settings
17. Notification Settings
18. Team/Members
19. Billing/Usage
20. Organization Settings

### Platform Admin

1. Admin Login/Guard
2. Organization List/Detail
3. Plan Management
4. Subscription Management
5. Job/Outbox Health

### Public Share

1. Read-only Report View
2. Expired/Revoked/Invalid Link Page

## 10.4 Dashboard Layout

بخش‌های پیشنهادی:

- Greeting + Organization Selector.
- چهار تا شش KPI Card.
- Health Overview با Status Breakdown.
- Open Incidents.
- SSL Expiring Soon.
- Missed Heartbeats.
- Recent Maintenance Activities.
- Recent Reports.
- Quick Actions.

تمام بخش‌ها از API واقعی تغذیه شوند.

## 10.5 Client Detail Layout

Tabs یا Sections:

- Overview
- Websites/Assets
- Monitors
- Incidents
- Maintenance History
- Reports

در Summary وضعیت کلی Client نمایش داده شود؛ اگر یکی از Websiteها Down است، وضعیت Green کلی نمایش داده نشود.

## 10.6 Monitor Detail Layout

- Current Status
- URL و Check Config
- Uptime و Coverage
- Latency Chart
- Check History
- Incident Timeline
- SSL Details
- Pause/Resume/Test Now

## 10.7 Report UX

- انتخاب Client و Period.
- نمایش Data Coverage و Preview Summary.
- انتخاب Locale.
- ساخت PDF.
- نمایش Generated Version.
- Share/Download/Revoke.
- هیچ متن AI-generated یا ادعای غیرمستند در Report وجود نداشته باشد.

## 10.8 Localization

- از i18n استاندارد Next.js استفاده شود.
- تمام متن UI از Translation Key بیاید.
- متن فارسی Right-aligned و Layout واقعاً RTL باشد.
- `dir` روی Root/Section مناسب تنظیم شود.
- ترتیب Icon و Button در RTL اصلاح شود.
- تاریخ/عدد/واحدها بر اساس Locale فرمت شوند.
- زبان Report مستقل از زبان فعلی Browser باشد.
- همه Validation/Error Messageها ترجمه شوند.

---

# 11. Non-Functional Requirements

## 11.1 Performance

اهداف اولیه در بار عادی:

- APIهای List معمولی: P95 کمتر از 500ms بدون احتساب Network Client.
- Dashboard Summary: P95 کمتر از 1s در Dataset آزمایشی هدف.
- UI Initial Load متناسب با Best Practiceهای Next.js.
- Check Timeout به‌صورت مستقل از API Request باشد.
- PDF Generation Async باشد و Request را Block نکند.

این اعداد هدف مهندسی هستند و باید اندازه‌گیری شوند؛ ادعای تأیید بدون Benchmark ممنوع است.

## 11.2 Availability

- Health Endpoint برای API و Worker.
- DB Connection Health.
- Queue Backlog Metric.
- Outbox Failure Metric.
- Graceful Shutdown برای Worker.
- Restart خودکار Container.
- Backup دیتابیس و تست Restore مستند.

## 11.3 Scalability

- Worker قابل Scale افقی.
- DB Index مناسب.
- Pagination Server-side.
- Check Results قابل Retention و Aggregation.
- No N+1 Query در Dashboard/Report.
- Queryهای Report از Indexهای مناسب استفاده کنند.

## 11.4 Observability

- Structured JSON Logs.
- Correlation/Request ID.
- Metrics برای تعداد Check، Failure، Duration، Job Queue و Notification.
- Health/Readiness Endpoint.
- Secretها و Tokenها Redact شوند.
- داده حساس Response Body وارد Logs نشود.

## 11.5 Accessibility

- Keyboard Navigation.
- Label واقعی برای Inputها.
- Contrast مناسب.
- Focus State مشخص.
- Status فقط با رنگ نمایش داده نشود.
- نمودارها Summary متنی/Accessible داشته باشند.

---

# 12. امنیت و حریم خصوصی

## 12.1 امنیت عمومی

- TLS اجباری در Production.
- Password Hash استاندارد.
- Secure Session Cookies.
- CSRF Protection.
- XSS Prevention و Output Encoding.
- SQL Injection Prevention با Parameterized Queries.
- Rate Limit روی Login، Signup، Public Share و Heartbeat.
- Content Security Policy مناسب.
- Security Headers.
- Dependency Scanning در CI.
- Secret Rotation Policy.

## 12.2 Multi-Tenant Isolation

- Tenant Context از Authentication معتبر Backend بیاید.
- تمام Repository Methodهای Tenant-scoped پارامتر Organization Context اجباری داشته باشند.
- Foreign Keyهای ترکیبی در جاهای لازم از Relationship بین Tenantها جلوگیری کنند.
- تست Cross-Tenant برای Read، Update، Delete، Export و Share.
- UUID به‌تنهایی Security Control نیست.
- Platform Admin مسیر مجزا و Audit شده داشته باشد.

## 12.3 SSRF

تمام الزامات بخش 6.6 اجباری‌اند و هیچ Release عمومی بدون این تست‌ها مجاز نیست.

## 12.4 Token Security

- Heartbeat Token و Share Token خام در DB ذخیره نشوند.
- Tokenها با CSPRNG ساخته شوند.
- Hash امن با Salt/Method مناسب.
- Token در URL عمومی طبیعتاً در مسیر Request وجود دارد؛ بنابراین Access Log باید Path Tokenدار را Redact کند.
- Token در Analytics/Referrer/Third-party Script نشت نکند.

## 12.5 File Upload

- فقط MIMEهای مجاز Logo.
- Max File Size.
- Image Dimension Limit.
- Malware Scan در صورت Provider/زیرساخت موجود؛ اگر نیست، ریسک در مستندات ذکر شود.
- فایل‌ها خارج از Web Root یا با Storage ACL ذخیره شوند.
- فایل اجرایی یا HTML مجاز نباشد.

## 12.6 Privacy

- حداقل‌سازی اطلاعات شخصی.
- امکان حذف/Export Organization بر اساس Policy.
- Retention مشخص.
- Report Public Link قابل Revoke.
- اطلاعات Contact Client فقط برای نیاز عملیاتی استفاده شود.
- متن Privacy Policy قبل از Production باید توسط مالک کسب‌وکار بازبینی شود؛ Gemini متن حقوقی تضمینی یا ادعای Compliance تولید نکند.

---

# 13. Billing و مدل تجاری اجرایی

## 13.1 قیمت‌های پیشنهادی اولیه

قیمت‌ها برای آزمایش بازارند و باید در Admin قابل تغییر باشند.

| Plan | Monthly Price | Monitor Limit | Seats | Branding |
|---|---:|---:|---:|---|
| Trial | $0 | 3 | 1 | ClientPulse Branding |
| Solo | $12 | 10 | 1 | Basic |
| Studio | $24 | 30 | 3 | White Label |
| Agency | $49 | 100 | 5 | Extended White Label |

## 13.2 Billing Rules

- قیمت و Currency از Plan خوانده شوند.
- قیمت در Minor Unit ذخیره شود (مثلاً USD cents)؛ Float برای Money ممنوع.
- Subscription Period و Trial Date به‌صورت UTC ذخیره شوند.
- Trial تمدید خودکار نداشته باشد مگر سیاستی صریح پیاده شود.
- Plan Change باید Proration Policy مشخص داشته باشد؛ در MVP می‌توان تغییر Plan را با شروع Period بعدی اعمال کرد.
- لغو اشتراک باید Effective Date داشته باشد.
- Admin Manual Override باید Audit شود.

## 13.3 Payment

- ClientPulse نباید Card Number یا CVV را دریافت/ذخیره کند.
- Checkout فقط از طریق Hosted Checkout Provider در صورت فعال بودن.
- Webhookها باید Signature Verify و Idempotency داشته باشند.
- Provider Eventهای قدیمی یا تکراری Subscription را به State نادرست نبرند.
- Providerهای در دسترس بر اساس کشور و امکان قانونی مالک محصول انتخاب شوند؛ Gemini نباید بدون تأیید این موضوع Stripe یا Provider خاصی را فعال فرض کند.

---

# 14. هزینه و زیرساخت اولیه

این بخش تخمین محصول است، نه تضمین قیمت Provider.

هدف اولیه معماری کم‌هزینه برای چند آژانس و حدود 100 تا 150 Website Monitor است.

پیشنهاد زیرساخت:

- یک VPS مستقل برای API، Worker و PostgreSQL در MVP کم‌ترافیک، مشروط به Backup و مانیتورینگ درست.
- ترجیح Production: دیتابیس Backup خارج از Host اصلی.
- Object Storage برای PDF و Logo.
- Email Provider در صورت پیکربندی.
- Telegram Bot Token در صورت پیکربندی.

بودجه هدف اولیه: حدود 10 تا 20 دلار در ماه، بدون احتساب هزینه‌های حقوقی، مالیاتی، Payment Fee، زمان توسعه و هزینه جذب مشتری. قبل از Production باید با قیمت روز سرویس‌دهنده و Region واقعی محاسبه شود.

**مهم:** از منابع و VPS فعلی پروژه‌های کاربر بدون تأیید استفاده نکن. ClientPulse باید Deployment مستقل و قابل خاموش/روشن کردن داشته باشد.

---

# 15. DevOps، Deployment و Backup

## 15.1 Docker Compose

حداقل سرویس‌ها:

- `web`
- `api`
- `worker`
- `postgres`
- `caddy` یا Reverse Proxy
- `mailpit` فقط Development

Storage Service می‌تواند External S3-Compatible باشد؛ در Dev Local Volume کافی است.

## 15.2 Environment Variables

حداقل موارد زیر را مستند کن:

```env
APP_ENV=development
APP_BASE_URL=http://localhost:3000
API_BASE_URL=http://localhost:8080
DEFAULT_LOCALE=en
DEFAULT_TIMEZONE=UTC

DATABASE_URL=postgres://clientpulse:change-me@postgres:5432/clientpulse?sslmode=disable

SESSION_SECRET=replace-with-long-random-secret
TOKEN_HASH_PEPPER=replace-with-long-random-secret

TELEGRAM_BOT_TOKEN=
TELEGRAM_BOT_USERNAME=

EMAIL_PROVIDER=none
SMTP_HOST=
SMTP_PORT=
SMTP_USERNAME=
SMTP_PASSWORD=
EMAIL_FROM=

STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=/data/storage
S3_ENDPOINT=
S3_BUCKET=
S3_REGION=
S3_ACCESS_KEY=
S3_SECRET_KEY=

PAYMENT_PROVIDER=manual
PAYMENT_WEBHOOK_SECRET=

WORKER_CONCURRENCY=10
MONITOR_DEFAULT_INTERVAL_SECONDS=300
MONITOR_DEFAULT_TIMEOUT_SECONDS=10
CHECK_RESULT_RETENTION_DAYS=90

PLATFORM_ADMIN_EMAIL=
PLATFORM_ADMIN_PASSWORD=
```

این‌ها نمونه اولیه‌اند. نام نهایی متغیرها باید با کد یکسان باشد. در Production برای Password/Secret از Secret Manager یا Environment امن استفاده شود؛ مقادیر بالا نباید Secret واقعی تلقی شوند.

## 15.3 Health Endpoints

- `GET /health/live`
- `GET /health/ready`
- `GET /health/worker` یا Metric مناسب برای Worker

Health Endpoint عمومی نباید اطلاعات حساس DB/Host را آشکار کند.

## 15.4 Backup

- PostgreSQL backup روزانه.
- نگهداری حداقل 7 نسخه روزانه و 4 نسخه هفتگی در Storage مستقل از VPS اصلی، به‌عنوان سیاست اولیه.
- تست Restore دوره‌ای.
- Backup Encryption در Storage خارجی.
- Backup و Restore Runbook.
- فایل PDF/Logo باید Backup/Retention مناسب داشته باشد.

## 15.5 Deployment

- Docker Imageها Pin و Reproducible باشند.
- Migration قبل از نسخه جدید با سیاست کنترل‌شده اجرا شود.
- Rollback Strategy برای API و DB Migration مستند باشد.
- Health Check قبل از اعلام Deploy موفق.
- Log Rotation فعال.
- Firewall فقط Portهای لازم را باز کند.
- PostgreSQL مستقیماً به اینترنت Public نشود.

---

# 16. Testing Strategy

## 16.1 Unit Tests

حداقل برای:

- Uptime Calculation.
- Incident State Machine.
- Failure/Recovery Threshold.
- Heartbeat Expiry.
- Plan Entitlement.
- Report Aggregation.
- Locale/Date Period Boundary.
- Token Generation/Hash/Verify.
- URL Validation/SSRF Guard.

## 16.2 Integration Tests

- Signup → Organization Creation.
- Create Client → Create Asset → Create Monitor.
- Worker Check → Check Result → Incident.
- Recovery Check → Incident Resolve.
- Heartbeat → Last Received → Missed Heartbeat Incident.
- Maintenance Activity → Report Snapshot.
- Generate PDF → Download.
- Share Token → Read → Revoke → Access Denied.
- Notification Outbox → Retry/Success.
- Plan Limit → Request Rejected.

## 16.3 Tenant Isolation Tests — Critical

دو Organization A و B بساز.

از User سازمان A تلاش کن:

- Client سازمان B را Read کند.
- Asset سازمان B را Update/Delete کند.
- Monitor سازمان B را Test/Pause کند.
- Incident سازمان B را باز کند.
- Report سازمان B را دانلود کند.
- IDهای B را در Bulk API وارد کند.
- با تغییر ID در URL/Body به داده B دسترسی پیدا کند.

همه باید با پاسخ مناسب 404/403 رد شوند، بدون اینکه وجود Resource خصوصی بی‌جهت افشا شود.

## 16.4 Security Tests

- SSRF IP/Hostname Block.
- Redirect Revalidation.
- DNS Rebinding.
- Userinfo URL.
- Oversized Response.
- Timeout.
- Token Leakage in Logs.
- Invalid/Expired Heartbeat Token.
- Brute Force Login Rate Limit.
- CSRF.
- XSS in Client Name/Activity/Report Text.
- Malicious Logo Upload.

## 16.5 E2E Tests

با Playwright یا ابزار مناسب:

1. Signup و Onboarding.
2. ساخت Client و Monitor.
3. مشاهده Check واقعی روی Test Server کنترل‌شده.
4. ایجاد Incident با Mock/Controlled Test Endpoint.
5. ثبت Activity.
6. تولید Report PDF.
7. مشاهده RTL فارسی.
8. Share و Revoke Report.
9. Role Restrictions.

تست‌های Network Monitor باید به Test Server خود پروژه متصل شوند؛ تست‌ها نباید URLهای تصادفی اینترنت یا IP داخلی محیط CI را ناخواسته اسکن کنند.

## 16.6 Load Tests

سناریوی حداقل:

- 150 Monitor با Interval پنج دقیقه.
- Worker Concurrency قابل تنظیم.
- بررسی DB Lock/Queue Backlog.
- بررسی تأخیر اجرای Check.
- بررسی مصرف CPU/RAM.
- بررسی اثر Report Generation هم‌زمان.

نتیجه Benchmark ثبت شود؛ بدون اجرای واقعی، ادعای پشتیبانی از 150 Monitor مطرح نشود.

---

# 17. Acceptance Criteria نسخه MVP

Release MVP زمانی قابل قبول است که همه موارد زیر انجام شده باشند:

### Account & Tenant

- [ ] User می‌تواند Signup/Login/Logout کند.
- [ ] Signup یک Organization و Owner Membership می‌سازد.
- [ ] User نمی‌تواند به داده Organization دیگر دسترسی پیدا کند.
- [ ] Roleهای Owner/Admin/Member اعمال می‌شوند.

### Client & Website

- [ ] Client قابل Create/Read/Update/Archive است.
- [ ] Website به Client متصل می‌شود.
- [ ] URLهای Local/Private/Metadata و Redirectهای ناامن رد می‌شوند.
- [ ] Monitor اولین Check واقعی را اجرا می‌کند.

### Monitoring

- [ ] Check دوره‌ای در Worker اجرا می‌شود.
- [ ] Status، Latency، Error و TLS ثبت می‌شوند.
- [ ] DNS و SSL بررسی می‌شوند.
- [ ] دو Failure متوالی Incident ایجاد می‌کند.
- [ ] Success بعدی Incident را Resolve می‌کند.
- [ ] Check History و Uptime نمایش داده می‌شوند.
- [ ] Pause/Resume کار می‌کند.

### Heartbeat

- [ ] کاربر Heartbeat می‌سازد.
- [ ] Token فقط یک‌بار نمایش داده می‌شود.
- [ ] Endpoint عمومی Heartbeat را ثبت می‌کند.
- [ ] Token نامعتبر رد می‌شود.
- [ ] Heartbeat از دست‌رفته Incident می‌سازد.
- [ ] Token قابل Rotate/Revoke است.

### Maintenance & Reports

- [ ] کاربر Activity ثبت می‌کند.
- [ ] Report فقط داده Period واقعی را نشان می‌دهد.
- [ ] Coverage و Insufficient Data درست نمایش داده می‌شوند.
- [ ] PDF انگلیسی و فارسی قابل خواندن است.
- [ ] Branding Organization در PDF اعمال می‌شود.
- [ ] Share Link امن، منقضی‌شونده و قابل لغو است.

### Notifications

- [ ] Notification Outbox و Idempotency کار می‌کنند.
- [ ] در صورت پیکربندی، Telegram پیام دریافت می‌کند.
- [ ] در صورت پیکربندی، Email ارسال می‌شود.
- [ ] در نبود Credential، سیستم خطای شفاف می‌دهد و ادعای ارسال موفق نمی‌کند.

### Billing & Admin

- [ ] Trial ایجاد می‌شود.
- [ ] Plan Limitها در Backend اعمال می‌شوند.
- [ ] Manual Subscription Activation کار می‌کند.
- [ ] Admin می‌تواند Plan/Status را مدیریت کند.
- [ ] تمام تغییرات مهم Audit می‌شوند.

### Operations

- [ ] Docker Compose از صفر بالا می‌آید.
- [ ] Migration و Seed اجرا می‌شوند.
- [ ] Health Checkها درست‌اند.
- [ ] Backup و Restore مستند و تست شده‌اند.
- [ ] CI Build/Test/Lint موفق است.
- [ ] README مراحل واقعی Run/Deploy را توضیح می‌دهد.

---

# 18. مراحل اجرایی پیشنهادی برای Gemini

Gemini باید این ترتیب را رعایت کند. پایان هر فاز باید قابل اجرا و قابل تست باشد.

## Phase 0 — Repository Audit & Foundation

خروجی:

- بررسی Repo و ثبت وضعیت.
- ساخت Architecture Decision Record مختصر.
- ایجاد ساختار Repo.
- Docker Compose برای Web/API/Worker/Postgres.
- Config و `.env.example`.
- Logging و Health Endpoint.
- Migration Framework.
- CI پایه.

Acceptance: پروژه با یک دستور در Development اجرا شود و Health Check پاسخ دهد.

## Phase 1 — Auth & Multi-Tenancy

خروجی:

- Signup/Login/Logout.
- Password Hashing و Session.
- Organization/Member/Role.
- Tenant Context در Backend.
- UI Onboarding.
- Tenant Isolation Tests.

Acceptance: دو Organization مستقل ساخته شوند و Cross-Tenant Tests موفق باشند.

## Phase 2 — Clients, Assets & Monitoring Core

خروجی:

- Client CRUD.
- Asset CRUD.
- URL Safety/SSRF Guard.
- Monitor Creation.
- Scheduler و Worker.
- HTTP/SSL/DNS Checks.
- Check Results.
- Dashboard و Monitor Detail.

Acceptance: یک سایت Test تحت کنترل پروژه به‌صورت واقعی در بازه زمان‌بندی‌شده بررسی شود.

## Phase 3 — Incidents & Heartbeats

خروجی:

- Incident State Machine.
- Notification Event Generation.
- Heartbeat Creation/Token/Endpoint.
- Missed Heartbeat Detection.
- Incident UI.

Acceptance: Failure و Recovery کنترل‌شده، Incident باز و بسته کنند؛ Heartbeat نیز Miss و Recovery را تشخیص دهد.

## Phase 4 — Maintenance & Reports

خروجی:

- Maintenance Activity CRUD.
- Report Aggregation.
- Snapshot Versioning.
- PDF Generator.
- White Label.
- Public Share Token.
- فارسی/انگلیسی و RTL برای Report.

Acceptance: یک Client آزمایشی گزارش واقعی و قابل دانلود دریافت کند و اطلاعات Demo با Real Data اشتباه نشود.

## Phase 5 — Notifications, Billing & Admin

خروجی:

- Telegram Connection.
- Email Adapter.
- Outbox/Retry.
- Plan/Trial/Entitlements.
- Manual Billing.
- Admin Console.

Acceptance: در صورت Credential واقعی، ارسال واقعی انجام شود؛ بدون Credential، وضعیت Unconfigured نمایش داده شود. Plan Limit سمت سرور اعمال شود.

## Phase 6 — Hardening & Release Candidate

خروجی:

- تست امنیت.
- تست بار.
- Backup/Restore.
- Observability.
- Error/Empty/Loading UI.
- Accessibility و RTL Review.
- Production Docker Compose.
- Documentation.

Acceptance: تمام Acceptance Criteria بخش 17 اجرا و نتیجه ثبت شود.

---

# 19. گزارش پیشرفت و مدیریت کار

در Repository از این ساختار استفاده کن:

```text
task/
  TASK-001-foundation.md
  TASK-002-auth-tenancy.md
  TASK-003-client-assets.md
  TASK-004-monitoring.md
  TASK-005-incidents-heartbeats.md
  TASK-006-maintenance-reports.md
  TASK-007-notifications-billing-admin.md
  TASK-008-release-hardening.md

task-report/
  TASK-001-report.md
  TASK-002-report.md
  ...
```

هر Task باید شامل موارد زیر باشد:

- هدف و محدوده.
- فایل‌ها/ماژول‌های مرتبط.
- Acceptance Criteria.
- Dependencies.
- تست‌های مورد انتظار.

هر Report باید شامل موارد زیر باشد:

- کارهای تکمیل‌شده.
- فایل‌ها و بخش‌های مهم تغییرکرده.
- تست‌های اجراشده و خروجی واقعی.
- مشکلات و محدودیت‌ها.
- کارهای باقی‌مانده.
- وضعیت نهایی: `PASS`, `PARTIAL`, `BLOCKED`.

فقط پس از اجرای تست، وضعیت PASS ثبت شود.

---

# 20. تصمیم‌های باز و فرضیات مجاز

این موارد را Gemini نباید با حدس به تصمیم پرریسک تبدیل کند؛ راهکار قابل تنظیم و کم‌ریسک انتخاب کند:

1. **دامنه نهایی:** فعلاً نامشخص؛ از `APP_BASE_URL` استفاده شود.
2. **Payment Provider:** نامشخص؛ Manual Billing و Interface آماده شود.
3. **Email Provider:** SMTP Adapter و قابلیت پیکربندی؛ بدون Credential ارسال واقعی ممکن نیست.
4. **Telegram Bot:** Token هنوز مشخص نیست؛ Feature قابل غیرفعال‌سازی.
5. **Storage Provider:** S3-Compatible در Production، Local در Development.
6. **Hosting Region:** هنوز انتخاب نشده؛ تنظیمات باید مستقل از Provider باشند.
7. **Legal Terms/Privacy:** Placeholder یا Draft برای بازبینی مالک؛ ادعای Legal Compliance ممنوع.
8. **Pricing:** قیمت‌های پیشنهادی اولیه و قابل تغییر.
9. **Brand Design:** از یک Design System تمیز و حرفه‌ای استفاده شود؛ لوگوی نهایی هنوز ارائه نشده است.
10. **Custom Domain برای آژانس:** خارج از MVP.

اگر یک تصمیم باز مانع توسعه نیست، آن را به شکل Configurable/Adapter حل کن و کار را متوقف نکن. اگر تصمیمی واقعاً مانع امنیت یا اجرای صحیح است، آن را در گزارش Blocker ثبت کن و یک گزینه امن موقت ارائه بده.

---

# 21. تعریف نهایی Done

یک قابلیت فقط زمانی Done است که:

1. UI واقعی دارد، اگر کاربر باید با آن تعامل کند.
2. API واقعی دارد.
3. Validation سمت Backend دارد.
4. Database Persistence دارد، اگر داده‌ای باید ماندگار باشد.
5. Authorization/Tenant Isolation دارد.
6. Error/Loading/Empty State دارد.
7. Unit/Integration/E2E Test متناسب دارد.
8. مستندات لازم دارد.
9. در Docker Compose اجرا شده است.
10. در گزارش پیشرفت با شواهد تست ثبت شده است.

**تعریف Done به معنی «کد نوشته شد» نیست؛ به معنی «قابلیت از مسیر واقعی کاربر تا دیتابیس و خروجی نهایی کار می‌کند و تست شده است» است.**

---

# 22. متن شروع کار برای Gemini

پس از قرار دادن این فایل در Root پروژه، متن زیر را به Gemini بده:

> این فایل `PRD.md` مشخصات مرجع محصول ClientPulse است. آن را کامل بخوان و تمام بخش‌های Functional Requirements، Security، Architecture، Acceptance Criteria و Implementation Phases را مبنا قرار بده.
>
> ابتدا Repository فعلی را بررسی کن و وضعیت آن را گزارش بده. سپس یک برنامه اجرایی مرحله‌ای بساز و بلافاصله Phase 0 را اجرا کن. فقط برنامه یا Mockup تولید نکن؛ کد واقعی، Migration، UI، API، Worker و Testها را متناسب با فاز پیاده‌سازی کن.
>
> پس از هر Phase، Build، Lint و Testهای مربوط را اجرا کن، مشکلات را رفع کن و در `task-report/` گزارش دقیق بنویس. تا وقتی یک Phase واقعاً قابل اجرا و Acceptance Criteria آن پاس نشده، آن را Complete اعلام نکن.
>
> تمام الزامات Multi-Tenant Isolation، SSRF Protection، Token Security و Report Privacy حیاتی‌اند و نباید برای سرعت حذف یا ساده‌سازی ناامن شوند. هیچ دامنه، Credential، Payment Provider یا داده واقعی را فرض نکن. از `.env.example` و Adapterهای قابل پیکربندی استفاده کن.
>
> زبان رابط کاربری از ابتدا انگلیسی و فارسی با RTL واقعی باشد. نام محصول ClientPulse است و دامنه نهایی هنوز مشخص نیست.
>
> اکنون Repository را بررسی کن، برنامه را بساز و توسعه را با Phase 0 شروع کن. در پایان هر پاسخ، دقیقاً بگو چه چیزی پیاده‌سازی شد، چه تستی اجرا شد، نتیجه چه بود و قدم اجرایی بعدی چیست. از ادعای انجام کاری که تست نشده خودداری کن.

---

**پایان PRD — ClientPulse v1.0**
