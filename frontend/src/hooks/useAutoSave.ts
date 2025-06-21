import { useState, useEffect, useRef, useCallback } from 'react'

export const useAutoSave = (
  data: any,
  saveFn: () => Promise<void>,
  delay: number = 30000 // Default 30 seconds
) => {
  const [hasChanges, setHasChanges] = useState(false)
  const [lastSavedData, setLastSavedData] = useState<any>(null)
  const [isSaving, setIsSaving] = useState(false)
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const lastSaveTimeRef = useRef<Date | null>(null)

  // Initialize lastSavedData on mount
  useEffect(() => {
    if (data && !lastSavedData) {
      setLastSavedData(JSON.parse(JSON.stringify(data)))
    }
  }, [data, lastSavedData])

  // Check for changes
  useEffect(() => {
    if (!data || !lastSavedData) return

    const dataStr = JSON.stringify(data)
    const lastSavedStr = JSON.stringify(lastSavedData)
    
    const changed = dataStr !== lastSavedStr
    setHasChanges(changed)

    // Clear existing timeout
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }

    // Set up auto-save if there are changes
    if (changed && delay > 0) {
      saveTimeoutRef.current = setTimeout(async () => {
        await performSave()
      }, delay)
    }

    // Cleanup
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current)
      }
    }
  }, [data, lastSavedData, delay])

  const performSave = useCallback(async () => {
    if (!hasChanges || isSaving) return

    setIsSaving(true)
    try {
      await saveFn()
      setLastSavedData(JSON.parse(JSON.stringify(data)))
      lastSaveTimeRef.current = new Date()
      setHasChanges(false)
    } catch (error) {
      console.error('Auto-save failed:', error)
      // Keep hasChanges true on error
    } finally {
      setIsSaving(false)
    }
  }, [data, hasChanges, isSaving, saveFn])

  const markSaved = useCallback(() => {
    setLastSavedData(JSON.parse(JSON.stringify(data)))
    lastSaveTimeRef.current = new Date()
    setHasChanges(false)
  }, [data])

  const saveNow = useCallback(async () => {
    // Clear any pending auto-save
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }
    
    await performSave()
  }, [performSave])

  const getLastSaveTime = useCallback(() => {
    return lastSaveTimeRef.current
  }, [])

  const getTimeSinceLastSave = useCallback(() => {
    if (!lastSaveTimeRef.current) return null
    
    const now = new Date()
    const diff = now.getTime() - lastSaveTimeRef.current.getTime()
    
    if (diff < 60000) {
      return `${Math.floor(diff / 1000)} seconds ago`
    } else if (diff < 3600000) {
      return `${Math.floor(diff / 60000)} minutes ago`
    } else {
      return `${Math.floor(diff / 3600000)} hours ago`
    }
  }, [])

  return {
    hasChanges,
    isSaving,
    markSaved,
    saveNow,
    getLastSaveTime,
    getTimeSinceLastSave
  }
}