const express = require('express');
const ResponseGenerator = require('../lib/response-generator');
const LatencySimulator = require('../lib/latency-simulator');
const ErrorSimulator = require('../lib/error-simulator');

const router = express.Router();
const generator = new ResponseGenerator();
const latency = new LatencySimulator();
const errorSim = new ErrorSimulator();

// Start time for uptime calculation
const startTime = Date.now();

// Basic health check
router.get('/', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'QT-1 Mock AI Provider',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        uptime: Date.now() - startTime
    });
});

// Detailed health status
router.get('/detailed', (req, res) => {
    const uptime = Date.now() - startTime;
    const memUsage = process.memoryUsage();
    
    // Test core components
    const componentStatus = {
        response_generator: testResponseGenerator(),
        latency_simulator: testLatencySimulator(),
        error_simulator: testErrorSimulator(),
        templates: testTemplateLoading(),
        personalities: testPersonalityLoading()
    };
    
    const allHealthy = Object.values(componentStatus).every(status => status.healthy);
    
    res.json({
        status: allHealthy ? 'healthy' : 'degraded',
        service: 'QT-1 Mock AI Provider',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        uptime: {
            milliseconds: uptime,
            seconds: Math.floor(uptime / 1000),
            minutes: Math.floor(uptime / 60000),
            hours: Math.floor(uptime / 3600000)
        },
        system: {
            memory: {
                used: memUsage.heapUsed,
                total: memUsage.heapTotal,
                external: memUsage.external,
                rss: memUsage.rss
            },
            cpu: process.cpuUsage(),
            node_version: process.version,
            platform: process.platform,
            arch: process.arch
        },
        components: componentStatus,
        configuration: {
            response_delay: {
                min: parseInt(process.env.RESPONSE_DELAY_MIN) || 100,
                max: parseInt(process.env.RESPONSE_DELAY_MAX) || 500
            },
            error_rate: parseFloat(process.env.ERROR_RATE) || 0,
            rate_limit: {
                max: parseInt(process.env.RATE_LIMIT_MAX) || 1000,
                window: parseInt(process.env.RATE_LIMIT_WINDOW) || 60000
            },
            streaming_enabled: process.env.ENABLE_STREAMING === 'true'
        },
        endpoints: {
            openai: [
                'POST /v1/chat/completions',
                'POST /v1/completions',
                'POST /v1/moderations'
            ],
            anthropic: [
                'POST /v1/messages',
                'POST /v1/complete'
            ],
            generic: [
                'POST /api/v1/chat',
                'POST /api/v1/complete',
                'POST /api/v1/analyze',
                'POST /api/v1/moderate',
                'POST /api/v1/tokens/count',
                'POST /api/v1/batch'
            ]
        }
    });
});

// Load test endpoint
router.get('/loadtest', async (req, res) => {
    const iterations = parseInt(req.query.iterations) || 10;
    const maxIterations = 100; // Prevent abuse
    const actualIterations = Math.min(iterations, maxIterations);
    
    const results = {
        iterations: actualIterations,
        start_time: new Date().toISOString(),
        tests: []
    };
    
    try {
        for (let i = 0; i < actualIterations; i++) {
            const testStart = Date.now();
            
            // Test response generation
            const testRequest = {
                messages: [{ role: 'user', content: `Test message ${i + 1}` }],
                model: 'test-model'
            };
            
            const response = generator.generateOpenAIResponse(testRequest);
            const generationTime = Date.now() - testStart;
            
            // Test latency simulation
            const latencyStart = Date.now();
            await latency.simulate({ complexity: 'simple' });
            const latencyTime = Date.now() - latencyStart;
            
            results.tests.push({
                iteration: i + 1,
                generation_time_ms: generationTime,
                latency_time_ms: latencyTime,
                total_time_ms: generationTime + latencyTime,
                response_size: JSON.stringify(response).length,
                success: true
            });
        }
    } catch (error) {
        results.tests.push({
            error: error.message,
            success: false
        });
    }
    
    const totalTime = Date.now() - new Date(results.start_time).getTime();
    const successfulTests = results.tests.filter(t => t.success).length;
    const avgGenerationTime = results.tests
        .filter(t => t.success)
        .reduce((sum, t) => sum + t.generation_time_ms, 0) / successfulTests;
    const avgLatencyTime = results.tests
        .filter(t => t.success)
        .reduce((sum, t) => sum + t.latency_time_ms, 0) / successfulTests;
    
    results.summary = {
        total_time_ms: totalTime,
        successful_tests: successfulTests,
        failed_tests: results.tests.length - successfulTests,
        success_rate: (successfulTests / results.tests.length) * 100,
        average_generation_time_ms: Math.round(avgGenerationTime),
        average_latency_time_ms: Math.round(avgLatencyTime),
        requests_per_second: Math.round((successfulTests / totalTime) * 1000)
    };
    
    results.end_time = new Date().toISOString();
    
    res.json(results);
});

// Test individual components
function testResponseGenerator() {
    try {
        const testRequest = {
            messages: [{ role: 'user', content: 'Health check test' }],
            model: 'test-model'
        };
        
        const response = generator.generateOpenAIResponse(testRequest);
        
        return {
            healthy: !!(response && response.choices && response.choices[0]),
            message: 'Response generator working normally',
            last_test: new Date().toISOString()
        };
    } catch (error) {
        return {
            healthy: false,
            message: `Response generator error: ${error.message}`,
            last_test: new Date().toISOString()
        };
    }
}

function testLatencySimulator() {
    try {
        const stats = latency.getStats();
        
        return {
            healthy: !!(stats && stats.minDelay >= 0 && stats.maxDelay >= 0),
            message: 'Latency simulator working normally',
            stats: stats,
            last_test: new Date().toISOString()
        };
    } catch (error) {
        return {
            healthy: false,
            message: `Latency simulator error: ${error.message}`,
            last_test: new Date().toISOString()
        };
    }
}

function testErrorSimulator() {
    try {
        const testError = errorSim.generateOpenAIError('test_error');
        
        return {
            healthy: !!(testError && testError.error),
            message: 'Error simulator working normally',
            last_test: new Date().toISOString()
        };
    } catch (error) {
        return {
            healthy: false,
            message: `Error simulator error: ${error.message}`,
            last_test: new Date().toISOString()
        };
    }
}

function testTemplateLoading() {
    try {
        const hasTemplates = generator.templates && 
                           Object.keys(generator.templates).length > 0;
        
        return {
            healthy: hasTemplates,
            message: hasTemplates ? 'Templates loaded successfully' : 'Using default templates',
            template_count: Object.keys(generator.templates || {}).length,
            available_types: Object.keys(generator.templates || {}),
            last_test: new Date().toISOString()
        };
    } catch (error) {
        return {
            healthy: false,
            message: `Template loading error: ${error.message}`,
            last_test: new Date().toISOString()
        };
    }
}

function testPersonalityLoading() {
    try {
        const hasPersonalities = generator.personalities && 
                                Object.keys(generator.personalities).length > 0;
        
        return {
            healthy: hasPersonalities,
            message: hasPersonalities ? 'Personalities loaded successfully' : 'Using default personalities',
            personality_count: Object.keys(generator.personalities || {}).length,
            available_personalities: Object.keys(generator.personalities || {}),
            last_test: new Date().toISOString()
        };
    } catch (error) {
        return {
            healthy: false,
            message: `Personality loading error: ${error.message}`,
            last_test: new Date().toISOString()
        };
    }
}

module.exports = router;