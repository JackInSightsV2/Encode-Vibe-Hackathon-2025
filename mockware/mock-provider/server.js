const express = require('express');
const cors = require('cors');
const { v4: uuidv4 } = require('uuid');
require('dotenv').config();

const unifiedMockRoutes = require('./routes/unified-mock');
const healthRoutes = require('./routes/health');

const ResponseGenerator = require('./lib/response-generator');
const LatencySimulator = require('./lib/latency-simulator');
const ErrorSimulator = require('./lib/error-simulator');

const app = express();

// Middleware
app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true, limit: '50mb' }));

// Request logging middleware with comprehensive details
app.use((req, res, next) => {
    const requestStart = Date.now();
    const timestamp = new Date().toISOString();
    const method = req.method;
    const url = req.url;
    const userAgent = req.get('User-Agent') || 'Unknown';
    const contentType = req.get('Content-Type') || 'Unknown';
    const contentLength = req.get('Content-Length') || '0';
    const authorization = req.get('Authorization') ? '[PRESENT]' : '[NONE]';
    const host = req.get('Host') || 'Unknown';
    const origin = req.get('Origin') || 'Unknown';
    const referer = req.get('Referer') || 'Unknown';
    const clientIP = req.ip || req.connection.remoteAddress || 'Unknown';
    
    console.log(`\n==================== REQUEST START ====================`);
    console.log(`🕐 [${timestamp}] ${method} ${url}`);
    console.log(`📡 Client: ${clientIP} | Host: ${host}`);
    console.log(`🔧 User-Agent: ${userAgent}`);
    console.log(`📄 Content-Type: ${contentType} | Length: ${contentLength} bytes`);
    console.log(`🔐 Authorization: ${authorization}`);
    console.log(`🌐 Origin: ${origin} | Referer: ${referer}`);
    
    // Log all headers in debug mode
    if (process.env.ENABLE_DETAILED_LOGGING === 'true') {
        console.log(`📋 Headers:`);
        Object.entries(req.headers).forEach(([key, value]) => {
            // Mask sensitive headers
            const maskedValue = key.toLowerCase().includes('auth') || key.toLowerCase().includes('key') 
                ? '[MASKED]' : value;
            console.log(`   ${key}: ${maskedValue}`);
        });
    }
    
    // Log request body with detailed analysis
    if (req.body && Object.keys(req.body).length > 0) {
        const bodyStr = JSON.stringify(req.body);
        const bodySize = Buffer.byteLength(bodyStr, 'utf8');
        
        console.log(`📦 Request Body Analysis:`);
        console.log(`   Size: ${bodySize} bytes`);
        console.log(`   Fields: ${Object.keys(req.body).length}`);
        console.log(`   Keys: [${Object.keys(req.body).join(', ')}]`);
        
        // Analyze specific AI request patterns
        if (req.body.messages) {
            console.log(`   💬 Messages: ${req.body.messages.length} message(s)`);
            req.body.messages.forEach((msg, idx) => {
                const msgLength = msg.content ? msg.content.length : 0;
                console.log(`      [${idx}] ${msg.role}: ${msgLength} chars`);
            });
        }
        
        if (req.body.model) {
            console.log(`   🤖 Model: ${req.body.model}`);
        }
        
        if (req.body.prompt) {
            console.log(`   📝 Prompt: ${req.body.prompt.length} chars`);
        }
        
        if (req.body.max_tokens) {
            console.log(`   🎯 Max Tokens: ${req.body.max_tokens}`);
        }
        
        if (req.body.temperature !== undefined) {
            console.log(`   🌡️ Temperature: ${req.body.temperature}`);
        }
        
        if (req.body.stream) {
            console.log(`   🌊 Streaming: ${req.body.stream}`);
        }
        
        // Log full body if small enough, otherwise truncate
        if (bodySize < 1000) {
            console.log(`   📄 Full Body: ${bodyStr}`);
        } else {
            const truncated = bodyStr.substring(0, 500) + '... [TRUNCATED]';
            console.log(`   📄 Body Preview: ${truncated}`);
        }
    } else {
        console.log(`📦 Request Body: [EMPTY]`);
    }
    
    // Intercept response to log response details
    const originalSend = res.send;
    const originalJson = res.json;
    
    res.send = function(body) {
        logResponse(req, res, body, requestStart);
        return originalSend.call(this, body);
    };
    
    res.json = function(obj) {
        logResponse(req, res, JSON.stringify(obj), requestStart);
        return originalJson.call(this, obj);
    };
    
    next();
});

