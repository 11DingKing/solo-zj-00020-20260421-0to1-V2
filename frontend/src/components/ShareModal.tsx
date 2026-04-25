import { useState } from "react";
import { FileItem } from "../types";
import { createShare } from "../services/api";

interface ShareModalProps {
  file: FileItem;
  onClose: () => void;
  onSuccess: () => void;
}

const ShareModal = ({ file, onClose, onSuccess }: ShareModalProps) => {
  const [expireDays, setExpireDays] = useState(7);
  const [accessCode, setAccessCode] = useState("");
  const [shareLink, setShareLink] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCreateShare = async () => {
    try {
      setLoading(true);
      const data = await createShare(
        file.id,
        expireDays,
        accessCode || undefined,
      );
      const baseUrl = window.location.origin;
      const link = `${baseUrl}/share/${data.code}`;
      setShareLink(link);
      onSuccess();
    } catch (err: any) {
      console.error("Create share failed:", err);
      alert(err.response?.data?.error || "创建分享失败");
    } finally {
      setLoading(false);
    }
  };

  const handleCopyLink = async () => {
    if (shareLink) {
      await navigator.clipboard.writeText(shareLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3 className="modal-title">分享文件</h3>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <div style={{ marginBottom: "16px" }}>
          <div style={{ fontWeight: "500", marginBottom: "8px" }}>
            {file.name}
          </div>
          <div style={{ fontSize: "14px", color: "#666" }}>
            {file.sizeFormatted}
          </div>
        </div>

        {!shareLink ? (
          <>
            <div className="form-group">
              <label className="form-label">过期时间</label>
              <select
                className="input"
                value={expireDays}
                onChange={(e) => setExpireDays(Number(e.target.value))}
              >
                <option value={1}>1 天</option>
                <option value={7}>7 天</option>
                <option value={30}>30 天</option>
                <option value={365}>365 天</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">
                提取码（可选，为空则无需提取码）
              </label>
              <input
                type="text"
                className="input"
                value={accessCode}
                onChange={(e) => setAccessCode(e.target.value)}
                placeholder="请输入提取码"
                maxLength={50}
              />
            </div>

            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={onClose}>
                取消
              </button>
              <button
                className="btn btn-primary"
                onClick={handleCreateShare}
                disabled={loading}
              >
                {loading ? "生成中..." : "生成分享链接"}
              </button>
            </div>
          </>
        ) : (
          <>
            <div className="form-group">
              <label className="form-label">分享链接</label>
              <div className="share-link">
                <input
                  type="text"
                  className="share-url"
                  value={shareLink}
                  readOnly
                />
                <button className="btn btn-primary" onClick={handleCopyLink}>
                  {copied ? "已复制" : "复制"}
                </button>
              </div>
            </div>
            {accessCode && (
              <div
                style={{
                  marginTop: "12px",
                  padding: "12px",
                  background: "#fffbe6",
                  borderRadius: "4px",
                }}
              >
                <div style={{ fontWeight: "500", marginBottom: "4px" }}>
                  提取码
                </div>
                <div
                  style={{
                    fontSize: "18px",
                    fontFamily: "monospace",
                    letterSpacing: "4px",
                  }}
                >
                  {accessCode}
                </div>
              </div>
            )}
            <div className="modal-footer">
              <button className="btn btn-primary" onClick={onClose}>
                完成
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default ShareModal;
