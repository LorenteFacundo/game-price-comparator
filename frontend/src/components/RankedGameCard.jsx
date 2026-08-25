function playerCount(players) {
  if (!players) return ''
  return new Intl.NumberFormat('es-AR', { notation: 'compact', maximumFractionDigits: 1 }).format(players)
}

export default function RankedGameCard({ game, onSearch, showPlayers = false }) {
  const players = playerCount(game.players)
  return (
    <article className="rank-card">
      <button type="button" onClick={() => onSearch(game.title)} aria-label={`Comparar precios de ${game.title}`}>
        <div className="rank-image">
          {game.image_url ? <img src={game.image_url} alt="" width="300" height="140" loading="lazy" /> : <div className="image-fallback" aria-hidden="true">✦</div>}
          <span className="rank-number">#{game.rank}</span>
        </div>
        <span className="rank-title">{game.title}</span>
        <span className="rank-action">{showPlayers && players ? `${players} jugando` : 'Comparar precios'} <b aria-hidden="true">↗</b></span>
      </button>
    </article>
  )
}