function logResponse(req, res, body, requestStart) {
    const responseTime = Date.now() - requestStart;
    const statusCode = res.statusCode;
    const responseSize = Buffer.byteLength(body || '', 'utf8');
    const timestamp = new Date().toISOString();
    
    console.log(`\n==================== RESPONSE ====================`);
    console.log(`🕐 [${timestamp}] Response sent`);
    console.log(`⚡ Processing Time: ${responseTime}ms`);
    console.log(`📊 Status: ${statusCode} ${getStatusDescription(statusCode)}`);
    console.log(`📦 Response Size: ${responseSize} bytes`);
    
    // Response headers
    console.log(`📋 Response Headers:`);
    const headers = res.getHeaders();
    Object.entries(headers).forEach(([key, value]) => {
        console.log(`   ${key}: ${value}`);
    });
    
    // Response body analysis
    try {
        const responseObj = JSON.parse(body || '{}');
        console.log(`📄 Response Analysis:`);
        console.log(`   Fields: ${Object.keys(responseObj).length}`);
        console.log(`   Keys: [${Object.keys(responseObj).join(', ')}]`);
        
        // AI-specific response analysis
        if (responseObj.choices) {
            console.log(`   🎯 Choices: ${responseObj.choices.length}`);
            responseObj.choices.forEach((choice, idx) => {
                const content = choice.message?.content || choice.text || '';
                console.log(`      [${idx}] ${choice.finish_reason}: ${content.length} chars`);
            });
        }
        
        if (responseObj.content) {
            console.log(`   💬 Content: ${Array.isArray(responseObj.content) ? responseObj.content.length + ' items' : responseObj.content.length + ' chars'}`);
        }
        
        if (responseObj.usage) {
            console.log(`   💰 Usage: ${responseObj.usage.prompt_tokens}+${responseObj.usage.completion_tokens}=${responseObj.usage.total_tokens} tokens`);
        }
        
        if (responseObj.error) {
            console.log(`   ❌ Error: ${responseObj.error.type} - ${responseObj.error.message}`);
        }
        
        // Log full response if small, otherwise truncate
        if (responseSize < 1000) {
            console.log(`   📄 Full Response: ${body}`);
        } else {
            const truncated = body.substring(0, 500) + '... [TRUNCATED]';
            console.log(`   📄 Response Preview: ${truncated}`);
        }
        
    } catch (e) {
        console.log(`   📄 Raw Response: ${body || '[EMPTY]'}`);
    }
    
    console.log(`==================== REQUEST END ====================\n`);
}

function getStatusDescription(code) {
    const descriptions = {
        200: 'OK',
        201: 'Created',
        400: 'Bad Request',
        401: 'Unauthorized',
        403: 'Forbidden',
        404: 'Not Found',
        429: 'Too Many Requests',
        500: 'Internal Server Error',
        502: 'Bad Gateway',
        503: 'Service Unavailable'
    };
    return descriptions[code] || 'Unknown';
}

// Rate limiting headers (for compatibility, but NO actual limiting)
app.use((req, res, next) => {
    // Set fake rate limit headers that indicate unlimited access
    res.set({
        'X-RateLimit-Limit': '999999',
        'X-RateLimit-Remaining': '999999',
        'X-RateLimit-Reset': Math.ceil((Date.now() + 3600000) / 1000)
    });
    next();
});

// Minimal error simulation (only if specifically enabled and very low rate)
app.use(async (req, res, next) => {
    const errorRate = parseFloat(process.env.ERROR_RATE) || 0;
    // Only simulate errors if explicitly enabled AND at a very low rate
    if (process.env.ENABLE_ERROR_SIMULATION === 'true' && Math.random() < Math.min(errorRate, 0.001)) {
        const errorSim = new ErrorSimulator();
        const error = errorSim.generateRandomError();
        return res.status(error.status).json(error.response);
    }
    next();
});

// Root endpoint with API documentation
app.get('/', (req, res) => {
    res.json({
        service: "QT-1 Mock AI Provider",
        version: "1.0.1-universal",
        build: "2025-01-21-universal-mock",
        description: "Universal mock AI provider for testing without API credits",
        endpoints: {
            "ANY /*": "Universal mock endpoint - detects request format and responds appropriately",
            "GET /health": "Basic health check",
            "GET /health/detailed": "Detailed health status",
            "GET /health/loadtest": "Load testing endpoint"
        },
        supported_formats: [
            "OpenAI Chat Completions (/v1/chat/completions)",
            "OpenAI Text Completions (/v1/completions)", 
            "OpenAI Moderation (/v1/moderations)",
            "Anthropic Messages (/v1/messages)",
            "Anthropic Complete (/v1/complete)",
            "Any other AI provider format"
        ],
        features: [
            "Multiple AI provider emulation",
            "Intelligent response generation",
            "Configurable latency simulation",
            "Error injection capabilities",
            "Rate limiting simulation",
            "Token usage calculation",
            "Streaming response support"
        ],
        configuration: {
            responseDelay: `${process.env.RESPONSE_DELAY_MIN}-${process.env.RESPONSE_DELAY_MAX}ms`,
            errorRate: `${(parseFloat(process.env.ERROR_RATE) * 100).toFixed(2)}%`,
            rateLimitWindow: `${process.env.RATE_LIMIT_WINDOW}ms`,
            rateLimitMax: process.env.RATE_LIMIT_MAX
        }
    });
});

