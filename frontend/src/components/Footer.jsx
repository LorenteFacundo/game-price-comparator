export default function Footer() {
  return (
    <footer style={styles.footer}>
      <p style={styles.text}>
        Desarrollado por <strong>Facundo Lorente</strong>
      </p>
      <div style={styles.linksContainer}>
        <a 
          href="https://github.com/lorentefacundo" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.link}
        >
          GitHub
        </a>
        <span style={styles.separator}>|</span>
        <a 
          href="https://www.linkedin.com/in/facundo-lorente-692206230/"
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.link}
        >
          LinkedIn
        </a>
      </div>
    </footer>
  );
}

// Unos estilos básicos en línea para que se vea prolijo sin tocar tu index.css
const styles = {
  footer: {
    marginTop: 'auto',
    padding: '2rem 1rem',
    textAlign: 'center',
    borderTop: '1px solid #333', /* Ajusta el color si tu fondo es muy claro */
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '8px'
  },
  text: {
    margin: 0,
    fontSize: '14px',
    color: '#ccc'
  },
  linksContainer: {
    display: 'flex',
    gap: '12px',
    alignItems: 'center'
  },
  link: {
    color: '#646cff',
    textDecoration: 'none',
    fontWeight: '500',
    fontSize: '14px'
  },
  separator: {
    color: '#555',
    fontSize: '12px'
  }
}