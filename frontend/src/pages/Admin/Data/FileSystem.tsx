// i18n-processed-v1.1.0 (no translatable strings)
import { FileSystemTable } from "@/components/file/FileSystemTable";
import { apiGetAdminFile } from "@/services/api/file";

export const Component = () => {
  return <FileSystemTable apiGetFiles={apiGetAdminFile} isadmin={true} />;
};
