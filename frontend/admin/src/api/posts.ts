import type {
  AdminPostsResponse,
  AdminStats,
  ErrorResponse,
  MeResponse,
  MessageResponse,
  PostDetail,
  PostDetailResponse,
  SavePostRequest,
  SettingsResponse,
  SiteSettings,
} from "../types/posts";

const jsonHeaders: HeadersInit = {
  "Content-Type": "application/json",
};

export class UnauthorizedError extends Error {
  constructor(message = "请先登录。") {
    super(message);
    this.name = "UnauthorizedError";
  }
}

export async function loginAdmin(username: string, password: string): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await fetch("/api/login", {
      method: "POST",
      headers: jsonHeaders,
      credentials: "same-origin",
      body: JSON.stringify({ username, password }),
    }),
  );
}

export async function logoutAdmin(): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await fetch("/api/logout", {
      method: "POST",
      credentials: "same-origin",
    }),
  );
}

export async function checkAdminSession(): Promise<MeResponse> {
  return readJSONResponse<MeResponse>(await authorizedFetch("/api/admin/me"));
}

export async function fetchAdminStats(): Promise<AdminStats> {
  return readJSONResponse<AdminStats>(await authorizedFetch("/api/admin/stats"));
}

export async function fetchPosts(params: {
  page?: number;
  pageSize?: number;
  search?: string;
  draft?: "all" | "true" | "false";
}): Promise<AdminPostsResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.pageSize) searchParams.set("pageSize", String(params.pageSize));
  if (params.search) searchParams.set("search", params.search);
  if (params.draft && params.draft !== "all") searchParams.set("draft", params.draft);
  const queryString = searchParams.toString();
  return readJSONResponse<AdminPostsResponse>(
    await authorizedFetch(`/api/admin/posts${queryString ? `?${queryString}` : ""}`),
  );
}

export async function fetchPost(postID: string): Promise<PostDetail> {
  const postDetailResponse = await readJSONResponse<PostDetailResponse>(
    await authorizedFetch(`/api/admin/posts/${encodeURIComponent(postID)}`),
  );
  return postDetailResponse.post;
}

export async function createPost(savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await authorizedFetch("/api/admin/posts", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function updatePost(postID: string, savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await authorizedFetch(`/api/admin/posts/${encodeURIComponent(postID)}`, {
      method: "PUT",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function deletePost(postID: string): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await authorizedFetch(`/api/admin/posts/${encodeURIComponent(postID)}`, {
      method: "DELETE",
    }),
  );
}

export async function previewMarkdown(markdown: string): Promise<string> {
  const previewResponse = await authorizedFetch("/api/admin/preview", {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ markdown }),
  });
  if (!previewResponse.ok) {
    throw new Error(await readErrorMessage(previewResponse));
  }
  return previewResponse.text();
}

export async function fetchSettings(): Promise<SiteSettings> {
  const settingsResponse = await readJSONResponse<SettingsResponse>(await authorizedFetch("/api/admin/settings"));
  return settingsResponse.settings;
}

export async function updateSettings(siteSettings: SiteSettings): Promise<SettingsResponse> {
  return readJSONResponse<SettingsResponse>(
    await authorizedFetch("/api/admin/settings", {
      method: "PUT",
      headers: jsonHeaders,
      body: JSON.stringify(siteSettings),
    }),
  );
}

async function authorizedFetch(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
  return fetch(input, {
    ...init,
    credentials: "same-origin",
  });
}

async function readJSONResponse<ResponseType>(response: Response): Promise<ResponseType> {
  const responseText = await response.text();
  if (response.status === 401) {
    throw new UnauthorizedError(readErrorMessageFromText(responseText) || "请先登录。");
  }
  if (!response.ok) {
    throw new Error(readErrorMessageFromText(responseText));
  }
  if (responseText.trim() === "") {
    throw new Error("接口没有返回内容。");
  }
  return JSON.parse(responseText) as ResponseType;
}

async function readErrorMessage(response: Response): Promise<string> {
  const responseText = await response.text();
  return readErrorMessageFromText(responseText);
}

function readErrorMessageFromText(responseText: string): string {
  if (responseText.trim() === "") {
    return "请求失败。";
  }
  try {
    const parsedResponse = JSON.parse(responseText) as unknown;
    if (isErrorResponse(parsedResponse)) {
      return parsedResponse.error;
    }
  } catch {
    return responseText;
  }
  return responseText;
}

function isErrorResponse(value: unknown): value is ErrorResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const errorCandidate = value as { error?: unknown };
  return typeof errorCandidate.error === "string";
}
