import { NamespacedName } from "@/components/codeblock/PodContainerDialog";
import { useEffect, useState } from "react";

/**
 * `useNamespacedState` is a custom hook that manages the state of a dialog based on a namespaced name.
 * @param namespacedName - The namespace and name of Pod
 * @param setNamespacedName - A function to set the namespaced name
 * @returns A tuple containing the dialog open state and a function to set the dialog open state
 */
const useNamespacedState = (
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

export default useNamespacedState;
