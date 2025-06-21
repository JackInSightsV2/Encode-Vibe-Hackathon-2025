const moderationScenarios = require('./scenarios/moderation');
const injectionScenarios = require('./scenarios/injection');
const piiScenarios = require('./scenarios/pii');
const relevanceScenarios = require('./scenarios/relevance');
const ddosScenarios = require('./scenarios/ddos');
const combinationScenarios = require('./scenarios/combination');
const edgeCaseScenarios = require('./scenarios/edge-cases');
const { v4: uuidv4 } = require('uuid');

class TestGenerator {
    constructor() {
        this.scenarios = {
            moderation: moderationScenarios,
            injection: injectionScenarios,
            pii: piiScenarios,
            relevance: relevanceScenarios,
            ddos: ddosScenarios,
            combination: combinationScenarios,
            edgeCase: edgeCaseScenarios
        };
    }

    generateTestMessage(category, subcategory) {
        const categoryScenarios = this.scenarios[category];
        if (!categoryScenarios) {
            return this.generateCleanMessage();
        }

        const subcategoryScenarios = categoryScenarios[subcategory];
        if (!subcategoryScenarios || subcategoryScenarios.length === 0) {
            return this.generateCleanMessage();
        }

        // Select random scenario from subcategory
        const randomIndex = Math.floor(Math.random() * subcategoryScenarios.length);
        let message = subcategoryScenarios[randomIndex];
        
        // Handle function-based scenarios (like cache busters)
        if (typeof message === 'function') {
            message = message();
        }
        
        return message;
    }

    generateCleanMessage() {
        const cleanMessages = [
            "What's the weather like today?",
            "Can you help me understand quantum computing?",
            "Tell me about the history of artificial intelligence",
            "How do I make a chocolate cake?",
            "What are the benefits of exercise?",
            "Explain the theory of relativity",
            "What's the best way to learn programming?",
            "Tell me about renewable energy",
            "How does the internet work?",
            "What are some tips for better sleep?"
        ];
        
        return cleanMessages[Math.floor(Math.random() * cleanMessages.length)];
    }

    generateMixedBatch(distribution, count) {
        const messages = [];
        
        // Calculate counts for each category
        const categoryCounts = {};
        let remaining = count;
        
        // Sort categories by percentage to handle rounding
        const sortedCategories = Object.entries(distribution)
            .sort((a, b) => b[1] - a[1]);
        
        for (const [category, percentage] of sortedCategories) {
            const categoryCount = Math.floor(count * (percentage / 100));
            categoryCounts[category] = categoryCount;
            remaining -= categoryCount;
        }
        
        // Distribute remaining messages to largest percentage categories
        for (let i = 0; i < remaining; i++) {
            const category = sortedCategories[i % sortedCategories.length][0];
            categoryCounts[category]++;
        }
        
        // Generate messages for each category
        for (const [category, categoryCount] of Object.entries(categoryCounts)) {
            for (let i = 0; i < categoryCount; i++) {
                if (category === 'clean') {
                    messages.push({
                        category: 'clean',
                        message: this.generateCleanMessage(),
                        expectedResult: 'pass'
                    });
                } else {
                    // Parse category and subcategory
                    const [mainCategory, subCategory] = category.split('.');
                    const message = this.generateTestMessage(mainCategory, subCategory || 'default');
                    
                    messages.push({
                        category,
                        message,
                        expectedResult: this.getExpectedResult(mainCategory)
                    });
                }
            }
        }
        
        // Shuffle messages
        return this.shuffle(messages);
    }

    getExpectedResult(category) {
        const blockCategories = ['moderation', 'injection', 'pii', 'combination'];
        return blockCategories.includes(category) ? 'block' : 'pass';
    }

    shuffle(array) {
        const shuffled = [...array];
        for (let i = shuffled.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
        }
        return shuffled;
    }

    getAvailableScenarios() {
        const scenarios = {};
        
        for (const [category, subcategories] of Object.entries(this.scenarios)) {
            scenarios[category] = {};
            for (const [subcategory, messages] of Object.entries(subcategories)) {
                scenarios[category][subcategory] = messages.length;
            }
        }
        
        return scenarios;
    }

    generateTestPayload(message, userConfig = {}) {
        const defaultPayload = {
            user_id: `test_user_${Math.floor(Math.random() * 1000)}`,
            session_id: `test_session_${Math.floor(Math.random() * 1000)}`,
            message: message,
            timestamp: new Date().toISOString()
        };
        
        // Merge with user configuration
        return { ...defaultPayload, ...userConfig, message };
    }

    generateDDoSPayload(type = 'standard') {
        switch (type) {
            case 'large':
                return {
                    ...this.generateTestPayload('A'.repeat(10000)),
                    metadata: Array(100).fill({ data: 'padding' })
                };
            
            case 'complex':
                return {
                    ...this.generateTestPayload('Test message'),
                    nested: this.generateNestedObject(10)
                };
            
            case 'cache-buster':
                return {
                    ...this.generateTestPayload('Test message'),
                    random: Math.random().toString(36).substring(7),
                    timestamp: Date.now(),
                    cacheBuster: uuidv4()
                };
            
            default:
                return this.generateTestPayload('Standard DDoS test message');
        }
    }

    generateNestedObject(depth) {
        if (depth <= 0) return 'leaf';
        
        return {
            level: depth,
            data: Array(5).fill(null).map(() => this.generateNestedObject(depth - 1))
        };
    }
}

module.exports = TestGenerator;