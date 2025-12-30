/**
 * Performance Optimization Composables
 * 
 * Vue 3 composables for performance optimization.
 */

import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { debounce, throttle } from '@/utils/performance'

// ============================================================================
// Intersection Observer for Lazy Loading
// ============================================================================

export function useIntersectionObserver(
  target: Ref<HTMLElement | undefined>,
  callback: IntersectionObserverCallback,
  options: IntersectionObserverInit = {}
) {
  let observer: IntersectionObserver | null = null

  onMounted(() => {
    if (target.value) {
      observer = new IntersectionObserver(callback, options)
      observer.observe(target.value)
    }
  })

  onUnmounted(() => {
    observer?.disconnect()
  })

  return { observer }
}

// ============================================================================
// Window Resize with Debounce
// ============================================================================

export function useWindowSize(debounceMs: number = 150) {
  const width = ref(window.innerWidth)
  const height = ref(window.innerHeight)

  const update = debounce(() => {
    width.value = window.innerWidth
    height.value = window.innerHeight
  }, debounceMs)

  onMounted(() => {
    window.addEventListener('resize', update)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', update)
  })

  return { width, height }
}

// ============================================================================
// Scroll Position with Throttle
// ============================================================================

export function useScrollPosition(throttleMs: number = 100) {
  const x = ref(0)
  const y = ref(0)

  const update = throttle(() => {
    x.value = window.scrollX
    y.value = window.scrollY
  }, throttleMs)

  onMounted(() => {
    window.addEventListener('scroll', update, { passive: true })
  })

  onUnmounted(() => {
    window.removeEventListener('scroll', update)
  })

  return { x, y }
}

// ============================================================================
// Media Query
// ============================================================================

export function useMediaQuery(query: string) {
  const matches = ref(false)
  let mediaQuery: MediaQueryList | null = null

  const update = () => {
    if (mediaQuery) {
      matches.value = mediaQuery.matches
    }
  }

  onMounted(() => {
    mediaQuery = window.matchMedia(query)
    update()
    mediaQuery.addEventListener('change', update)
  })

  onUnmounted(() => {
    mediaQuery?.removeEventListener('change', update)
  })

  return matches
}

// ============================================================================
// Async State Management
// ============================================================================

export function useAsyncState<T>(
  fn: () => Promise<T>,
  initial: T
) {
  const state = ref(initial) as Ref<T>
  const loading = ref(false)
  const error = ref<Error | null>(null)

  const execute = async () => {
    loading.value = true
    error.value = null

    try {
      state.value = await fn()
    } catch (e) {
      error.value = e as Error
    } finally {
      loading.value = false
    }
  }

  return { state, loading, error, execute }
}

// ============================================================================
// Virtual Scrolling Helper
// ============================================================================

export function useVirtualList<T>(
  items: Ref<T[]>,
  itemHeight: number,
  containerHeight: number
) {
  const scrollTop = ref(0)

  const visibleCount = Math.ceil(containerHeight / itemHeight)
  const startIndex = computed(() => Math.floor(scrollTop.value / itemHeight))
  const endIndex = computed(() => Math.min(
    startIndex.value + visibleCount + 1, // Buffer of 1
    items.value.length
  ))

  const visibleItems = computed(() =>
    items.value.slice(startIndex.value, endIndex.value)
  )

  const offsetY = computed(() => startIndex.value * itemHeight)
  const totalHeight = computed(() => items.value.length * itemHeight)

  const handleScroll = throttle((e: Event) => {
    scrollTop.value = (e.target as HTMLElement).scrollTop
  }, 16) // ~60fps

  return {
    visibleItems,
    offsetY,
    totalHeight,
    handleScroll
  }
}
