// Test setup file for Vitest
// Runs before each test file

import '@testing-library/jest-dom';
import { vi, afterEach } from 'vitest';

// Mock WebSocket for tests
global.WebSocket = class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.OPEN;
  onopen: ((this: WebSocket, ev: Event) => unknown) | null = null;
  onmessage: ((this: WebSocket, ev: MessageEvent<unknown>) => unknown) | null = null;
  onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null;
  onerror: ((this: WebSocket, ev: Event) => unknown) | null = null;

  constructor(
    public url: string | URL,
    protocols?: string | string[]
  ) {
    void protocols;
  }

  send(data: string | ArrayBufferLike | Blob | ArrayBufferView): void {
    void data;
  }
  close(code?: number, reason?: string): void {
    void code;
    void reason;
  }
} as unknown as typeof WebSocket;

// Mock localStorage for tests
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
  length: 0,
  key: vi.fn(),
};

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Clean up after each test
afterEach(() => {
  vi.clearAllMocks();
});
