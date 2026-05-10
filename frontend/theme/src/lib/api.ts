export interface PostSummary {
  id: string;
  title: string;
  thumbnail: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  publicUrl: string;
  tags: string[];
  wordCount: number;
}

export interface PublicPostDetail {
  id: string;
  title: string;
  thumbnail: string;
  date: string;
  description: string;
  seoTitle: string;
  seoDescription: string;
  url: string;
  publicUrl: string;
  tags: string[];
  html: string;
}

export interface PublicSiteSettings {
  title: string;
  description: string;
  iconUrl: string;
  commentEnabled: boolean;
  giscusRepo: string;
  giscusRepoId: string;
  giscusCategory: string;
  giscusCategoryId: string;
  themeDefault: "auto" | "light" | "dark";
  font: "default" | "douyin-sans";
}

interface PostsResponse {
  posts: PostSummary[];
}

interface PublicPostResponse {
  post: PublicPostDetail;
}

interface PublicSiteResponse {
  site: PublicSiteSettings;
}

export async function fetchPosts(): Promise<PostSummary[]> {
  const response = await fetch("/api/posts", { credentials: "same-origin" });
  if (!response.ok) {
    throw new Error(await readError(response));
  }
  const data = (await response.json()) as PostsResponse;
  return data.posts;
}

export async function fetchSite(): Promise<PublicSiteSettings> {
  const response = await fetch("/api/site", { credentials: "same-origin" });
  if (!response.ok) {
    throw new Error(await readError(response));
  }
  const data = (await response.json()) as PublicSiteResponse;
  return data.site;
}

export async function fetchPost(postID: string): Promise<PublicPostDetail> {
  const response = await fetch(`/api/posts/${encodeURIComponent(postID)}`, { credentials: "same-origin" });
  if (!response.ok) {
    throw new Error(await readError(response));
  }
  const data = (await response.json()) as PublicPostResponse;
  return data.post;
}

async function readError(response: Response): Promise<string> {
  const text = await response.text();
  if (text.trim() === "") return "请求失败";
  try {
    const parsed = JSON.parse(text) as { error?: string };
    return parsed.error || text;
  } catch {
    return text;
  }
}
