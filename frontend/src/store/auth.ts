import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User } from "@/types";

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  activeOrgId: string | null;
  activeProjectId: string | null;
  setAuth: (access: string, refresh: string, user: User) => void;
  setTokens: (access: string, refresh: string) => void;
  setActiveOrg: (id: string | null) => void;
  setActiveProject: (id: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      activeOrgId: null,
      activeProjectId: null,
      setAuth: (accessToken, refreshToken, user) =>
        set({ accessToken, refreshToken, user }),
      setTokens: (accessToken, refreshToken) =>
        set({ accessToken, refreshToken }),
      setActiveOrg: (activeOrgId) => set({ activeOrgId }),
      setActiveProject: (activeProjectId) => set({ activeProjectId }),
      logout: () =>
        set({
          accessToken: null,
          refreshToken: null,
          user: null,
          activeOrgId: null,
          activeProjectId: null,
        }),
    }),
    { name: "pulse-auth" }
  )
);