// Version endpoint
app.get('/version', (req, res) => {
    res.json({
        service: "QT-1 Mock AI Provider",
        version: "1.0.1-universal",
        build: "2025-01-21-universal-mock", 
        commit: "universal-mock-endpoint",
        architecture: "Single Universal Endpoint",
        node_version: process.version,
        uptime: process.uptime(),
        environment: process.env.NODE_ENV || 'development',
        features: [
            "Universal AI Provider Emulation",
            "Auto-Detection (OpenAI, Anthropic, Custom)",
            "Intelligent Content Analysis",
            "Realistic Latency Simulation",
            "Configurable Error Injection", 
            "Rate Limiting Simulation",
            "Token Usage Calculation",
            "Streaming Response Support",
            "Health Monitoring & Metrics"
        ],
        capabilities: {
            "content_moderation": ["violence", "hate", "explicit", "self-harm"],
            "pii_detection": ["email", "phone", "ssn", "credit_card"],
            "prompt_injection": ["direct", "system", "hypothetical", "encoded"],
            "response_formats": ["openai", "anthropic", "generic"],
            "streaming": true,
            "error_simulation": true,
            "rate_limiting": true
        },
        configuration: {
            response_delay: `${process.env.RESPONSE_DELAY_MIN || 100}-${process.env.RESPONSE_DELAY_MAX || 500}ms`,
            error_rate: `${(parseFloat(process.env.ERROR_RATE || 0) * 100).toFixed(2)}%`,
            rate_limit: `${process.env.RATE_LIMIT_MAX || 1000} requests per ${process.env.RATE_LIMIT_WINDOW || 60000}ms`,
            streaming: process.env.ENABLE_STREAMING === 'true'
        },
        last_updated: new Date().toISOString()
    });
});

// Mount route handlers
app.use('/health', healthRoutes);

// Catch-all mock handler for any AI provider endpoint (must be last)
app.use('/', unifiedMockRoutes);

// 404 handler (this won't be reached due to catch-all above, but kept for completeness)
app.use('*', (req, res) => {
    res.status(404).json({
        error: {
            type: "not_found",
            message: `Endpoint not found: ${req.method} ${req.originalUrl}`,
            available_endpoints: [
                "POST /v1/chat/completions",
                "POST /v1/messages",
                "POST /api/v1/chat",
                "GET /health"
            ]
        }
    });
});

// Error handler
app.use((error, req, res, next) => {
    console.error('Server error:', error);
    res.status(500).json({
        error: {
            type: "internal_server_error",
            message: "An internal server error occurred",
            details: process.env.NODE_ENV === 'development' ? error.message : undefined
        }
    });
});

const PORT = process.env.PORT || 8081;
const server = app.listen(PORT, () => {
    console.log(`🤖 QT-1 Mock AI Provider v1.0.1-universal running on port ${PORT}`);
    console.log(`🔧 Build: 2025-01-21-universal-mock`);
    console.log(`📊 Configuration:`);
    console.log(`  - Response Delay: ${process.env.RESPONSE_DELAY_MIN || 100}-${process.env.RESPONSE_DELAY_MAX || 500}ms`);
    console.log(`  - Error Rate: ${(parseFloat(process.env.ERROR_RATE || 0) * 100).toFixed(2)}%`);
    console.log(`  - Rate Limit: ${process.env.RATE_LIMIT_MAX || 1000} requests per ${process.env.RATE_LIMIT_WINDOW || 60000}ms`);
    console.log(`  - Streaming: ${process.env.ENABLE_STREAMING === 'true' ? 'Enabled' : 'Disabled'}`);
    console.log(`🌐 Available at: http://localhost:${PORT}`);
    console.log(`📖 API docs: http://localhost:${PORT}`);
    console.log(`🔍 Version info: http://localhost:${PORT}/version`);
    console.log(`💊 Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('SIGTERM received, shutting down gracefully...');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('SIGINT received, shutting down gracefully...');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});