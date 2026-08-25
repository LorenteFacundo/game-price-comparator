export default function SearchPanel({ query, onQueryChange, onSearch, loading, currency, onCurrencyChange, steamMode, onSteamModeChange }) {
  function submit(event) {
    event.preventDefault()
    if (query.trim()) onSearch(query.trim())
  }

  return (
    <form className="search-panel" onSubmit={submit}>
      <div className="search-copy">
        <span className="eyebrow">Radar de precios</span>
        <h1>Encontrá el precio justo.<br /><em>Sin vueltas.</em></h1>
        <p>Comparamos tiendas confiables, conservamos la moneda original y resaltamos las ofertas que realmente importan.</p>
      </div>

      <div className="search-controls">
        <label className="search-field" htmlFor="game-search">
          <span>¿Qué querés jugar?</span>
          <input
            id="game-search"
            name="game"
            type="search"
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Ej.: Hades, Clair Obscur, Elden Ring…"
            autoComplete="off"
            spellCheck="false"
            disabled={loading}
          />
        </label>
        <button className="search-button" type="submit" disabled={loading || !query.trim()}>
          <span>{loading ? 'Buscando…' : 'Buscar ofertas'}</span>
          <span aria-hidden="true">↗</span>
        </button>
      </div>

      <div className="search-options" aria-label="Preferencias de búsqueda">
        <fieldset>
          <legend>Mostrar en</legend>
          <div className="segmented-control">
            {['ARS', 'USD'].map((option) => (
              <button key={option} className={currency === option ? 'is-active' : ''} type="button" aria-pressed={currency === option} onClick={() => onCurrencyChange(option)}>{option}</button>
            ))}
          </div>
        </fieldset>
        <fieldset>
          <legend>Precio de Steam</legend>
          <div className="segmented-control">
            <button className={steamMode === 'regional' ? 'is-active' : ''} type="button" aria-pressed={steamMode === 'regional'} onClick={() => onSteamModeChange('regional')}>Argentina</button>
            <button className={steamMode === 'global' ? 'is-active' : ''} type="button" aria-pressed={steamMode === 'global'} onClick={() => onSteamModeChange('global')}>Global</button>
          </div>
        </fieldset>
      </div>
    </form>
  )
}
