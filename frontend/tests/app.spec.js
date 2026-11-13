import { expect, test } from '@playwright/test'

const results = {
  query: 'Hades',
  usd_rate: 1370,
  official_rate: 1200,
  card_rate: 1452,
  tax_rate: 0.21,
	rates_updated_at: '2026-08-28T18:29:00Z',
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

const deals = {
  updated_at: '2026-08-28T18:30:00Z',
  official_rate: 1200,
  card_rate: 1452,
  tax_rate: 0.21,
  rates_updated_at: '2026-08-28T18:29:00Z',
  featured: [
	{ id: 'deal-1', title: 'Balatro', image_url: '', store_name: 'Epic Games Store', price: 6.49, regular: 14.99, currency: 'USD', discount_percent: 57, url: 'https://store.epicgames.com/', score: 84, reasons: ['96% positivas', 'Mínimo histórico'] },
	{ id: 'deal-1-steam', title: 'Balatro', image_url: '', store_name: 'Steam', price: 7500, regular: 15000, currency: 'ARS', discount_percent: 50, url: 'https://store.steampowered.com/', score: 80, reasons: ['Popular'] },
  ],
  free: [{ id: 'deal-2', title: 'Free Favorite', image_url: '', store_name: 'Steam', price: 0, regular: 20, currency: 'USD', discount_percent: 100, url: 'https://store.steampowered.com/', score: 72, reasons: ['Gratis ahora'] }],
  discounts: [{ id: 'deal-3', title: 'Deep Discount', image_url: '', store_name: 'Microsoft Store', price: 2, regular: 20, currency: 'USD', discount_percent: 90, url: 'https://www.microsoft.com/', score: 50, reasons: [] }],
}
const discover = {
  updated_at: '2026-08-28T18:31:00Z',
  popular: [{ id: '730', title: 'Counter-Strike 2', image_url: '', steam_url: 'https://store.steampowered.com/app/730/', rank: 1 }],
  most_played: [{ id: '570', title: 'Dota 2', image_url: '', steam_url: 'https://store.steampowered.com/app/570/', rank: 2, players: 800000 }],
}

async function mockAPI(page, overrides = {}) {
  await page.route('**/api/health**', (route) => route.fulfill({ json: { status: 'ok', service: 'pricepulse-api', version: 'test123', checked_at: '2026-08-28T18:32:00Z', providers: ['Steam', 'IsThereAnyDeal', 'Bluelytics'] } }))
  await page.route('**/api/deals**', (route) => route.fulfill({ json: overrides.deals || deals }))
  await page.route('**/api/discover**', (route) => route.fulfill({ json: overrides.discover || discover }))
  await page.route('**/api/search**', (route) => route.fulfill({ json: overrides.results || results }))
}

test('price reports include the selected game and store', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await page.getByRole('button', { name: '¿Precio incorrecto?' }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('Balatro', { exact: true })).toBeVisible()
  await expect(dialog.getByText(/Epic Games Store/)).toBeVisible()
  await expect(dialog.getByText(/reporte público en GitHub/i)).toBeVisible()
  await dialog.getByLabel('Mensaje').fill('En la tienda figura otro precio.')
  await page.evaluate(() => { window.open = (url) => { window.__pricePulseReport = url } })
  await dialog.getByRole('button', { name: /preparar reporte/i }).click()
  const reportURL = await page.evaluate(() => window.__pricePulseReport)
  const reportBody = new URL(reportURL).searchParams.get('body')
  expect(reportBody).toContain('Balatro')
  expect(reportBody).toContain('Epic Games Store')
})

test('footer exposes service status and changelog', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByText('API online')).toBeVisible()
  await page.getByRole('button', { name: 'Novedades' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByText('Ofertas top tier')).toBeVisible()
})

test('landing explains the comparison purpose at a glance', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '¿Dónde está más barato?' })).toBeVisible()
  await expect(page.getByText('Steam', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('Epic', { exact: true }).first()).toBeVisible()
  await expect(page.getByText('Microsoft', { exact: true }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Comparar precios', exact: true })).toBeVisible()
})

test('search is shareable and shows only the real regional Steam price', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.locator('#results-title')).toHaveText('Hades')
  await expect(page.getByText('$\u00a08.799').first()).toBeVisible()
  await expect(page.getByText('regional · ARS')).toBeVisible()
  await expect(page.getByText('ARS + IMP · USD × 1.452 ARS/USD (oficial 1.200 ARS/USD + IVA 21%) = total final. ARS regional no se reconvierte.')).toBeVisible()
  await expect(page.getByText('GOG', { exact: true })).toBeVisible()
  await expect(page.getByText('$\u00a014.505').first()).toBeVisible()
})

