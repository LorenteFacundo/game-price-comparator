import { expect, test } from '@playwright/test'

const results = {
  query: 'Hades',
  usd_rate: 1370,
  official_rate: 1200,
  tax_rate: 0.21,
  warnings: [],
  results: [{
    id: 'hades', title: 'Hades', image_url: 'https://cdn.cloudflare.steamstatic.com/steam/apps/1145360/header.jpg',
    best_deal: { store_name: 'Steam', price: 8799, regular: 17599, currency: 'ARS', discount_percent: 50, url: 'https://store.steampowered.com/app/1145360/', on_sale: true, is_regional: true },
    prices: [
      { store_name: 'Steam', price: 8799, regular: 17599, currency: 'ARS', discount_percent: 50, url: 'https://store.steampowered.com/app/1145360/', on_sale: true, is_regional: true },
      { store_name: 'GOG', price: 9.99, regular: 24.99, currency: 'USD', discount_percent: 60, url: 'https://www.gog.com/game/hades', on_sale: true, is_regional: false },
    ],
  }],
}

const deals = { deals: [{ id: 'deal-1', title: 'Balatro', image_url: '', store_name: 'Epic Games Store', price: 6.49, regular: 14.99, currency: 'USD', discount_percent: 57, url: 'https://store.epicgames.com/', is_near_low: true }] }
const discover = {
  popular: [{ id: '730', title: 'Counter-Strike 2', image_url: '', steam_url: 'https://store.steampowered.com/app/730/', rank: 1 }],
  most_played: [{ id: '570', title: 'Dota 2', image_url: '', steam_url: 'https://store.steampowered.com/app/570/', rank: 2, players: 800000 }],
}

async function mockAPI(page) {
  await page.route('**/api/deals**', (route) => route.fulfill({ json: deals }))
  await page.route('**/api/discover**', (route) => route.fulfill({ json: discover }))
  await page.route('**/api/search**', (route) => route.fulfill({ json: results }))
}

test('search is shareable and shows only the real regional Steam price', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  await expect(page.getByText('$\u00a08.799').first()).toBeVisible()
  await expect(page.getByText('regional · ARS')).toBeVisible()
  await expect(page.getByText('USD: final estimado con IVA 21% · ARS: precio publicado por tienda')).toBeVisible()
  await expect(page.getByText('GOG', { exact: true })).toBeVisible()
})

test('Steam rankings open a price comparison', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Populares en Steam' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Más jugados en Steam' })).toBeVisible()
  await page.getByRole('button', { name: /comparar precios de counter-strike 2/i }).click()
  await expect(page.getByRole('heading', { name: 'Counter-Strike 2', exact: true })).toBeVisible()
})

test('mobile layout has no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockAPI(page)
  await page.goto('/?q=Hades')
  await expect(page.getByRole('heading', { name: 'Hades', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
})
