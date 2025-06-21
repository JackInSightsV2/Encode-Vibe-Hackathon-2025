class ResponseAnalyzer {
    constructor() {
        this.moderationPatterns = this.initializeModerationPatterns();
        this.injectionPatterns = this.initializeInjectionPatterns();
        this.piiPatterns = this.initializePIIPatterns();
        this.relevanceKeywords = this.initializeRelevanceKeywords();
    }

    analyzeRequest(request) {
        const message = this.extractMessage(request);
        
        const analysis = {
            originalMessage: message,
            messageLength: message.length,
            wordCount: message.split(/\s+/).length,
            detectedLanguage: this.detectLanguage(message),
            categories: [],
            confidence: {},
            isModerated: false,
            containsPII: false,
            isInjection: false,
            isOffTopic: false,
            severity: 'low'
        };

        // Run all analysis checks
        this.checkModeration(message, analysis);
        this.checkPII(message, analysis);
        this.checkInjection(message, analysis);
        this.checkRelevance(message, analysis);
        this.calculateOverallSeverity(analysis);

        return analysis;
    }

    extractMessage(request) {
        // Extract message from different request formats
        if (typeof request === 'string') {
            return request;
        }
        
        // OpenAI format
        if (request.messages && Array.isArray(request.messages)) {
            const userMessages = request.messages
                .filter(m => m.role === 'user')
                .map(m => m.content)
                .join(' ');
            return userMessages;
        }
        
        // Anthropic format
        if (request.messages && Array.isArray(request.messages)) {
            return request.messages
                .filter(m => m.role === 'user')
                .map(m => m.content)
                .join(' ');
        }
        
        // Generic formats
        if (request.message) {
            return request.message;
        }
        if (request.prompt) {
            return request.prompt;
        }
        if (request.text) {
            return request.text;
        }
        if (request.input) {
            return request.input;
        }
        
        // Fallback: stringify the entire request
        return JSON.stringify(request);
    }

    checkModeration(message, analysis) {
        const messageLower = message.toLowerCase();
        let maxConfidence = 0;
        
        for (const [category, patterns] of Object.entries(this.moderationPatterns)) {
            let categoryConfidence = 0;
            
            for (const pattern of patterns) {
                if (pattern.test(messageLower)) {
                    categoryConfidence = Math.max(categoryConfidence, 0.8);
                    analysis.categories.push(`moderation_${category}`);
                }
            }
            
            if (categoryConfidence > 0) {
                analysis.confidence[`moderation_${category}`] = categoryConfidence;
                maxConfidence = Math.max(maxConfidence, categoryConfidence);
            }
        }
        
        if (maxConfidence > 0.6) {
            analysis.isModerated = true;
            analysis.severity = maxConfidence > 0.8 ? 'high' : 'medium';
        }
    }

    checkPII(message, analysis) {
        let maxConfidence = 0;
        
        for (const [type, pattern] of Object.entries(this.piiPatterns)) {
            const matches = message.match(pattern);
            if (matches) {
                analysis.categories.push(`pii_${type}`);
                const confidence = this.calculatePIIConfidence(type, matches[0]);
                analysis.confidence[`pii_${type}`] = confidence;
                maxConfidence = Math.max(maxConfidence, confidence);
            }
        }
        
        if (maxConfidence > 0.7) {
            analysis.containsPII = true;
            analysis.severity = 'medium';
        }
    }

    checkInjection(message, analysis) {
        const messageLower = message.toLowerCase();
        let maxConfidence = 0;
        
        for (const [category, patterns] of Object.entries(this.injectionPatterns)) {
            let categoryConfidence = 0;
            
            for (const pattern of patterns) {
                if (pattern.test(messageLower)) {
                    categoryConfidence = Math.max(categoryConfidence, 0.9);
                    analysis.categories.push(`injection_${category}`);
                }
            }
            
            if (categoryConfidence > 0) {
                analysis.confidence[`injection_${category}`] = categoryConfidence;
                maxConfidence = Math.max(maxConfidence, categoryConfidence);
            }
        }
        
        // Check for encoded content
        if (this.hasEncodedContent(message)) {
            analysis.categories.push('injection_encoded');
            analysis.confidence.injection_encoded = 0.8;
            maxConfidence = Math.max(maxConfidence, 0.8);
        }
        
        if (maxConfidence > 0.7) {
            analysis.isInjection = true;
            analysis.severity = 'high';
        }
    }

    checkRelevance(message, analysis) {
        const messageLower = message.toLowerCase();
        const words = messageLower.split(/\s+/);
        
        // Check for AI/technology relevance
        const relevantKeywords = this.relevanceKeywords.relevant;
        const irrelevantKeywords = this.relevanceKeywords.irrelevant;
        
        let relevantScore = 0;
        let irrelevantScore = 0;
        
        for (const word of words) {
            if (relevantKeywords.includes(word)) {
                relevantScore += 1;
            }
            if (irrelevantKeywords.includes(word)) {
                irrelevantScore += 1;
            }
        }
        
        const totalWords = words.length;
        const relevanceRatio = relevantScore / Math.max(totalWords, 1);
        const irrelevanceRatio = irrelevantScore / Math.max(totalWords, 1);
        
        if (irrelevanceRatio > 0.3 || (relevanceRatio < 0.1 && totalWords > 5)) {
            analysis.isOffTopic = true;
            analysis.categories.push('relevance_offtopic');
            analysis.confidence.relevance_offtopic = Math.min(irrelevanceRatio + (1 - relevanceRatio), 1);
        }
    }

    calculatePIIConfidence(type, match) {
        switch (type) {
            case 'email':
                return this.validateEmail(match) ? 0.95 : 0.6;
            case 'phone':
                return this.validatePhone(match) ? 0.9 : 0.7;
            case 'ssn':
                return this.validateSSN(match) ? 0.98 : 0.8;
            case 'credit_card':
                return this.validateCreditCard(match) ? 0.95 : 0.7;
            default:
                return 0.8;
        }
    }

    validateEmail(email) {
        const parts = email.split('@');
        return parts.length === 2 && parts[0].length > 0 && parts[1].includes('.');
    }

    validatePhone(phone) {
        const cleaned = phone.replace(/\D/g, '');
        return cleaned.length >= 10 && cleaned.length <= 15;
    }

    validateSSN(ssn) {
        const cleaned = ssn.replace(/\D/g, '');
        return cleaned.length === 9;
    }

    validateCreditCard(card) {
        const cleaned = card.replace(/\D/g, '');
        return cleaned.length >= 13 && cleaned.length <= 19;
    }

    hasEncodedContent(message) {
        // Check for Base64
        const base64Pattern = /^[A-Za-z0-9+/]*={0,2}$/;
        const words = message.split(/\s+/);
        
        for (const word of words) {
            if (word.length > 10 && base64Pattern.test(word)) {
                try {
                    const decoded = Buffer.from(word, 'base64').toString('utf8');
                    if (decoded.length > 0 && /[a-zA-Z]/.test(decoded)) {
                        return true;
                    }
                } catch (e) {
                    // Not valid base64
                }
            }
        }
        
        // Check for hex encoding
        const hexPattern = /^[0-9a-fA-F]+$/;
        for (const word of words) {
            if (word.length > 20 && word.length % 2 === 0 && hexPattern.test(word)) {
                return true;
            }
        }
        
        return false;
    }

    detectLanguage(message) {
        // Simple language detection based on character patterns
        const languages = {
            english: /^[a-zA-Z0-9\s.,!?'"()-]+$/,
            chinese: /[\u4e00-\u9fff]/,
            japanese: /[\u3040-\u309f\u30a0-\u30ff]/,
            korean: /[\uac00-\ud7af]/,
            arabic: /[\u0600-\u06ff]/,
            russian: /[\u0400-\u04ff]/,
            german: /[äöüß]/i,
            french: /[àâäçéèêëïîôùûüÿ]/i,
            spanish: /[áéíóúüñ]/i
        };
        
        for (const [lang, pattern] of Object.entries(languages)) {
            if (pattern.test(message)) {
                return lang;
            }
        }
        
        return 'unknown';
    }

    calculateOverallSeverity(analysis) {
        const severityScores = {
            low: 1,
            medium: 2,
            high: 3
        };
        
        let maxSeverity = 'low';
        
        if (analysis.isModerated || analysis.isInjection) {
            maxSeverity = 'high';
        } else if (analysis.containsPII) {
            maxSeverity = 'medium';
        }
        
        analysis.severity = maxSeverity;
    }

    initializeModerationPatterns() {
        return {
            violence: [
                /\b(kill|murder|destroy|eliminate|hurt|harm|attack|assault)\b/gi,
                /\b(bomb|weapon|gun|knife|explosive)\b/gi,
                /\b(death|die|dead|corpse)\b/gi
            ],
            hate: [
                /\b(hate|despise|loathe)\s+(all|every|most)\s+\w+/gi,
                /\b(racist|sexist|homophobic|bigot)\b/gi,
                /\b(inferior|subhuman|scum)\b/gi
            ],
            explicit: [
                /\b(porn|sex|naked|nude|explicit)\b/gi,
                /\b(erotic|sexual|adult)\b/gi
            ],
            selfharm: [
                /\b(suicide|kill\s+myself|end\s+my\s+life)\b/gi,
                /\b(cut\s+myself|hurt\s+myself)\b/gi
            ]
        };
    }

    initializeInjectionPatterns() {
        return {
            direct: [
                /ignore\s+(all\s+)?(previous|prior)\s+instructions/gi,
                /forget\s+(everything|all)\s+(above|before)/gi,
                /you\s+are\s+now\s+(in\s+)?(\w+\s+)?mode/gi,
                /act\s+as\s+(a\s+)?(?!assistant|ai)\w+/gi,
                /pretend\s+to\s+be/gi,
                /roleplay\s+as/gi
            ],
            system: [
                /system\s+message/gi,
                /developer\s+mode/gi,
                /admin\s+mode/gi,
                /jailbreak/gi,
                /DAN\s+mode/gi,
                /override/gi
            ],
            hypothetical: [
                /in\s+a\s+hypothetical/gi,
                /imagine\s+if/gi,
                /what\s+if/gi,
                /suppose/gi
            ]
        };
    }

    initializePIIPatterns() {
        return {
            email: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g,
            phone: /\b(?:\+?1[-.\s]?)?(?:\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4})\b/g,
            ssn: /\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b/g,
            credit_card: /\b(?:\d{4}[-\s]?){3}\d{4}\b/g
        };
    }

    initializeRelevanceKeywords() {
        return {
            relevant: [
                'ai', 'artificial', 'intelligence', 'machine', 'learning', 'neural', 'network',
                'algorithm', 'computer', 'technology', 'software', 'programming', 'code',
                'data', 'model', 'train', 'predict', 'analyze', 'help', 'assist', 'question',
                'answer', 'explain', 'tell', 'know', 'understand', 'information'
            ],
            irrelevant: [
                'pizza', 'food', 'recipe', 'cooking', 'weather', 'sports', 'movie', 'music',
                'celebrity', 'gossip', 'fashion', 'shopping', 'travel', 'vacation', 'hobby',
                'pet', 'animal', 'plant', 'garden', 'car', 'driving', 'personal', 'family'
            ]
        };
    }
}

module.exports = ResponseAnalyzer;