import { useEffect, useMemo, useState } from "react";

import { SiteHead, SiteLayout, siteName } from "../src/components/site-layout";
import { fetchPosts, fetchSite, type PostSummary, type PublicSiteSettings } from "../src/lib/api";

export default function ArchivePage() {
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

  const wordCount = useMemo(
    () => posts.reduce((total, post) => total + (post.wordCount || countVisibleCharacters(post.description)), 0),
    [posts],
  );
  const title = `归档 - ${siteName(site)}`;

  return (
    <>
      <SiteHead site={site} title={title} description={site?.description} canonicalPath="/archive.html" />
      <SiteLayout site={site} pageClassName="page-archive">
        <section className="archive">
          <header className="archive-header">
            <h1>归档</h1>
            <p className="archive-stats">
              共 {posts.length} 篇文章，约 {wordCount} 字
            </p>
          </header>
          {error ? <p className="form-error">{error}</p> : null}
          <div className="archive-list">
            {hasLoaded && !error && posts.length === 0 ? <p className="empty">还没有文章。</p> : null}
            {posts.map((post) => (
              <article className="archive-item" key={post.id}>
                {post.thumbnail ? (
                  <a className="archive-thumbnail" href={post.publicUrl}>
                    <img src={post.thumbnail} alt="" loading="lazy" />
                  </a>
                ) : null}
                <div className="archive-content">
                  <time dateTime={post.date}>{post.date}</time>
                  <h2>
                    <a href={post.publicUrl}>{post.title}</a>
                  </h2>
                </div>
              </article>
            ))}
          </div>
        </section>
      </SiteLayout>
    </>
  );
}

function countVisibleCharacters(text: string): number {
  return Array.from(text).filter((character) => !/\s/.test(character)).length;
}
