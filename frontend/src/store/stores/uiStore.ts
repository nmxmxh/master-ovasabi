import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { UIState } from '../types/stateArchitecture';

interface UIStore {
  // UI State
  state: UIState;

  // Actions
  setView: (view: string) => void;
  setActiveTab: (tab: number) => void;
  toggleDetails: () => void;
  setLoading: (loading: boolean) => void;
  setTheme: (theme: Partial<UIState['theme']>) => void;
  toggleModal: (modal: string) => void;
  setNavigation: (path: string) => void;

  // Getters
  getCurrentView: () => string;
  getTheme: () => UIState['theme'];
  isModalOpen: (modal: string) => boolean;

  // Debugging
  getStateSnapshot: () => any;
  logStateChange: (action: string, details?: any) => void;
}

const defaultUIState: UIState = {
  currentView: 'campaigns',
  activeTab: 0,
  showDetails: false,
  isLoading: false,
  theme: {
    primary: '#0f0',
    secondary: '#333',
    background: '#000',
    text: '#fff'
  },
  modals: {},
  navigation: {
    currentPath: '/',
    history: ['/']
  }
};

export const useUIStore = create<UIStore>()(
  devtools(
    (set, get) => ({
      state: defaultUIState,

      // Set current view
      setView: (view: string) => {
        console.log('[UIStore] 🎯 Setting view:', { view, previous: get().state.currentView });
        set(
          state => ({
            state: {
              ...state.state,
              currentView: view
            }
          }),
          false,
          'setView'
        );
      },

      // Set active tab
      setActiveTab: (tab: number) => {
        console.log('[UIStore] 📑 Setting active tab:', { tab, previous: get().state.activeTab });
        set(
          state => ({
            state: {
              ...state.state,
              activeTab: tab
            }
          }),
          false,
          'setActiveTab'
        );
      },

      // Toggle details visibility
      toggleDetails: () => {
        const current = get().state.showDetails;
        console.log('[UIStore] 👁️ Toggling details:', { showDetails: !current });
        set(
          state => ({
            state: {
              ...state.state,
              showDetails: !state.state.showDetails
            }
          }),
          false,
          'toggleDetails'
        );
      },

      // Set loading state
      setLoading: (loading: boolean) => {
        console.log('[UIStore] ⏳ Setting loading state:', {
          loading,
          previous: get().state.isLoading
        });
        set(
          state => ({
            state: {
              ...state.state,
              isLoading: loading
            }
          }),
          false,
          'setLoading'
        );
      },

      // Set theme
      setTheme: (theme: Partial<UIState['theme']>) => {
        console.log('[UIStore] 🎨 Setting theme:', { theme, previous: get().state.theme });
        set(
          state => ({
            state: {
              ...state.state,
              theme: {
                ...state.state.theme,
                ...theme
              }
            }
          }),
          false,
          'setTheme'
        );
      },

      // Toggle modal
      toggleModal: (modal: string) => {
        const current = get().state.modals[modal] || false;
        console.log('[UIStore] 🪟 Toggling modal:', { modal, open: !current });
        set(
          state => ({
            state: {
              ...state.state,
              modals: {
                ...state.state.modals,
                [modal]: !current
              }
            }
          }),
          false,
          'toggleModal'
        );
      },

      // Set navigation
      setNavigation: (path: string) => {
        const current = get().state.navigation.currentPath;
        console.log('[UIStore] 🧭 Setting navigation:', { path, previous: current });
        set(
          state => ({
            state: {
              ...state.state,
              navigation: {
                currentPath: path,
                history: [...state.state.navigation.history, path]
              }
            }
          }),
          false,
          'setNavigation'
        );
      },

      // Getters
      getCurrentView: () => get().state.currentView,
      getTheme: () => get().state.theme,
      isModalOpen: (modal: string) => get().state.modals[modal] || false,

      // Debugging
      getStateSnapshot: () => {
        const state = get();
        return {
          ui: state.state,
          timestamp: new Date().toISOString()
        };
      },

      logStateChange: (action: string, details?: any) => {
        const snapshot = get().getStateSnapshot();
        console.log(`[UIStore] 📊 State Change: ${action}`, {
          action,
          details,
          currentState: snapshot,
          timestamp: new Date().toISOString()
        });
      }
    }),
    {
      name: 'ui-store'
    }
  )
);
