/**
 * Comprehensive tests for storage synchronization hooks
 * 
 * Tests cover:
 * - Initial value loading from storage
 * - Value updates and persistence
 * - Error handling (JSON parse errors, storage unavailable)
 * - Callback memoization
 * - Edge cases (null values, empty strings, etc.)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { StorageAdapter, TypedStorage } from '../../persistence/types';
import {
  useStorageSync,
  useBooleanStorageSync,
  useJsonStorageSync,
  useTypedStorageSync,
} from '../useStorageSync';

// ============================================================================
// Mock Factories
// ============================================================================

const createMockAdapter = (overrides: Partial<StorageAdapter> = {}): StorageAdapter => ({
  isAvailable: vi.fn(() => true),
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  ...overrides,
});

const createMockTypedStorage = <T>(overrides: Partial<TypedStorage<T>> = {}): TypedStorage<T> => ({
  get: vi.fn(),
  set: vi.fn(),
  ...overrides,
});

// ============================================================================
// useStorageSync Tests
// ============================================================================

describe('useStorageSync', () => {
  let mockAdapter: StorageAdapter;

  beforeEach(() => {
    mockAdapter = createMockAdapter();
  });

  describe('initial value loading', () => {
    it('should load initial value from storage when available', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('stored-value');
      
      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      expect(result.current[0]).toBe('stored-value');
      expect(mockAdapter.getItem).toHaveBeenCalledWith('test-key');
    });

    it('should use default value when storage returns null', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      expect(result.current[0]).toBe('default-value');
    });

    it('should use default value when storage is unavailable', () => {
      (mockAdapter.isAvailable as ReturnType<typeof vi.fn>).mockReturnValue(false);

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      expect(result.current[0]).toBe('default-value');
      expect(mockAdapter.getItem).not.toHaveBeenCalled();
    });

    it('should use default value when stored value is empty string', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('');

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      expect(result.current[0]).toBe('');
    });
  });

  describe('value updates', () => {
    it('should update state and persist to storage', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      act(() => {
        result.current[1]('new-value');
      });

      expect(result.current[0]).toBe('new-value');
      expect(mockAdapter.setItem).toHaveBeenCalledWith('test-key', 'new-value');
    });

    it('should handle multiple consecutive updates', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      act(() => {
        result.current[1]('value-1');
      });

      act(() => {
        result.current[1]('value-2');
      });

      act(() => {
        result.current[1]('value-3');
      });

      expect(result.current[0]).toBe('value-3');
      expect(mockAdapter.setItem).toHaveBeenCalledTimes(3);
    });

    it('should persist empty string values', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      act(() => {
        result.current[1]('');
      });

      expect(result.current[0]).toBe('');
      expect(mockAdapter.setItem).toHaveBeenCalledWith('test-key', '');
    });
  });

  describe('callback memoization', () => {
    it('should return stable setter function across renders', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(() =>
        useStorageSync(mockAdapter, 'test-key', 'default-value')
      );

      const firstSetter = result.current[1];
      
      rerender();
      
      const secondSetter = result.current[1];

      expect(firstSetter).toBe(secondSetter);
    });

    it('should create new setter when adapter changes', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(
        ({ adapter }) => useStorageSync(adapter, 'test-key', 'default-value'),
        { initialProps: { adapter: mockAdapter } }
      );

      const firstSetter = result.current[1];

      const newAdapter = createMockAdapter();
      (newAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);
      
      rerender({ adapter: newAdapter });

      const secondSetter = result.current[1];

      expect(firstSetter).not.toBe(secondSetter);
    });

    it('should create new setter when key changes', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(
        ({ key }) => useStorageSync(mockAdapter, key, 'default-value'),
        { initialProps: { key: 'key-1' } }
      );

      const firstSetter = result.current[1];

      rerender({ key: 'key-2' });

      const secondSetter = result.current[1];

      expect(firstSetter).not.toBe(secondSetter);
    });
  });
});

// ============================================================================
// useBooleanStorageSync Tests
// ============================================================================

describe('useBooleanStorageSync', () => {
  let mockAdapter: StorageAdapter;

  beforeEach(() => {
    mockAdapter = createMockAdapter();
  });

  describe('initial value loading', () => {
    it('should load true value from storage', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('true');

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', false)
      );

      expect(result.current[0]).toBe(true);
    });

    it('should load false value from storage', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('false');

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', true)
      );

      expect(result.current[0]).toBe(false);
    });

    it('should use default value when storage returns null', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', true)
      );

      expect(result.current[0]).toBe(true);
    });

    it('should use default value when storage is unavailable', () => {
      (mockAdapter.isAvailable as ReturnType<typeof vi.fn>).mockReturnValue(false);

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', false)
      );

      expect(result.current[0]).toBe(false);
    });

    it('should treat any non-"true" string as false', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('invalid');

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', true)
      );

      expect(result.current[0]).toBe(false);
    });

    it('should treat empty string as false', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('');

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', true)
      );

      expect(result.current[0]).toBe(false);
    });
  });

  describe('value updates', () => {
    it('should persist true as "true" string', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', false)
      );

      act(() => {
        result.current[1](true);
      });

      expect(result.current[0]).toBe(true);
      expect(mockAdapter.setItem).toHaveBeenCalledWith('bool-key', 'true');
    });

    it('should persist false as "false" string', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', true)
      );

      act(() => {
        result.current[1](false);
      });

      expect(result.current[0]).toBe(false);
      expect(mockAdapter.setItem).toHaveBeenCalledWith('bool-key', 'false');
    });

    it('should handle toggling between true and false', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', false)
      );

      act(() => {
        result.current[1](true);
      });

      act(() => {
        result.current[1](false);
      });

      act(() => {
        result.current[1](true);
      });

      expect(result.current[0]).toBe(true);
      expect(mockAdapter.setItem).toHaveBeenCalledTimes(3);
    });
  });

  describe('callback memoization', () => {
    it('should return stable setter function across renders', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(() =>
        useBooleanStorageSync(mockAdapter, 'bool-key', false)
      );

      const firstSetter = result.current[1];
      
      rerender();
      
      const secondSetter = result.current[1];

      expect(firstSetter).toBe(secondSetter);
    });
  });
});

// ============================================================================
// useJsonStorageSync Tests
// ============================================================================

describe('useJsonStorageSync', () => {
  let mockAdapter: StorageAdapter;

  beforeEach(() => {
    mockAdapter = createMockAdapter();
  });

  describe('initial value loading', () => {
    it('should load and parse JSON object from storage', () => {
      const storedData = { name: 'test', count: 42 };
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(JSON.stringify(storedData));

      const { result } = renderHook(() =>
        useJsonStorageSync<{ name: string; count: number }>(mockAdapter, 'json-key', { name: '', count: 0 })
      );

      expect(result.current[0]).toEqual(storedData);
    });

    it('should load and parse JSON array from storage', () => {
      const storedData = [1, 2, 3, 4, 5];
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(JSON.stringify(storedData));

      const { result } = renderHook(() =>
        useJsonStorageSync<number[]>(mockAdapter, 'json-key', [])
      );

      expect(result.current[0]).toEqual(storedData);
    });

    it('should load and parse nested JSON structures', () => {
      const storedData = { user: { name: 'John', settings: { theme: 'dark' } } };
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(JSON.stringify(storedData));

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof storedData>(mockAdapter, 'json-key', { user: { name: '', settings: { theme: 'light' } } })
      );

      expect(result.current[0]).toEqual(storedData);
    });

    it('should use default value when storage returns null', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const defaultValue = { items: [] as string[] };

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof defaultValue>(mockAdapter, 'json-key', defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });

    it('should use default value when storage is unavailable', () => {
      (mockAdapter.isAvailable as ReturnType<typeof vi.fn>).mockReturnValue(false);
      const defaultValue = { data: 'default' };

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof defaultValue>(mockAdapter, 'json-key', defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });
  });

  describe('error handling', () => {
    it('should use default value when JSON parsing fails', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('invalid json {{{');
      const defaultValue = { valid: true };

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof defaultValue>(mockAdapter, 'json-key', defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });

    it('should use default value when stored value is empty string', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue('');
      const defaultValue = { valid: true };

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof defaultValue>(mockAdapter, 'json-key', defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });

    it('should warn when JSON serialization fails on update', () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const circularObj: Record<string, unknown> = { name: 'test' };
      circularObj.self = circularObj;

      const { result } = renderHook(() =>
        useJsonStorageSync<Record<string, unknown>>(mockAdapter, 'json-key', {})
      );

      act(() => {
        result.current[1](circularObj);
      });

      expect(consoleSpy).toHaveBeenCalled();
      expect(consoleSpy.mock.calls[0][0]).toContain('Failed to serialize value');

      consoleSpy.mockRestore();
    });
  });

  describe('value updates', () => {
    it('should serialize and persist object to storage', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const newValue = { id: 1, name: 'updated' };

      const { result } = renderHook(() =>
        useJsonStorageSync<typeof newValue>(mockAdapter, 'json-key', { id: 0, name: '' })
      );

      act(() => {
        result.current[1](newValue);
      });

      expect(result.current[0]).toEqual(newValue);
      expect(mockAdapter.setItem).toHaveBeenCalledWith('json-key', JSON.stringify(newValue));
    });

    it('should serialize and persist array to storage', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const newValue = ['a', 'b', 'c'];

      const { result } = renderHook(() =>
        useJsonStorageSync<string[]>(mockAdapter, 'json-key', [])
      );

      act(() => {
        result.current[1](newValue);
      });

      expect(result.current[0]).toEqual(newValue);
      expect(mockAdapter.setItem).toHaveBeenCalledWith('json-key', JSON.stringify(newValue));
    });

    it('should handle null values', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useJsonStorageSync<string | null>(mockAdapter, 'json-key', 'default')
      );

      act(() => {
        result.current[1](null);
      });

      expect(result.current[0]).toBeNull();
      expect(mockAdapter.setItem).toHaveBeenCalledWith('json-key', 'null');
    });

    it('should handle primitive values', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result } = renderHook(() =>
        useJsonStorageSync<number>(mockAdapter, 'json-key', 0)
      );

      act(() => {
        result.current[1](42);
      });

      expect(result.current[0]).toBe(42);
      expect(mockAdapter.setItem).toHaveBeenCalledWith('json-key', '42');
    });
  });

  describe('callback memoization', () => {
    it('should return stable setter function across renders', () => {
      (mockAdapter.getItem as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(() =>
        useJsonStorageSync<{ value: number }>(mockAdapter, 'json-key', { value: 0 })
      );

      const firstSetter = result.current[1];
      
      rerender();
      
      const secondSetter = result.current[1];

      expect(firstSetter).toBe(secondSetter);
    });
  });
});

// ============================================================================
// useTypedStorageSync Tests
// ============================================================================

describe('useTypedStorageSync', () => {
  let mockTypedStorage: TypedStorage<{ name: string; count: number }>;

  beforeEach(() => {
    mockTypedStorage = createMockTypedStorage<{ name: string; count: number }>();
  });

  describe('initial value loading', () => {
    it('should load initial value from typed storage', () => {
      const storedValue = { name: 'test', count: 42 };
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(storedValue);

      const { result } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, { name: '', count: 0 })
      );

      expect(result.current[0]).toEqual(storedValue);
      expect(mockTypedStorage.get).toHaveBeenCalled();
    });

    it('should use default value when typed storage returns null', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const defaultValue = { name: 'default', count: 0 };

      const { result } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });

    it('should use default value when typed storage returns undefined', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(undefined);
      const defaultValue = { name: 'default', count: 0 };

      const { result } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, defaultValue)
      );

      expect(result.current[0]).toEqual(defaultValue);
    });
  });

  describe('value updates', () => {
    it('should update state and persist to typed storage', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const newValue = { name: 'updated', count: 100 };

      const { result } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, { name: '', count: 0 })
      );

      act(() => {
        result.current[1](newValue);
      });

      expect(result.current[0]).toEqual(newValue);
      expect(mockTypedStorage.set).toHaveBeenCalledWith(newValue);
    });

    it('should handle partial updates', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);
      const initialValue = { name: 'initial', count: 0 };

      const { result } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, initialValue)
      );

      const updatedValue = { name: 'updated', count: 0 };
      act(() => {
        result.current[1](updatedValue);
      });

      expect(result.current[0]).toEqual(updatedValue);
    });

    it('should handle complex object updates', () => {
      interface ComplexType {
        nested: { value: string };
        items: number[];
      }
      
      const mockComplexStorage = createMockTypedStorage<ComplexType>();
      (mockComplexStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);
      
      const newValue: ComplexType = {
        nested: { value: 'deep' },
        items: [1, 2, 3],
      };

      const { result } = renderHook(() =>
        useTypedStorageSync(mockComplexStorage, { nested: { value: '' }, items: [] })
      );

      act(() => {
        result.current[1](newValue);
      });

      expect(result.current[0]).toEqual(newValue);
      expect(mockComplexStorage.set).toHaveBeenCalledWith(newValue);
    });
  });

  describe('callback memoization', () => {
    it('should return stable setter function across renders', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(() =>
        useTypedStorageSync(mockTypedStorage, { name: '', count: 0 })
      );

      const firstSetter = result.current[1];
      
      rerender();
      
      const secondSetter = result.current[1];

      expect(firstSetter).toBe(secondSetter);
    });

    it('should create new setter when typed storage changes', () => {
      (mockTypedStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);

      const { result, rerender } = renderHook(
        ({ storage }) => useTypedStorageSync(storage, { name: '', count: 0 }),
        { initialProps: { storage: mockTypedStorage } }
      );

      const firstSetter = result.current[1];

      const newStorage = createMockTypedStorage<{ name: string; count: number }>();
      (newStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(null);
      
      rerender({ storage: newStorage });

      const secondSetter = result.current[1];

      expect(firstSetter).not.toBe(secondSetter);
    });
  });

  describe('edge cases', () => {
    it('should handle string type', () => {
      const mockStringStorage = createMockTypedStorage<string>();
      (mockStringStorage.get as ReturnType<typeof vi.fn>).mockReturnValue('stored');

      const { result } = renderHook(() =>
        useTypedStorageSync(mockStringStorage, 'default')
      );

      expect(result.current[0]).toBe('stored');
    });

    it('should handle number type', () => {
      const mockNumberStorage = createMockTypedStorage<number>();
      (mockNumberStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(42);

      const { result } = renderHook(() =>
        useTypedStorageSync(mockNumberStorage, 0)
      );

      expect(result.current[0]).toBe(42);
    });

    it('should handle array type', () => {
      const mockArrayStorage = createMockTypedStorage<string[]>();
      (mockArrayStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(['a', 'b']);

      const { result } = renderHook(() =>
        useTypedStorageSync(mockArrayStorage, [])
      );

      expect(result.current[0]).toEqual(['a', 'b']);
    });

    it('should handle boolean type', () => {
      const mockBoolStorage = createMockTypedStorage<boolean>();
      (mockBoolStorage.get as ReturnType<typeof vi.fn>).mockReturnValue(true);

      const { result } = renderHook(() =>
        useTypedStorageSync(mockBoolStorage, false)
      );

      expect(result.current[0]).toBe(true);
    });
  });
});
