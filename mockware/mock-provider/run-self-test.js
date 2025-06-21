#!/usr/bin/env node

const { spawn } = require('child_process');
const { performance } = require('perf_hooks');

class SelfTestRunner {
    constructor() {
        this.serverProcess = null;
        this.isServerReady = false;
    }

    async runCompleteTest() {
        console.log('🚀 QT-1 Mock Provider - Complete Self-Test');
        console.log('==========================================');
        console.log('');

        try {
            // Start the server
            await this.startServer();
            
            // Wait a moment for server to fully initialize
            await this.sleep(2000);
            
            // Run the self-test
            await this.runSelfTest();
            
        } catch (error) {
            console.error('❌ Test runner failed:', error.message);
            process.exit(1);
        } finally {
            // Always cleanup
            await this.stopServer();
        }
        
        console.log('\\n🎯 Complete self-test finished successfully!');
    }

    async startServer() {
        console.log('🔄 Starting mock provider server...');
        
        return new Promise((resolve, reject) => {
            // Start server process
            this.serverProcess = spawn('node', ['server.js'], {
                stdio: ['ignore', 'pipe', 'pipe'],
                cwd: __dirname
            });

            let output = '';
            
            // Capture stdout
            this.serverProcess.stdout.on('data', (data) => {
                const text = data.toString();
                output += text;
                
                // Look for server ready message
                if (text.includes('Mock AI Provider running')) {
                    console.log('✅ Server started successfully!');
                    this.isServerReady = true;
                    resolve();
                }
            });

            // Capture stderr
            this.serverProcess.stderr.on('data', (data) => {
                console.error('Server stderr:', data.toString());
            });

            // Handle process exit
            this.serverProcess.on('close', (code) => {
                if (!this.isServerReady) {
                    reject(new Error(`Server failed to start (exit code: ${code})`));
                }
            });

            // Handle process errors
            this.serverProcess.on('error', (error) => {
                reject(new Error(`Failed to start server: ${error.message}`));
            });

            // Timeout after 10 seconds
            setTimeout(() => {
                if (!this.isServerReady) {
                    reject(new Error('Server start timeout (10 seconds)'));
                }
            }, 10000);
        });
    }

    async runSelfTest() {
        console.log('\\n🧪 Running comprehensive self-test...');
        
        return new Promise((resolve, reject) => {
            const testProcess = spawn('node', ['self-test.js'], {
                stdio: ['ignore', 'inherit', 'inherit'],
                cwd: __dirname
            });

            testProcess.on('close', (code) => {
                if (code === 0) {
                    resolve();
                } else {
                    reject(new Error(`Self-test failed with exit code: ${code}`));
                }
            });

            testProcess.on('error', (error) => {
                reject(new Error(`Failed to run self-test: ${error.message}`));
            });
        });
    }

    async stopServer() {
        if (this.serverProcess && !this.serverProcess.killed) {
            console.log('\\n🔄 Stopping server...');
            
            // Send SIGTERM for graceful shutdown
            this.serverProcess.kill('SIGTERM');
            
            // Wait a moment for graceful shutdown
            await this.sleep(1000);
            
            // Force kill if still running
            if (!this.serverProcess.killed) {
                this.serverProcess.kill('SIGKILL');
            }
            
            console.log('✅ Server stopped.');
        }
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Handle cleanup on process exit
process.on('SIGINT', async () => {
    console.log('\\n🛑 Received SIGINT, cleaning up...');
    if (global.testRunner) {
        await global.testRunner.stopServer();
    }
    process.exit(0);
});

process.on('SIGTERM', async () => {
    console.log('\\n🛑 Received SIGTERM, cleaning up...');
    if (global.testRunner) {
        await global.testRunner.stopServer();
    }
    process.exit(0);
});

// Run if executed directly
if (require.main === module) {
    const runner = new SelfTestRunner();
    global.testRunner = runner; // Store globally for cleanup
    
    runner.runCompleteTest().catch(error => {
        console.error('\\n❌ Complete test failed:', error.message);
        process.exit(1);
    });
}

module.exports = SelfTestRunner;