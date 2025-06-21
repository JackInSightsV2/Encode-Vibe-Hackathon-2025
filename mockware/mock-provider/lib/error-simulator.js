class ErrorSimulator {
    constructor() {
        this.baseErrorRate = parseFloat(process.env.ERROR_RATE) || 0;
        this.errorTypes = this.initializeErrorTypes();
    }

    generateRandomError() {
        const errorCategories = Object.keys(this.errorTypes);
        const category = errorCategories[Math.floor(Math.random() * errorCategories.length)];
        const errors = this.errorTypes[category];
        const error = errors[Math.floor(Math.random() * errors.length)];
        
        return {
            status: error.status,
            response: error.response
        };
    }

    generateOpenAIError(type = 'internal_error') {
        const errorMap = {
            'rate_limit': {
                status: 429,
                response: {
                    error: {
                        message: "Rate limit reached for requests",
                        type: "rate_limit_exceeded",
                        param: null,
                        code: "rate_limit_exceeded"
                    }
                }
            },
            'invalid_request': {
                status: 400,
                response: {
                    error: {
                        message: "Invalid request format or missing required parameters",
                        type: "invalid_request_error",
                        param: null,
                        code: null
                    }
                }
            },
            'authentication': {
                status: 401,
                response: {
                    error: {
                        message: "Incorrect API key provided",
                        type: "invalid_request_error",
                        param: null,
                        code: "invalid_api_key"
                    }
                }
            },
            'insufficient_quota': {
                status: 429,
                response: {
                    error: {
                        message: "You exceeded your current quota, please check your plan and billing details",
                        type: "insufficient_quota",
                        param: null,
                        code: "insufficient_quota"
                    }
                }
            },
            'model_not_found': {
                status: 404,
                response: {
                    error: {
                        message: "The model `invalid-model` does not exist",
                        type: "invalid_request_error",
                        param: null,
                        code: "model_not_found"
                    }
                }
            },
            'context_length_exceeded': {
                status: 400,
                response: {
                    error: {
                        message: "This model's maximum context length is 4096 tokens",
                        type: "invalid_request_error",
                        param: "messages",
                        code: "context_length_exceeded"
                    }
                }
            },
            'content_filter': {
                status: 400,
                response: {
                    error: {
                        message: "Your request was rejected as a result of our safety system",
                        type: "invalid_request_error",
                        param: null,
                        code: "content_filter"
                    }
                }
            },
            'server_error': {
                status: 500,
                response: {
                    error: {
                        message: "The server had an error while processing your request",
                        type: "server_error",
                        param: null,
                        code: null
                    }
                }
            },
            'service_unavailable': {
                status: 503,
                response: {
                    error: {
                        message: "The engine is currently overloaded, please try again later",
                        type: "server_error",
                        param: null,
                        code: "service_unavailable"
                    }
                }
            },
            'timeout': {
                status: 408,
                response: {
                    error: {
                        message: "Request timeout",
                        type: "timeout_error",
                        param: null,
                        code: "timeout"
                    }
                }
            },
            'internal_error': {
                status: 500,
                response: {
                    error: {
                        message: "Internal server error occurred",
                        type: "server_error",
                        param: null,
                        code: "internal_error"
                    }
                }
            }
        };

        return errorMap[type] || errorMap['internal_error'];
    }

    generateAnthropicError(type = 'internal_server_error') {
        const errorMap = {
            'invalid_request_error': {
                status: 400,
                response: {
                    type: "error",
                    error: {
                        type: "invalid_request_error",
                        message: "Invalid request: missing required field"
                    }
                }
            },
            'authentication_error': {
                status: 401,
                response: {
                    type: "error",
                    error: {
                        type: "authentication_error",
                        message: "Invalid API key"
                    }
                }
            },
            'permission_error': {
                status: 403,
                response: {
                    type: "error",
                    error: {
                        type: "permission_error",
                        message: "Permission denied"
                    }
                }
            },
            'not_found_error': {
                status: 404,
                response: {
                    type: "error",
                    error: {
                        type: "not_found_error",
                        message: "Model not found"
                    }
                }
            },
            'rate_limit_error': {
                status: 429,
                response: {
                    type: "error",
                    error: {
                        type: "rate_limit_error",
                        message: "Rate limit exceeded"
                    }
                }
            },
            'api_error': {
                status: 500,
                response: {
                    type: "error",
                    error: {
                        type: "api_error",
                        message: "Internal server error"
                    }
                }
            },
            'overloaded_error': {
                status: 529,
                response: {
                    type: "error",
                    error: {
                        type: "overloaded_error",
                        message: "Service is temporarily overloaded"
                    }
                }
            },
            'internal_server_error': {
                status: 500,
                response: {
                    type: "error",
                    error: {
                        type: "api_error",
                        message: "An unexpected error occurred"
                    }
                }
            }
        };

        return errorMap[type] || errorMap['internal_server_error'];
    }

    generateGenericError(type = 'internal_error') {
        const errorMap = {
            'invalid_request': {
                status: 400,
                response: {
                    error: {
                        type: "invalid_request",
                        message: "The request is invalid or malformed",
                        code: "INVALID_REQUEST"
                    }
                }
            },
            'authentication_failed': {
                status: 401,
                response: {
                    error: {
                        type: "authentication_failed",
                        message: "Authentication failed",
                        code: "AUTH_FAILED"
                    }
                }
            },
            'rate_limited': {
                status: 429,
                response: {
                    error: {
                        type: "rate_limited",
                        message: "Too many requests",
                        code: "RATE_LIMITED"
                    }
                }
            },
            'processing_error': {
                status: 500,
                response: {
                    error: {
                        type: "processing_error",
                        message: "Error processing request",
                        code: "PROCESSING_ERROR"
                    }
                }
            },
            'analysis_error': {
                status: 500,
                response: {
                    error: {
                        type: "analysis_error",
                        message: "Error during text analysis",
                        code: "ANALYSIS_ERROR"
                    }
                }
            },
            'moderation_error': {
                status: 500,
                response: {
                    error: {
                        type: "moderation_error",
                        message: "Error during moderation check",
                        code: "MODERATION_ERROR"
                    }
                }
            },
            'token_error': {
                status: 500,
                response: {
                    error: {
                        type: "token_error",
                        message: "Error counting tokens",
                        code: "TOKEN_ERROR"
                    }
                }
            },
            'batch_error': {
                status: 500,
                response: {
                    error: {
                        type: "batch_error",
                        message: "Error processing batch request",
                        code: "BATCH_ERROR"
                    }
                }
            },
            'internal_error': {
                status: 500,
                response: {
                    error: {
                        type: "internal_error",
                        message: "Internal server error",
                        code: "INTERNAL_ERROR"
                    }
                }
            }
        };

        return errorMap[type] || errorMap['internal_error'];
    }

    // Simulate network-level errors
    simulateNetworkError() {
        const networkErrors = [
            {
                status: 502,
                response: {
                    error: {
                        type: "bad_gateway",
                        message: "Bad gateway",
                        code: "BAD_GATEWAY"
                    }
                }
            },
            {
                status: 503,
                response: {
                    error: {
                        type: "service_unavailable",
                        message: "Service temporarily unavailable",
                        code: "SERVICE_UNAVAILABLE"
                    }
                }
            },
            {
                status: 504,
                response: {
                    error: {
                        type: "gateway_timeout",
                        message: "Gateway timeout",
                        code: "GATEWAY_TIMEOUT"
                    }
                }
            },
            {
                status: 408,
                response: {
                    error: {
                        type: "request_timeout",
                        message: "Request timeout",
                        code: "REQUEST_TIMEOUT"
                    }
                }
            }
        ];

        return networkErrors[Math.floor(Math.random() * networkErrors.length)];
    }

    // Check if an error should occur based on configuration
    shouldError() {
        return Math.random() < this.baseErrorRate;
    }

    // Generate error with specific probability
    generateConditionalError(probability = null) {
        const errorProbability = probability !== null ? probability : this.baseErrorRate;
        
        if (Math.random() < errorProbability) {
            return this.generateRandomError();
        }
        
        return null;
    }

    // Get error statistics
    getErrorStats() {
        return {
            baseErrorRate: this.baseErrorRate,
            availableErrorTypes: Object.keys(this.errorTypes),
            totalErrorVariants: Object.values(this.errorTypes)
                .reduce((sum, errors) => sum + errors.length, 0)
        };
    }

    initializeErrorTypes() {
        return {
            client_errors: [
                {
                    status: 400,
                    response: {
                        error: {
                            type: "invalid_request_error",
                            message: "Invalid JSON in request body"
                        }
                    }
                },
                {
                    status: 401,
                    response: {
                        error: {
                            type: "authentication_error",
                            message: "Invalid or missing API key"
                        }
                    }
                },
                {
                    status: 403,
                    response: {
                        error: {
                            type: "permission_error",
                            message: "Insufficient permissions for this operation"
                        }
                    }
                },
                {
                    status: 404,
                    response: {
                        error: {
                            type: "not_found_error",
                            message: "Requested resource not found"
                        }
                    }
                }
            ],
            rate_limiting: [
                {
                    status: 429,
                    response: {
                        error: {
                            type: "rate_limit_exceeded",
                            message: "Rate limit exceeded for API key"
                        }
                    }
                },
                {
                    status: 429,
                    response: {
                        error: {
                            type: "quota_exceeded",
                            message: "Monthly quota exceeded"
                        }
                    }
                }
            ],
            server_errors: [
                {
                    status: 500,
                    response: {
                        error: {
                            type: "internal_server_error",
                            message: "An unexpected error occurred"
                        }
                    }
                },
                {
                    status: 502,
                    response: {
                        error: {
                            type: "bad_gateway",
                            message: "Upstream service error"
                        }
                    }
                },
                {
                    status: 503,
                    response: {
                        error: {
                            type: "service_unavailable",
                            message: "Service temporarily unavailable"
                        }
                    }
                }
            ],
            timeout_errors: [
                {
                    status: 408,
                    response: {
                        error: {
                            type: "request_timeout",
                            message: "Request timed out"
                        }
                    }
                },
                {
                    status: 504,
                    response: {
                        error: {
                            type: "gateway_timeout",
                            message: "Gateway timeout occurred"
                        }
                    }
                }
            ]
        };
    }
}

module.exports = ErrorSimulator;