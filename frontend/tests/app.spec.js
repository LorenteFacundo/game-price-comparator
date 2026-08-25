import { expect, test } from '@playwright/test'

const results = {
  query: 'Hades',
  usd_rate: 1370,
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

const deals = { deals: [{ id: 'deal-1', title: 'Balatro', image_url: '', store_name: 'Steam', price: 6.49, regular: 14.99, currency: 'USD', discount_percent: 57, url: 'https://store.steampowered.com/', is_near_low: true }] }

async function mockAPI(page) {
  await page.route('**/api/deals**', (route) => route.fulfill({ json: deals }))
  await page.route('**/api/search**', (route) => route.fulfill({ json: results }))
}

test('search is shareable and displays correctly converted offers', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.getByRole('heading', { name: /coincidencias para “hades”/i })).toBeVisible()
  await expect(page.getByText('$\u00a08.799').first()).toBeVisible()
  await expect(page.getByText('GOG', { exact: true })).toBeVisible()
})

test('mobile layout has no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockAPI(page)
  await page.goto('/?q=Hades')
  await expect(page.getByRole('heading', { name: /coincidencias para “hades”/i })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
})
