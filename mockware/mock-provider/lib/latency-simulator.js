class LatencySimulator {
    constructor() {
        this.minDelay = parseInt(process.env.RESPONSE_DELAY_MIN) || 100;
        this.maxDelay = parseInt(process.env.RESPONSE_DELAY_MAX) || 500;
        this.networkPatterns = this.initializeNetworkPatterns();
    }

    async simulate(options = {}) {
        const delay = this.calculateDelay(options);
        
        if (delay > 0) {
            await this.sleep(delay);
        }
        
        return delay;
    }

    calculateDelay(options = {}) {
        const {
            requestSize = 0,
            responseSize = 0,
            complexity = 'normal',
            networkCondition = 'normal',
            model = 'default'
        } = options;

        let baseDelay = this.getRandomDelay();
        
        // Adjust based on request complexity
        baseDelay *= this.getComplexityMultiplier(complexity);
        
        // Adjust based on model type
        baseDelay *= this.getModelMultiplier(model);
        
        // Adjust based on request/response size
        baseDelay += this.getSizeDelay(requestSize, responseSize);
        
        // Apply network conditions
        baseDelay *= this.getNetworkMultiplier(networkCondition);
        
        // Add some randomness to make it realistic
        baseDelay *= (0.8 + Math.random() * 0.4); // ±20% variance
        
        return Math.round(Math.max(0, baseDelay));
    }

    getRandomDelay() {
        // Generate delay with realistic distribution (not uniform)
        // Most responses are quick, with occasional slower ones
        const random = Math.random();
        
        if (random < 0.7) {
            // 70% of requests are in the lower range
            return this.minDelay + (this.maxDelay - this.minDelay) * 0.3 * Math.random();
        } else if (random < 0.9) {
            // 20% are in the middle range
            return this.minDelay + (this.maxDelay - this.minDelay) * (0.3 + 0.4 * Math.random());
        } else {
            // 10% are in the upper range (slow responses)
            return this.minDelay + (this.maxDelay - this.minDelay) * (0.7 + 0.3 * Math.random());
        }
    }

    getComplexityMultiplier(complexity) {
        const multipliers = {
            'simple': 0.7,      // Simple questions
            'normal': 1.0,      // Regular complexity
            'complex': 1.5,     // Complex reasoning
            'very_complex': 2.0, // Very complex tasks
            'coding': 1.8,      // Code generation
            'creative': 1.3,    // Creative writing
            'analysis': 1.6     // Data analysis
        };
        
        return multipliers[complexity] || 1.0;
    }

    getModelMultiplier(model) {
        // Different models have different response times
        const modelMultipliers = {
            'gpt-3.5-turbo': 0.8,
            'gpt-4': 1.5,
            'gpt-4-turbo': 1.2,
            'gpt-4o': 1.0,
            'claude-3-haiku': 0.7,
            'claude-3-sonnet': 1.0,
            'claude-3-opus': 1.8,
            'claude-3.5-sonnet': 1.1,
            'local': 0.5,       // Local models are faster
            'mock': 0.6         // Mock responses are quick
        };
        
        // Find matching model (case insensitive, partial match)
        const modelLower = model.toLowerCase();
        for (const [key, multiplier] of Object.entries(modelMultipliers)) {
            if (modelLower.includes(key.toLowerCase())) {
                return multiplier;
            }
        }
        
        return 1.0;
    }

    getSizeDelay(requestSize, responseSize) {
        // Add delay based on input/output size
        const requestDelay = Math.min(requestSize / 1000, 100); // Max 100ms for large requests
        const responseDelay = Math.min(responseSize / 500, 50);  // Max 50ms for large responses
        
        return requestDelay + responseDelay;
    }

    getNetworkMultiplier(condition) {
        const conditions = {
            'excellent': 0.6,   // Fast connection
            'good': 0.8,        // Good connection
            'normal': 1.0,      // Normal connection
            'slow': 1.5,        // Slow connection
            'poor': 2.0,        // Poor connection
            'timeout': 10.0     // Connection issues
        };
        
        return conditions[condition] || 1.0;
    }

    // Simulate streaming response delay
    async simulateStreaming(totalTokens, options = {}) {
        const {
            tokensPerChunk = 10,
            baseChunkDelay = 50,
            model = 'default'
        } = options;
        
        const chunks = Math.ceil(totalTokens / tokensPerChunk);
        const delays = [];
        
        for (let i = 0; i < chunks; i++) {
            let chunkDelay = baseChunkDelay;
            
            // First chunk is usually slower (initial processing)
            if (i === 0) {
                chunkDelay *= 2;
            }
            
            // Apply model-specific streaming speed
            chunkDelay *= this.getModelStreamingMultiplier(model);
            
            // Add some variance
            chunkDelay *= (0.8 + Math.random() * 0.4);
            
            delays.push(Math.round(chunkDelay));
        }
        
        return delays;
    }

    getModelStreamingMultiplier(model) {
        const streamingMultipliers = {
            'gpt-3.5-turbo': 0.8,
            'gpt-4': 1.2,
            'claude-3-haiku': 0.7,
            'claude-3-sonnet': 1.0,
            'claude-3-opus': 1.3
        };
        
        const modelLower = model.toLowerCase();
        for (const [key, multiplier] of Object.entries(streamingMultipliers)) {
            if (modelLower.includes(key.toLowerCase())) {
                return multiplier;
            }
        }
        
        return 1.0;
    }

    // Simulate network jitter
    async simulateJitter(baseDelay, jitterPercent = 0.1) {
        const jitter = baseDelay * jitterPercent * (Math.random() - 0.5) * 2;
        const finalDelay = Math.max(0, baseDelay + jitter);
        
        await this.sleep(finalDelay);
        return finalDelay;
    }

    // Simulate timeout scenarios
    async simulateTimeout(timeoutMs = 30000) {
        const shouldTimeout = Math.random() < 0.001; // 0.1% chance
        
        if (shouldTimeout) {
            await this.sleep(timeoutMs);
            throw new Error('Request timeout');
        }
    }

    // Pattern-based delays for realistic simulation
    getPatternDelay(pattern = 'normal') {
        const patterns = this.networkPatterns[pattern];
        if (!patterns) return this.getRandomDelay();
        
        const randomPattern = patterns[Math.floor(Math.random() * patterns.length)];
        return randomPattern.delay + (Math.random() - 0.5) * randomPattern.variance;
    }

    initializeNetworkPatterns() {
        return {
            burst: [
                { delay: 50, variance: 20 },   // Very fast
                { delay: 100, variance: 30 },  // Fast
                { delay: 200, variance: 50 }   // Normal
            ],
            sustained: [
                { delay: 150, variance: 40 },  // Consistent medium
                { delay: 200, variance: 60 },  // Consistent normal
                { delay: 250, variance: 80 }   // Consistent slow
            ],
            degraded: [
                { delay: 500, variance: 200 }, // Slow
                { delay: 1000, variance: 500 }, // Very slow
                { delay: 2000, variance: 1000 } // Extremely slow
            ],
            unstable: [
                { delay: 50, variance: 500 },   // Highly variable
                { delay: 200, variance: 800 },  // Very unstable
                { delay: 1000, variance: 2000 } // Chaotic
            ]
        };
    }

    // Helper method for sleeping
    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    // Get delay statistics for monitoring
    getStats() {
        return {
            minDelay: this.minDelay,
            maxDelay: this.maxDelay,
            averageDelay: (this.minDelay + this.maxDelay) / 2,
            patterns: Object.keys(this.networkPatterns)
        };
    }

    // Simulate progressive delay (for rate limiting scenarios)
    async simulateProgressiveDelay(requestCount, baseDelay = null) {
        if (baseDelay === null) {
            baseDelay = this.getRandomDelay();
        }
        
        // Increase delay based on request count (simulating rate limiting)
        const progressiveMultiplier = Math.min(1 + (requestCount / 100), 5); // Max 5x delay
        const finalDelay = baseDelay * progressiveMultiplier;
        
        await this.sleep(finalDelay);
        return finalDelay;
    }
}

module.exports = LatencySimulator;