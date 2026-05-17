import Head from "next/head";
import type { AppProps } from "next/app";
import { useEffect, useState } from "react";

import {
  adminFaviconChangeEventName,
  defaultFaviconHref,
  faviconHrefFromChangeEvent,
  fetchSiteFaviconHref,
  syncDocumentFavicon,
} from "@/lib/favicon";
import "@/styles/globals.css";

export default function App({ Component, pageProps }: AppProps) {
  const [faviconHref, setFaviconHref] = useState(defaultFaviconHref);

  useEffect(() => {
    let mounted = true;
    const handleFaviconChange = (event: Event) => {
      const nextFaviconHref = faviconHrefFromChangeEvent(event);
      setFaviconHref(nextFaviconHref);
      syncDocumentFavicon(nextFaviconHref);
    };
    window.addEventListener(adminFaviconChangeEventName, handleFaviconChange);
    void fetchSiteFaviconHref()
      .then((nextFaviconHref) => {
        if (!mounted) return;
        setFaviconHref(nextFaviconHref);
        syncDocumentFavicon(nextFaviconHref);
      })
      .catch(() => {
        if (mounted) syncDocumentFavicon(defaultFaviconHref);
      });
    return () => {
      mounted = false;
      window.removeEventListener(adminFaviconChangeEventName, handleFaviconChange);
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
