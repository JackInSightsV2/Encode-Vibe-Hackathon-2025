const express = require('express');
const ResponseGenerator = require('../lib/response-generator');
const LatencySimulator = require('../lib/latency-simulator');
const ErrorSimulator = require('../lib/error-simulator');

const router = express.Router();
const generator = new ResponseGenerator();
const latency = new LatencySimulator();
const errorSim = new ErrorSimulator();

// Universal mock endpoint - handles any AI provider request format
// Skip health endpoints and root documentation
router.all('/*', async (req, res, next) => {
    // Skip health endpoints and root docs
    if (req.path === '/' || req.path.startsWith('/health')) {
        return next();
    }
    try {
        const processingStart = Date.now();
        console.log(`\n🔧 ================ MOCK PROCESSING START ================`);
        console.log(`📝 Processing mock request: ${req.method} ${req.path}`);
        
        // Determine response format based on request path and body
        const formatDetectionStart = Date.now();
        const responseFormat = detectResponseFormat(req);
        const formatDetectionTime = Date.now() - formatDetectionStart;
        
        console.log(`🎯 Format Detection: ${responseFormat} (${formatDetectionTime}ms)`);
        console.log(`📊 Request Analysis:`);
        console.log(`   Path: ${req.path}`);
        console.log(`   Method: ${req.method}`);
        console.log(`   Body Keys: [${Object.keys(req.body || {}).join(', ')}]`);
        
        // Log format detection details
        const path = req.path.toLowerCase();
        const body = req.body || {};
        console.log(`🔍 Format Detection Logic:`);
        console.log(`   Path analysis: "${path}"`);
        if (path.includes('/chat/completions')) console.log(`   ✅ Matched path: /chat/completions -> openai-chat`);
        else if (path.includes('/completions')) console.log(`   ✅ Matched path: /completions -> openai-completion`);
        else if (path.includes('/moderations')) console.log(`   ✅ Matched path: /moderations -> openai-moderation`);
        else if (path.includes('/messages')) console.log(`   ✅ Matched path: /messages -> anthropic-messages`);
        else if (path.includes('/complete')) console.log(`   ✅ Matched path: /complete -> anthropic-complete`);
        else {
            console.log(`   🔍 No path match, analyzing body structure:`);
            if (body.messages && Array.isArray(body.messages)) {
                console.log(`   ✅ Found messages array -> chat format`);
                if (body.model && body.model.includes('claude')) console.log(`   ✅ Claude model -> anthropic-messages`);
                else if (body.model && body.model.includes('gpt')) console.log(`   ✅ GPT model -> openai-chat`);
                else console.log(`   ✅ Default -> openai-chat`);
            } else if (body.prompt && typeof body.prompt === 'string') {
                console.log(`   ✅ Found prompt string -> completion format`);
                if (body.model && body.model.includes('claude')) console.log(`   ✅ Claude model -> anthropic-complete`);
                else console.log(`   ✅ Default -> openai-completion`);
            } else if (body.input && typeof body.input === 'string') {
                console.log(`   ✅ Found input string -> moderation format`);
            } else {
                console.log(`   ✅ No specific pattern -> generic format`);
            }
        }
        
        // Simulate latency with detailed logging
        console.log(`⏱️ Simulating latency...`);
        const latencyStart = Date.now();
        const delayOptions = {
            complexity: 'normal',
            requestSize: JSON.stringify(req.body || {}).length
        };
        console.log(`   Request size: ${delayOptions.requestSize} bytes`);
        console.log(`   Complexity: ${delayOptions.complexity}`);
        
        await latency.simulate(delayOptions);
        const latencyTime = Date.now() - latencyStart;
        console.log(`   ✅ Latency simulated: ${latencyTime}ms`);

        // Generate appropriate mock response with detailed logging
        console.log(`🤖 Generating ${responseFormat} response...`);
        const generationStart = Date.now();
        let response;
        
        switch (responseFormat) {
            case 'openai-chat':
                console.log(`   🎯 Generating OpenAI Chat Completion...`);
                response = generator.generateOpenAIResponse(req.body || {});
                console.log(`   ✅ Generated response with ${response.choices?.length || 0} choices`);
                break;
            case 'openai-completion':
                console.log(`   🎯 Generating OpenAI Text Completion...`);
                response = generateOpenAICompletion(req.body || {});
                console.log(`   ✅ Generated completion response`);
                break;
            case 'openai-moderation':
                console.log(`   🎯 Generating OpenAI Moderation Response...`);
                response = generateModerationResponse(req.body || {});
                console.log(`   ✅ Generated moderation response with ${response.results?.length || 0} results`);
                break;
            case 'anthropic-messages':
                console.log(`   🎯 Generating Anthropic Messages Response...`);
                response = generator.generateAnthropicResponse(req.body || {});
                console.log(`   ✅ Generated Anthropic response`);
                break;
            case 'anthropic-complete':
                console.log(`   🎯 Generating Anthropic Completion...`);
                response = generateAnthropicCompletion(req.body || {});
                console.log(`   ✅ Generated Anthropic completion`);
                break;
            default:
                console.log(`   🎯 Generating Generic Response...`);
                response = generator.generateGenericResponse(req.body || {});
                console.log(`   ✅ Generated generic response`);
        }
        
        const generationTime = Date.now() - generationStart;
        console.log(`   ⚡ Generation time: ${generationTime}ms`);
        
        // Log response details
        console.log(`📦 Response Generated:`);
        console.log(`   Type: ${responseFormat}`);
        console.log(`   Size: ${JSON.stringify(response).length} chars`);
        console.log(`   Keys: [${Object.keys(response).join(', ')}]`);
        
        if (response.choices) {
            console.log(`   Choices: ${response.choices.length}`);
            response.choices.forEach((choice, idx) => {
                const content = choice.message?.content || choice.text || '';
                console.log(`      [${idx}] ${choice.finish_reason}: "${content.substring(0, 50)}${content.length > 50 ? '...' : ''}"`);
            });
        }
        
        if (response.usage) {
            console.log(`   Usage: ${response.usage.total_tokens} tokens (${response.usage.prompt_tokens} + ${response.usage.completion_tokens})`);
        }
        
        // Handle streaming if requested
        if (req.body?.stream) {
            console.log(`🌊 Initiating streaming response...`);
            const totalProcessingTime = Date.now() - processingStart;
            console.log(`🔧 Total processing time: ${totalProcessingTime}ms`);
            console.log(`🔧 ================ MOCK PROCESSING END ================\n`);
            return handleStreaming(req, res, response, responseFormat);
        }
        
        const totalProcessingTime = Date.now() - processingStart;
        console.log(`🔧 Processing Summary:`);
        console.log(`   Format Detection: ${formatDetectionTime}ms`);
        console.log(`   Latency Simulation: ${latencyTime}ms`);
        console.log(`   Response Generation: ${generationTime}ms`);
        console.log(`   Total Processing: ${totalProcessingTime}ms`);
        console.log(`🔧 ================ MOCK PROCESSING END ================\n`);
        
        res.json(response);
        
    } catch (error) {
        console.error(`❌ Mock API error in ${req.path}:`, error);
        console.error(`   Error type: ${error.constructor.name}`);
        console.error(`   Error message: ${error.message}`);
        console.error(`   Stack trace: ${error.stack}`);
        
        const errorResponse = errorSim.generateRandomError();
        console.log(`🚨 Sending error response: ${errorResponse.status} - ${errorResponse.response.error?.message || 'Unknown error'}`);
        res.status(errorResponse.status).json(errorResponse.response);
    }
});

