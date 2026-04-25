export interface User {
  id: number;
  username: string;
  email?: string;
}

export interface FileItem {
  id: number;
  name: string;
  size: number;
  sizeFormatted: string;
  downloadCount: number;
  createdAt: string;
  type: string;
}

export interface FolderItem {
  id: number;
  name: string;
  createdAt: string;
}

export interface BreadcrumbItem {
  id: number;
  name: string;
}

export interface ShareLink {
  code: string;
  fileId: number;
  fileName: string;
  fileSize: number;
  accessCode?: string;
  expiresAt: string;
  downloadCount: number;
  isActive: boolean;
  createdAt: string;
}

export interface StorageInfo {
  used: number;
  max: number;
  usedFormatted: string;
  maxFormatted: string;
  percentage: number;
}

export interface UploadProgress {
  sessionId: string;
  fileId: number;
  fileName: string;
  progress: number;
  status: "pending" | "uploading" | "completed" | "error";
}

export interface RecycleItem {
  id: number;
  name: string;
  type: "file" | "folder";
  size?: number;
  sizeFormatted?: string;
  deletedAt: string;
  remainingDays: number;
  originalParentId?: number;
  originalFolderId?: number;
}

export interface ApiError {
  error: string;
}
