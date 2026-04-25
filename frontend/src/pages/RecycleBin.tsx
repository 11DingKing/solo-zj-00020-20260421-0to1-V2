import { useState, useEffect } from "react";
import { useAuth } from "../context/AuthContext";
import {
  getRecycleList,
  restoreFile,
  restoreFolder,
  permanentlyDeleteFile,
  permanentlyDeleteFolder,
  emptyRecycleBin,
} from "../services/api";
import { RecycleItem } from "../types";

const RecycleBin = () => {
  const { user, logout } = useAuth();
  const [items, setItems] = useState<RecycleItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState<{
    message: string;
    type: "success" | "error" | "warning";
  } | null>(null);

  useEffect(() => {
    loadRecycleList();
  }, []);

  const loadRecycleList = async () => {
    try {
      setLoading(true);
      const data = await getRecycleList();
      setItems(data.items || []);
    } catch (err) {
      console.error("Failed to load recycle list:", err);
    } finally {
      setLoading(false);
    }
  };

  const showToast = (
    message: string,
    type: "success" | "error" | "warning",
  ) => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 3000);
  };

  const handleRestore = async (item: RecycleItem) => {
    if (
      !window.confirm(`确定要恢复${item.type === "folder" ? "文件夹" : "文件"} "${item.name}" 吗？`)
    ) {
      return;
    }
    try {
      let result;
      if (item.type === "folder") {
        result = await restoreFolder(item.id);
      } else {
        result = await restoreFile(item.id);
      }
      loadRecycleList();
      if (result.warning) {
        showToast(result.warning, "warning");
      } else {
        showToast(
          `${item.type === "folder" ? "文件夹" : "文件"}恢复成功`,
          "success",
        );
      }
    } catch (err: any) {
      showToast(err.response?.data?.error || "恢复失败", "error");
    }
  };

  const handlePermanentlyDelete = async (item: RecycleItem) => {
    if (
      !window.confirm(
        `确定要永久删除${item.type === "folder" ? "文件夹" : "文件"} "${item.name}" 吗？此操作不可撤销！`,
      )
    ) {
      return;
    }
    try {
      if (item.type === "folder") {
        await permanentlyDeleteFolder(item.id);
      } else {
        await permanentlyDeleteFile(item.id);
      }
      loadRecycleList();
      showToast(
        `${item.type === "folder" ? "文件夹" : "文件"}已永久删除`,
        "success",
      );
    } catch (err: any) {
      showToast(err.response?.data?.error || "删除失败", "error");
    }
  };

  const handleEmptyRecycleBin = async () => {
    if (items.length === 0) {
      showToast("回收站为空", "warning");
      return;
    }
    if (
      !window.confirm(
        `确定要清空回收站吗？所有 ${items.length} 个项目将被永久删除，此操作不可撤销！`,
      )
    ) {
      return;
    }
    try {
      await emptyRecycleBin();
      loadRecycleList();
      showToast("回收站已清空", "success");
    } catch (err: any) {
      showToast(err.response?.data?.error || "清空失败", "error");
    }
  };

  const getFileIcon = (type: string) => {
    if (type === "folder") return "📁";
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

  const formatDeletedDate = (deletedAt: string) => {
    return new Date(deletedAt).toLocaleString("zh-CN");
  };

  const getRemainingDaysClass = (days: number) => {
    if (days <= 3) return "danger";
    if (days <= 7) return "warning";
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
            <div className="logo">
              <a href="/" style={{ color: "inherit", textDecoration: "none" }}>
                ☁️ 云盘
              </a>
            </div>
            <div className="user-info">
              <span className="username">你好, {user?.username}</span>
              <button
                className="btn btn-secondary btn-sm"
                style={{ marginLeft: "12px" }}
                onClick={() => (window.location.href = "/")}
              >
                返回文件列表
              </button>
              <button
                className="btn btn-secondary btn-sm"
                onClick={logout}
              >
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
        <div className="card" style={{ marginBottom: "24px" }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <h2 style={{ margin: 0 }}>🗑️ 回收站</h2>
            <button
              className="btn btn-danger"
              onClick={handleEmptyRecycleBin}
              disabled={items.length === 0}
            >
              清空回收站
            </button>
          </div>
          <p style={{ color: "#666", marginTop: "8px", marginBottom: 0 }}>
            回收站中的文件将保留 30 天，过期后将被自动永久删除
          </p>
        </div>

        {items.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">🗑️</div>
            <div>回收站为空</div>
            <div style={{ fontSize: "14px", color: "#999", marginTop: "8px" }}>
              已删除的文件将显示在这里
            </div>
          </div>
        ) : (
          <div className="file-list">
            {items.map((item) => (
              <div
                key={`${item.type}-${item.id}`}
                className={`file-item ${item.type === "folder" ? "folder-item" : ""}`}
              >
                <div className="file-actions">
                  <button
                    className="btn btn-success btn-sm action-btn"
                    onClick={() => handleRestore(item)}
                    title="恢复"
                  >
                    恢复
                  </button>
                  <button
                    className="btn btn-danger btn-sm action-btn"
                    onClick={() => handlePermanentlyDelete(item)}
                    title="永久删除"
                  >
                    永久删除
                  </button>
                </div>
                <div className="file-icon">{getFileIcon(item.type)}</div>
                <div className="file-name">{item.name}</div>
                <div className="file-info">
                  <div className="file-info-item">
                    {item.type === "folder" ? "文件夹" : item.sizeFormatted}
                  </div>
                  <div className="file-info-item">
                    删除时间: {formatDeletedDate(item.deletedAt)}
                  </div>
                  <div
                    className={`file-info-item remaining-days ${getRemainingDaysClass(item.remainingDays)}`}
                  >
                    剩余: {item.remainingDays} 天
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {toast && (
        <div className={`toast ${toast.type}`}>{toast.message}</div>
      )}
    </div>
  );
};

export default RecycleBin;
