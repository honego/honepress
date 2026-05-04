export interface PostSummary {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  publicUrl: string;
  comments: boolean;
  tags: string[];
}

export interface PostDetail {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  aliases: string[];
  tags: string[];
  comments: boolean;
  body: string;
}

export interface SavePostRequest {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  aliases: string[];
  tags: string[];
  comments: boolean;
  body: string;
}

export interface SiteSettings {
  title: string;
  description: string;
  baseUrl: string;
  language: string;
  iconUrl: string;
  githubUrl: string;
  telegramUrl: string;
  commentEnabled: boolean;
  commentProvider: string;
  giscusRepo: string;
  giscusRepoId: string;
  giscusCategory: string;
  giscusCategoryId: string;
  giscusMapping: string;
  giscusStrict: string;
  giscusReactionsEnabled: string;
  giscusEmitMetadata: string;
  giscusInputPosition: string;
  giscusTheme: string;
  giscusLang: string;
  themeDefault: "auto" | "light" | "dark";
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
