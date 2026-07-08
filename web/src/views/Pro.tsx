import { useEffect, useRef, useState } from "react";
import { useStore } from "@nanostores/react";
import { useTranslation } from "react-i18next";
import "../i18n";
import { ArrowRight, Check, Plus, Loader2 } from "lucide-react";
import { $user } from "../store/auth";
import StaticFooter from "../components/common/StaticFooter";
import {
  checkSession,
  getBillingStatus,
  createCheckout,
  createPortal,
  type BillingStatus,
} from "../api/client";

type Plan = "monthly" | "yearly";
type TFn = (key: string) => string;

export default function Pro() {
  const { t } = useTranslation();
  const user = useStore($user);
  const [billing, setBilling] = useState<BillingStatus | null>(null);
  const [ready, setReady] = useState(false);
  const [pending, setPending] = useState<Plan | "portal" | null>(null);
  const pricingRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let alive = true;
    (async () => {
      const u = await checkSession();
      if (u !== undefined) $user.set(u);
      if (u) {
        const bill = await getBillingStatus();
        if (alive) setBilling(bill);
      }
      if (alive) setReady(true);
    })();
    return () => {
      alive = false;
    };
  }, []);

  const hasSub = billing?.hasSubscription ?? false;

  const handleCheckout = async (plan: Plan) => {
    if (!user) {
      window.location.href = "/login";
      return;
    }
    setPending(plan);
    const url = await createCheckout(plan);
    if (url) window.location.href = url;
    else setPending(null);
  };

  const handlePortal = async () => {
    setPending("portal");
    const url = await createPortal();
    if (url) window.location.href = url;
    else setPending(null);
  };

  const scrollToPricing = () =>
    pricingRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });

  const features: { title: string; desc: string }[] = [
    {
      title: t("pro.features.readingRoom.title"),
      desc: t("pro.features.readingRoom.desc"),
    },
    {
      title: t("pro.features.customDomain.title"),
      desc: t("pro.features.customDomain.desc"),
    },
    {
      title: t("pro.features.themes.title"),
      desc: t("pro.features.themes.desc"),
    },
    {
      title: t("pro.features.fonts.title"),
      desc: t("pro.features.fonts.desc"),
    },
    {
      title: t("pro.features.layouts.title"),
      desc: t("pro.features.layouts.desc"),
    },
    {
      title: t("pro.features.social.title"),
      desc: t("pro.features.social.desc"),
    },
  ];

  const includes = [
    t("pro.pricing.includes.readingRoom"),
    t("pro.pricing.includes.customDomain"),
    t("pro.pricing.includes.themes"),
    t("pro.pricing.includes.social"),
    t("pro.pricing.includes.future"),
  ];

  const faqs = [
    { q: t("pro.faq.q1"), a: t("pro.faq.a1") },
    { q: t("pro.faq.q2"), a: t("pro.faq.a2") },
    { q: t("pro.faq.q3"), a: t("pro.faq.a3") },
    { q: t("pro.faq.q4"), a: t("pro.faq.a4") },
  ];

  return (
    <div className="min-h-screen bg-surface-100 dark:bg-surface-900 text-surface-900 dark:text-surface-100 antialiased">
      {/* Nav */}
      <nav className="sticky top-3 sm:top-4 z-50 px-3 sm:px-5">
        <div className="max-w-[30rem] mx-auto flex items-center justify-between gap-3 bg-white dark:bg-surface-800 ring-1 ring-surface-200 dark:ring-white/10 rounded-full pl-3.5 pr-2 py-2 shadow-md shadow-surface-900/5 dark:shadow-surface-900/15">
          <a href="/" className="group flex items-center gap-2">
            <img
              src="/logo.svg"
              alt="Margin"
              className="w-6 h-6 transition-transform duration-300 ease-out group-hover:rotate-[-6deg]"
            />
            <span className="text-[13px] font-semibold tracking-tight">
              Margin{" "}
              <span className="text-primary-600 dark:text-primary-400">
                Pro
              </span>
            </span>
          </a>
          <div className="flex items-center gap-1">
            <a
              href={user ? "/home" : "/login"}
              className="text-[13px] font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white transition-colors px-3 py-1.5 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
            >
              {user ? t("pro.nav.openApp") : t("pro.nav.signIn")}
            </a>
            <button
              onClick={hasSub ? handlePortal : scrollToPricing}
              className="inline-flex items-center gap-1.5 text-[13px] font-semibold pl-3.5 pr-4 py-1.5 bg-primary-600 text-white rounded-full hover:bg-primary-500 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
            >
              {hasSub ? t("pro.hero.ctaManage") : t("pro.hero.ctaUpgrade")}
            </button>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <header>
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 pt-16 md:pt-24 pb-16 md:pb-20 text-center">
          <h1
            className="font-display font-semibold tracking-[-0.03em] leading-[1.04] text-surface-900 dark:text-white text-[clamp(2.25rem,6vw,4.25rem)] max-w-[18ch] mx-auto"
            style={{ textWrap: "balance" }}
          >
            {t("pro.hero.title")}
          </h1>
          <p className="mt-7 mx-auto max-w-[40rem] text-[16px] md:text-[18px] leading-[1.65] text-surface-600 dark:text-surface-300">
            {t("pro.hero.lede")}
          </p>

          <div className="mt-9 flex flex-wrap justify-center items-center gap-3 min-h-[46px]">
            {!ready ? (
              <Loader2 size={20} className="animate-spin text-surface-400" />
            ) : hasSub ? (
              <>
                <span className="inline-flex items-center gap-2 px-5 py-3 text-[14.5px] font-medium text-emerald-700 dark:text-emerald-400">
                  <Check size={16} />
                  {t("pro.hero.activeNote")}
                </span>
                <button
                  onClick={handlePortal}
                  className="group inline-flex items-center gap-2 px-5 py-3 text-[14.5px] font-medium text-surface-700 dark:text-surface-200 bg-white dark:bg-surface-800 rounded-full border border-surface-200 dark:border-surface-700/60 hover:border-surface-300 dark:hover:border-surface-600 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
                >
                  {pending === "portal" && (
                    <Loader2 size={15} className="animate-spin" />
                  )}
                  {t("pro.hero.ctaManage")}
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={scrollToPricing}
                  className="group inline-flex items-center gap-2 px-6 py-3 text-[14.5px] font-semibold bg-primary-600 text-white rounded-full hover:bg-primary-500 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
                >
                  {user ? t("pro.hero.ctaUpgrade") : t("pro.hero.ctaSignIn")}
                  <ArrowRight
                    size={15}
                    className="transition-transform duration-200 ease-out group-hover:translate-x-0.5"
                  />
                </button>
                <a
                  href="#pricing"
                  onClick={(e) => {
                    e.preventDefault();
                    scrollToPricing();
                  }}
                  className="px-5 py-3 text-[14.5px] font-medium text-surface-600 dark:text-surface-300 hover:text-surface-900 dark:hover:text-white transition-colors"
                >
                  {t("pro.pricing.heading")}
                </a>
              </>
            )}
          </div>
          <p className="mt-5 text-[13px] text-surface-500 dark:text-surface-400">
            {t("pro.hero.note")}
          </p>
        </div>

        <div className="max-w-[68rem] mx-auto px-5 sm:px-8">
          <div className="h-px bg-surface-200 dark:bg-surface-800/80" />
        </div>
      </header>

      {/* Features */}
      <section>
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-20 md:py-28">
          <div className="max-w-[42rem] mx-auto text-center mb-12 md:mb-16">
            <p className="font-display text-[1.75rem] md:text-[2.25rem] leading-[1.15] tracking-[-0.02em] text-surface-900 dark:text-white">
              {t("pro.features.subheading")}
            </p>
          </div>
          <dl className="max-w-[46rem] mx-auto border-t border-surface-200 dark:border-surface-800">
            {features.map((f) => (
              <div
                key={f.title}
                className="grid grid-cols-1 sm:grid-cols-[16rem_minmax(0,1fr)] gap-2 sm:gap-10 py-5 sm:py-6 border-b border-surface-200 dark:border-surface-800"
              >
                <dt className="font-display font-semibold text-[1.05rem] tracking-tight text-surface-900 dark:text-white">
                  {f.title}
                </dt>
                <dd className="text-[15px] leading-[1.65] text-surface-600 dark:text-surface-400 max-w-[40rem] sm:pt-1.5">
                  {f.desc}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      {/* Pricing */}
      <section
        id="pricing"
        ref={pricingRef}
        className="border-t border-surface-200 dark:border-surface-800 scroll-mt-24"
      >
        <div className="max-w-[68rem] mx-auto px-5 sm:px-8 py-20 md:py-28">
          <div className="max-w-[42rem] mx-auto text-center mb-12 md:mb-16">
            <p className="font-display text-[1.75rem] md:text-[2.25rem] leading-[1.15] tracking-[-0.02em] text-surface-900 dark:text-white">
              {t("pro.pricing.subheading")}
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-5 max-w-[48rem] mx-auto items-stretch">
            <PlanCard
              name={t("pro.pricing.monthly")}
              price={t("pro.pricing.monthlyPrice")}
              per={t("pro.pricing.perMonth")}
              plan="monthly"
              highlighted={false}
              hasSub={hasSub}
              currentPlan={billing?.plan}
              pending={pending}
              ready={ready}
              onChoose={handleCheckout}
              onManage={handlePortal}
              t={t}
            />
            <PlanCard
              name={t("pro.pricing.yearly")}
              price={t("pro.pricing.yearlyPrice")}
              per={t("pro.pricing.perYear")}
              badge={t("pro.pricing.yearlyNote")}
              subnote={t("pro.pricing.billedYearly")}
              plan="yearly"
              highlighted
              hasSub={hasSub}
              currentPlan={billing?.plan}
              pending={pending}
              ready={ready}
              onChoose={handleCheckout}
              onManage={handlePortal}
              t={t}
            />
          </div>

          <div className="mt-12 pt-8 border-t border-surface-200 dark:border-surface-800 max-w-[48rem] mx-auto">
            <ul className="grid sm:grid-cols-2 gap-x-10 gap-y-3">
              {includes.map((item) => (
                <li
                  key={item}
                  className="flex items-center gap-2.5 text-[14.5px] text-surface-700 dark:text-surface-300"
                >
                  <Check
                    size={16}
                    strokeWidth={2}
                    className="shrink-0 text-primary-600 dark:text-primary-400"
                  />
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section className="border-t border-surface-200 dark:border-surface-800">
        <div className="max-w-[52rem] mx-auto px-5 sm:px-8 py-20 md:py-28">
          <div className="border-t border-surface-200 dark:border-surface-800">
            {faqs.map((item) => (
              <details
                key={item.q}
                className="group border-b border-surface-200 dark:border-surface-800 py-5"
              >
                <summary className="flex cursor-pointer items-center justify-between gap-4 list-none font-display text-[1.05rem] font-semibold tracking-tight text-surface-900 dark:text-white">
                  {item.q}
                  <Plus
                    size={18}
                    className="shrink-0 text-surface-400 transition-transform duration-200 group-open:rotate-45"
                  />
                </summary>
                <p className="mt-3 text-[15px] leading-[1.7] text-surface-600 dark:text-surface-400 max-w-[44rem]">
                  {item.a}
                </p>
              </details>
            ))}
          </div>
        </div>
      </section>

      <StaticFooter />
    </div>
  );
}

function PlanCard({
  name,
  price,
  per,
  badge,
  subnote,
  plan,
  highlighted,
  hasSub,
  currentPlan,
  pending,
  ready,
  onChoose,
  onManage,
  t,
}: {
  name: string;
  price: string;
  per: string;
  badge?: string;
  subnote?: string;
  plan: Plan;
  highlighted: boolean;
  hasSub: boolean;
  currentPlan?: string;
  pending: Plan | "portal" | null;
  ready: boolean;
  onChoose: (plan: Plan) => void;
  onManage: () => void;
  t: TFn;
}) {
  const isCurrent = hasSub && currentPlan === plan;

  return (
    <div
      className={`relative flex flex-col rounded-2xl p-6 md:p-7 bg-white dark:bg-surface-800/40 border ${
        highlighted
          ? "border-primary-500 ring-1 ring-primary-500"
          : "border-surface-200 dark:border-surface-800"
      }`}
    >
      {badge && (
        <span className="absolute -top-2.5 right-6 inline-flex items-center rounded-full bg-primary-600 text-white px-2.5 py-1 text-[11px] font-semibold">
          {badge}
        </span>
      )}
      <p className="text-[13px] font-semibold uppercase tracking-wider text-surface-500 dark:text-surface-400">
        {name}
      </p>
      <div className="mt-3 flex items-baseline gap-1">
        <span className="font-display text-[2.75rem] font-semibold tracking-[-0.02em] text-surface-900 dark:text-white leading-none">
          {price}
        </span>
        <span className="text-[15px] text-surface-500 dark:text-surface-400">
          {per}
        </span>
      </div>
      <p className="mt-2 min-h-[1.25rem] text-[13px] text-surface-500 dark:text-surface-400">
        {subnote ?? " "}
      </p>

      <div className="mt-6">
        {!ready ? (
          <div className="h-11 rounded-lg bg-surface-100 dark:bg-surface-700/50 animate-pulse" />
        ) : isCurrent ? (
          <button
            onClick={onManage}
            className="w-full h-11 inline-flex items-center justify-center gap-2 rounded-full text-[14px] font-medium bg-surface-100 dark:bg-surface-700 text-surface-700 dark:text-surface-200 hover:bg-surface-200 dark:hover:bg-surface-600 transition-colors"
          >
            {pending === "portal" ? (
              <Loader2 size={15} className="animate-spin" />
            ) : (
              <Check size={15} />
            )}
            {t("pro.pricing.currentPlan")}
          </button>
        ) : (
          <button
            onClick={() => onChoose(plan)}
            disabled={hasSub}
            className={`w-full h-11 inline-flex items-center justify-center gap-2 rounded-full text-[14px] font-semibold transition-colors disabled:opacity-40 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/50 ${
              highlighted
                ? "bg-primary-600 text-white hover:bg-primary-500"
                : "bg-surface-900 dark:bg-white text-white dark:text-surface-900 hover:opacity-90"
            }`}
          >
            {pending === plan && <Loader2 size={15} className="animate-spin" />}
            {t("pro.pricing.choosePlan")}
          </button>
        )}
      </div>
    </div>
  );
}