test('currency controls keep USD prices in dollars and ARS prices with taxes', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/?q=Hades&steam=regional')
  await expect(page.getByRole('button', { name: 'ARS + IMP', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'USD', exact: true }).click()
  await expect(page.getByText(/US\$\s?9,99/).first()).toBeVisible()
  await expect(page.getByText('USD · Sólo precios publicados en USD para Argentina. No se convierte ningún precio.')).toBeVisible()
  await expect(page.locator('.game-card').getByText('Steam', { exact: true })).not.toBeVisible()
})

test('Steam rankings open a price comparison', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Populares en Steam' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Más jugados en Steam' })).toBeVisible()
  await page.getByRole('button', { name: /comparar precios de counter-strike 2/i }).click()
  await expect(page.getByRole('heading', { name: 'Counter-Strike 2', exact: true })).toBeVisible()
})

test('offers switch between curated, free and biggest discount views', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Balatro' })).toBeVisible()
  await page.getByRole('tab', { name: /gratis ahora/i }).click()
  await expect(page.getByRole('button', { name: 'Free Favorite' })).toBeVisible()
  await page.getByRole('tab', { name: /más descuento/i }).click()
  await expect(page.getByRole('button', { name: 'Deep Discount' })).toBeVisible()
})

test('offers obey ARS plus tax and USD modes', async ({ page }) => {
	await mockAPI(page)
	await page.goto('/')
	const balatro = page.locator('.deal-card').filter({ hasText: 'Balatro' }).first()
	await expect(balatro).toContainText('9.423')
	await expect(balatro).toContainText('total con IVA 21%')
	await page.getByRole('button', { name: 'USD', exact: true }).click()
	await expect(page.locator('.deal-card').filter({ hasText: 'Balatro' }).first()).toContainText(/US\$\s?6,49/)
})

test('store filters preserve a regional variant of the same game', async ({ page }) => {
	await mockAPI(page)
	await page.goto('/')
	await page.getByRole('button', { name: 'Steam', exact: true }).click()
	const balatro = page.locator('.deal-card').filter({ hasText: 'Balatro' })
	await expect(balatro).toHaveCount(1)
	await expect(balatro).toContainText('7.500')
	await expect(balatro).toContainText('precio regional publicado')
})

test('a free search price remains visible and wins over a paid store', async ({ page }) => {
	const freeResults = { ...results, results: [{ ...results.results[0], best_deal: { store_name: 'Steam', price: 0, regular: 10, currency: 'USD', url: 'https://store.steampowered.com/' }, prices: [{ store_name: 'Steam', price: 0, regular: 10, currency: 'USD', url: 'https://store.steampowered.com/' }, { store_name: 'Epic Games Store', price: 10, regular: 20, currency: 'USD', url: 'https://store.epicgames.com/' }] }] }
	await mockAPI(page, { results: freeResults })
	await page.goto('/?q=Hades')
	await expect(page.locator('.best-summary')).toContainText('Gratis')
	await expect(page.getByText('Gratis', { exact: true }).first()).toBeVisible()
})

test('an empty successful search has an explicit result state', async ({ page }) => {
	await mockAPI(page, { results: { ...results, results: [] } })
	await page.goto('/?q=JuegoInexistente')
	await expect(page.getByRole('heading', { name: 'No encontramos “JuegoInexistente”' })).toBeVisible()
	await expect(page.getByRole('heading', { name: 'Populares en Steam' })).not.toBeVisible()
})

test('status shows independent freshness timestamps', async ({ page }) => {
	await mockAPI(page)
	await page.goto('/?q=Hades')
	await page.getByRole('button', { name: 'Estado' }).click()
	const dialog = page.getByRole('dialog')
	await expect(dialog.getByText('Búsqueda', { exact: true })).toBeVisible()
	await expect(dialog.getByText('Ofertas', { exact: true })).toBeVisible()
	await expect(dialog.getByText('Rankings', { exact: true })).toBeVisible()
	await expect(dialog.getByText('Cotización', { exact: true })).toBeVisible()
})

test('mobile layout has no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockAPI(page)
  await page.goto('/?q=Hades')
  await expect(page.locator('#results-title')).toHaveText('Hades')
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
})

