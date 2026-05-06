import type {
  ErrorResponse,
  MessageResponse,
  PostDetail,
  PostDetailResponse,
  PostsResponse,
  PostSummary,
  SavePostRequest,
  SettingsResponse,
  SiteSettings,
} from "../types/post";

const jsonHeaders: HeadersInit = {
  "Content-Type": "application/json",
};

export class UnauthorizedError extends Error {
  constructor(message = "请先登录后台。") {
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

export async function checkAdminSession(): Promise<void> {
  await readJSONResponse<{ status: string }>(await authorizedFetch("/api/health"));
}

export async function fetchPosts(): Promise<PostSummary[]> {
  const postsResponse = await readJSONResponse<PostsResponse>(await authorizedFetch("/api/posts"));
  return postsResponse.posts;
}

export async function fetchPost(postID: string): Promise<PostDetail> {
  const postDetailResponse = await readJSONResponse<PostDetailResponse>(
    await authorizedFetch(`/api/posts/${encodeURIComponent(postID)}`),
  );
  return postDetailResponse.post;
}

export async function createPost(savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await authorizedFetch("/api/posts", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function updatePost(postID: string, savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await authorizedFetch(`/api/posts/${encodeURIComponent(postID)}`, {
      method: "PUT",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function deletePost(postID: string): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await authorizedFetch(`/api/posts/${encodeURIComponent(postID)}`, {
      method: "DELETE",
    }),
  );
}

export async function previewMarkdown(markdown: string): Promise<string> {
  const previewResponse = await authorizedFetch("/api/preview", {
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
  const settingsResponse = await readJSONResponse<SettingsResponse>(await authorizedFetch("/api/settings"));
  return settingsResponse.settings;
}

export async function updateSettings(siteSettings: SiteSettings): Promise<SettingsResponse> {
  return readJSONResponse<SettingsResponse>(
    await authorizedFetch("/api/settings", {
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
    throw new UnauthorizedError(readErrorMessageFromText(responseText) || "请先登录后台。");
  }
  if (!response.ok) {
    throw new Error(readErrorMessageFromText(responseText));
  }
  if (responseText.trim() === "") {
    throw new Error("接口没有返回内容。");
  }
  const parsedResponse = JSON.parse(responseText) as unknown;
  return parsedResponse as ResponseType;
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
