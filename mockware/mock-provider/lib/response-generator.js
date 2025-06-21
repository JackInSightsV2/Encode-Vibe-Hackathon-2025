const fs = require('fs');
const path = require('path');
const { v4: uuidv4 } = require('uuid');
const ResponseAnalyzer = require('./response-analyzer');
const TokenCounter = require('./token-counter');

class ResponseGenerator {
    constructor() {
        this.analyzer = new ResponseAnalyzer();
        this.tokenCounter = new TokenCounter();
        this.templates = this.loadTemplates();
        this.personalities = this.loadPersonalities();
    }

    loadTemplates() {
        const templatesDir = path.join(__dirname, '../responses/templates');
        const templates = {};
        
        try {
            const files = ['clean.json', 'triggered.json', 'pii.json', 'injected.json', 'errors.json'];
            
            for (const file of files) {
                const filePath = path.join(templatesDir, file);
                if (fs.existsSync(filePath)) {
                    const key = path.basename(file, '.json');
                    templates[key] = JSON.parse(fs.readFileSync(filePath, 'utf8'));
                }
            }
        } catch (error) {
            console.warn('Could not load response templates:', error.message);
            // Return default templates
            templates.clean = this.getDefaultCleanTemplates();
            templates.triggered = this.getDefaultTriggeredTemplates();
            templates.pii = this.getDefaultPIITemplates();
            templates.injected = this.getDefaultInjectedTemplates();
            templates.errors = this.getDefaultErrorTemplates();
        }
        
        return templates;
    }

    loadPersonalities() {
        const personalitiesDir = path.join(__dirname, '../responses/personalities');
        const personalities = {};
        
        try {
            const files = ['helpful.json', 'creative.json', 'technical.json'];
            
            for (const file of files) {
                const filePath = path.join(personalitiesDir, file);
                if (fs.existsSync(filePath)) {
                    const key = path.basename(file, '.json');
                    personalities[key] = JSON.parse(fs.readFileSync(filePath, 'utf8'));
                }
            }
        } catch (error) {
            console.warn('Could not load personalities:', error.message);
            personalities.helpful = this.getDefaultHelpfulPersonality();
            personalities.creative = this.getDefaultCreativePersonality();
            personalities.technical = this.getDefaultTechnicalPersonality();
        }
        
        return personalities;
    }

    // OpenAI-compatible response generation
    generateOpenAIResponse(request) {
        const analysis = this.analyzer.analyzeRequest(request);
        const responseType = this.determineResponseType(analysis);
        
        const responseTemplate = this.selectTemplate(responseType, analysis);
        const personality = this.selectPersonality(request.model);
        
        const content = this.generateContent(analysis, responseTemplate, personality);
        const tokens = this.tokenCounter.countTokens(request, content);
        
        return {
            id: `chatcmpl-${uuidv4().replace(/-/g, '')}`,
            object: "chat.completion",
            created: Math.floor(Date.now() / 1000),
            model: request.model || "gpt-3.5-turbo",
            choices: [{
                index: 0,
                message: {
                    role: "assistant",
                    content: content
                },
                finish_reason: "stop"
            }],
            usage: {
                prompt_tokens: tokens.input,
                completion_tokens: tokens.output,
                total_tokens: tokens.total
            },
            system_fingerprint: `fp_${Math.random().toString(36).substring(2, 15)}`
        };
    }

    // Anthropic-compatible response generation
    generateAnthropicResponse(request) {
        const analysis = this.analyzer.analyzeRequest(request);
        const responseType = this.determineResponseType(analysis);
        
        const responseTemplate = this.selectTemplate(responseType, analysis);
        const personality = this.selectPersonality(request.model);
        
        const content = this.generateContent(analysis, responseTemplate, personality);
        const tokens = this.tokenCounter.countTokens(request, content);
        
        return {
            id: `msg_${uuidv4().replace(/-/g, '')}`,
            type: "message",
            role: "assistant",
            content: [{
                type: "text",
                text: content
            }],
            model: request.model || "claude-3-sonnet-20240229",
            stop_reason: "end_turn",
            stop_sequence: null,
            usage: {
                input_tokens: tokens.input,
                output_tokens: tokens.output
            }
        };
    }

