const fs = require('fs').promises;
const path = require('path');

class HTMLReporter {
    async generate(results, testId) {
        const reportDir = process.env.REPORT_OUTPUT_DIR || './reports';
        await fs.mkdir(reportDir, { recursive: true });
        
        const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Load Test Report - ${testId}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        
        .header {
            background: #2c3e50;
            color: white;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .metric-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: #3498db;
        }
        
        .metric-label {
            color: #7f8c8d;
            font-size: 0.9em;
            margin-top: 5px;
        }
        
        .section {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        .section h2 {
            margin-top: 0;
            color: #2c3e50;
            border-bottom: 2px solid #ecf0f1;
            padding-bottom: 10px;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        
        th, td {
            padding: 10px;
            text-align: left;
            border-bottom: 1px solid #ecf0f1;
        }
        
        th {
            background: #ecf0f1;
            font-weight: 600;
        }
        
        .success { color: #27ae60; }
        .warning { color: #f39c12; }
        .danger { color: #e74c3c; }
        
        .progress-bar {
            width: 100%;
            height: 20px;
            background: #ecf0f1;
            border-radius: 10px;
            overflow: hidden;
            margin: 10px 0;
        }
        
        .progress-fill {
            height: 100%;
            background: #3498db;
            transition: width 0.3s ease;
        }
        
        .error-box {
            background: #fee;
            border: 1px solid #fcc;
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        
        .chart-container {
            height: 300px;
            margin: 20px 0;
        }
        
        .timestamp {
            color: #7f8c8d;
            font-size: 0.9em;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div class="header">
        <h1>Load Test Report</h1>
        <p>Test ID: ${testId}</p>
        <p>Generated: ${new Date().toISOString()}</p>
    </div>
    
    <div class="summary-grid">
        <div class="metric-card">
            <div class="metric-value">${results.totalRequests.toLocaleString()}</div>
            <div class="metric-label">Total Requests</div>
        </div>
        
        <div class="metric-card">
            <div class="metric-value ${results.statistics?.successRate > 95 ? 'success' : results.statistics?.successRate > 90 ? 'warning' : 'danger'}">
                ${results.statistics?.successRate.toFixed(1)}%
            </div>
            <div class="metric-label">Success Rate</div>
        </div>
        
        <div class="metric-card">
            <div class="metric-value">${results.statistics?.avgResponseTime.toFixed(0)}ms</div>
            <div class="metric-label">Avg Response Time</div>
        </div>
        
        <div class="metric-card">
            <div class="metric-value">${results.statistics?.requestsPerSecond.toFixed(1)}</div>
            <div class="metric-label">Requests/Second</div>
        </div>
    </div>
    
    <div class="section">
        <h2>Response Time Distribution</h2>
        <canvas id="responseTimeChart"></canvas>
    </div>
    
    <div class="section">
        <h2>Status Code Distribution</h2>
        <canvas id="statusCodeChart"></canvas>
    </div>
    
    <div class="section">
        <h2>Category Performance</h2>
        <table>
            <thead>
                <tr>
                    <th>Category</th>
                    <th>Total</th>
                    <th>Passed</th>
                    <th>Blocked</th>
                    <th>Errors</th>
                    <th>Success Rate</th>
                </tr>
            </thead>
            <tbody>
                ${Object.entries(results.categories).map(([category, stats]) => `
                    <tr>
                        <td><strong>${category}</strong></td>
                        <td>${stats.total}</td>
                        <td class="success">${stats.passed}</td>
                        <td class="warning">${stats.blocked}</td>
                        <td class="danger">${stats.errors}</td>
                        <td>
                            <div class="progress-bar">
                                <div class="progress-fill" style="width: ${(stats.passed / stats.total * 100)}%"></div>
                            </div>
                            ${((stats.passed / stats.total) * 100).toFixed(1)}%
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    </div>
    
    <div class="section">
        <h2>Performance Metrics</h2>
        <table>
            <tr>
                <td><strong>Min Response Time:</strong></td>
                <td>${results.statistics?.minResponseTime}ms</td>
                <td><strong>P50 (Median):</strong></td>
                <td>${results.statistics?.p50}ms</td>
            </tr>
            <tr>
                <td><strong>Max Response Time:</strong></td>
                <td>${results.statistics?.maxResponseTime}ms</td>
                <td><strong>P95:</strong></td>
                <td>${results.statistics?.p95}ms</td>
            </tr>
            <tr>
                <td><strong>Test Duration:</strong></td>
                <td>${(results.duration / 1000).toFixed(2)}s</td>
                <td><strong>P99:</strong></td>
                <td>${results.statistics?.p99}ms</td>
            </tr>
        </table>
    </div>
    
    ${results.errors.length > 0 ? `
    <div class="section">
        <h2>Errors (${results.errors.length} total)</h2>
        ${results.errors.slice(0, 20).map(error => `
            <div class="error-box">
                <div class="timestamp">${error.timestamp}</div>
                <strong>Category:</strong> ${error.category || 'N/A'}<br>
                <strong>Error:</strong> ${error.error || error.message}<br>
                ${error.expected ? `<strong>Expected:</strong> ${error.expected}, <strong>Actual:</strong> ${error.actual}` : ''}
            </div>
        `).join('')}
        ${results.errors.length > 20 ? `<p>... and ${results.errors.length - 20} more errors</p>` : ''}
    </div>
    ` : ''}
    
    <script>
        // Response Time Chart
        const rtCtx = document.getElementById('responseTimeChart').getContext('2d');
        const responseTimes = ${JSON.stringify(results.responseTimes || [])};
        const rtBuckets = createHistogram(responseTimes, 20);
        
        new Chart(rtCtx, {
            type: 'bar',
            data: {
                labels: rtBuckets.labels,
                datasets: [{
                    label: 'Response Time Distribution',
                    data: rtBuckets.data,
                    backgroundColor: '#3498db'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Number of Requests'
                        }
                    },
                    x: {
                        title: {
                            display: true,
                            text: 'Response Time (ms)'
                        }
                    }
                }
            }
        });
        
        // Status Code Chart
        const scCtx = document.getElementById('statusCodeChart').getContext('2d');
        const statusCodes = ${JSON.stringify(results.statusCodes || {})};
        
        new Chart(scCtx, {
            type: 'doughnut',
            data: {
                labels: Object.keys(statusCodes),
                datasets: [{
                    data: Object.values(statusCodes),
                    backgroundColor: [
                        '#27ae60', // 200
                        '#f39c12', // 403
                        '#e74c3c', // 500
                        '#9b59b6', // Others
                        '#34495e'
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'right'
                    }
                }
            }
        });
        
        function createHistogram(data, bins) {
            if (!data.length) return { labels: [], data: [] };
            
            const min = Math.min(...data);
            const max = Math.max(...data);
            const binWidth = (max - min) / bins;
            const histogram = new Array(bins).fill(0);
            const labels = [];
            
            for (let i = 0; i < bins; i++) {
                const start = Math.round(min + i * binWidth);
                const end = Math.round(min + (i + 1) * binWidth);
                labels.push(\`\${start}-\${end}\`);
            }
            
            data.forEach(value => {
                const binIndex = Math.min(Math.floor((value - min) / binWidth), bins - 1);
                histogram[binIndex]++;
            });
            
            return { labels, data: histogram };
        }
    </script>
</body>
</html>
        `;
        
        const reportPath = path.join(reportDir, `${testId}.html`);
        await fs.writeFile(reportPath, html);
        console.log(`HTML report generated: ${reportPath}`);
    }
}

module.exports = HTMLReporter;