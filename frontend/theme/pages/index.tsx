import { useEffect, useState } from "react";

import { SiteHead, SiteLayout } from "../src/components/site-layout";
import { fetchPosts, fetchSite, type PostSummary, type PublicSiteSettings } from "../src/lib/api";

export default function HomePage() {
  const [posts, setPosts] = useState<PostSummary[]>([]);
  const [site, setSite] = useState<PublicSiteSettings | null>(null);
  const [hasLoaded, setHasLoaded] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let mounted = true;
    async function load() {
      try {
        const [loadedPosts, loadedSite] = await Promise.all([fetchPosts(), fetchSite()]);
        if (!mounted) return;
        setPosts(loadedPosts);
        setSite(loadedSite);
      } catch (loadError) {
        if (mounted) setError(loadError instanceof Error ? loadError.message : "Load failed");
      } finally {
        if (mounted) setHasLoaded(true);
      }
    }
    void load();
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <>
      <SiteHead site={site} canonicalPath="/" />
      <SiteLayout site={site} pageClassName="page-home">
        {site?.description ? (
          <section className="page-lead">
            <p className="site-kicker">{site.description}</p>
          </section>
        ) : null}
        <section className="page-body post-list" aria-label="文章">
          {error ? <p className="form-error">{error}</p> : null}
          {hasLoaded && !error && posts.length === 0 ? <p className="empty">还没有文章。</p> : null}
          {posts.map((post) => (
            <article className="page-entry post-item" key={post.id}>
              {post.thumbnail ? (
                <a className="post-thumbnail" href={post.publicUrl}>
                  <img src={post.thumbnail} alt="" loading="lazy" />
                </a>
              ) : null}
              <div className="post-content">
                <time dateTime={post.date}>{post.date}</time>
                <h3>
                  <a href={post.publicUrl}>{post.title}</a>
                </h3>
                <p>{post.description}</p>
              </div>
            </article>
          ))}
        </section>
      </SiteLayout>
    </>
  );
}
