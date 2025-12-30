/**
 * Performance Optimization Utilities
 * 
 * Provides utilities for debouncing, throttling, memoization,
 * and lazy loading to improve Vue 3 application performance.
 */

// ============================================================================
// Type Definitions
// ============================================================================

export type AnyFunction = (...args: unknown[]) => unknown

// ============================================================================
// Debounce - Delays function execution until after wait milliseconds
// ============================================================================

export function debounce<T extends AnyFunction>(
  fn: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return function (this: unknown, ...args: Parameters<T>) {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
    }

    timeoutId = setTimeout(() => {
      fn.apply(this, args)
      timeoutId = null
    }, wait)
  }
}

// ============================================================================
// Throttle - Limits function execution to once every wait milliseconds
// ============================================================================

export function throttle<T extends AnyFunction>(
  fn: T,
  wait: number
): (...args: Parameters<T>) => void {
  let lastTime = 0
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return function (this: unknown, ...args: Parameters<T>) {
    const now = Date.now()
    const remaining = wait - (now - lastTime)

    if (remaining <= 0) {
      if (timeoutId !== null) {
        clearTimeout(timeoutId)
        timeoutId = null
      }
      lastTime = now
      fn.apply(this, args)
    } else if (timeoutId === null) {
      timeoutId = setTimeout(() => {
        lastTime = Date.now()
        timeoutId = null
        fn.apply(this, args)
      }, remaining)
    }
  }
}

// ============================================================================
// Memoize - Caches function results based on arguments
// ============================================================================()

export function memoize<T extends AnyFunction>(
  fn: T,
  keyGenerator?: (...args: Parameters<T>) => string
): T {
  const cache = new Map<string, ReturnType<T>>()

  return ((...args: Parameters<T>) => {
    const key = keyGenerator
      ? keyGenerator(...args)
      : JSON.stringify(args)

    if (cache.has(key)) {
      return cache.get(key)!
    }

    const result = fn(...args) as ReturnType<T>
    cache.set(key, result)
    return result
  }) as T
}

// ============================================================================
// Lazy ref - Creates a ref that initializes lazily
// ============================================================================

import { ref, type Ref } from 'vue'

export function lazyRef<T>(factory: () => T): Ref<T> {
  const value = ref<T>() as Ref<T>
  let initialized = false

  return new Proxy(value, {
    get(target, prop) {
      if (!initialized) {
        target.value = factory()
        initialized = true
      }
      return target[prop as keyof Ref<T>]
    },
    set(target, prop, newValue) {
      if (!initialized) {
        target.value = factory()
        initialized = true
      }
      target[prop as keyof Ref<T>] = newValue
      return true
    }
  })
}

// ============================================================================
// Batch updates - Batches multiple updates into a single tick
// ============================================================================()

let batchUpdatePending = false
const batchUpdateQueue: Array<() => void> = []

export function batchUpdate(fn: () => void): void {
  batchUpdateQueue.push(fn)

  if (!batchUpdatePending) {
    batchUpdatePending = true
    Promise.resolve().then(() => {
      const queue = batchUpdateQueue.splice(0)
      batchUpdatePending = false
      queue.forEach(fn => fn())
    })
  }
}

// ============================================================================
// Performance measurement
// ============================================================================()

export class PerformanceMeasure {
  private marks: Map<string, number> = new Map()

  mark(name: string): void {
    this.marks.set(name, performance.now())
  }

  measure(startMark: string, endMark?: string): number {
    const start = this.marks.get(startMark)
    if (start === undefined) {
      console.warn(`Mark "${startMark}" not found`)
      return 0
    }

    const end = endMark
      ? this.marks.get(endMark) ?? performance.now()
      : performance.now()

    return end - start
  }

  clear(): void {
    this.marks.clear()
  }
}

// ============================================================================
// Async queue - Limits concurrent async operations
// ============================================================================()

export class AsyncQueue<T> {
  private queue: Array<() => Promise<T>> = []
  private activeCount = 0

  constructor(private concurrency: number = 1) {}

  async add<R>(fn: () => Promise<R>): Promise<R> {
    return new Promise((resolve, reject) => {
      this.queue.push(async () => {
        try {
          const result = await fn()
          resolve(result as R)
        } catch (error) {
          reject(error)
        }
      })
      this.process()
    })
  }

  private async process(): Promise<void> {
    if (this.activeCount >= this.concurrency || this.queue.length === 0) {
      return
    }

    this.activeCount++
    const fn = this.queue.shift()!

    try {
      await fn()
    } finally {
      this.activeCount--
      this.process()
    }
  }
}
