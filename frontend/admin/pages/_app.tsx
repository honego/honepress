import Head from "next/head";
import type { AppProps } from "next/app";
import { useEffect, useState } from "react";

import "@/styles/globals.css";

const defaultFaviconHref = "/honepress-black.svg";
const faviconChangeEventName = "honepress-admin-favicon-change";

export default function App({ Component, pageProps }: AppProps) {
  const [faviconHref, setFaviconHref] = useState(defaultFaviconHref);

  useEffect(() => {
    let mounted = true;
    const handleFaviconChange = (event: Event) => {
      const nextFaviconHref = faviconHrefFromEvent(event);
      setFaviconHref(nextFaviconHref);
      syncDocumentFavicon(nextFaviconHref);
    };
    window.addEventListener(faviconChangeEventName, handleFaviconChange);
    void fetch("/api/site", { credentials: "same-origin" })
      .then((response) => (response.ok ? response.json() : null))
      .then((responseBody: { site?: { iconUrl?: string } } | null) => {
        if (!mounted) return;
        const nextFaviconHref = responseBody?.site?.iconUrl?.trim() || defaultFaviconHref;
        setFaviconHref(nextFaviconHref);
        syncDocumentFavicon(nextFaviconHref);
      })
      .catch(() => {
        if (mounted) syncDocumentFavicon(defaultFaviconHref);
      });
    return () => {
      mounted = false;
      window.removeEventListener(faviconChangeEventName, handleFaviconChange);
    };
  }, []);

  return (
    <>
      <Head>
        <link rel="icon" href={faviconHref} />
      </Head>
      <Component {...pageProps} />
    </>
  );
}

function faviconHrefFromEvent(event: Event): string {
  const iconUrl = (event as CustomEvent<string>).detail;
  return typeof iconUrl === "string" ? iconUrl.trim() || defaultFaviconHref : defaultFaviconHref;
}

function syncDocumentFavicon(href: string) {
  if (typeof document === "undefined" || !href) return;
  let favicon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (!favicon) {
    favicon = document.createElement("link");
    favicon.rel = "icon";
    document.head.appendChild(favicon);
  }
  if (favicon.getAttribute("href") !== href) {
    favicon.setAttribute("href", href);
  }
}
