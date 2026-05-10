import { useRouter } from "next/router";
import { useEffect, useMemo, useState } from "react";

import { SiteHead, SiteLayout, siteName } from "../src/components/site-layout";
import { fetchPost, fetchSite, type PublicPostDetail, type PublicSiteSettings } from "../src/lib/api";

export default function PostPage() {
  const router = useRouter();
  const [post, setPost] = useState<PublicPostDetail | null>(null);
  const [site, setSite] = useState<PublicSiteSettings | null>(null);
  const [error, setError] = useState("");

  const postID = useMemo(() => {
    if (typeof router.query.id === "string") return router.query.id;
    if (typeof window === "undefined") return "";
    const pathPostID = window.location.pathname.replace(/^\/+/, "");
    return pathPostID === "posts" || pathPostID === "posts.html" ? "" : pathPostID;
  }, [router.query.id]);

  useEffect(() => {
    let mounted = true;
    async function load() {
      if (!postID) return;
      try {
        const [loadedPost, loadedSite] = await Promise.all([fetchPost(postID), fetchSite()]);
        if (!mounted) return;
        setPost(loadedPost);
        setSite(loadedSite);
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
    if (!site.giscusRepo || !site.giscusRepoId || !site.giscusCategory || !site.giscusCategoryId) return;
    const container = document.querySelector("[data-giscus-comments]");
    if (!container || container.querySelector("script[data-giscus-script]")) return;

    const script = document.createElement("script");
    script.src = "https://giscus.app/client.js";
    script.async = true;
    script.crossOrigin = "anonymous";
    script.setAttribute("data-giscus-script", "true");
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

  const siteTitle = siteName(site);
  const pageTitle = post ? postPageTitle(post, siteTitle) : siteTitle;
  const pageDescription = post ? postPageDescription(post) : site?.description;

  function goBack() {
    if (window.history.length > 1) {
      window.history.back();
      return;
    }
    window.location.assign("/");
  }

  return (
    <>
      <SiteHead
        site={site}
        title={pageTitle}
        description={pageDescription}
        canonicalPath={post?.publicUrl}
        type="article"
      />
      <SiteLayout site={site} pageClassName="page-post">
        <article className="article">
          <section className="page-lead">
            <button type="button" className="back-link" onClick={goBack}>
              返回
            </button>
          </section>
          {error ? <p className="form-error">{error}</p> : null}
          {post ? (
            <>
              <header className="page-body page-entry article-header">
                <time dateTime={post.date}>发布于 {post.date}</time>
                <h1>{post.title}</h1>
                <p>{post.description}</p>
                {post.tags.length > 0 ? (
                  <div className="post-tags">
                    {post.tags.map((tag) => (
                      <span key={tag}>{tag}</span>
                    ))}
                  </div>
                ) : null}
              </header>
              <div className="content" dangerouslySetInnerHTML={{ __html: post.html }} />
            </>
          ) : null}
        </article>
        {post && site?.commentEnabled ? <section className="comments" data-giscus-comments /> : null}
      </SiteLayout>
    </>
  );
}

function postPageTitle(post: PublicPostDetail, siteTitle: string): string {
  const customTitle = post.seoTitle.trim();
  if (customTitle) return customTitle;
  const postTitle = post.title.trim();
  if (!postTitle || postTitle === siteTitle) return siteTitle;
  return `${postTitle} - ${siteTitle}`;
}

function postPageDescription(post: PublicPostDetail): string {
  const customDescription = post.seoDescription.trim();
  if (customDescription) return customDescription;
  return post.description.trim() || post.title.trim();
}
