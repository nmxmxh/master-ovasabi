// TypeScript type definition for window.mediaStreaming including shutdown method

declare global {
  interface MediaStreamingAPI {
    connect: () => void;
    connectToCampaign: (campaignId: string, contextId: string, peerId: string) => void;
    send: (message: any) => void;
    onMessage: (callback: (data: any) => void) => void;
    onState: (callback: (state: string) => void) => void;
    isConnected: () => boolean;
    getURL: () => string;
    shutdown: () => void;
  }

  interface Window {
    mediaStreaming?: MediaStreamingAPI;
  }
}

export {}; // Ensures this file is treated as a module
