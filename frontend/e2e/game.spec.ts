import { test, expect } from '@playwright/test'

test.describe('Dixit Game', () => {
  test('creates and joins a room with 3 players', async ({ browser }) => {
    // Create 3 browser contexts for 3 players
    const player1Context = await browser.newContext()
    const player2Context = await browser.newContext()
    const player3Context = await browser.newContext()

    const player1 = await player1Context.newPage()
    const player2 = await player2Context.newPage()
    const player3 = await player3Context.newPage()

    // Player 1 creates a room
    await player1.goto('/')
    await player1.fill('input[placeholder="Enter your name"]', 'Alice')
    await player1.click('button:has-text("Continue")')
    await player1.click('button:has-text("Create New Room")')

    // Wait for room page and get the room code
    await player1.waitForURL(/\/room\/[A-Z]{4}/)
    const roomCode = await player1.locator('.font-mono.text-white').first().textContent()
    expect(roomCode).toMatch(/^[A-Z]{4}$/)

    // Player 2 joins the room
    await player2.goto('/')
    await player2.fill('input[placeholder="Enter your name"]', 'Bob')
    await player2.click('button:has-text("Continue")')
    await player2.fill('input[placeholder="Enter room code"]', roomCode!)
    await player2.click('button:has-text("Join Room")')

    // Player 2 should be in the room
    await player2.waitForURL(/\/room\/[A-Z]{4}/)

    // Player 3 joins the room
    await player3.goto('/')
    await player3.fill('input[placeholder="Enter your name"]', 'Charlie')
    await player3.click('button:has-text("Continue")')
    await player3.fill('input[placeholder="Enter room code"]', roomCode!)
    await player3.click('button:has-text("Join Room")')

    // Player 3 should be in the room
    await player3.waitForURL(/\/room\/[A-Z]{4}/)

    // Verify all players see each other in the lobby
    for (const page of [player1, player2, player3]) {
      await expect(page.locator('text=Alice')).toBeVisible()
      await expect(page.locator('text=Bob')).toBeVisible()
      await expect(page.locator('text=Charlie')).toBeVisible()
    }

    // Verify only host (Player 1) can see the Start Game button
    await expect(player1.locator('button:has-text("Start Game")')).toBeVisible()

    // Clean up
    await player1Context.close()
    await player2Context.close()
    await player3Context.close()
  })

  test('homepage displays correctly', async ({ page }) => {
    await page.goto('/')

    // Check title
    await expect(page.locator('h1:has-text("Dixit")')).toBeVisible()

    // Check name input
    await expect(page.locator('input[placeholder="Enter your name"]')).toBeVisible()

    // Check continue button
    await expect(page.locator('button:has-text("Continue")')).toBeVisible()
  })

  test('can enter name and see room options', async ({ page }) => {
    await page.goto('/')

    // Enter name
    await page.fill('input[placeholder="Enter your name"]', 'TestPlayer')
    await page.click('button:has-text("Continue")')

    // Should see room options
    await expect(page.locator('text=Welcome, TestPlayer!')).toBeVisible()
    await expect(page.locator('button:has-text("Create New Room")')).toBeVisible()
    await expect(page.locator('input[placeholder="Enter room code"]')).toBeVisible()
  })
})
