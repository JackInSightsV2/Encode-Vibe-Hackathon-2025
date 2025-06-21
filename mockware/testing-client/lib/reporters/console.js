class ConsoleReporter {
    async generate(results, testId) {
        console.log('\n' + '='.repeat(80));
        console.log(`LOAD TEST RESULTS - ${testId}`);
        console.log('='.repeat(80));
        
        // Summary
        console.log('\nSUMMARY:');
        console.log(`Total Requests: ${results.totalRequests}`);
        console.log(`Successful: ${results.successfulRequests} (${((results.successfulRequests / results.totalRequests) * 100).toFixed(2)}%)`);
        console.log(`Blocked: ${results.blockedRequests} (${((results.blockedRequests / results.totalRequests) * 100).toFixed(2)}%)`);
        console.log(`Failed: ${results.failedRequests} (${((results.failedRequests / results.totalRequests) * 100).toFixed(2)}%)`);
        console.log(`Duration: ${(results.duration / 1000).toFixed(2)}s`);
        
        // Performance metrics
        if (results.statistics) {
            console.log('\nPERFORMANCE:');
            console.log(`Requests/Second: ${results.statistics.requestsPerSecond.toFixed(2)}`);
            console.log(`Avg Response Time: ${results.statistics.avgResponseTime.toFixed(2)}ms`);
            console.log(`Min Response Time: ${results.statistics.minResponseTime}ms`);
            console.log(`Max Response Time: ${results.statistics.maxResponseTime}ms`);
            console.log(`P50: ${results.statistics.p50}ms`);
            console.log(`P95: ${results.statistics.p95}ms`);
            console.log(`P99: ${results.statistics.p99}ms`);
        }
        
        // Status code distribution
        console.log('\nSTATUS CODES:');
        for (const [code, count] of Object.entries(results.statusCodes)) {
            console.log(`${code}: ${count} (${((count / results.totalRequests) * 100).toFixed(2)}%)`);
        }
        
        // Category breakdown
        console.log('\nCATEGORY BREAKDOWN:');
        for (const [category, stats] of Object.entries(results.categories)) {
            console.log(`\n${category.toUpperCase()}:`);
            console.log(`  Total: ${stats.total}`);
            console.log(`  Passed: ${stats.passed} (${((stats.passed / stats.total) * 100).toFixed(2)}%)`);
            console.log(`  Blocked: ${stats.blocked} (${((stats.blocked / stats.total) * 100).toFixed(2)}%)`);
            console.log(`  Errors: ${stats.errors} (${((stats.errors / stats.total) * 100).toFixed(2)}%)`);
        }
        
        // Errors
        if (results.errors.length > 0) {
            console.log('\nERRORS (first 10):');
            results.errors.slice(0, 10).forEach((error, index) => {
                console.log(`\n${index + 1}. ${error.timestamp}`);
                console.log(`   Category: ${error.category || 'N/A'}`);
                console.log(`   Message: ${error.message || error.error}`);
                if (error.expected && error.actual) {
                    console.log(`   Expected: ${error.expected}, Actual: ${error.actual}`);
                }
            });
            
            if (results.errors.length > 10) {
                console.log(`\n... and ${results.errors.length - 10} more errors`);
            }
        }
        
        console.log('\n' + '='.repeat(80));
        console.log('Test completed at:', new Date().toISOString());
        console.log('='.repeat(80) + '\n');
    }
}

module.exports = ConsoleReporter;