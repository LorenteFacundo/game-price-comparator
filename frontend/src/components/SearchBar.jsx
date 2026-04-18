import { useState } from 'react'

export default function SearchBar({ onSearch, loading }) {
  const [query, setQuery] = useState('')

  function handleSubmit(e) {
    e.preventDefault()
    if (query.trim()) onSearch(query.trim())
  }

  return (
    <form onSubmit={handleSubmit} style={{
      display: 'flex', gap: '10px', width: '100%', maxWidth: '600px'
    }}>
      <input
        value={query}
        onChange={e => setQuery(e.target.value)}
        placeholder="Buscar juego... ej: Hades, Elden Ring"
        disabled={loading}
        style={{
          flex: 1,
          background: 'var(--surface)',
          border: '1px solid var(--border2)',
          borderRadius: 'var(--radius)',
          padding: '12px 18px',
          fontSize: '15px',
          color: 'var(--text)',
          outline: 'none',
        }}
      />
      <button
        type="submit"
        disabled={loading || !query.trim()}
        style={{
          background: 'linear-gradient(135deg, var(--accent), var(--accent2))',
          border: 'none',
          borderRadius: 'var(--radius)',
          padding: '12px 24px',
          fontSize: '15px',
          fontWeight: 600,
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