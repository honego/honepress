import Head from "next/head";
import Link from "next/link";
import { useRouter } from "next/router";
import { useEffect, useMemo, useState } from "react";

import { checkSession, fetchPost, fetchSite, PublicPostDetail, PublicSiteSettings } from "../src/lib/api";

export default function PostPage() {
  const router = useRouter();
  const [post, setPost] = useState<PublicPostDetail | null>(null);
  const [site, setSite] = useState<PublicSiteSettings | null>(null);
  const [error, setError] = useState("");

  const postID = useMemo(() => {
    if (typeof router.query.id === "string") return router.query.id;
    if (typeof window === "undefined") return "";
    const pathPostID = window.location.pathname.replace(/^\/+/, "");
    return pathPostID === "posts" ? "" : pathPostID;
  }, [router.query.id]);

  useEffect(() => {
    let mounted = true;
    async function load() {
      if (!(await checkSession())) {
        window.location.replace("/login");
        return;
      }
      if (!postID) return;
      try {
        const [loadedPost, loadedSite] = await Promise.all([fetchPost(postID), fetchSite()]);
        if (mounted) {
          setPost(loadedPost);
          setSite(loadedSite);
        }
      } catch (loadError) {
        if (mounted) setError(loadError instanceof Error ? loadError.message : "Load failed");
      }
    }
    if (router.isReady) void load();
    return () => {
      mounted = false;
    };
  }, [postID, router.isReady]);

  useEffect(() => {
    if (!post || !site?.commentEnabled) return;
    const container = document.querySelector("[data-giscus-comments]");
    if (!container || container.querySelector("script")) return;
    const script = document.createElement("script");
    script.src = "https://giscus.app/client.js";
    script.async = true;
    script.crossOrigin = "anonymous";
    script.setAttribute("data-repo", site.giscusRepo);
    script.setAttribute("data-repo-id", site.giscusRepoId);
    script.setAttribute("data-category", site.giscusCategory);
    script.setAttribute("data-category-id", site.giscusCategoryId);
    script.setAttribute("data-mapping", "pathname");
    script.setAttribute("data-strict", "0");
    script.setAttribute("data-reactions-enabled", "1");
    script.setAttribute("data-emit-metadata", "0");
    script.setAttribute("data-input-position", "bottom");
    script.setAttribute("data-theme", "preferred_color_scheme");
    script.setAttribute("data-lang", "zh-CN");
    container.appendChild(script);
  }, [post, site]);

  useEffect(() => {
    if (!site) return;
    document.documentElement.dataset.theme = site.themeDefault;
    document.documentElement.dataset.font = site.font;
  }, [site]);

  const siteTitle = site?.title || "HonePress";
  const pageTitle = post ? postPageTitle(post, siteTitle) : `Post - ${siteTitle}`;
  const pageDescription = post ? postPageDescription(post) : site?.description || "";
  const seoImage = site?.iconUrl || "";

  return (
    <>
      <Head>
        <title>{pageTitle}</title>
        {pageDescription ? <meta name="description" content={pageDescription} /> : null}
        {post ? <link rel="canonical" href={post.publicUrl} /> : null}
        {post ? <meta property="og:type" content="article" /> : null}
        {post ? <meta property="og:title" content={pageTitle} /> : null}
        {post && pageDescription ? <meta property="og:description" content={pageDescription} /> : null}
        {post ? <meta property="og:url" content={post.publicUrl} /> : null}
        <meta property="og:site_name" content={siteTitle} />
        {seoImage ? <meta property="og:image" content={seoImage} /> : null}
        <meta name="twitter:card" content={seoImage ? "summary_large_image" : "summary"} />
        {post ? <meta name="twitter:title" content={pageTitle} /> : null}
        {post && pageDescription ? <meta name="twitter:description" content={pageDescription} /> : null}
        {seoImage ? <meta name="twitter:image" content={seoImage} /> : null}
      </Head>
      <main className="site-shell">
        <header className="site-header">
          <Link href="/" className="brand-row">
            <img src={site?.iconUrl || "/honepress-black.svg"} alt="" />
            <span>{siteTitle}</span>
          </Link>
          <nav>
            <Link href="/">Home</Link>
          </nav>
        </header>
        {error ? <p className="form-error">{error}</p> : null}
        {!post && !error ? <p className="muted">Loading post...</p> : null}
        {post ? (
          <article className="article">
            <header>
              <time dateTime={post.date}>{post.date}</time>
              <h1>{post.title}</h1>
              <p>{post.description}</p>
              {post.tags.length > 0 ? (
                <div className="tag-row">
                  {post.tags.map((tag) => (
                    <span key={tag}>{tag}</span>
                  ))}
                </div>
              ) : null}
            </header>
            <div className="content" dangerouslySetInnerHTML={{ __html: post.html }} />
            {site?.commentEnabled &&
            site.giscusRepo &&
            site.giscusRepoId &&
            site.giscusCategory &&
            site.giscusCategoryId ? (
              <section className="comments" data-giscus-comments />
            ) : null}
          </article>
        ) : null}
      </main>
    </>
  );
}

function postPageTitle(post: PublicPostDetail, siteTitle: string) {
  const customTitle = post.seoTitle.trim();
  if (customTitle) return customTitle;
  const postTitle = post.title.trim();
  if (!postTitle || postTitle === siteTitle) return siteTitle;
  return `${postTitle} - ${siteTitle}`;
}

function postPageDescription(post: PublicPostDetail) {
  const customDescription = post.seoDescription.trim();
  if (customDescription) return customDescription;
  return post.description.trim() || post.title.trim();
}
