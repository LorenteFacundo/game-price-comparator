import { useEffect, useState } from 'react'

export function useLocalStorage(key, initialValue) {
  const [value, setValue] = useState(() => {
    try {
      const saved = window.localStorage.getItem(key)
      return saved ? JSON.parse(saved) : initialValue
    } catch {
      return initialValue
    }
  })

	useEffect(() => {
		try {
			window.localStorage.setItem(key, JSON.stringify(value))
		} catch {
			// El almacenamiento puede estar bloqueado o lleno; el estado en memoria sigue funcionando.
		}
  }, [key, value])

  return [value, setValue]
}
