'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

const navItems = [
  { href: '/', label: 'Home', icon: '🏠' },
  { href: '/concepts', label: 'Concepts', icon: '💡' },
  { href: '/generate', label: 'Generate', icon: '🎨' },
  { href: '/review', label: 'Review', icon: '✅' },
  { href: '/export', label: 'Export', icon: '📦' },
]

export default function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="fixed left-0 top-0 h-full w-64 bg-slate-900 border-r border-slate-700 p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">Card Set Studio</h1>
        <p className="text-sm text-slate-400 mt-1">AI-powered Dixit cards</p>
      </div>

      <nav className="space-y-2">
        {navItems.map((item) => {
          const isActive = pathname === item.href ||
            (item.href !== '/' && pathname.startsWith(item.href))

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                isActive
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-300 hover:bg-slate-800 hover:text-white'
              }`}
            >
              <span className="text-xl">{item.icon}</span>
              <span className="font-medium">{item.label}</span>
            </Link>
          )
        })}
      </nav>

      <div className="absolute bottom-6 left-6 right-6">
        <div className="bg-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-semibold text-slate-300 mb-2">Services</h3>
          <div className="space-y-2 text-xs">
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Ollama</span>
              <span className="w-2 h-2 rounded-full bg-green-500"></span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Stable Diffusion</span>
              <span className="w-2 h-2 rounded-full bg-green-500"></span>
            </div>
          </div>
        </div>
      </div>
    </aside>
  )
}
