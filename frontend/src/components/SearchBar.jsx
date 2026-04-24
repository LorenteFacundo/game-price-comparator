import { useState } from 'react'

export default function SearchBar({ onSearch, loading }) {
  const [query, setQuery] = useState('')

  function handleSubmit(e) {
    e.preventDefault()
    if (query.trim()) onSearch(query.trim())
  }

  return (
    <form
      onSubmit={handleSubmit}
      style={{
        display: 'flex',
        gap: '10px',
        width: '100%',
      }}
    >
      <label style={{ flex: 1 }}>
        <span style={{
          display: 'block',
          fontSize: '12px',
          color: 'var(--muted)',
          marginBottom: '8px',
        }}>
          Buscar juego
        </span>
        <input
          aria-label="Buscar juego"
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Ej: Hades, Elden Ring, Pragmata"
          disabled={loading}
          style={{
            width: '100%',
            background: 'var(--surface)',
            border: '1px solid var(--border2)',
            borderRadius: 'var(--radius)',
            padding: '14px 18px',
            fontSize: '15px',
            color: 'var(--text)',
            outline: 'none',
          }}
        />
      </label>
      <button
        type="submit"
        disabled={loading || !query.trim()}
        style={{
          alignSelf: 'end',
          background: 'linear-gradient(135deg, var(--accent), var(--accent2))',
          border: 'none',
          borderRadius: 'var(--radius)',
          padding: '14px 24px',
          fontSize: '15px',
          fontWeight: 700,
          color: '#fff',
          opacity: loading || !query.trim() ? 0.5 : 1,
          transition: 'opacity 0.2s',
        }}
      >
        {loading ? 'Buscando...' : 'Buscar'}
      </button>
    </form>
  )
}
