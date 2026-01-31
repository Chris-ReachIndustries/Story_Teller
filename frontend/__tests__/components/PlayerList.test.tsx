import { render, screen } from '@testing-library/react'
import PlayerList from '@/components/PlayerList'
import { PlayerInfo } from '@/lib/types'

describe('PlayerList', () => {
  const mockPlayers: PlayerInfo[] = [
    { id: '1', name: 'Alice', playerType: 'human', connected: true, score: 10 },
    { id: '2', name: 'Bob', playerType: 'human', connected: true, score: 5 },
    { id: '3', name: 'Charlie', playerType: 'human', connected: false, score: 3 },
  ]

  it('renders all players', () => {
    render(<PlayerList players={mockPlayers} />)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('Charlie')).toBeInTheDocument()
  })

  it('shows host badge for host player', () => {
    render(<PlayerList players={mockPlayers} hostId="1" />)
    expect(screen.getByText('Host')).toBeInTheDocument()
  })

  it('shows storyteller badge for storyteller', () => {
    render(<PlayerList players={mockPlayers} storytellerId="2" />)
    expect(screen.getByText('Storyteller')).toBeInTheDocument()
  })

  it('shows scores when showScores is true', () => {
    render(<PlayerList players={mockPlayers} showScores />)
    expect(screen.getByText('10')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('shows submission status when showStatus is true', () => {
    const playersWithStatus: PlayerInfo[] = [
      { id: '1', name: 'Alice', playerType: 'human', connected: true, score: 0, hasSubmitted: true },
      { id: '2', name: 'Bob', playerType: 'human', connected: true, score: 0, hasVoted: true },
    ]
    render(<PlayerList players={playersWithStatus} showStatus />)
    expect(screen.getByText('✓ Submitted')).toBeInTheDocument()
    expect(screen.getByText('✓ Voted')).toBeInTheDocument()
  })

  it('shows bot badge for bot players', () => {
    const playersWithBot: PlayerInfo[] = [
      { id: '1', name: 'Alice', playerType: 'human', connected: true, score: 10 },
      { id: '2', name: 'Bot Aurora', playerType: 'bot', connected: true, score: 5 },
    ]
    render(<PlayerList players={playersWithBot} />)
    expect(screen.getByText('Bot')).toBeInTheDocument()
  })

  it('does not show bot badge for human players', () => {
    const humanPlayers: PlayerInfo[] = [
      { id: '1', name: 'Alice', playerType: 'human', connected: true, score: 10 },
      { id: '2', name: 'Bob', playerType: 'human', connected: true, score: 5 },
    ]
    render(<PlayerList players={humanPlayers} />)
    expect(screen.queryByText('Bot')).not.toBeInTheDocument()
  })

  it('applies disconnected styling to offline players', () => {
    render(<PlayerList players={mockPlayers} />)
    const charlieItem = screen.getByText('Charlie').closest('li')
    expect(charlieItem).toHaveClass('opacity-50')
  })
})
