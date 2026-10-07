// Responsive layout store for tiled/narrow windows
// Three modes: full (>1024px), medium (768-1024px), narrow (<768px)

import { isSidebarCollapsed, toggleSidebarCollapsed } from './uiState.svelte'

export type LayoutMode = 'full' | 'medium' | 'narrow'
export type ResponsiveView = 'default' | 'viewer' | 'sidebar'

let layoutMode = $state<LayoutMode>('full')
let responsiveView = $state<ResponsiveView>('default')

let narrowMql: MediaQueryList | null = null
let mediumMql: MediaQueryList | null = null

function updateMode() {
  if (!narrowMql || !mediumMql) return

  if (narrowMql.matches) {
    layoutMode = 'narrow'
  } else if (mediumMql.matches) {
    layoutMode = 'medium'
  } else {
    layoutMode = 'full'
  }

  // Reset overlays when entering full mode
  if (layoutMode === 'full') {
    responsiveView = 'default'
  }
}

export function initLayout() {
  narrowMql = window.matchMedia('(max-width: 767px)')
  mediumMql = window.matchMedia('(max-width: 1024px)')

  updateMode()

  narrowMql.addEventListener('change', updateMode)
  mediumMql.addEventListener('change', updateMode)
}

export function getLayoutMode(): LayoutMode {
  return layoutMode
}

export function getResponsiveView(): ResponsiveView {
  return responsiveView
}

export function isResponsive(): boolean {
  return layoutMode !== 'full'
}

export function showViewer() {
  if (layoutMode === 'full') return
  responsiveView = 'viewer'
}

export function hideViewer() {
  if (layoutMode === 'full') return
  responsiveView = 'default'
}

export function showSidebar() {
  if (layoutMode !== 'narrow') return
  responsiveView = 'sidebar'
}

export function hideSidebar() {
  if (layoutMode !== 'narrow') return
  if (responsiveView === 'sidebar') {
    responsiveView = 'default'
  }
}

// True when a view's left sidebar (default: the active view's) is collapsed.
// Narrow mode ignores the collapse flag: there the sidebar is always an
// on-demand overlay.
export function isSidebarHidden(view?: string): boolean {
  return layoutMode !== 'narrow' && isSidebarCollapsed(view)
}

// Single toggle action for the active view's sidebar: opens/closes the
// overlay in narrow mode, otherwise flips the persisted per-view collapse flag.
// App moves pane focus off a sidebar once it is hidden.
export function toggleActiveSidebar() {
  if (layoutMode !== 'narrow') {
    toggleSidebarCollapsed()
  } else if (responsiveView === 'sidebar') {
    hideSidebar()
  } else {
    showSidebar()
  }
}
