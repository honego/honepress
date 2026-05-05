export interface PostSummary {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  publicUrl: string;
  tags: string[];
}

export interface PostDetail {
  id: string;
  title: string;
  icon: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  aliases: string[];
  tags: string[];
  body: string;
}

export interface SavePostRequest {
  id: string;
  title: string;
  icon: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  aliases: string[];
  tags: string[];
  body: string;
}

export interface SiteSettings {
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

export interface PostsResponse {
  posts: PostSummary[];
}

export interface PostDetailResponse {
  post: PostDetail;
  message?: string;
}

export interface MessageResponse {
  message: string;
}

export interface SettingsResponse {
  settings: SiteSettings;
  message?: string;
}

export interface ErrorResponse {
  error: string;
}
