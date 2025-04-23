import { globalFixedLayout } from "@/utils/store";
import { useSetAtom } from "jotai";
import { useEffect } from "react";

/**
 * `useFixedLayout` is a custom hook that sets the global fixed layout state to true when the component mounts,
 * and resets it to false when the component unmounts.
 * @returns {void}
 *
 * @example
 *
 * ```tsx
 * import useFixedLayout from "@/hooks/useFixedLayout";
 *
 * const MyComponent = () => {
 *   useFixedLayout();
 *  return <div>My Component</div>;
 * };
 * ```
 */
const useFixedLayout = () => {
  const setFixedLayout = useSetAtom(globalFixedLayout);

  useEffect(() => {
    setFixedLayout(true);
    return () => {
      setFixedLayout(false);
    };
  });
};

export default useFixedLayout;
