// Global variables
let socket;
let currentTestId = null;
let responseTimeChart = null;
let testProfiles = {};

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    initializeSocket();
    loadTestProfiles();
    initializeChart();
    updateDistributionTotal();
    initializeStatusIndicators();
    startStatusUpdateTimer();
});

// Initialize Socket.IO connection
function initializeSocket() {
    socket = io();
    
    socket.on('connect', () => {
        addLog('Connected to test server', 'success');
        updateWebSocketStatus(true);
    });
    
    socket.on('disconnect', () => {
        addLog('Disconnected from test server', 'error');
        updateWebSocketStatus(false);
    });
    
    socket.on('test-started', (data) => {
        currentTestId = data.testId;
        updateStatus('running', 'Test started: ' + data.testId);
        document.getElementById('startTest').disabled = true;
        document.getElementById('stopTest').disabled = false;
        document.getElementById('testProgress').style.display = 'block';
    });
    
    socket.on('test-log', (data) => {
        // Handle detailed logs from the load runner
        addLog(data.message, data.type);
    });
    
    socket.on('test-progress', (data) => {
        if (data.status === 'running' && data.progress) {
            updateProgress(data.progress);
            updateMetrics(data.progress);
        } else if (data.status === 'completed') {
            updateStatus('completed', 'Test completed successfully');
            displayResults(data.results);
            resetTestControls();
        } else if (data.status === 'error') {
            updateStatus('error', 'Test failed: ' + data.error);
            resetTestControls();
        }
    });
    
    socket.on('test-error', (data) => {
        addLog('Test error: ' + data.error, 'error');
        updateStatus('error', 'Test failed');
        resetTestControls();
    });
}

