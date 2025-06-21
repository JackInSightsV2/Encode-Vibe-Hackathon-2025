const axios = require('axios');
const TestGenerator = require('./test-generator');

class LoadRunner {
    constructor(config, socket = null) {
        this.config = config;
        this.socket = socket;
        this.generator = new TestGenerator();
        this.isRunning = false;
        this.startTime = null;
        this.results = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            blockedRequests: 0,
            errors: [],
            responseTimes: [],
            statusCodes: {},
            categories: {},
            startTime: null,
            endTime: null
        };
        this.concurrentRequests = 0;
    }

    async execute() {
        this.isRunning = true;
        this.startTime = Date.now();
        this.results.startTime = new Date().toISOString();
        
        const startMessage = `🚀 ==================== LOAD TEST EXECUTION START ====================`;
        console.log(`\n${startMessage}`);
        this.emitLog(startMessage, 'info');
        
        const configDetails = [
            `🕐 Start Time: ${this.results.startTime}`,
            `📊 Test Configuration:`,
            `   Test Type: ${this.config.testType}`,
            `   Target URL: ${this.config.middlewareUrl}`,
            `   Total Requests: ${this.config.totalRequests || 'N/A'}`,
            `   Requests/Second: ${this.config.requestsPerSecond || 'N/A'}`,
            `   Duration: ${this.config.duration || 'N/A'}s`,
            `   Distribution: ${JSON.stringify(this.config.distribution || {})}`,
            `   Timeout: ${this.config.requestTimeout || 30000}ms`
        ];
        
        configDetails.forEach(line => {
            console.log(line);
            this.emitLog(line, 'info');
        });
        
        this.emit('test-progress', { status: 'started', config: this.config });
        
        try {
            const { testType, totalRequests, requestsPerSecond, duration, distribution } = this.config;
            
            const executionMessage = `🎯 Executing ${testType} test...`;
            console.log(`\n${executionMessage}`);
            this.emitLog(executionMessage, 'info');
            
            if (testType === 'burst') {
                const burstMessage = `🚀 Running BURST test: ${totalRequests} requests over ${duration}s`;
                console.log(burstMessage);
                this.emitLog(burstMessage, 'info');
                await this.runBurstTest(totalRequests, duration);
            } else if (testType === 'sustained') {
                const sustainedMessage = `⚡ Running SUSTAINED test: ${requestsPerSecond} RPS for ${duration}s`;
                console.log(sustainedMessage);
                this.emitLog(sustainedMessage, 'info');
                await this.runSustainedTest(requestsPerSecond, duration);
            } else if (testType === 'mixed') {
                const mixedMessage = `🎭 Running MIXED test: ${totalRequests} requests at ${requestsPerSecond} RPS`;
                console.log(mixedMessage);
                this.emitLog(mixedMessage, 'info');
                await this.runMixedTest(distribution, totalRequests, requestsPerSecond);
            } else if (testType === 'spike') {
                const spikeMessage = `📈 Running SPIKE test with pattern: ${JSON.stringify(this.config.spikePattern)}`;
                console.log(spikeMessage);
                this.emitLog(spikeMessage, 'info');
                await this.runSpikeTest(this.config.spikePattern);
            } else {
                throw new Error(`Unknown test type: ${testType}`);
            }
            
            this.results.endTime = new Date().toISOString();
            this.results.duration = Date.now() - this.startTime;
            this.calculateStatistics();
            
            const completionMessages = [
                `✅ ==================== LOAD TEST EXECUTION COMPLETE ====================`,
                `🕐 End Time: ${this.results.endTime}`,
                `⏱️ Total Duration: ${this.results.duration}ms (${(this.results.duration / 1000).toFixed(2)}s)`,
                `📊 Final Results Summary:`,
                `   Total Requests: ${this.results.totalRequests}`,
                `   Successful: ${this.results.successfulRequests} (${this.results.statistics.successRate.toFixed(2)}%)`,
                `   Failed: ${this.results.failedRequests} (${this.results.statistics.errorRate.toFixed(2)}%)`,
                `   Blocked: ${this.results.blockedRequests} (${this.results.statistics.blockRate.toFixed(2)}%)`,
                `   Avg Response Time: ${this.results.statistics.avgResponseTime.toFixed(2)}ms`,
                `   P95 Response Time: ${this.results.statistics.p95}ms`,
                `   Requests/Second: ${this.results.statistics.requestsPerSecond.toFixed(2)}`,
                `   Errors: ${this.results.errors.length}`
            ];
            
            completionMessages.forEach(line => {
                console.log(line);
                this.emitLog(line, 'success');
            });
            
            // Log error details if any
            if (this.results.errors.length > 0) {
                const errorHeader = `❌ ERROR DETAILS:`;
                console.log(`\n${errorHeader}`);
                this.emitLog(errorHeader, 'error');
                
                this.results.errors.slice(0, 10).forEach((error, idx) => {
                    const errorLine = `   [${idx + 1}] ${error.timestamp}: ${error.error || error.category} - ${error.message || 'No message'}`;
                    console.log(errorLine);
                    this.emitLog(errorLine, 'error');
                    
                    if (error.status) {
                        const statusLine = `       Status: ${error.status}`;
                        console.log(statusLine);
                        this.emitLog(statusLine, 'error');
                    }
                    if (error.expected && error.actual) {
                        const expectationLine = `       Expected: ${error.expected}, Got: ${error.actual}`;
                        console.log(expectationLine);
                        this.emitLog(expectationLine, 'error');
                    }
                });
                if (this.results.errors.length > 10) {
                    const moreErrorsLine = `   ... and ${this.results.errors.length - 10} more errors`;
                    console.log(moreErrorsLine);
                    this.emitLog(moreErrorsLine, 'error');
                }
            }
            
            // Log status code breakdown
            if (Object.keys(this.results.statusCodes).length > 0) {
                const statusHeader = `📈 STATUS CODE BREAKDOWN:`;
                console.log(`\n${statusHeader}`);
                this.emitLog(statusHeader, 'info');
                
                Object.entries(this.results.statusCodes).forEach(([code, count]) => {
                    const percentage = ((count / this.results.totalRequests) * 100).toFixed(2);
                    const statusLine = `   ${code}: ${count} requests (${percentage}%)`;
                    console.log(statusLine);
                    this.emitLog(statusLine, 'info');
                });
            }
            
            // Log category breakdown
            if (Object.keys(this.results.categories).length > 0) {
                const categoryHeader = `🏷️ CATEGORY BREAKDOWN:`;
                console.log(`\n${categoryHeader}`);
                this.emitLog(categoryHeader, 'info');
                
                Object.entries(this.results.categories).forEach(([category, stats]) => {
                    const categoryLine = `   ${category}: ${stats.total} total, ${stats.passed} passed, ${stats.blocked} blocked, ${stats.errors} errors`;
                    console.log(categoryLine);
                    this.emitLog(categoryLine, 'info');
                });
            }
            
            this.emit('test-progress', { 
                status: 'completed', 
                results: this.results 
            });
            
            return this.results;
            
        } catch (error) {
            const errorMessages = [
                `💥 ==================== LOAD TEST EXECUTION FAILED ====================`,
                `❌ Error Type: ${error.constructor.name}`,
                `❌ Error Message: ${error.message}`,
                `❌ Error Stack: ${error.stack}`,
                `📊 Partial Results at Failure:`,
                `   Requests Completed: ${this.results.totalRequests}`,
                `   Duration: ${Date.now() - this.startTime}ms`
            ];
            
            errorMessages.forEach(line => {
                console.error(line);
                this.emitLog(line, 'error');
            });
            
            this.results.errors.push({
                timestamp: new Date().toISOString(),
                error: error.message,
                stack: error.stack
            });
            this.emit('test-progress', { 
                status: 'error', 
                error: error.message 
            });
            throw error;
        } finally {
            this.isRunning = false;
            const endMessage = `🏁 ==================== LOAD TEST SESSION END ====================`;
            console.log(endMessage);
            this.emitLog(endMessage, 'info');
        }
    }

    async runBurstTest(totalRequests, duration) {
        const messages = this.generator.generateMixedBatch(
            this.config.distribution || { clean: 100 }, 
            totalRequests
        );
        
        const batchSize = Math.min(this.config.maxConcurrent || 100, totalRequests);
        const batches = Math.ceil(totalRequests / batchSize);
        const delayBetweenBatches = duration * 1000 / batches;
        
        for (let i = 0; i < batches && this.isRunning; i++) {
            const start = i * batchSize;
            const end = Math.min(start + batchSize, messages.length);
            const batch = messages.slice(start, end);
            
            await this.sendBatch(batch);
            
            if (i < batches - 1) {
                await this.delay(delayBetweenBatches);
            }
            
            this.emitProgress();
        }
    }

    async runSustainedTest(requestsPerSecond, durationSeconds) {
        const intervalMs = 1000 / requestsPerSecond;
        const endTime = Date.now() + (durationSeconds * 1000);
        
        while (Date.now() < endTime && this.isRunning) {
            const message = this.generateMessage();
            this.sendRequest(message);
            
            await this.delay(intervalMs);
            
            // Emit progress every second
            if (this.results.totalRequests % requestsPerSecond === 0) {
                this.emitProgress();
            }
        }
        
        // Wait for all pending requests
        while (this.concurrentRequests > 0) {
            await this.delay(100);
        }
    }

    async runMixedTest(distribution, totalRequests, requestsPerSecond) {
        const messages = this.generator.generateMixedBatch(distribution, totalRequests);
        const intervalMs = 1000 / requestsPerSecond;
        
        for (const messageData of messages) {
            if (!this.isRunning) break;
            
            await this.sendRequest(messageData);
            await this.delay(intervalMs);
            
            if (this.results.totalRequests % 10 === 0) {
                this.emitProgress();
            }
        }
        
        // Wait for all pending requests
        while (this.concurrentRequests > 0) {
            await this.delay(100);
        }
    }

    async runSpikeTest(spikePattern) {
        for (const phase of spikePattern) {
            if (!this.isRunning) break;
            
            console.log(`Spike phase: ${phase.requestsPerSecond} RPS for ${phase.duration}s`);
            await this.runSustainedTest(phase.requestsPerSecond, phase.duration);
        }
    }

    async sendBatch(messages) {
        const promises = messages.map(messageData => this.sendRequest(messageData));
        await Promise.allSettled(promises);
    }

    async sendRequest(messageData) {
        this.concurrentRequests++;
        this.results.totalRequests++;
        
        const requestId = `req_${this.results.totalRequests.toString().padStart(4, '0')}`;
        const startTime = Date.now();
        let payload = this.generator.generateTestPayload(
            messageData.message,
            this.config.payloadOverrides
        );
        
        const requestStartMessage = `📡 ${requestId}: ${messageData.category} | "${messageData.message.substring(0, 50)}${messageData.message.length > 50 ? '...' : ''}" | Expected: ${messageData.expectedResult}`;
        console.log(`\n📡 ==================== REQUEST ${requestId} START ====================`);
        this.emitLog(requestStartMessage, 'info');
        
        try {
            // Determine the correct endpoint and format based on the target
            let endpoint = '/chat'; // Default for QT-1 middleware
            
            // If testing directly against mock provider, use OpenAI format
            if (this.config.middlewareUrl.includes('8081')) {
                endpoint = '/v1/chat/completions';
                this.emitLog(`${requestId}: 🤖 Mock provider detected -> OpenAI format`, 'info');
                
                // Convert to OpenAI chat completions format
                payload = {
                    model: 'gpt-3.5-turbo',
                    messages: [
                        {
                            role: 'user',
                            content: messageData.message
                        }
                    ],
                    max_tokens: 1000,
                    temperature: 0.7
                };
            } else {
                this.emitLog(`${requestId}: 🛡️ Middleware detected -> QT-1 format`, 'info');
            }
            
            const httpStartTime = Date.now();
            this.emitLog(`${requestId}: 🚀 Sending POST to ${this.config.middlewareUrl}${endpoint}`, 'info');
            
            const response = await axios.post(
                `${this.config.middlewareUrl}${endpoint}`,
                payload,
                {
                    timeout: this.config.requestTimeout || 30000
                    // Removed validateStatus - now only 2xx codes are considered successful
                }
            );
            
            const httpTime = Date.now() - httpStartTime;
            const responseTime = Date.now() - startTime;
            this.results.responseTimes.push(responseTime);
            
            // Track status codes
            const status = response.status;
            this.results.statusCodes[status] = (this.results.statusCodes[status] || 0) + 1;
            
            const successMessage = `✅ ${requestId}: ${status} ${this.getStatusText(status)} | ${responseTime}ms | ${JSON.stringify(response.data).length} chars`;
            console.log(`\n✅ ==================== REQUEST ${requestId} SUCCESS ====================`);
            this.emitLog(successMessage, 'success');
            
            // Analyze response body for detailed console logging
            if (response.data) {
                const responseStr = JSON.stringify(response.data);
                console.log(`📊 Response Analysis:`);
                console.log(`   Status: ${status} ${this.getStatusText(status)}`);
                console.log(`   Headers: ${Object.keys(response.headers).length} header(s)`);
                console.log(`   Content-Type: ${response.headers['content-type'] || 'Unknown'}`);
                console.log(`   Content-Length: ${response.headers['content-length'] || 'Unknown'}`);
                console.log(`   Body Size: ${responseStr.length} chars`);
                console.log(`   Body Fields: [${Object.keys(response.data).join(', ')}]`);
                
                if (response.data.choices) {
                    console.log(`   OpenAI Choices: ${response.data.choices.length}`);
                    response.data.choices.forEach((choice, idx) => {
                        const content = choice.message?.content || choice.text || '';
                        console.log(`      [${idx}] ${choice.finish_reason}: "${content.substring(0, 50)}${content.length > 50 ? '...' : ''}"`);
                    });
                }
                
                if (response.data.usage) {
                    console.log(`   Token Usage: ${response.data.usage.total_tokens} (${response.data.usage.prompt_tokens}+${response.data.usage.completion_tokens})`);
                }
                
                if (response.data.error) {
                    console.log(`   ⚠️ Response contains error: ${response.data.error.message || 'Unknown error'}`);
                }
                
                if (responseStr.length < 500) {
                    console.log(`   📄 Full Response: ${responseStr}`);
                } else {
                    console.log(`   📄 Response Preview: ${responseStr.substring(0, 200)}... [TRUNCATED]`);
                }
            }
            
            // Track categories
            const category = messageData.category;
            if (!this.results.categories[category]) {
                this.results.categories[category] = {
                    total: 0,
                    passed: 0,
                    blocked: 0,
                    errors: 0
                };
            }
            this.results.categories[category].total++;
            
            if (status === 200) {
                this.results.successfulRequests++;
                this.results.categories[category].passed++;
                console.log(`   ✅ Request marked as SUCCESSFUL`);
            } else if (status === 403) {
                this.results.blockedRequests++;
                this.results.categories[category].blocked++;
                console.log(`   🚫 Request marked as BLOCKED`);
            } else {
                this.results.failedRequests++;
                this.results.categories[category].errors++;
                console.log(`   ❌ Request marked as FAILED (unexpected status)`);
            }
            
            // Check if result matches expectation
            const expectedBlock = messageData.expectedResult === 'block';
            const wasBlocked = status === 403;
            
            console.log(`🎯 Expectation Analysis:`);
            console.log(`   Expected: ${messageData.expectedResult}`);
            console.log(`   Actual: ${wasBlocked ? 'block' : 'pass'}`);
            console.log(`   Match: ${expectedBlock === wasBlocked ? '✅ YES' : '❌ NO'}`);
            
            if (expectedBlock !== wasBlocked) {
                const mismatchMessage = `⚠️ ${requestId}: EXPECTATION MISMATCH - Expected ${messageData.expectedResult}, got ${wasBlocked ? 'block' : 'pass'}`;
                console.log(`   ⚠️ EXPECTATION MISMATCH - Adding to error list`);
                this.emitLog(mismatchMessage, 'warning');
                
                this.results.errors.push({
                    timestamp: new Date().toISOString(),
                    category: messageData.category,
                    message: messageData.message.substring(0, 100) + '...',
                    expected: messageData.expectedResult,
                    actual: wasBlocked ? 'block' : 'pass',
                    status: status
                });
            }
            
        } catch (error) {
            const responseTime = Date.now() - startTime;
            this.results.responseTimes.push(responseTime);
            this.results.failedRequests++;
            
            const errorMessage = `💥 ${requestId}: ${error.constructor.name} | ${error.message} | ${responseTime}ms`;
            console.log(`\n💥 ==================== REQUEST ${requestId} FAILED ====================`);
            this.emitLog(errorMessage, 'error');
            
            // Detailed console logging for errors
            console.log(`❌ Error Analysis:`);
            console.log(`   Error Type: ${error.constructor.name}`);
            console.log(`   Error Code: ${error.code || 'Unknown'}`);
            console.log(`   Error Message: ${error.message}`);
            
            if (error.response) {
                console.log(`   HTTP Status: ${error.response.status} ${error.response.statusText}`);
                console.log(`   Response Headers: ${Object.keys(error.response.headers).length} header(s)`);
                if (error.response.data) {
                    const errorData = JSON.stringify(error.response.data);
                    console.log(`   Error Response: ${errorData.length > 200 ? errorData.substring(0, 200) + '...' : errorData}`);
                }
                
                // Emit detailed error for specific HTTP errors
                if (error.response.status) {
                    this.emitLog(`${requestId}: HTTP ${error.response.status} ${error.response.statusText}`, 'error');
                }
            } else if (error.request) {
                console.log(`   Network Error: No response received`);
                console.log(`   Request Details: ${error.request._currentUrl || 'Unknown URL'}`);
                this.emitLog(`${requestId}: Network error - no response received`, 'error');
            } else {
                console.log(`   Setup Error: ${error.message}`);
                this.emitLog(`${requestId}: Setup error - ${error.message}`, 'error');
            }
            
            if (error.stack) {
                console.log(`   Stack Trace: ${error.stack.split('\n').slice(0, 3).join(' | ')}`);
            }
            
            this.results.errors.push({
                timestamp: new Date().toISOString(),
                error: error.message,
                category: messageData.category,
                message: messageData.message.substring(0, 100) + '...',
                code: error.code,
                status: error.response?.status
            });
            
            if (this.results.categories[messageData.category]) {
                this.results.categories[messageData.category].errors++;
            }
        } finally {
            this.concurrentRequests--;
            const statsMessage = `📊 Stats: ${this.results.successfulRequests}✅ ${this.results.failedRequests}❌ ${this.results.blockedRequests}🚫 | Active: ${this.concurrentRequests}`;
            console.log(statsMessage);
            this.emitLog(statsMessage, 'info');
            console.log(`==================== REQUEST ${requestId} END ====================\n`);
        }
    }

    generateMessage() {
        const distribution = this.config.distribution || { clean: 100 };
        const categories = Object.entries(distribution);
        
        // Generate weighted random selection
        const random = Math.random() * 100;
        let cumulative = 0;
        
        for (const [category, percentage] of categories) {
            cumulative += percentage;
            if (random <= cumulative) {
                if (category === 'clean') {
                    return {
                        category: 'clean',
                        message: this.generator.generateCleanMessage(),
                        expectedResult: 'pass'
                    };
                } else {
                    const [mainCategory, subCategory] = category.split('.');
                    return {
                        category,
                        message: this.generator.generateTestMessage(mainCategory, subCategory),
                        expectedResult: this.generator.getExpectedResult(mainCategory)
                    };
                }
            }
        }
        
        // Fallback to clean message
        return {
            category: 'clean',
            message: this.generator.generateCleanMessage(),
            expectedResult: 'pass'
        };
    }

    calculateStatistics() {
        // Always calculate basic statistics
        const totalRequests = this.results.totalRequests || 0;
        const duration = this.results.duration || 1;
        
        this.results.statistics = {
            requestsPerSecond: totalRequests / (duration / 1000),
            successRate: totalRequests > 0 ? (this.results.successfulRequests / totalRequests) * 100 : 0,
            blockRate: totalRequests > 0 ? (this.results.blockedRequests / totalRequests) * 100 : 0,
            errorRate: totalRequests > 0 ? (this.results.failedRequests / totalRequests) * 100 : 0
        };
        
        // Calculate response time statistics only if we have data
        if (this.results.responseTimes.length > 0) {
            // Sort response times for percentile calculation
            const sorted = [...this.results.responseTimes].sort((a, b) => a - b);
            
            this.results.statistics.avgResponseTime = sorted.reduce((a, b) => a + b, 0) / sorted.length;
            this.results.statistics.minResponseTime = sorted[0];
            this.results.statistics.maxResponseTime = sorted[sorted.length - 1];
            this.results.statistics.p50 = sorted[Math.floor(sorted.length * 0.5)];
            this.results.statistics.p95 = sorted[Math.floor(sorted.length * 0.95)];
            this.results.statistics.p99 = sorted[Math.floor(sorted.length * 0.99)];
        } else {
            // Set default values when no response times available
            this.results.statistics.avgResponseTime = 0;
            this.results.statistics.minResponseTime = 0;
            this.results.statistics.maxResponseTime = 0;
            this.results.statistics.p50 = 0;
            this.results.statistics.p95 = 0;
            this.results.statistics.p99 = 0;
        }
    }

    emitProgress() {
        if (this.socket) {
            const progress = {
                totalRequests: this.results.totalRequests,
                successfulRequests: this.results.successfulRequests,
                failedRequests: this.results.failedRequests,
                blockedRequests: this.results.blockedRequests,
                concurrentRequests: this.concurrentRequests,
                elapsedTime: Date.now() - this.startTime,
                requestsPerSecond: this.results.totalRequests / ((Date.now() - this.startTime) / 1000)
            };
            
            this.emit('test-progress', { status: 'running', progress });
        }
    }

    emit(event, data) {
        if (this.socket) {
            this.socket.emit(event, data);
        }
    }

    stop() {
        this.isRunning = false;
        this.emit('test-progress', { status: 'stopped' });
    }

    getStatus() {
        return {
            isRunning: this.isRunning,
            progress: {
                totalRequests: this.results.totalRequests,
                successfulRequests: this.results.successfulRequests,
                failedRequests: this.results.failedRequests,
                blockedRequests: this.results.blockedRequests,
                elapsedTime: Date.now() - this.startTime
            }
        };
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    getStatusText(status) {
        const statusTexts = {
            200: 'OK',
            201: 'Created',
            400: 'Bad Request',
            401: 'Unauthorized',
            403: 'Forbidden',
            404: 'Not Found',
            429: 'Too Many Requests',
            500: 'Internal Server Error',
            502: 'Bad Gateway',
            503: 'Service Unavailable',
            504: 'Gateway Timeout'
        };
        return statusTexts[status] || 'Unknown';
    }

    emitLog(message, type = 'info') {
        if (this.socket) {
            this.socket.emit('test-log', { message, type, timestamp: new Date().toISOString() });
        }
    }
}

module.exports = LoadRunner;