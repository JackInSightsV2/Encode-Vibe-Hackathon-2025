const express = require('express');
const socketIo = require('socket.io');
const cors = require('cors');
const path = require('path');
const bodyParser = require('body-parser');
const { v4: uuidv4 } = require('uuid');
require('dotenv').config();

const LoadRunner = require('./lib/load-runner');
const TestGenerator = require('./lib/test-generator');
const { ConsoleReporter, HTMLReporter, JSONReporter } = require('./lib/reporters');

const app = express();
const server = require('http').createServer(app);
const io = socketIo(server, { 
    cors: { 
        origin: "*",
        methods: ["GET", "POST"]
    } 
});

// Middleware
app.use(cors());
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));
app.use(express.static('public'));

// Store active tests
const activeTests = new Map();

// WebSocket connection handling
io.on('connection', (socket) => {
    console.log('Client connected:', socket.id);
    
    socket.on('start-test', async (config) => {
        try {
            const testId = uuidv4();
            const testConfig = {
                ...config,
                testId,
                middlewareUrl: process.env.MIDDLEWARE_URL,
                maxConcurrent: parseInt(process.env.MAX_CONCURRENT_REQUESTS),
                requestTimeout: parseInt(process.env.REQUEST_TIMEOUT)
            };
            
            console.log('Starting test:', testId, testConfig);
            
            // Create load runner with socket for real-time updates
            const runner = new LoadRunner(testConfig, socket);
            activeTests.set(testId, runner);
            
            // Start the test
            socket.emit('test-started', { testId });
            
            const results = await runner.execute();
            
            // Generate reports
            const reporters = [
                new ConsoleReporter(),
                new HTMLReporter(),
                new JSONReporter()
            ];
            
            for (const reporter of reporters) {
                await reporter.generate(results, testId);
            }
            
            socket.emit('test-completed', { testId, results });
            activeTests.delete(testId);
            
        } catch (error) {
            console.error('Test execution error:', error);
            socket.emit('test-error', { 
                error: error.message,
                stack: error.stack 
            });
        }
    });
    
    socket.on('stop-test', (testId) => {
        const runner = activeTests.get(testId);
        if (runner) {
            runner.stop();
            activeTests.delete(testId);
            socket.emit('test-stopped', { testId });
        }
    });
    
    socket.on('disconnect', () => {
        console.log('Client disconnected:', socket.id);
    });
});

