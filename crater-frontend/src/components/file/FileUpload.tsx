// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import instance from "@/services/axios";
import { useQueryClient } from "@tanstack/react-query";
import { UploadIcon } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface ProgressProps {
  progress: number;
}

const FloatingProgress: React.FC<ProgressProps> = ({ progress }) => {
  const { t } = useTranslation();

  return (
    <div
      style={{
        position: "fixed",
        top: "20px",
        right: "20px",
        zIndex: 1000,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      <progress
        value={progress}
        max="100"
        style={{ width: "200px", marginBottom: "5px" }}
      />
      <span>{t("fileUpload.progressLabel", { progress: progress })}</span>
    </div>
  );
};

interface FileUploadProps {
  uploadPath: string;
  disabled: boolean;
}

const FileUpload: React.FC<FileUploadProps> = ({ uploadPath, disabled }) => {
  const { t } = useTranslation();
  const [isUploading, setIsUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const queryClient = useQueryClient();

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { files } = event.currentTarget;
    if (files && files.length) {
      const Files = Array.from(files);
      handleUpload(Files);
    }
  };

  const handleUpload = async (Files: File[]) => {
    if (Files.length == 0) return;
    setIsUploading(true);
    setProgress(0);
    const filename = Files[0].name.split("/").pop();
    if (filename === undefined) return null;
    const filedataBuffer = await Files[0].arrayBuffer();
    try {
      await instance
        .put(`/ss${uploadPath}/${filename}`, filedataBuffer, {
          onUploadProgress(progressEvent) {
            const { loaded, total } = progressEvent;
            if (total !== undefined && total > 0) {
              const percentCompleted = Math.round((loaded * 100) / total);
              setProgress(percentCompleted);
            }
          },
        })
        .then(() => {
          toast.success(t("fileUpload.successMessage"));
          void queryClient.invalidateQueries({
            queryKey: ["data", "filesystem", uploadPath],
          });
          setTimeout(() => setProgress(0), 500);
        })
        .catch((error) => {
          toast.info(error);
        });
    } finally {
      setIsUploading(false);
      setTimeout(() => setProgress(0), 500);
    }
  };

  const handleClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  return (
    <>
      <Button
        variant="outline"
        size="icon"
        className="h-8 w-8"
        title={t("fileUpload.buttonTitle")}
        disabled={isUploading || disabled}
        onClick={handleClick}
      >
        <UploadIcon className="size-4" />
      </Button>
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        style={{ display: "none" }}
      />
      {progress > 0 && <FloatingProgress progress={progress} />}
    </>
  );
};

export default FileUpload;
