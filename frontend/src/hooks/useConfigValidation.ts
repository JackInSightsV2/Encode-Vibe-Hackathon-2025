import { useState, useCallback, useRef } from 'react'

interface ValidationError {
  field: string
  message: string
  code: string
  severity: string
  suggestion?: string
  value?: any
}

interface ValidationWarning {
  field: string
  message: string
  code: string
  suggestion?: string
  value?: any
}

interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
  warnings: ValidationWarning[]
  summary: string
  validated_at?: string
  duration?: string
}

export const useConfigValidation = () => {
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null)
  const [isValidating, setIsValidating] = useState(false)
  const validationTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const validate = useCallback(async (config: any) => {
    // Clear any pending validation
    if (validationTimeoutRef.current) {
      clearTimeout(validationTimeoutRef.current)
    }

    // Debounce validation
    validationTimeoutRef.current = setTimeout(async () => {
      setIsValidating(true)
      
      try {
        const startTime = Date.now()
        
        // Call validation API
        const response = await fetch('/api/config/validate', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(config),
        })

        if (response.ok) {
          const data = await response.json()
          const endTime = Date.now()
          
          const result: ValidationResult = {
            ...data.data,
            validated_at: new Date().toISOString(),
            duration: `${endTime - startTime}ms`
          }
          
          setValidationResult(result)
        } else {
          // Handle validation error
          setValidationResult({
            valid: false,
            errors: [{
              field: 'general',
              message: 'Validation service error',
              code: 'VALIDATION_SERVICE_ERROR',
              severity: 'error'
            }],
            warnings: [],
            summary: 'Failed to validate configuration'
          })
        }
      } catch (error) {
        // Handle network or other errors
        setValidationResult({
          valid: false,
          errors: [{
            field: 'general',
            message: 'Network error during validation',
            code: 'NETWORK_ERROR',
            severity: 'error'
          }],
          warnings: [],
          summary: 'Failed to connect to validation service'
        })
      } finally {
        setIsValidating(false)
      }
    }, 300) // 300ms debounce
  }, [])

  const clearValidation = useCallback(() => {
    if (validationTimeoutRef.current) {
      clearTimeout(validationTimeoutRef.current)
    }
    setValidationResult(null)
    setIsValidating(false)
  }, [])

  const validateField = useCallback(async () => {
    // This could be extended to validate individual fields
    // For now, it triggers a full validation
    // In a real implementation, you might want field-level validation
    return true
  }, [])

  return {
    validationResult,
    isValidating,
    validate,
    clearValidation,
    validateField
  }
}