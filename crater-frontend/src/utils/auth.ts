export interface UserInfo {
  id: string;
  role: "admin" | "user" | "viewer"; // admin | power user | user | viewer
  token: string;
}

interface AuthProvider {
  isAuthenticated: boolean;
  userInfo: UserInfo;
  signin(username: string): Promise<void>;
  signout(): Promise<void>;
}

/**
 * This represents some generic auth provider API, like Firebase.
 */
export const fakeAuthProvider: AuthProvider = {
  isAuthenticated: false,
  userInfo: {
    id: "",
    role: "viewer",
    token: "",
  },
  async signin(id: string) {
    await new Promise((r) => setTimeout(r, 500)); // fake delay
    fakeAuthProvider.isAuthenticated = true;
    fakeAuthProvider.userInfo.id = id;
    fakeAuthProvider.userInfo.role = "admin";
  },
  async signout() {
    await new Promise((r) => setTimeout(r, 500)); // fake delay
    fakeAuthProvider.isAuthenticated = false;
    fakeAuthProvider.userInfo.role = "viewer";
    fakeAuthProvider.userInfo.id = "";
  },
};
