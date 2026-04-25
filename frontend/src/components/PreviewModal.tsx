interface PreviewModalProps {
  fileData: any;
  onClose: () => void;
}

const PreviewModal = ({ fileData, onClose }: PreviewModalProps) => {
  const { file, type, content } = fileData;

  const renderPreview = () => {
    if (type === "image") {
      return (
        <img
          src={fileData.filePath || fileData.previewUrl}
          alt={file.name}
          className="preview-image"
        />
      );
    }

    if (type === "text" && content) {
      return <div className="preview-text">{content}</div>;
    }

    return (
      <div className="preview-info">
        <div>
          <span className="preview-label">文件名：</span>
          <span>{file.name}</span>
        </div>
        <div>
          <span className="preview-label">文件大小：</span>
          <span>{file.sizeFormatted}</span>
        </div>
        <div>
          <span className="preview-label">文件类型：</span>
          <span>{type}</span>
        </div>
        <div>
          <span className="preview-label">上传时间：</span>
          <span>{new Date(file.createdAt).toLocaleString()}</span>
        </div>
        <div>
          <span className="preview-label">下载次数：</span>
          <span>{file.downloadCount}</span>
        </div>
      </div>
    );
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal"
        onClick={(e) => e.stopPropagation()}
        style={{ maxWidth: "700px" }}
      >
        <div className="modal-header">
          <h3 className="modal-title">文件预览 - {file.name}</h3>
          <button className="modal-close" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="preview-content">{renderPreview()}</div>

        <div className="modal-footer">
          <button className="btn btn-secondary" onClick={onClose}>
            关闭
          </button>
        </div>
      </div>
    </div>
  );
};

export default PreviewModal;
