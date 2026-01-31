import { render, screen, fireEvent } from '@testing-library/react'
import Card from '@/components/Card'

describe('Card', () => {
  const mockCard = {
    id: 'card-001',
    image: '/cards/images/card-001.svg',
  }

  it('renders card image', () => {
    render(<Card card={mockCard} />)
    const img = screen.getByRole('img')
    expect(img).toHaveAttribute('src', mockCard.image)
  })

  it('handles click when onClick provided', () => {
    const handleClick = jest.fn()
    render(<Card card={mockCard} onClick={handleClick} />)

    const card = screen.getByRole('img').parentElement
    fireEvent.click(card!)

    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('does not handle click when disabled', () => {
    const handleClick = jest.fn()
    render(<Card card={mockCard} onClick={handleClick} disabled />)

    const card = screen.getByRole('img').parentElement
    fireEvent.click(card!)

    expect(handleClick).not.toHaveBeenCalled()
  })

  it('shows selected state', () => {
    render(<Card card={mockCard} selected />)
    expect(screen.getByText('✓')).toBeInTheDocument()
  })

  it('shows overlay when provided', () => {
    render(<Card card={mockCard} showOverlay="Your Card" />)
    expect(screen.getByText('Your Card')).toBeInTheDocument()
  })

  it('applies correct size classes', () => {
    const { rerender } = render(<Card card={mockCard} size="small" />)
    let container = screen.getByRole('img').parentElement
    expect(container).toHaveClass('w-24', 'h-32')

    rerender(<Card card={mockCard} size="large" />)
    container = screen.getByRole('img').parentElement
    expect(container).toHaveClass('w-40', 'h-56')
  })
})
