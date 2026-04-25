import { useState, useRef, useCallback } from "react";
import { uploadInit, uploadChunk, uploadComplete } from "../services/api";
import { UploadProgress } from "../types";

const CHUNK_SIZE = 2 * 1024 * 1024;

interface UploaderProps {
  currentFolderId: number;
  onUploadStart: (progress: UploadProgress) => void;
  onUploadProgress: (sessionId: string, progress: number) => void;
  onUploadComplete: (sessionId: string, success: boolean) => void;
}

const Uploader = ({
  currentFolderId,
  onUploadStart,
  onUploadProgress,
  onUploadComplete,
}: UploaderProps) => {
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (files: FileList) => {
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      uploadFile(file);
    }
  };

  const uploadFile = async (file: File) => {
    let sessionId = "";
    let fileId = 0;
    let success = false;

    try {
      const totalChunks = Math.ceil(file.size / CHUNK_SIZE);

      const initResult = await uploadInit(
        file.name,
        file.size,
        currentFolderId || undefined,
      );

      sessionId = initResult.sessionId;
      fileId = initResult.fileId;

      onUploadStart({
        sessionId,
        fileId,
        fileName: file.name,
        progress: 0,
        status: "uploading",
      });

      for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunk = file.slice(start, end);

        await uploadChunk(sessionId, i, chunk);

        const progress = ((i + 1) / totalChunks) * 100;
        onUploadProgress(sessionId, progress);
      }

      await uploadComplete(sessionId, fileId);
      success = true;
    } catch (err) {
      console.error("Upload failed:", err);
      success = false;
    } finally {
      onUploadComplete(sessionId, success);
    }
  };

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelect(e.dataTransfer.files);
    }
  }, []);

  const handleClick = () => {
    fileInputRef.current?.click();
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      handleFileSelect(e.target.files);
    }
  };

  return (
    <div
      className={`upload-area ${isDragOver ? "drag-over" : ""}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onClick={handleClick}
    >
      <input
        ref={fileInputRef}
        type="file"
        multiple
        style={{ display: "none" }}
        onChange={handleInputChange}
      />
      <div className="upload-icon">📤</div>
      <div className="upload-text">拖拽文件到此处或点击选择文件上传</div>
      <div style={{ fontSize: "12px", color: "#999", marginTop: "8px" }}>
        支持大文件分片上传，每片 2MB
      </div>
    </div>
  );
};

export default Uploader;
