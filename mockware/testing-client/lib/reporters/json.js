const fs = require('fs').promises;
const path = require('path');

class JSONReporter {
    async generate(results, testId) {
        const reportDir = process.env.REPORT_OUTPUT_DIR || './reports';
        await fs.mkdir(reportDir, { recursive: true });
        
        const report = {
            testId,
            timestamp: new Date().toISOString(),
            summary: {
                totalRequests: results.totalRequests,
                successfulRequests: results.successfulRequests,
                blockedRequests: results.blockedRequests,
                failedRequests: results.failedRequests,
                duration: results.duration,
                startTime: results.startTime,
                endTime: results.endTime
            },
            statistics: results.statistics || {},
            statusCodes: results.statusCodes || {},
            categories: results.categories || {},
            errors: results.errors || [],
            responseTimes: {
                raw: results.responseTimes || [],
                histogram: this.createHistogram(results.responseTimes || [], 50)
            },
            metadata: {
                generatedAt: new Date().toISOString(),
                reportVersion: '1.0',
                testFramework: 'QT-1 Mockware'
            }
        };
        
        const reportPath = path.join(reportDir, `${testId}.json`);
        await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
        console.log(`JSON report generated: ${reportPath}`);
        
        // Also create a summary file for quick access
        const summaryPath = path.join(reportDir, `${testId}-summary.json`);
        const summary = {
            testId,
            timestamp: report.timestamp,
            totalRequests: report.summary.totalRequests,
            successRate: results.statistics?.successRate || 0,
            avgResponseTime: results.statistics?.avgResponseTime || 0,
            requestsPerSecond: results.statistics?.requestsPerSecond || 0,
            duration: report.summary.duration,
            errorCount: report.errors.length
        };
        
        await fs.writeFile(summaryPath, JSON.stringify(summary, null, 2));
    }
    
    createHistogram(data, bins) {
        if (!data.length) return { bins: [], counts: [] };
        
        const min = Math.min(...data);
        const max = Math.max(...data);
        const binWidth = (max - min) / bins;
        const histogram = new Array(bins).fill(0);
        const binEdges = [];
        
        for (let i = 0; i <= bins; i++) {
            binEdges.push(min + i * binWidth);
        }
        
        data.forEach(value => {
            const binIndex = Math.min(Math.floor((value - min) / binWidth), bins - 1);
            histogram[binIndex]++;
        });
        
        return {
            binEdges,
            counts: histogram,
            binWidth,
            min,
            max
        };
    }
}

module.exports = JSONReporter;