// Load test profiles
async function loadTestProfiles() {
    try {
        const response = await fetch('/api/profiles');
        const data = await response.json();
        testProfiles = data.profiles;
        
        const select = document.getElementById('testProfile');
        Object.entries(testProfiles).forEach(([key, profile]) => {
            const option = document.createElement('option');
            option.value = key;
            option.textContent = profile.name;
            option.title = profile.description;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load test profiles:', error);
    }
}

// Load selected profile
function loadProfile() {
    const profileKey = document.getElementById('testProfile').value;
    if (!profileKey) return;
    
    const profile = testProfiles[profileKey];
    if (!profile) return;
    
    // Set test type
    document.getElementById('testType').value = profile.testType;
    updateTestTypeOptions();
    
    // Set test parameters
    if (profile.totalRequests) {
        document.getElementById('totalRequests').value = profile.totalRequests;
    }
    if (profile.duration) {
        document.getElementById('duration').value = profile.duration;
        document.getElementById('burstDuration').value = profile.duration;
    }
    if (profile.requestsPerSecond) {
        document.getElementById('requestsPerSecond').value = profile.requestsPerSecond;
    }
    
    // Set distribution
    if (profile.distribution) {
        // Clear all distribution inputs first
        document.querySelectorAll('[id^="dist-"]').forEach(input => {
            input.value = 0;
        });
        
        // Set profile distribution values
        Object.entries(profile.distribution).forEach(([category, percentage]) => {
            const input = document.getElementById('dist-' + category);
            if (input) {
                input.value = percentage;
            }
        });
        
        updateDistributionTotal();
    }
}

// Update test type options visibility
function updateTestTypeOptions() {
    const testType = document.getElementById('testType').value;
    
    document.getElementById('burstOptions').style.display = 
        testType === 'burst' ? 'block' : 'none';
    document.getElementById('sustainedOptions').style.display = 
        testType === 'sustained' ? 'block' : 'none';
}

// Update distribution total
function updateDistributionTotal() {
    const inputs = document.querySelectorAll('[id^="dist-"]');
    let total = 0;
    
    inputs.forEach(input => {
        total += parseInt(input.value) || 0;
    });
    
    document.getElementById('distributionTotal').textContent = total;
    const warning = document.getElementById('distributionWarning');
    warning.style.display = total !== 100 ? 'inline' : 'none';
}

// Start test
function startTest() {
    const distribution = {};
    document.querySelectorAll('[id^="dist-"]').forEach(input => {
        const category = input.id.replace('dist-', '');
        const value = parseInt(input.value) || 0;
        if (value > 0) {
            distribution[category] = value;
        }
    });
    
    const total = Object.values(distribution).reduce((sum, val) => sum + val, 0);
    if (total !== 100) {
        alert('Distribution must total 100%');
        return;
    }
    
    const testType = document.getElementById('testType').value;
    const config = {
        testType,
        distribution,
        middlewareUrl: document.getElementById('middlewareUrl').value
    };
    
    // Add type-specific parameters
    if (testType === 'burst') {
        config.totalRequests = parseInt(document.getElementById('totalRequests').value);
        config.duration = parseInt(document.getElementById('burstDuration').value);
    } else if (testType === 'sustained') {
        config.requestsPerSecond = parseInt(document.getElementById('requestsPerSecond').value);
        config.duration = parseInt(document.getElementById('duration').value);
    } else if (testType === 'mixed') {
        config.totalRequests = parseInt(document.getElementById('totalRequests').value) || 1000;
        config.requestsPerSecond = parseInt(document.getElementById('requestsPerSecond').value) || 10;
    }
    
    // Clear previous results
    document.getElementById('resultsPanel').style.display = 'none';
    document.getElementById('logContainer').innerHTML = '';
    resetChart();
    
    addLog('Starting test with config: ' + JSON.stringify(config, null, 2), 'info');
    socket.emit('start-test', config);
}

// Stop test
function stopTest() {
    if (currentTestId) {
        socket.emit('stop-test', currentTestId);
        addLog('Stopping test...', 'warning');
    }
}

// Reset test controls
function resetTestControls() {
    document.getElementById('startTest').disabled = false;
    document.getElementById('stopTest').disabled = true;
    document.getElementById('testProgress').style.display = 'none';
    currentTestId = null;
}

// Update status display
function updateStatus(status, message) {
    const statusEl = document.getElementById('testStatus');
    statusEl.className = 'status-' + status;
    
    const icons = {
        idle: '⏸',
        running: '▶',
        completed: '✅',
        error: '❌'
    };
    
    statusEl.innerHTML = `
        <span class="status-icon">${icons[status] || '?'}</span>
        <span class="status-text">${message || status}</span>
    `;
}

// Update progress bar
function updateProgress(progress) {
    const percentage = Math.round((progress.totalRequests / (progress.totalRequests + 100)) * 100);
    const progressFill = document.querySelector('.progress-fill');
    const progressText = document.querySelector('.progress-text');
    
    progressFill.style.width = percentage + '%';
    progressText.textContent = percentage + '%';
}

// Update metrics display
function updateMetrics(progress) {
    document.getElementById('totalRequestsMetric').textContent = 
        progress.totalRequests.toLocaleString();
    
    const successRate = progress.totalRequests > 0 
        ? ((progress.successfulRequests / progress.totalRequests) * 100).toFixed(1)
        : 0;
    document.getElementById('successRateMetric').textContent = successRate + '%';
    
    document.getElementById('rpsMetric').textContent = 
        progress.requestsPerSecond.toFixed(1);
    
    // Update chart with mock data (real implementation would track actual response times)
    if (responseTimeChart) {
        const timestamp = new Date().toLocaleTimeString();
        responseTimeChart.data.labels.push(timestamp);
        responseTimeChart.data.datasets[0].data.push(Math.random() * 100 + 50);
        
        // Keep only last 20 data points
        if (responseTimeChart.data.labels.length > 20) {
            responseTimeChart.data.labels.shift();
            responseTimeChart.data.datasets[0].data.shift();
        }
        
        responseTimeChart.update();
    }
}

// Initialize Chart.js
function initializeChart() {
    const ctx = document.getElementById('responseTimeChart').getContext('2d');
    responseTimeChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Response Time (ms)',
                data: [],
                borderColor: '#3498db',
                backgroundColor: 'rgba(52, 152, 219, 0.1)',
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Response Time (ms)'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'Time'
                    }
                }
            }
        }
    });
}

// Reset chart
function resetChart() {
    if (responseTimeChart) {
        responseTimeChart.data.labels = [];
        responseTimeChart.data.datasets[0].data = [];
        responseTimeChart.update();
    }
}

// Display test results
function displayResults(results) {
    const resultsPanel = document.getElementById('resultsPanel');
    const testResults = document.getElementById('testResults');
    
    resultsPanel.style.display = 'block';
    
    testResults.innerHTML = `
        <div class="results-summary">
            <h3>Test Summary</h3>
            <div class="results-grid">
                <div class="result-item">
                    <div class="result-label">Total Requests</div>
                    <div class="result-value">${results.totalRequests.toLocaleString()}</div>
                </div>
                <div class="result-item">
                    <div class="result-label">Success Rate</div>
                    <div class="result-value">${results.statistics?.successRate.toFixed(2)}%</div>
                </div>
                <div class="result-item">
                    <div class="result-label">Average Response Time</div>
                    <div class="result-value">${results.statistics?.avgResponseTime.toFixed(0)}ms</div>
                </div>
                <div class="result-item">
                    <div class="result-label">Requests/Second</div>
                    <div class="result-value">${results.statistics?.requestsPerSecond.toFixed(2)}</div>
                </div>
                <div class="result-item">
                    <div class="result-label">P95 Response Time</div>
                    <div class="result-value">${results.statistics?.p95}ms</div>
                </div>
                <div class="result-item">
                    <div class="result-label">Errors</div>
                    <div class="result-value">${results.errors.length}</div>
                </div>
            </div>
        </div>
    `;
    
    addLog('Test completed successfully. View full report above.', 'success');
}

