export interface PostSummary {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  publicUrl: string;
  englishPublicUrl: string;
  comments: boolean;
  translation: boolean;
  translationStatus: string;
}

export interface PostDetail {
  id: string;
  title: string;
  date: string;
  description: string;
  draft: boolean;
  url: string;
  aliases: string[];
  comments: boolean;
  translation: boolean;
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
  comments: boolean;
  translation: boolean;
  body: string;
}

export interface SiteSettings {
  title: string;
  description: string;
  baseUrl: string;
  language: string;
  githubUrl: string;
  telegramUrl: string;
  commentEnabled: boolean;
  translationEnabled: boolean;
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
