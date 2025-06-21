module.exports = {
    largePayload: [
        // Massive text blocks
        "A".repeat(10000),
        "SPAM ".repeat(5000),
        "X".repeat(50000),
        "Lorem ipsum dolor sit amet ".repeat(1000),
        
        // Repeated patterns
        "Test message. ".repeat(2000),
        "DDoS attack simulation! ".repeat(500),
        "1234567890".repeat(1000),
        
        // Complex nested structures (as strings)
        JSON.stringify({ data: Array(1000).fill({ nested: "data" }) }),
        JSON.stringify({ level1: { level2: { level3: { level4: { level5: "deep" } } } } }),
        
        // Unicode spam
        "🔥".repeat(5000),
        "😀😃😄😁😆😅😂🤣".repeat(500),
        "★☆".repeat(2000),
        
        // Mixed content spam
        ("URGENT!!! " + "A".repeat(100) + " BUY NOW!!! ").repeat(50),
        ("!@#$%^&*()_+" + "Z".repeat(50)).repeat(100)
    ],
    
    rapidFire: [
        // Simple rapid messages
        "Test 1", "Test 2", "Test 3", "Test 4", "Test 5",
        "Message A", "Message B", "Message C", "Message D",
        "Request 001", "Request 002", "Request 003",
        
        // Incrementing patterns
        ...Array(100).fill(null).map((_, i) => `Rapid request ${i}`),
        ...Array(50).fill(null).map((_, i) => `Burst message #${i}`),
        
        // Timestamp-based
        ...Array(20).fill(null).map(() => `Request at ${Date.now()}`),
        
        // Random IDs
        ...Array(30).fill(null).map(() => `Request ID: ${Math.random()}`),
    ],
    
    slowloris: [
        // Incomplete requests (simulated)
        "This is a very slow message that...",
        "Starting to type something but...",
        "Incomplete request that never...",
        "Partial message without...",
        
        // Long processing requests
        "Calculate the factorial of 1000000",
        "Generate a list of all prime numbers up to 10000000",
        "Solve this complex equation: " + "x^2 + ".repeat(1000),
        
        // Hanging requests
        "Process this and wait for next instruction...",
        "First part of multi-part request...",
        "Begin transaction...",
        "Start processing..."
    ],
    
    cacheBuster: [
        // Unique parameters each time
        () => `Request with timestamp: ${Date.now()}`,
        () => `Query with random ID: ${Math.random()}`,
        () => `Message with UUID: ${require('uuid').v4()}`,
        () => `Dynamic content: ${new Date().toISOString()}`,
        
        // Random suffixes
        () => `Standard request ${Math.floor(Math.random() * 1000000)}`,
        () => `Cache test ${Buffer.from(Math.random().toString()).toString('base64')}`,
        
        // Unique combinations
        () => `User ${Math.floor(Math.random() * 1000)} session ${Math.floor(Math.random() * 1000)}`,
        () => `Request from IP 192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
        
        // Query parameter variations
        () => `Search for term_${Date.now()}_${Math.random()}`,
        () => `Get data with filter=${Math.random()}&sort=${Date.now()}`
    ],
    
    resourceExhaustion: [
        // CPU intensive requests
        "Calculate pi to 1 million decimal places",
        "Find all permutations of the alphabet",
        "Generate all possible 10-character passwords",
        "Compute fibonacci sequence up to position 1000000",
        
        // Memory intensive
        "Create a list of all integers from 1 to 100000000",
        "Generate a dictionary with 1 million entries",
        "Build a matrix of size 10000x10000",
        
        // Complex parsing
        "Parse this: " + "(((" + "nested".repeat(100) + ")))",
        "Evaluate: " + Array(100).fill("1+").join("") + "1",
        "Process XML: " + "<tag>".repeat(1000) + "data" + "</tag>".repeat(1000),
        
        // Regex bombs
        "Match this pattern: " + "a?".repeat(30) + "a".repeat(30),
        "Find all occurrences of: (a+)+b in " + "a".repeat(100)
    ],
    
    protocolAbuse: [
        // HTTP method fuzzing (as message content)
        "GET /../../etc/passwd",
        "POST /%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
        "PUT /admin/config",
        "DELETE /api/users/*",
        
        // Header injection attempts (as content)
        "Message with headers: \\r\\nX-Injected: true\\r\\n",
        "Content with CRLF: \\r\\n\\r\\nNew request",
        
        // Path traversal
        "../../../etc/passwd",
        "..\\..\\..\\windows\\system32\\config",
        "%2e%2e%2f%2e%2e%2f%2e%2e%2f",
        
        // SQL injection in messages
        "'; DROP TABLE users; --",
        "1' OR '1'='1",
        "admin'--",
        "1; DELETE FROM messages WHERE 1=1;",
        
        // XSS attempts
        "<script>alert('XSS')</script>",
        "<img src=x onerror=alert('XSS')>",
        "javascript:alert('XSS')",
        "<iframe src='javascript:alert(`xss`)'>"
    ]
};