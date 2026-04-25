import { FileItem, FolderItem } from "../types";

interface FileListProps {
  files: FileItem[];
  folders: FolderItem[];
  onFolderClick: (folderId: number) => void;
  onFilePreview: (file: FileItem) => void;
  onFileDownload: (file: FileItem) => void;
  onFileShare: (file: FileItem) => void;
  onFileDelete: (file: FileItem) => void;
  onFolderDelete: (folder: FolderItem) => void;
}

const FileList = ({
  files,
  folders,
  onFolderClick,
  onFilePreview,
  onFileDownload,
  onFileShare,
  onFileDelete,
  onFolderDelete,
}: FileListProps) => {
  const getFileIcon = (type: string) => {
    const icons: Record<string, string> = {
      image: "🖼️",
      text: "📄",
      document: "📑",
      video: "🎬",
      audio: "🎵",
      archive: "📦",
      other: "📁",
    };
    return icons[type] || icons.other;
  };

  const getFolderIcon = () => "📁";

  if (files.length === 0 && folders.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-icon">📂</div>
        <div>当前文件夹为空</div>
        <div style={{ fontSize: "14px", color: "#999", marginTop: "8px" }}>
          点击"上传文件"或"新建文件夹"开始使用
        </div>
      </div>
    );
  }

  return (
    <div className="file-list">
      {folders.map((folder) => (
        <div
          key={`folder-${folder.id}`}
          className="folder-item"
          onClick={() => onFolderClick(folder.id)}
        >
          <div className="file-actions" onClick={(e) => e.stopPropagation()}>
            <button
              className="btn btn-danger btn-sm action-btn"
              onClick={() => onFolderDelete(folder)}
            >
              删除
            </button>
          </div>
          <div className="file-icon">{getFolderIcon()}</div>
          <div className="file-name">{folder.name}</div>
          <div className="file-info">
            <div className="file-info-item">文件夹</div>
            <div className="file-info-item">
              {new Date(folder.createdAt).toLocaleDateString()}
            </div>
          </div>
        </div>
      ))}

      {files.map((file) => (
        <div
          key={`file-${file.id}`}
          className="file-item"
          onClick={() => onFilePreview(file)}
        >
          <div className="file-actions" onClick={(e) => e.stopPropagation()}>
            <button
              className="btn btn-secondary btn-sm action-btn"
              onClick={() => onFileDownload(file)}
            >
              下载
            </button>
            <button
              className="btn btn-secondary btn-sm action-btn"
              onClick={() => onFileShare(file)}
            >
              分享
            </button>
            <button
              className="btn btn-danger btn-sm action-btn"
              onClick={() => onFileDelete(file)}
            >
              删除
            </button>
          </div>
          <div className="file-icon">{getFileIcon(file.type)}</div>
          <div className="file-name">{file.name}</div>
          <div className="file-info">
            <div className="file-info-item">{file.sizeFormatted}</div>
            <div className="file-info-item">下载: {file.downloadCount}</div>
            <div className="file-info-item">
              {new Date(file.createdAt).toLocaleDateString()}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};

export default FileList;
