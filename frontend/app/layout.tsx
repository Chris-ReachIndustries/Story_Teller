import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { GameProvider } from '@/lib/gameContext'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'Dixit Online',
  description: 'A multiplayer online Dixit-style game',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <GameProvider>
          <main className="min-h-screen">
            {children}
          </main>
        </GameProvider>
      </body>
    </html>
  )
}