// REST API endpoints
app.post('/api/test/start', async (req, res) => {
    try {
        const testId = uuidv4();
        const testConfig = {
            ...req.body,
            testId,
            middlewareUrl: process.env.MIDDLEWARE_URL,
            maxConcurrent: parseInt(process.env.MAX_CONCURRENT_REQUESTS),
            requestTimeout: parseInt(process.env.REQUEST_TIMEOUT)
        };
        
        const runner = new LoadRunner(testConfig);
        activeTests.set(testId, runner);
        
        // Start test asynchronously
        runner.execute().then(results => {
            activeTests.delete(testId);
        }).catch(error => {
            console.error('Test error:', error);
            activeTests.delete(testId);
        });
        
        res.json({ testId, status: 'started' });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/test/status/:id', (req, res) => {
    const runner = activeTests.get(req.params.id);
    if (runner) {
        res.json(runner.getStatus());
    } else {
        res.status(404).json({ error: 'Test not found' });
    }
});

app.get('/api/test/results/:id', (req, res) => {
    const resultsPath = path.join(process.env.REPORT_OUTPUT_DIR || './reports', `${req.params.id}.json`);
    res.sendFile(resultsPath, { root: __dirname }, (err) => {
        if (err) {
            console.error('Error serving JSON report:', err);
            res.status(404).json({ error: 'Report not found' });
        }
    });
});

app.get('/api/test/results/:id.html', (req, res) => {
    const resultsPath = path.join(process.env.REPORT_OUTPUT_DIR || './reports', `${req.params.id}.html`);
    res.sendFile(resultsPath, { root: __dirname }, (err) => {
        if (err) {
            console.error('Error serving HTML report:', err);
            res.status(404).json({ error: 'Report not found' });
        }
    });
});

app.get('/api/test/results/:id.json', (req, res) => {
    const resultsPath = path.join(process.env.REPORT_OUTPUT_DIR || './reports', `${req.params.id}.json`);
    res.sendFile(resultsPath, { root: __dirname }, (err) => {
        if (err) {
            console.error('Error serving JSON report:', err);
            res.status(404).json({ error: 'Report not found' });
        }
    });
});

app.get('/api/test/active', (req, res) => {
    const active = Array.from(activeTests.keys()).map(id => ({
        id,
        status: activeTests.get(id).getStatus()
    }));
    res.json(active);
});

app.get('/api/scenarios', (req, res) => {
    const generator = new TestGenerator();
    res.json(generator.getAvailableScenarios());
});

app.get('/api/profiles', (req, res) => {
    const profiles = require('./config/test-profiles.json');
    res.json(profiles);
});

// Test data management endpoints
app.get('/api/test-data', (req, res) => {
    try {
        const fs = require('fs');
        const testData = {};
        
        // Load all scenario files
        const scenarioDir = path.join(__dirname, 'lib', 'scenarios');
        const files = fs.readdirSync(scenarioDir);
        
        files.forEach(file => {
            if (file.endsWith('.js')) {
                const category = file.replace('.js', '');
                try {
                    delete require.cache[require.resolve(path.join(scenarioDir, file))];
                    testData[category] = require(path.join(scenarioDir, file));
                } catch (error) {
                    console.error(`Error loading ${file}:`, error);
                }
            }
        });
        
        res.json(testData);
    } catch (error) {
        console.error('Error loading test data:', error);
        res.status(500).json({ error: 'Failed to load test data' });
    }
});

app.post('/api/test-data', (req, res) => {
    try {
        const fs = require('fs');
        const testData = req.body;
        
        // Save each category to its respective file
        const scenarioDir = path.join(__dirname, 'lib', 'scenarios');
        
        Object.keys(testData).forEach(category => {
            const filePath = path.join(scenarioDir, `${category}.js`);
            const data = testData[category];
            
            // Generate JavaScript module content
            const moduleContent = `module.exports = ${JSON.stringify(data, null, 4)};`;
            
            // Write to file
            fs.writeFileSync(filePath, moduleContent, 'utf8');
            
            // Clear require cache so changes are reflected immediately
            delete require.cache[require.resolve(filePath)];
        });
        
        res.json({ success: true, message: 'Test data saved successfully' });
    } catch (error) {
        console.error('Error saving test data:', error);
        res.status(500).json({ error: 'Failed to save test data' });
    }
});

// Health check
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'QT-1 Load Testing Client',
        version: '1.0.1-hotfix',
        activeTests: activeTests.size,
        uptime: process.uptime(),
        timestamp: new Date().toISOString()
    });
});

// Version info endpoint
app.get('/version', (req, res) => {
    res.json({
        service: 'QT-1 Load Testing Client',
        version: '1.0.1-hotfix',
        build: '2025-01-21-endpoint-fix',
        commit: 'load-testing-mock-provider-fix',
        node_version: process.version,
        uptime: process.uptime(),
        environment: process.env.NODE_ENV || 'development',
        features: [
            'Socket.IO Real-time Updates',
            'Multiple Test Patterns (Burst, Sustained, Mixed, Spike)',
            'Mock Provider Auto-Detection (Port 8081)',
            'OpenAI Format Auto-Conversion',
            'Comprehensive Reporting (HTML, JSON, Console)',
            'Real-time Metrics & Charts',
            '500+ Test Scenarios'
        ],
        fixes: [
            'Mock Provider endpoint detection',
            'OpenAI format conversion for port 8081',
            'Statistics calculation with zero response times',
            'Proper error handling and validation'
        ],
        last_updated: new Date().toISOString(),
        active_tests: activeTests.size
    });
});

const PORT = process.env.PORT || 3000;
server.listen(PORT, () => {
    console.log(`🧪 QT-1 Load Testing Client v1.0.1-hotfix running on port ${PORT}`);
    console.log(`🔧 Build: 2025-01-21-endpoint-fix`);
    console.log(`📊 Features: Mock Provider Auto-Detection, OpenAI Format Conversion`);
    console.log(`🌐 Dashboard: http://localhost:${PORT}`);
    console.log(`🔍 Version info: http://localhost:${PORT}/version`);
    console.log(`💊 Health check: http://localhost:${PORT}/health`);
    console.log(`🎯 Configuration:`);
    console.log(`  - Default Middleware: ${process.env.MIDDLEWARE_URL || 'http://localhost:8080'}`);
    console.log(`  - Mock Provider: ${process.env.MOCK_PROVIDER_URL || 'http://localhost:8081'}`);
    console.log(`  - Max Concurrent: ${process.env.MAX_CONCURRENT_REQUESTS || 100}`);
});