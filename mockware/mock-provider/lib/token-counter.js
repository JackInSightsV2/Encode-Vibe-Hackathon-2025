class TokenCounter {
    constructor() {
        this.inputRate = parseFloat(process.env.TOKEN_RATE_INPUT) || 0.0001;
        this.outputRate = parseFloat(process.env.TOKEN_RATE_OUTPUT) || 0.0002;
    }

    countTokens(request, response) {
        const inputTokens = this.estimateTokens(this.extractRequestText(request));
        const outputTokens = this.estimateTokens(response);
        
        return {
            input: inputTokens,
            output: outputTokens,
            total: inputTokens + outputTokens,
            cost: {
                input: inputTokens * this.inputRate,
                output: outputTokens * this.outputRate,
                total: (inputTokens * this.inputRate) + (outputTokens * this.outputRate)
            }
        };
    }

    extractRequestText(request) {
        let text = '';
        
        // Handle different request formats
        if (typeof request === 'string') {
            return request;
        }
        
        // OpenAI format
        if (request.messages && Array.isArray(request.messages)) {
            text = request.messages
                .map(m => m.content || '')
                .join(' ');
        }
        
        // Add system prompt if present
        if (request.system) {
            text = request.system + ' ' + text;
        }
        
        // Anthropic format
        if (request.messages && Array.isArray(request.messages)) {
            text = request.messages
                .map(m => {
                    if (Array.isArray(m.content)) {
                        return m.content.map(c => c.text || '').join(' ');
                    }
                    return m.content || '';
                })
                .join(' ');
        }
        
        // Generic formats
        if (request.message) {
            text += ' ' + request.message;
        }
        if (request.prompt) {
            text += ' ' + request.prompt;
        }
        if (request.text) {
            text += ' ' + request.text;
        }
        if (request.input) {
            text += ' ' + request.input;
        }
        
        return text.trim();
    }

    estimateTokens(text) {
        if (!text || typeof text !== 'string') {
            return 0;
        }
        
        // Simple token estimation algorithm
        // This is a rough approximation, real tokenizers are more complex
        
        // Remove extra whitespace
        const cleaned = text.replace(/\s+/g, ' ').trim();
        
        if (cleaned.length === 0) {
            return 0;
        }
        
        // Basic word-based estimation
        const words = cleaned.split(' ');
        let tokenCount = 0;
        
        for (const word of words) {
            if (word.length === 0) continue;
            
            // Short words (1-4 chars) = 1 token
            if (word.length <= 4) {
                tokenCount += 1;
            }
            // Medium words (5-8 chars) = 1-2 tokens
            else if (word.length <= 8) {
                tokenCount += Math.ceil(word.length / 4);
            }
            // Long words = more tokens
            else {
                tokenCount += Math.ceil(word.length / 3);
            }
            
            // Special handling for certain patterns
            if (this.isCodeLike(word)) {
                tokenCount += Math.ceil(word.length / 2);
            }
            
            if (this.hasSpecialChars(word)) {
                tokenCount += Math.ceil(word.length / 4);
            }
        }
        
        // Add tokens for punctuation and special formatting
        const punctuationCount = (text.match(/[.,!?;:()[\]{}"'`]/g) || []).length;
        tokenCount += Math.ceil(punctuationCount / 2);
        
        // Add tokens for newlines and formatting
        const newlineCount = (text.match(/\n/g) || []).length;
        tokenCount += newlineCount;
        
        // Minimum of 1 token for non-empty text
        return Math.max(1, Math.round(tokenCount));
    }

    isCodeLike(word) {
        // Check if word looks like code
        const codePatterns = [
            /^[a-zA-Z_][a-zA-Z0-9_]*\(/,  // Function calls
            /^[a-zA-Z_][a-zA-Z0-9_]*\./,  // Method calls
            /^[A-Z_][A-Z0-9_]+$/,         // Constants
            /^0x[0-9a-fA-F]+$/,           // Hex numbers
            /^[a-zA-Z]+:\/\//,            // URLs
            /^[{}[\]<>]/,                 // Brackets
            /^[=+\-*/%&|^~!<>]+$/         // Operators
        ];
        
        return codePatterns.some(pattern => pattern.test(word));
    }

    hasSpecialChars(word) {
        // Check for special characters that might affect tokenization
        return /[^\w\s.,!?;:()[\]{}"'`-]/.test(word);
    }

    calculateCost(tokens) {
        return {
            input: tokens.input * this.inputRate,
            output: tokens.output * this.outputRate,
            total: (tokens.input * this.inputRate) + (tokens.output * this.outputRate)
        };
    }

    // Model-specific token counting (if needed)
    countTokensForModel(request, response, model) {
        const baseTokens = this.countTokens(request, response);
        
        // Apply model-specific multipliers if needed
        const modelMultipliers = {
            'gpt-4': 1.2,
            'gpt-4-turbo': 1.1,
            'claude-3-opus': 1.3,
            'claude-3-sonnet': 1.0,
            'claude-3-haiku': 0.8
        };
        
        const multiplier = modelMultipliers[model] || 1.0;
        
        return {
            input: Math.round(baseTokens.input * multiplier),
            output: Math.round(baseTokens.output * multiplier),
            total: Math.round(baseTokens.total * multiplier),
            cost: {
                input: baseTokens.cost.input * multiplier,
                output: baseTokens.cost.output * multiplier,
                total: baseTokens.cost.total * multiplier
            }
        };
    }

    // Estimate tokens for streaming responses
    estimateStreamingTokens(partialResponse) {
        return this.estimateTokens(partialResponse);
    }

    // Get token statistics
    getTokenStats(text) {
        const tokens = this.estimateTokens(text);
        const chars = text.length;
        const words = text.split(/\s+/).length;
        
        return {
            tokens,
            characters: chars,
            words,
            tokensPerWord: words > 0 ? tokens / words : 0,
            tokensPerChar: chars > 0 ? tokens / chars : 0,
            efficiency: tokens / Math.max(chars, 1) // Lower is more efficient
        };
    }
}

module.exports = TokenCounter;