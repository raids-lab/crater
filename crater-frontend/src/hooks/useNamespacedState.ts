import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import { useEffect, useState } from "react";

export const useNamespacedState = (
  namespacedName: NamespacedName,
  setNamespacedName: (namespacedName: NamespacedName) => void,
) => {
  const [isOpen, setIsOpen] = useState(false);

  // if namespacedName is set, open the dialog
  // if dialog is closed, reset the namespacedName
  useEffect(() => {
    if (namespacedName) {
      setIsOpen(true);
    }
  }, [namespacedName]);

  useEffect(() => {
    if (!isOpen) {
      setNamespacedName(undefined);
    }
  }, [isOpen, setNamespacedName]);

  return [isOpen, setIsOpen] as const;
};
