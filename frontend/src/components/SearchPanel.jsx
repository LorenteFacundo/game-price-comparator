export default function SearchPanel({ query, onQueryChange, onSearch, loading, currency, onCurrencyChange }) {
  function submit(event) {
    event.preventDefault()
    if (query.trim()) onSearch(query.trim())
  }

  return (
    <form className="search-panel" onSubmit={submit}>
      <div className="search-copy"><h1>Buscá un juego</h1></div>

      <div className="search-controls">
        <label className="search-field" htmlFor="game-search">
          <span>Juego</span>
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
          <span>{loading ? 'Buscando…' : 'Buscar'}</span>
          <span aria-hidden="true">↗</span>
        </button>
      </div>

      <div className="search-options" aria-label="Preferencias de búsqueda">
        <fieldset>
          <legend>Moneda</legend>
          <div className="segmented-control">
            {[{ value: 'ARS', label: 'ARS + IMP' }, { value: 'USD', label: 'USD' }].map((option) => (
              <button key={option.value} className={currency === option.value ? 'is-active' : ''} type="button" aria-pressed={currency === option.value} onClick={() => onCurrencyChange(option.value)}>{option.label}</button>
            ))}
          </div>
        </fieldset>
      </div>
    </form>
  )
}
