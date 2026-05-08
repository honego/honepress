import Head from "next/head";
import Link from "next/link";
import { useEffect, useState } from "react";

import { checkSession, fetchPosts, fetchSite, PostSummary, PublicSiteSettings } from "../src/lib/api";

export default function HomePage() {
  const [posts, setPosts] = useState<PostSummary[]>([]);
  const [site, setSite] = useState<PublicSiteSettings | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let mounted = true;
    async function load() {
      if (!(await checkSession())) {
        window.location.replace("/login");
        return;
      }
      try {
        const [loadedPosts, loadedSite] = await Promise.all([fetchPosts(), fetchSite()]);
        if (mounted) {
          setPosts(loadedPosts);
          setSite(loadedSite);
        }
      } catch (loadError) {
        if (mounted) setError(loadError instanceof Error ? loadError.message : "Load failed");
      } finally {
        if (mounted) setIsLoading(false);
      }
    }
    void load();
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    if (!site) return;
    document.documentElement.dataset.theme = site.themeDefault;
    document.documentElement.dataset.font = site.font;
  }, [site]);

  const siteTitle = site?.title || "HonePress";

  return (
    <>
      <Head>
        <title>{siteTitle}</title>
        {site?.description ? <meta name="description" content={site.description} /> : null}
      </Head>
      <main className="site-shell">
        <header className="site-header">
          <Link href="/" className="brand-row">
            <img src={site?.iconUrl || "/honepress-black.svg"} alt="" />
            <span>{siteTitle}</span>
          </Link>
          <nav>
            <Link href="/admin/dashboard">Admin</Link>
          </nav>
        </header>
        <section className="hero">
          <p>{site?.description || "Personal publishing"}</p>
          <h1>{siteTitle}</h1>
        </section>
        <section className="post-list" aria-label="Posts">
          {isLoading ? <p className="muted">Loading posts...</p> : null}
          {error ? <p className="form-error">{error}</p> : null}
          {!isLoading && posts.length === 0 ? <p className="muted">No posts yet.</p> : null}
          {posts.map((post) => (
            <article className="post-row" key={post.id}>
              {post.thumbnail ? (
                <Link className="post-thumb" href={post.publicUrl}>
                  <img src={post.thumbnail} alt="" loading="lazy" />
                </Link>
              ) : null}
              <div className="post-row-body">
                <time dateTime={post.date}>{post.date}</time>
                <h2>
                  <Link href={post.publicUrl}>{post.title}</Link>
                </h2>
                <p>{post.description}</p>
                {post.tags.length > 0 ? (
                  <div className="tag-row">
                    {post.tags.map((tag) => (
                      <span key={tag}>{tag}</span>
                    ))}
                  </div>
                ) : null}
              </div>
            </article>
          ))}
        </section>
      </main>
    </>
  );
}