test('a slow cold start explains that the server is waking up', async ({ page }) => {
  await mockAPI(page)
  await page.route('**/api/deals**', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 5500))
    await route.fulfill({ json: deals })
  })
  await page.goto('/')
  const notice = page.getByText(/despertando el servidor/i)
  await expect(notice).toBeVisible({ timeout: 6000 })
  await expect(notice).toBeHidden({ timeout: 8000 })
  await expect(page.getByText('Balatro').first()).toBeVisible()
})

test('rankings show the Steam price and the cheapest store', async ({ page }) => {
  const priced = {
    ...discover,
    official_rate: 1200,
    card_rate: 1452,
    tax_rate: 0.21,
    popular: [
      { id: '1145360', title: 'Hades', image_url: 'https://images.invalid/hades.jpg', steam_url: '', rank: 1, steam_price: { store_name: 'Steam', price: 10, currency: 'USD' }, best_price: { store_name: 'GOG', price: 5, currency: 'USD' } },
      { id: '730', title: 'Counter-Strike 2', image_url: '', steam_url: '', rank: 2, steam_price: { store_name: 'Steam', price: 0, currency: 'USD' }, best_price: { store_name: 'Steam', price: 0, currency: 'USD' } },
    ],
  }
  await mockAPI(page, { discover: priced })
  await page.goto('/')
  const hades = page.getByRole('button', { name: 'Comparar precios de Hades' })
  await expect(hades).toContainText('Steam')
  await expect(hades).toContainText(/14\.520/)
  await expect(hades).toContainText('Más barato · GOG')
  await expect(hades).toContainText(/7\.260/)
  await expect(hades.locator('.image-fallback')).toBeVisible()
  const cs = page.getByRole('button', { name: 'Comparar precios de Counter-Strike 2' })
  await expect(cs).toContainText('Gratis')
  await expect(cs).toContainText('Steam es la opción más barata')
})

test('footer credits the data sources and the author', async ({ page }) => {
  await mockAPI(page)
  await page.goto('/')
  const footer = page.locator('footer')
  await expect(footer.getByRole('link', { name: 'IsThereAnyDeal' })).toHaveAttribute('href', 'https://isthereanydeal.com/')
  await expect(footer.getByRole('link', { name: 'LinkedIn' })).toHaveAttribute('href', /linkedin\.com\/in\//)
  await expect(footer.getByRole('link', { name: 'GitHub' })).toHaveAttribute('href', 'https://github.com/LorenteFacundo')
  await expect(footer.getByRole('link', { name: /Bot de Discord/ })).toHaveAttribute('href', /discord\.com\/oauth2\/authorize/)
})

test('USD mode never shows peso prices as dollars in the rankings', async ({ page }) => {
  const priced = {
    ...discover,
    official_rate: 1200,
    card_rate: 1452,
    tax_rate: 0.21,
    popular: [{ id: '1', title: 'Peso Game', image_url: '', steam_url: '', rank: 1, steam_price: { store_name: 'Steam', price: 12268.52, currency: 'ARS' }, best_price: { store_name: 'Steam', price: 12268.52, currency: 'ARS' } }],
  }
  await mockAPI(page, { discover: priced })
  await page.goto('/')
  await page.getByRole('button', { name: 'USD', exact: true }).click()
  const card = page.getByRole('button', { name: 'Comparar precios de Peso Game' })
  await expect(card).toContainText(/\$\s?12\.269/)
  await expect(card).not.toContainText('US$')
  await expect(card).not.toContainText('12,268')
})

test('a suspiciously low price is shown with a warning but never wins', async ({ page }) => {
  const suspicious = {
    ...results,
    results: [{
      id: 'hades', title: 'Hades', image_url: '',
      prices: [
        { store_name: 'Microsoft Store', price: 359, regular: 359, currency: 'ARS', discount_percent: 0, url: 'https://ms.example/hades', warning: 'Precio inusualmente bajo: puede ser otro producto. Verificalo en la tienda.' },
        { store_name: 'Steam', price: 7700, regular: 7700, currency: 'ARS', discount_percent: 0, url: 'https://steam.example/hades', is_regional: true },
        { store_name: 'Epic Games Store', price: 7800, regular: 7800, currency: 'ARS', discount_percent: 0, url: 'https://epic.example/hades' },
      ],
    }],
  }
  await mockAPI(page, { results: suspicious })
  await page.goto('/?q=Hades')
  await expect(page.locator('.best-summary strong')).toContainText(/7\.700/)
  await expect(page.getByText(/inusualmente bajo/)).toBeVisible()
  await expect(page.locator('.price-row.is-best')).toContainText('Steam')
})
