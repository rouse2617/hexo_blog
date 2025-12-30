import { ref, watch, onUnmounted } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'

const THEME_STORAGE_KEY = 'aiops-theme'

const currentTheme = ref<ThemeMode>('auto')
const systemTheme = ref<'light' | 'dark'>('light')
const appliedTheme = ref<'light' | 'dark'>('light')

// Media query for system theme preference
let mediaQuery: MediaQueryList | null = null

function updateSystemTheme() {
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    systemTheme.value = 'dark'
  } else {
    systemTheme.value = 'light'
  }
}

function applyTheme(mode: ThemeMode) {
  const effectiveTheme = mode === 'auto' ? systemTheme.value : mode
  appliedTheme.value = effectiveTheme

  document.documentElement.classList.remove('light', 'dark')
  document.documentElement.classList.add(effectiveTheme)

  // Update Element Plus theme
  if (effectiveTheme === 'dark') {
    document.documentElement.setAttribute('data-theme', 'dark')
  } else {
    document.documentElement.setAttribute('data-theme', 'light')
  }
}

export function useTheme() {
  const isDark = ref(false)
  const isLight = ref(true)

  // Initialize theme
  function initialize() {
    // Load saved theme from localStorage
    const savedTheme = localStorage.getItem(THEME_STORAGE_KEY) as ThemeMode | null
    if (savedTheme && ['light', 'dark', 'auto'].includes(savedTheme)) {
      currentTheme.value = savedTheme
    }

    // Setup system theme detection
    updateSystemTheme()

    if (window.matchMedia) {
      mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', handleSystemThemeChange)
    }

    // Apply initial theme
    applyTheme(currentTheme.value)
    updateComputedValues()
  }

  function handleSystemThemeChange() {
    updateSystemTheme()
    if (currentTheme.value === 'auto') {
      applyTheme('auto')
      updateComputedValues()
    }
  }

  function updateComputedValues() {
    isDark.value = appliedTheme.value === 'dark'
    isLight.value = appliedTheme.value === 'light'
  }

  function setTheme(mode: ThemeMode) {
    currentTheme.value = mode
    localStorage.setItem(THEME_STORAGE_KEY, mode)
    applyTheme(mode)
    updateComputedValues()
  }

  function toggleTheme() {
    if (currentTheme.value === 'light') {
      setTheme('dark')
    } else if (currentTheme.value === 'dark') {
      setTheme('auto')
    } else {
      setTheme('light')
    }
  }

  function cycleTheme() {
    if (currentTheme.value === 'light') {
      setTheme('dark')
    } else if (currentTheme.value === 'dark') {
      setTheme('auto')
    } else {
      setTheme('light')
    }
  }

  // Cleanup
  onUnmounted(() => {
    if (mediaQuery) {
      mediaQuery.removeEventListener('change', handleSystemThemeChange)
    }
  })

  // Watch for theme changes
  watch(currentTheme, (newMode) => {
    applyTheme(newMode)
    updateComputedValues()
  })

  return {
    currentTheme,
    appliedTheme,
    systemTheme,
    isDark,
    isLight,
    setTheme,
    toggleTheme,
    cycleTheme,
    initialize
  }
}

// Auto-initialize on import
let initialized = false
export function ensureThemeInitialized() {
  if (!initialized) {
    const { initialize } = useTheme()
    initialize()
    initialized = true
  }
}
