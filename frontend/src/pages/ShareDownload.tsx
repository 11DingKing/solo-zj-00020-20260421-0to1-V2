import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { getShareInfo, downloadShare } from "../services/api";
import { ShareLink } from "../types";

const ShareDownload = () => {
  const { code } = useParams<{ code: string }>();
  const [share, setShare] = useState<ShareLink | null>(null);
  const [accessCode, setAccessCode] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [downloading, setDownloading] = useState(false);
  const [needsAccessCode, setNeedsAccessCode] = useState(false);

  useEffect(() => {
    if (code) {
      loadShareInfo();
    }
  }, [code]);

  const loadShareInfo = async () => {
    try {
      setLoading(true);
      const data = await getShareInfo(code!);
      setShare(data.share);
      setNeedsAccessCode(data.needsAccessCode);
    } catch (err: any) {
      setError(err.response?.data?.error || "分享链接不存在或已过期");
    } finally {
      setLoading(false);
    }
  };

  const handleDownload = async () => {
    if (!share) return;

    try {
      setDownloading(true);
      const blob = await downloadShare(share.code, accessCode || undefined);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = share.fileName;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      setError(err.response?.data?.error || "下载失败");
    } finally {
      setDownloading(false);
    }
  };

  const formatSize = (size: number) => {
    const units = ["B", "KB", "MB", "GB"];
    let i = 0;
    let s = size;
    while (s >= 1024 && i < units.length - 1) {
      s /= 1024;
      i++;
    }
    return `${s.toFixed(1)} ${units[i]}`;
  };

  if (loading) {
    return (
      <div className="share-page">
        <div className="share-card">
          <div className="loading">加载中...</div>
        </div>
      </div>
    );
  }

  if (error && !share) {
    return (
      <div className="share-page">
        <div className="share-card">
          <div className="expired-message">{error}</div>
        </div>
      </div>
    );
  }

  if (!share) {
    return null;
  }

  return (
    <div className="share-page">
      <div className="share-card">
        <div style={{ fontSize: "48px", marginBottom: "16px" }}>
          {getIconForType(share.fileName)}
        </div>
        <div className="share-file-name">{share.fileName}</div>
        <div className="share-file-info">
          {formatSize(share.fileSize)} · 过期时间:{" "}
          {new Date(share.expiresAt).toLocaleString()}
        </div>

        {needsAccessCode && (
          <div className="access-code-input">
            <label className="form-label">提取码</label>
            <input
              type="text"
              className="input"
              value={accessCode}
              onChange={(e) => setAccessCode(e.target.value)}
              placeholder="请输入提取码"
              maxLength={50}
            />
          </div>
        )}

        {error && <div className="error-message">{error}</div>}

        <button
          className="btn btn-primary form-btn"
          onClick={handleDownload}
          disabled={downloading}
        >
          {downloading ? "下载中..." : "下载文件"}
        </button>
      </div>
    </div>
  );
};

const getIconForType = (filename: string) => {
  const ext = filename.split(".").pop()?.toLowerCase() || "";
  const imageExts = ["jpg", "jpeg", "png", "gif", "bmp", "webp", "svg"];
  const textExts = [
    "txt",
    "html",
    "css",
    "js",
    "json",
    "md",
    "go",
    "py",
    "java",
    "c",
    "cpp",
    "rs",
    "ts",
    "tsx",
    "jsx",
    "vue",
  ];
  const docExts = ["pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx"];
  const videoExts = ["mp4", "avi", "mkv", "mov", "webm"];
  const audioExts = ["mp3", "wav", "flac", "aac", "ogg"];
  const archiveExts = ["zip", "rar", "7z", "tar", "gz"];

  if (imageExts.includes(ext)) return "🖼️";
  if (textExts.includes(ext)) return "📄";
  if (docExts.includes(ext)) return "📑";
  if (videoExts.includes(ext)) return "🎬";
  if (audioExts.includes(ext)) return "🎵";
  if (archiveExts.includes(ext)) return "📦";
  return "📁";
};

export default ShareDownload;