    // Generic response generation
    generateGenericResponse(request) {
        const analysis = this.analyzer.analyzeRequest(request);
        const responseType = this.determineResponseType(analysis);
        
        const responseTemplate = this.selectTemplate(responseType, analysis);
        const personality = this.selectPersonality(request.model);
        
        const content = this.generateContent(analysis, responseTemplate, personality);
        const tokens = this.tokenCounter.countTokens(request, content);
        
        return {
            id: uuidv4(),
            response: content,
            model: request.model || "generic-ai-v1",
            timestamp: new Date().toISOString(),
            tokens: {
                input: tokens.input,
                output: tokens.output,
                total: tokens.total
            },
            metadata: {
                response_type: responseType,
                analysis: analysis.categories,
                personality: personality.name
            }
        };
    }

    determineResponseType(analysis) {
        // Determine response type based on analysis
        if (analysis.isModerated) {
            return 'triggered';
        } else if (analysis.containsPII) {
            return 'pii';
        } else if (analysis.isInjection) {
            return 'injected';
        } else if (analysis.isOffTopic) {
            return 'clean'; // Still respond normally to off-topic
        } else {
            return 'clean';
        }
    }

    selectTemplate(responseType, analysis) {
        const templates = this.templates[responseType] || this.templates.clean;
        
        // Select template based on specific categories if available
        if (analysis.categories.length > 0) {
            const categoryTemplates = templates.filter(t => 
                analysis.categories.some(cat => t.categories?.includes(cat))
            );
            if (categoryTemplates.length > 0) {
                return categoryTemplates[Math.floor(Math.random() * categoryTemplates.length)];
            }
        }
        
        // Default random selection
        return templates[Math.floor(Math.random() * templates.length)];
    }

    selectPersonality(model) {
        // Select personality based on model
        if (model?.includes('gpt-4')) {
            return this.personalities.technical || this.personalities.helpful;
        } else if (model?.includes('claude')) {
            return this.personalities.helpful || this.personalities.creative;
        } else if (model?.includes('creative') || model?.includes('instruct')) {
            return this.personalities.creative || this.personalities.helpful;
        } else {
            return this.personalities.helpful;
        }
    }

    generateContent(analysis, template, personality) {
        let content = template.content;
        
        // Apply personality modifications
        if (personality.modifiers) {
            content = this.applyPersonalityModifiers(content, personality.modifiers);
        }
        
        // Replace placeholders
        content = this.replacePlaceholders(content, analysis);
        
        // Add personality-specific prefix/suffix
        if (personality.prefix && Math.random() < 0.3) {
            content = personality.prefix + " " + content;
        }
        if (personality.suffix && Math.random() < 0.2) {
            content = content + " " + personality.suffix;
        }
        
        return content;
    }

    applyPersonalityModifiers(content, modifiers) {
        if (modifiers.formal && Math.random() < 0.5) {
            content = "I would like to point out that " + content.toLowerCase();
        }
        if (modifiers.casual && Math.random() < 0.5) {
            content = content.replace(/\.$/, '') + ", you know?";
        }
        if (modifiers.technical && Math.random() < 0.3) {
            content = "From a technical perspective, " + content.toLowerCase();
        }
        if (modifiers.creative && Math.random() < 0.4) {
            content = "Interestingly, " + content.toLowerCase();
        }
        
        return content;
    }

