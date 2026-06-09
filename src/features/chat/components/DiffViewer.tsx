"use client"

import dynamic from "next/dynamic"

// react-diff-viewer-continued nggak SSR-safe (refer ke window). Load di client only.
const ReactDiffViewer = dynamic(() => import("react-diff-viewer-continued"), {
  ssr: false,
})

interface DiffViewerProps {
  oldValue: string
  newValue: string
  splitView?: boolean
  oldTitle?: string
  newTitle?: string
}

export function DiffViewer({
  oldValue,
  newValue,
  splitView = false,
  oldTitle = "Before",
  newTitle = "After",
}: DiffViewerProps) {
  return (
    <div className="my-2 overflow-hidden rounded-lg border border-border text-xs">
      <ReactDiffViewer
        oldValue={oldValue}
        newValue={newValue}
        splitView={splitView}
        leftTitle={oldTitle}
        rightTitle={newTitle}
        compareMethod={"diffWords" as never}
        useDarkTheme
      />
    </div>
  )
}
