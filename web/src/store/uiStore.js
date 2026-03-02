import { create } from 'zustand'

export const useUIStore = create((set) => ({
  panelOpen: false,
  setPanelOpen: (panelOpen) => set({ panelOpen }),
}))