    replacePlaceholders(content, analysis) {
        // Replace common placeholders
        content = content.replace(/\{timestamp\}/g, new Date().toISOString());
        content = content.replace(/\{random\}/g, Math.random().toString(36).substring(7));
        
        // Replace analysis-specific placeholders
        if (analysis.messageLength) {
            content = content.replace(/\{message_length\}/g, analysis.messageLength);
        }
        if (analysis.detectedLanguage) {
            content = content.replace(/\{language\}/g, analysis.detectedLanguage);
        }
        
        return content;
    }

    // Default templates when files are not available
    getDefaultCleanTemplates() {
        return [
            {
                content: "I'm happy to help you with that! Let me provide you with some information.",
                categories: ["helpful", "informative"]
            },
            {
                content: "That's an interesting question. Here's what I can tell you about that topic.",
                categories: ["engaging", "educational"]
            },
            {
                content: "I understand what you're asking about. Let me break this down for you.",
                categories: ["clear", "structured"]
            },
            {
                content: "Thank you for your question. I'll do my best to provide a helpful response.",
                categories: ["polite", "professional"]
            },
            {
                content: "Based on my knowledge, I can offer the following insights on this topic.",
                categories: ["knowledgeable", "confident"]
            },
            {
                content: "This is a great question that touches on several important aspects.",
                categories: ["comprehensive", "analytical"]
            },
            {
                content: "I appreciate you bringing this up. Let me share some relevant information.",
                categories: ["appreciative", "collaborative"]
            },
            {
                content: "From what I understand about this subject, here are the key points to consider.",
                categories: ["organized", "thoughtful"]
            }
        ];
    }

    getDefaultTriggeredTemplates() {
        return [
            {
                content: "I can't and won't provide information or assistance with harmful, illegal, or dangerous activities.",
                categories: ["safety", "refusal"]
            },
            {
                content: "I'm not able to help with content that could cause harm to individuals or groups.",
                categories: ["protection", "ethical"]
            },
            {
                content: "I notice your request contains content that goes against my guidelines. I'd be happy to help with something else instead.",
                categories: ["redirection", "alternative"]
            },
            {
                content: "I can't assist with that particular request, but I'm here to help with many other topics.",
                categories: ["boundaries", "helpful"]
            }
        ];
    }

    getDefaultPIITemplates() {
        return [
            {
                content: "I notice your message contains what appears to be personal information. For privacy and security reasons, I'd recommend not sharing sensitive details like that.",
                categories: ["privacy", "security"]
            },
            {
                content: "I see some personal information in your message. I should mention that it's generally best to avoid sharing such details online.",
                categories: ["awareness", "protection"]
            }
        ];
    }

    getDefaultInjectedTemplates() {
        return [
            {
                content: "I notice you're trying to change how I operate, but I'm designed to maintain consistent behavior and follow my guidelines.",
                categories: ["consistency", "design"]
            },
            {
                content: "I understand you might be testing my responses, but I'll continue to operate within my intended parameters.",
                categories: ["testing", "parameters"]
            }
        ];
    }

    getDefaultErrorTemplates() {
        return [
            {
                content: "I apologize, but I encountered an error while processing your request. Please try again.",
                categories: ["error", "retry"]
            }
        ];
    }

    getDefaultHelpfulPersonality() {
        return {
            name: "helpful",
            description: "Friendly and professional assistant",
            modifiers: {
                formal: true,
                polite: true
            },
            prefix: "I'd be happy to help.",
            suffix: "Is there anything else you'd like to know?"
        };
    }

    getDefaultCreativePersonality() {
        return {
            name: "creative",
            description: "Imaginative and engaging assistant",
            modifiers: {
                creative: true,
                casual: true
            },
            prefix: "Here's an interesting perspective:",
            suffix: "What do you think about that?"
        };
    }

    getDefaultTechnicalPersonality() {
        return {
            name: "technical",
            description: "Precise and analytical assistant",
            modifiers: {
                technical: true,
                formal: true
            },
            prefix: "From a technical standpoint:",
            suffix: "Let me know if you need more specific details."
        };
    }
}

module.exports = ResponseGenerator;