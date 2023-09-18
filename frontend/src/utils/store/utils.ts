import { AtomEffect } from "recoil";

/**
 * Save Recoil State to Local Storage.
 *
 * Usage:
 *
 * ```
 * atom({
 *  ...
 *  effects: [localStorageEffect(key)]
 * });
 * ```
 *
 * @param key local storage key
 */
export const localStorageEffect =
  <T>(key: string): AtomEffect<T> =>
  ({ setSelf, onSet }) => {
    const savedValue = localStorage.getItem(key);
    if (savedValue != null) {
      setSelf(JSON.parse(savedValue) as T);
    }
    onSet((newValue, _, isReset) => {
      isReset
        ? localStorage.removeItem(key)
        : localStorage.setItem(key, JSON.stringify(newValue));
    });
  };
