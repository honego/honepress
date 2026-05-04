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

export async function fetchPosts(): Promise<PostSummary[]> {
  const postsResponse = await readJSONResponse<PostsResponse>(await fetch("/api/posts"));
  return postsResponse.posts;
}

export async function fetchPost(postID: string): Promise<PostDetail> {
  const postDetailResponse = await readJSONResponse<PostDetailResponse>(
    await fetch(`/api/posts/${encodeURIComponent(postID)}`),
  );
  return postDetailResponse.post;
}

export async function createPost(savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await fetch("/api/posts", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function updatePost(postID: string, savePostRequest: SavePostRequest): Promise<PostDetailResponse> {
  return readJSONResponse<PostDetailResponse>(
    await fetch(`/api/posts/${encodeURIComponent(postID)}`, {
      method: "PUT",
      headers: jsonHeaders,
      body: JSON.stringify(savePostRequest),
    }),
  );
}

export async function deletePost(postID: string): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await fetch(`/api/posts/${encodeURIComponent(postID)}`, {
      method: "DELETE",
    }),
  );
}

export async function previewMarkdown(markdown: string): Promise<string> {
  const previewResponse = await fetch("/api/preview", {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ markdown }),
  });
  if (!previewResponse.ok) {
    throw new Error(await readErrorMessage(previewResponse));
  }
  return previewResponse.text();
}

export async function renderSite(): Promise<MessageResponse> {
  return readJSONResponse<MessageResponse>(
    await fetch("/api/render", {
      method: "POST",
    }),
  );
}

export async function fetchSettings(): Promise<SiteSettings> {
  const settingsResponse = await readJSONResponse<SettingsResponse>(await fetch("/api/settings"));
  return settingsResponse.settings;
}

export async function updateSettings(siteSettings: SiteSettings): Promise<SettingsResponse> {
  return readJSONResponse<SettingsResponse>(
    await fetch("/api/settings", {
      method: "PUT",
      headers: jsonHeaders,
      body: JSON.stringify(siteSettings),
    }),
  );
}

async function readJSONResponse<ResponseType>(response: Response): Promise<ResponseType> {
  const responseText = await response.text();
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