function detectResponseFormat(req) {
    const path = req.path.toLowerCase();
    const body = req.body || {};
    
    // Detect by path
    if (path.includes('/chat/completions')) return 'openai-chat';
    if (path.includes('/completions')) return 'openai-completion';
    if (path.includes('/moderations')) return 'openai-moderation';
    if (path.includes('/messages')) return 'anthropic-messages';
    if (path.includes('/complete')) return 'anthropic-complete';
    
    // Detect by request body structure
    if (body.messages && Array.isArray(body.messages)) {
        // Could be OpenAI chat or Anthropic messages
        if (body.model && body.model.includes('claude')) return 'anthropic-messages';
        if (body.model && body.model.includes('gpt')) return 'openai-chat';
        return 'openai-chat'; // Default to OpenAI format
    }
    
    if (body.prompt && typeof body.prompt === 'string') {
        if (body.model && body.model.includes('claude')) return 'anthropic-complete';
        return 'openai-completion';
    }
    
    if (body.input && typeof body.input === 'string') {
        return 'openai-moderation';
    }
    
    // Default to generic
    return 'generic';
}

function generateOpenAICompletion(request) {
    // Convert to chat format and then to completion format
    const chatRequest = {
        model: request.model || 'text-davinci-003',
        messages: [{ role: 'user', content: request.prompt || '' }]
    };
    
    const chatResponse = generator.generateOpenAIResponse(chatRequest);
    
    return {
        id: chatResponse.id,
        object: "text_completion",
        created: chatResponse.created,
        model: chatResponse.model,
        choices: [{
            text: chatResponse.choices[0].message.content,
            index: 0,
            logprobs: null,
            finish_reason: chatResponse.choices[0].finish_reason
        }],
        usage: chatResponse.usage
    };
}

