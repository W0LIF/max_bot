export type MaxUser = {
  id: number
  first_name?: string
  name?: string
}

export type MaxWebApp = {
  initData: string
  initDataUnsafe?: { user?: MaxUser }
  ready?: () => void
  expand?: () => void
  close?: () => void
  shareMessage?: (url: string) => void
}

declare global {
  interface Window {
    WebApp?: MaxWebApp
  }
}

export function initWebApp(): void {
  if (!window.WebApp?.initData) return
  window.WebApp?.ready?.()
  window.WebApp?.expand?.()
}

export function getInitData(): string {
  return window.WebApp?.initData?.trim() ?? ''
}

export function isMaxWebApp(): boolean {
  return Boolean(window.WebApp)
}

export function closeWebApp(): void {
  window.WebApp?.close?.()
}

export function shareWebApp(url: string): boolean {
  if (!window.WebApp?.shareMessage) return false
  window.WebApp.shareMessage(url)
  return true
}