import { useState, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import FileList from "../components/FileList";
import Uploader from "../components/Uploader";
import Breadcrumb from "../components/Breadcrumb";
import ShareModal from "../components/ShareModal";
import PreviewModal from "../components/PreviewModal";
import {
  getFiles,
  getStorageInfo,
  getBreadcrumb,
  createFolder,
  deleteFile,
  deleteFolder,
  downloadFile,
  previewFile,
} from "../services/api";
import {
  FileItem,
  FolderItem,
  BreadcrumbItem,
  StorageInfo,
  UploadProgress,
} from "../types";

const Dashboard = () => {
  const { user, logout } = useAuth();
  const [currentFolderId, setCurrentFolderId] = useState(0);
  const [files, setFiles] = useState<FileItem[]>([]);
  const [folders, setFolders] = useState<FolderItem[]>([]);
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbItem[]>([]);
  const [storage, setStorage] = useState<StorageInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [showUploader, setShowUploader] = useState(false);
  const [showCreateFolder, setShowCreateFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [uploadProgresses, setUploadProgresses] = useState<UploadProgress[]>(
    [],
  );
  const [shareFile, setShareFile] = useState<FileItem | null>(null);
  const [previewFileData, setPreviewFileData] = useState<any>(null);
  const [toast, setToast] = useState<{
    message: string;
    type: "success" | "error";
  } | null>(null);

  useEffect(() => {
    loadData();
  }, [currentFolderId]);

  const loadData = async () => {
    try {
      setLoading(true);
      const [filesData, breadcrumbData, storageData] = await Promise.all([
        getFiles(currentFolderId),
        getBreadcrumb(currentFolderId),
        getStorageInfo(),
      ]);
      setFiles(filesData.files || []);
      setFolders(filesData.folders || []);
      setBreadcrumb(breadcrumbData);
      setStorage(storageData);
    } catch (err) {
      console.error("Failed to load data:", err);
    } finally {
      setLoading(false);
    }
  };

  const showToast = (message: string, type: "success" | "error") => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
  };

  const handleNavigateFolder = (folderId: number) => {
    setCurrentFolderId(folderId);
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    try {
      await createFolder(newFolderName, currentFolderId || undefined);
      setNewFolderName("");
      setShowCreateFolder(false);
      loadData();
      showToast("文件夹创建成功", "success");
    } catch (err: any) {
      showToast(err.response?.data?.error || "创建失败", "error");
    }
  };

  const handleDeleteFile = async (file: FileItem) => {
    if (!window.confirm(`确定要删除文件 "${file.name}" 吗？`)) return;
    try {
      await deleteFile(file.id);
      loadData();
      showToast("文件删除成功", "success");
    } catch (err: any) {
      showToast(err.response?.data?.error || "删除失败", "error");
    }
  };

  const handleDeleteFolder = async (folder: FolderItem) => {
    if (!window.confirm(`确定要删除文件夹 "${folder.name}" 吗？`)) return;
    try {
      await deleteFolder(folder.id);
      loadData();
      showToast("文件夹删除成功", "success");
    } catch (err: any) {
      showToast(err.response?.data?.error || "删除失败", "error");
    }
  };

  const handleDownloadFile = async (file: FileItem) => {
    try {
      const blob = await downloadFile(file.id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = file.name;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      showToast(err.response?.data?.error || "下载失败", "error");
    }
  };

  const handlePreviewFile = async (file: FileItem) => {
    try {
      const data = await previewFile(file.id);
      setPreviewFileData({ ...data, file });
    } catch (err: any) {
      showToast(err.response?.data?.error || "预览失败", "error");
    }
  };

  const handleUploadStart = (progress: UploadProgress) => {
    setUploadProgresses((prev) => [...prev, progress]);
  };

  const handleUploadProgress = (sessionId: string, progress: number) => {
    setUploadProgresses((prev) =>
      prev.map((p) => (p.sessionId === sessionId ? { ...p, progress } : p)),
    );
  };

  const handleUploadComplete = (sessionId: string, success: boolean) => {
    setUploadProgresses((prev) =>
      prev.map((p) =>
        p.sessionId === sessionId
          ? { ...p, status: success ? "completed" : ("error" as const) }
          : p,
      ),
    );
    if (success) {
      loadData();
      showToast("文件上传成功", "success");
    }
    setTimeout(() => {
      setUploadProgresses((prev) =>
        prev.filter((p) => p.sessionId !== sessionId),
      );
    }, 3000);
  };

  const getProgressColorClass = () => {
    if (!storage) return "normal";
    if (storage.percentage >= 90) return "danger";
    if (storage.percentage >= 70) return "warning";
    return "normal";
  };

  if (loading) {
    return (
      <div className="loading">
        <div style={{ fontSize: "48px", marginBottom: "16px" }}>⏳</div>
        <div>加载中...</div>
      </div>
    );
  }

  return (
    <div>
      <header className="header">
        <div className="container">
          <div className="header-content">
            <div className="logo">☁️ 云盘</div>
            <div className="user-info">
              <span className="username">你好, {user?.username}</span>
              <button
                className="btn btn-secondary btn-sm"
                onClick={() => (window.location.href = "/recycle")}
              >
                🗑️ 回收站
              </button>
              <button className="btn btn-secondary btn-sm" onClick={logout}>
                退出登录
              </button>
            </div>
          </div>
        </div>
      </header>

      <main
        className="container"
        style={{ paddingTop: "24px", paddingBottom: "24px" }}
      >
        {storage && (
          <div className="storage-bar card">
            <div className="storage-info">
              <span>存储空间</span>
              <span className="storage-percentage">
                {storage.usedFormatted} / {storage.maxFormatted} (
                {storage.percentage.toFixed(1)}%)
              </span>
            </div>
            <div className="progress-bar">
              <div
                className={`progress-fill ${getProgressColorClass()}`}
                style={{ width: `${Math.min(storage.percentage, 100)}%` }}
              />
            </div>
          </div>
        )}

        <Breadcrumb items={breadcrumb} onNavigate={handleNavigateFolder} />

        <div className="toolbar">
          <div className="toolbar-left">
            <button
              className="btn btn-primary"
              onClick={() => setShowUploader(!showUploader)}
            >
              📤 上传文件
            </button>
            <button
              className="btn btn-secondary"
              onClick={() => setShowCreateFolder(true)}
            >
              📁 新建文件夹
            </button>
          </div>
        </div>

        {showUploader && (
          <Uploader
            currentFolderId={currentFolderId}
            onUploadStart={handleUploadStart}
            onUploadProgress={handleUploadProgress}
            onUploadComplete={handleUploadComplete}
          />
        )}

        {uploadProgresses.length > 0 && (
          <div className="upload-progress-list">
            {uploadProgresses.map((p) => (
              <div key={p.sessionId} className="upload-progress-item">
                <div className="upload-progress-info">
                  <div className="upload-progress-name">{p.fileName}</div>
                  <div className="upload-progress-bar-container">
                    <div
                      className="upload-progress-bar"
                      style={{ width: `${p.progress}%` }}
                    />
                  </div>
                </div>
                <div className="upload-progress-status">
                  {p.status === "completed" && "✅"}
                  {p.status === "error" && "❌"}
                  {p.status === "uploading" && `${p.progress.toFixed(0)}%`}
                </div>
              </div>
            ))}
          </div>
        )}

        <FileList
          files={files}
          folders={folders}
          onFolderClick={handleNavigateFolder}
          onFilePreview={handlePreviewFile}
          onFileDownload={handleDownloadFile}
          onFileShare={(file) => setShareFile(file)}
          onFileDelete={handleDeleteFile}
          onFolderDelete={handleDeleteFolder}
        />
      </main>

      {showCreateFolder && (
        <div
          className="modal-overlay"
          onClick={() => setShowCreateFolder(false)}
        >
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">新建文件夹</h3>
              <button
                className="modal-close"
                onClick={() => setShowCreateFolder(false)}
              >
                ×
              </button>
            </div>
            <div className="create-folder-input">
              <input
                type="text"
                className="input"
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                placeholder="文件夹名称"
                autoFocus
                onKeyDown={(e) => e.key === "Enter" && handleCreateFolder()}
              />
              <button className="btn btn-primary" onClick={handleCreateFolder}>
                创建
              </button>
            </div>
          </div>
        </div>
      )}

      {shareFile && (
        <ShareModal
          file={shareFile}
          onClose={() => setShareFile(null)}
          onSuccess={() => {
            showToast("分享链接已生成", "success");
          }}
        />
      )}

      {previewFileData && (
        <PreviewModal
          fileData={previewFileData}
          onClose={() => setPreviewFileData(null)}
        />
      )}

      {toast && <div className={`toast ${toast.type}`}>{toast.message}</div>}
    </div>
  );
};

export default Dashboard;