function generateModerationResponse(request) {
    const input = request.input || '';
    const analysis = generator.analyzer.analyzeRequest({ message: input });
    
    return {
        id: `modr-${Math.random().toString(36).substring(2, 15)}`,
        model: request.model || 'text-moderation-latest',
        results: [{
            flagged: analysis.isModerated,
            categories: {
                sexual: analysis.categories.includes('moderation_explicit'),
                hate: analysis.categories.includes('moderation_hate'),
                harassment: analysis.categories.includes('moderation_hate'),
                'self-harm': analysis.categories.includes('moderation_selfharm'),
                'sexual/minors': false,
                'hate/threatening': analysis.categories.includes('moderation_hate'),
                'violence/graphic': analysis.categories.includes('moderation_violence'),
                'self-harm/intent': analysis.categories.includes('moderation_selfharm'),
                'self-harm/instructions': analysis.categories.includes('moderation_selfharm'),
                'harassment/threatening': analysis.categories.includes('moderation_hate'),
                violence: analysis.categories.includes('moderation_violence')
            },
            category_scores: {
                sexual: analysis.confidence.moderation_explicit || 0,
                hate: analysis.confidence.moderation_hate || 0,
                harassment: analysis.confidence.moderation_hate || 0,
                'self-harm': analysis.confidence.moderation_selfharm || 0,
                'sexual/minors': 0,
                'hate/threatening': analysis.confidence.moderation_hate || 0,
                'violence/graphic': analysis.confidence.moderation_violence || 0,
                'self-harm/intent': analysis.confidence.moderation_selfharm || 0,
                'self-harm/instructions': analysis.confidence.moderation_selfharm || 0,
                'harassment/threatening': analysis.confidence.moderation_hate || 0,
                violence: analysis.confidence.moderation_violence || 0
            }
        }]
    };
}

function generateAnthropicCompletion(request) {
    // Convert to messages format and then to completion format
    const messagesRequest = {
        model: request.model || 'claude-3-sonnet-20240229',
        messages: [{ role: 'user', content: request.prompt || '' }],
        max_tokens: request.max_tokens_to_sample || request.max_tokens || 1000
    };
    
    const messagesResponse = generator.generateAnthropicResponse(messagesRequest);
    
    return {
        completion: messagesResponse.content[0].text,
        stop_reason: messagesResponse.stop_reason,
        model: messagesResponse.model,
        log_id: messagesResponse.id
    };
}

function handleStreaming(req, res, response, format) {
    res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*'
    });
    
    let content;
    if (format.includes('openai')) {
        content = response.choices[0].message?.content || response.choices[0].text || '';
    } else if (format.includes('anthropic')) {
        content = response.content?.[0]?.text || response.completion || '';
    } else {
        content = response.response || '';
    }
    
    const chunks = content.split(' ');
    let chunkIndex = 0;
    
    const sendChunk = () => {
        if (chunkIndex < chunks.length) {
            const chunkText = chunks[chunkIndex] + (chunkIndex < chunks.length - 1 ? ' ' : '');
            
            let streamData;
            if (format === 'openai-chat') {
                streamData = {
                    id: response.id,
                    object: "chat.completion.chunk",
                    created: response.created,
                    model: response.model,
                    choices: [{
                        index: 0,
                        delta: { content: chunkText },
                        finish_reason: null
                    }]
                };
            } else if (format === 'anthropic-messages') {
                streamData = {
                    type: "content_block_delta",
                    index: 0,
                    delta: { type: "text_delta", text: chunkText }
                };
            } else {
                streamData = { chunk: chunkText, done: false };
            }
            
            res.write(`data: ${JSON.stringify(streamData)}\n\n`);
            chunkIndex++;
            
            setTimeout(sendChunk, 50 + Math.random() * 150);
        } else {
            // Send final chunk
            if (format === 'openai-chat') {
                const finalChunk = {
                    id: response.id,
                    object: "chat.completion.chunk",
                    created: response.created,
                    model: response.model,
                    choices: [{
                        index: 0,
                        delta: {},
                        finish_reason: "stop"
                    }]
                };
                res.write(`data: ${JSON.stringify(finalChunk)}\n\n`);
            }
            
            res.write('data: [DONE]\n\n');
            res.end();
        }
    };
    
    sendChunk();
}

module.exports = router;