import axios from "axios";

const API_BASE_URL =
  import.meta.env.VITE_API_URL || "http://localhost:8080/api";

const api = axios.create({
  baseURL: API_BASE_URL,
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  if (config.data instanceof FormData) {
    delete config.headers["Content-Type"];
  } else if (config.data && config.headers["Content-Type"] === undefined) {
    config.headers["Content-Type"] = "application/json";
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("token");
      window.location.href = "/login";
    }
    return Promise.reject(error);
  },
);

export const getFiles = (folderId?: number) => {
  const params = folderId && folderId > 0 ? { folderId } : {};
  return api.get("/files", { params }).then((res) => res.data);
};

export const getStorageInfo = () => {
  return api.get("/files/storage").then((res) => res.data);
};

export const getBreadcrumb = (folderId?: number) => {
  const params = folderId && folderId > 0 ? { folderId } : {};
  return api.get("/files/breadcrumb", { params }).then((res) => res.data);
};

export const uploadInit = (
  filename: string,
  fileSize: number,
  folderId?: number,
) => {
  const formData = new FormData();
  formData.append("filename", filename);
  formData.append("fileSize", fileSize.toString());
  if (folderId && folderId > 0) {
    formData.append("folderId", folderId.toString());
  }
  return api.post("/files/upload/init", formData).then((res) => res.data);
};

export const uploadChunk = (
  sessionId: string,
  chunkIndex: number,
  chunk: Blob,
) => {
  const formData = new FormData();
  formData.append("sessionId", sessionId);
  formData.append("chunkIndex", chunkIndex.toString());
  formData.append("chunk", chunk);
  return api.post("/files/upload/chunk", formData).then((res) => res.data);
};

export const uploadComplete = (sessionId: string, fileId: number) => {
  const formData = new FormData();
  formData.append("sessionId", sessionId);
  formData.append("fileId", fileId.toString());
  return api.post("/files/upload/complete", formData).then((res) => res.data);
};

export const downloadFile = (fileId: number) => {
  return api
    .get(`/files/download/${fileId}`, {
      responseType: "blob",
    })
    .then((res) => res.data);
};

export const previewFile = (fileId: number) => {
  return api.get(`/files/preview/${fileId}`).then((res) => res.data);
};

export const deleteFile = (fileId: number) => {
  return api.delete(`/files/${fileId}`).then((res) => res.data);
};

export const createFolder = (name: string, parentId?: number) => {
  return api
    .post("/folders", {
      name,
      parentId,
    })
    .then((res) => res.data);
};

export const deleteFolder = (folderId: number) => {
  return api.delete(`/folders/${folderId}`).then((res) => res.data);
};

export const createShare = (
  fileId: number,
  expireDays: number,
  accessCode?: string,
) => {
  return api
    .post("/shares", {
      fileId,
      expireDays,
      accessCode,
    })
    .then((res) => res.data);
};

export const getShares = () => {
  return api.get("/shares").then((res) => res.data);
};

export const deleteShare = (code: string) => {
  return api.delete(`/shares/${code}`).then((res) => res.data);
};

export const getShareInfo = (code: string) => {
  return api.get(`/shares/${code}`).then((res) => res.data);
};

export const downloadShare = (code: string, accessCode?: string) => {
  return api
    .post(
      `/shares/${code}/download`,
      {
        accessCode,
      },
      {
        responseType: "blob",
      },
    )
    .then((res) => res.data);
};

export const getRecycleList = () => {
  return api.get("/recycle").then((res) => res.data);
};

export const restoreFile = (fileId: number) => {
  return api.post(`/recycle/restore/file/${fileId}`).then((res) => res.data);
};

export const restoreFolder = (folderId: number) => {
  return api.post(`/recycle/restore/folder/${folderId}`).then((res) => res.data);
};

export const permanentlyDeleteFile = (fileId: number) => {
  return api.delete(`/recycle/file/${fileId}`).then((res) => res.data);
};

export const permanentlyDeleteFolder = (folderId: number) => {
  return api.delete(`/recycle/folder/${folderId}`).then((res) => res.data);
};

export const emptyRecycleBin = () => {
  return api.delete("/recycle/empty").then((res) => res.data);
};

export default api;
