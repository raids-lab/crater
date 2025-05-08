// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";

export const FileSizeComponent = ({ size }: { size: number }) => {
  const { t } = useTranslation();

  const formatSize = (size: number) => {
    if (size < 1024) {
      return size + t("fileSize.bytes");
    } else if (size < 1024 * 1024) {
      return (size / 1024).toFixed(2) + t("fileSize.kilobytes");
    } else if (size < 1024 * 1024 * 1024) {
      return (size / (1024 * 1024)).toFixed(2) + t("fileSize.megabytes");
    } else {
      return (size / (1024 * 1024 * 1024)).toFixed(2) + t("fileSize.gigabytes");
    }
  };

  return <>{formatSize(size)}</>;
};
