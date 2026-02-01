'use client'

import { useState } from 'react'

interface PromptEditorProps {
  value: string
  onChange: (value: string) => void
  onSave?: () => void
  onCancel?: () => void
  placeholder?: string
  showActions?: boolean
  disabled?: boolean
}

export default function PromptEditor({
  value,
  onChange,
  onSave,
  onCancel,
  placeholder = 'Enter prompt...',
  showActions = true,
  disabled = false,
}: PromptEditorProps) {
  const [isFocused, setIsFocused] = useState(false)

  return (
    <div className={`rounded-lg border transition-colors ${
      isFocused ? 'border-primary-500 bg-slate-700' : 'border-slate-600 bg-slate-700'
    }`}>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => setIsFocused(true)}
        onBlur={() => setIsFocused(false)}
        placeholder={placeholder}
        disabled={disabled}
        rows={4}
        className="w-full px-4 py-3 bg-transparent text-white placeholder-slate-400 focus:outline-none resize-none disabled:opacity-50"
      />

      {showActions && (onSave || onCancel) && (
        <div className="flex justify-end gap-2 px-3 pb-3">
          {onCancel && (
            <button
              onClick={onCancel}
              disabled={disabled}
              className="px-3 py-1.5 bg-slate-600 hover:bg-slate-500 text-white rounded text-sm transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
          )}
          {onSave && (
            <button
              onClick={onSave}
              disabled={disabled}
              className="px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm font-medium transition-colors disabled:opacity-50"
            >
              Save
            </button>
          )}
        </div>
      )}
    </div>
  )
}
