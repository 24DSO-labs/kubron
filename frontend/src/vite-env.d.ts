// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 'web' — сборка для kubron-server (`vite build --mode web`); не задано — десктоп (Wails) */
  readonly VITE_TARGET?: 'web'
}

declare module '*.vue' {
    import type {DefineComponent} from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}

// Declare types for external modules that might not have proper type definitions
declare module 'xterm' {
  export * from 'xterm/xterm'
}

declare module 'xterm-addon-fit' {
  export * from 'xterm-addon-fit/lib/xterm-addon-fit'
}

declare module 'monaco-editor' {
  export * from 'monaco-editor/esm/vs/editor/editor.api'
}