// Add log entry
function addLog(message, type = 'info') {
    const logContainer = document.getElementById('logContainer');
    const timestamp = new Date().toLocaleTimeString();
    
    const entry = document.createElement('div');
    entry.className = `log-entry log-${type}`;
    entry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span> ${message}`;
    
    logContainer.appendChild(entry);
    
    if (document.getElementById('autoScroll').checked) {
        logContainer.scrollTop = logContainer.scrollHeight;
    }
}

// Clear logs
function clearLogs() {
    document.getElementById('logContainer').innerHTML = '';
}

// Download report
function downloadReport(format) {
    if (!currentTestId) {
        alert('No test results available');
        return;
    }
    
    // Create a download URL for the specific format
    const baseUrl = window.location.origin;
    let url;
    
    if (format === 'html') {
        url = `${baseUrl}/api/test/results/${currentTestId}.html`;
    } else if (format === 'json') {
        url = `${baseUrl}/api/test/results/${currentTestId}.json`;
    } else {
        alert('Invalid format requested');
        return;
    }
    
    // Try to download the file
    fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            return response.blob();
        })
        .then(blob => {
            // Create a download link
            const downloadUrl = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = downloadUrl;
            link.download = `test-report-${currentTestId}.${format}`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            window.URL.revokeObjectURL(downloadUrl);
            
            addLog(`Downloaded ${format.toUpperCase()} report: test-report-${currentTestId}.${format}`, 'success');
        })
        .catch(error => {
            console.error('Download failed:', error);
            addLog(`Download failed: ${error.message}`, 'error');
            
            // Fallback: try opening in new tab
            try {
                window.open(url, '_blank');
                addLog(`Opened report in new tab (download failed)`, 'warning');
            } catch (fallbackError) {
                addLog(`Both download and fallback failed: ${fallbackError.message}`, 'error');
            }
        });
}

// Attach event listeners to distribution inputs
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[id^="dist-"]').forEach(input => {
        input.addEventListener('input', updateDistributionTotal);
    });
});

// Initialize status indicators
function initializeStatusIndicators() {
    // Set browser info
    const browserInfo = navigator.userAgent.includes('Chrome') ? 'Chrome' : 
                       navigator.userAgent.includes('Firefox') ? 'Firefox' :
                       navigator.userAgent.includes('Safari') ? 'Safari' : 'Unknown';
    document.getElementById('browserInfo').textContent = browserInfo;
    
    // Check mock provider status
    checkMockProviderStatus();
}

// Start status update timer
let startTime = Date.now();
function startStatusUpdateTimer() {
    setInterval(() => {
        updateUptime();
        updateLastUpdateTime();
        checkMockProviderStatus();
    }, 1000);
}

// Update uptime display
function updateUptime() {
    const uptime = Date.now() - startTime;
    const hours = Math.floor(uptime / 3600000);
    const minutes = Math.floor((uptime % 3600000) / 60000);
    const seconds = Math.floor((uptime % 60000) / 1000);
    
    const formattedUptime = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    document.getElementById('uptimeDisplay').textContent = formattedUptime;
}

// Update last update time
function updateLastUpdateTime() {
    const now = new Date();
    const timeString = now.toLocaleTimeString();
    document.getElementById('lastUpdateTime').textContent = timeString;
}

// Check mock provider status
async function checkMockProviderStatus() {
    try {
        const response = await fetch('http://localhost:8081/health');
        const data = await response.json();
        
        if (response.ok && data.status === 'healthy') {
            document.getElementById('mockProviderStatus').textContent = '🟢';
            document.getElementById('mockProviderVersion').textContent = `v${data.version || '1.0.0'} ✓`;
        } else {
            document.getElementById('mockProviderStatus').textContent = '🟡';
            document.getElementById('mockProviderVersion').textContent = 'Unhealthy';
        }
    } catch (error) {
        document.getElementById('mockProviderStatus').textContent = '🔴';
        document.getElementById('mockProviderVersion').textContent = 'Disconnected';
    }
}

// Update WebSocket status
function updateWebSocketStatus(connected) {
    document.getElementById('websocketStatus').textContent = connected ? 'Connected ✓' : 'Disconnected ✗';
}