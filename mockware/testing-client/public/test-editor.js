// Global variables
let currentCategory = null;
let currentSubcategory = null;
let testData = {};
let hasUnsavedChanges = false;

// Category configurations
const categoryConfig = {
    moderation: {
        name: 'Moderation Tests',
        subcategories: ['violence', 'hateSpeech', 'explicit', 'harmful', 'spam']
    },
    injection: {
        name: 'Injection Tests',
        subcategories: ['direct', 'encoded', 'sophisticated', 'creative', 'multilingual']
    },
    pii: {
        name: 'PII Detection Tests',
        subcategories: ['email', 'phone', 'ssn', 'creditCard', 'mixed']
    },
    ddos: {
        name: 'DDoS Tests',
        subcategories: ['largePayload', 'rapidFire', 'cacheBuster', 'resourceExhaustion']
    },
    combination: {
        name: 'Combination Tests',
        subcategories: ['piiAndInjection', 'violenceAndMisspelling', 'hateAndEncoding', 'multiVector', 'obfuscationMix', 'contextSwitching']
    },
    'edge-cases': {
        name: 'Edge Case Tests',
        subcategories: ['unicodeAbuse', 'boundaryTesting', 'parsingAttacks', 'semanticTricks', 'polyglotAttacks']
    },
    relevance: {
        name: 'Relevance Tests',
        subcategories: ['offTopic', 'contextDrift', 'irrelevant', 'nonsensical']
    }
};

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    setupCategoryButtons();
    loadAllTestData();
    setupKeyboardShortcuts();
    setupBeforeUnload();
});

// Setup category buttons
function setupCategoryButtons() {
    const categoryButtons = document.getElementById('categoryButtons');
    
    Object.keys(categoryConfig).forEach(category => {
        const btn = document.createElement('div');
        btn.className = 'category-btn';
        btn.dataset.category = category;
        btn.textContent = categoryConfig[category].name;
        btn.addEventListener('click', () => selectCategory(category));
        categoryButtons.appendChild(btn);
    });
}

// Select a category
function selectCategory(category) {
    if (hasUnsavedChanges && !confirm('You have unsaved changes. Continue anyway?')) {
        return;
    }
    
    currentCategory = category;
    currentSubcategory = null;
    
    // Update category button states
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.category === category);
    });
    
    // Show subcategory selector
    const subcategorySelector = document.getElementById('subcategorySelector');
    const subcategoryButtons = document.getElementById('subcategoryButtons');
    
    subcategorySelector.style.display = 'block';
    subcategoryButtons.innerHTML = '';
    
    categoryConfig[category].subcategories.forEach(subcategory => {
        const btn = document.createElement('div');
        btn.className = 'subcategory-btn';
        btn.dataset.subcategory = subcategory;
        btn.textContent = formatSubcategoryName(subcategory);
        btn.addEventListener('click', () => selectSubcategory(subcategory));
        subcategoryButtons.appendChild(btn);
    });
    
    document.getElementById('editorPanel').classList.remove('active');
}

// Select a subcategory
function selectSubcategory(subcategory) {
    currentSubcategory = subcategory;
    
    // Update subcategory button states
    document.querySelectorAll('.subcategory-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.subcategory === subcategory);
    });
    
    showEditor();
}

// Show the editor panel
function showEditor() {
    const editorPanel = document.getElementById('editorPanel');
    editorPanel.classList.add('active');
    
    updateCategoryTitle();
    loadCurrentData();
    updateStats();
    renderRequestsList();
    updateBulkExport();
}

// Update category title
function updateCategoryTitle() {
    const title = document.getElementById('categoryTitle');
    title.textContent = `- ${categoryConfig[currentCategory].name} > ${formatSubcategoryName(currentSubcategory)}`;
}

// Format subcategory name for display
function formatSubcategoryName(subcategory) {
    return subcategory
        .replace(/([A-Z])/g, ' $1')
        .replace(/^./, str => str.toUpperCase())
        .trim();
}

// Load all test data
async function loadAllTestData() {
    try {
        const response = await fetch('/api/test-data');
        if (response.ok) {
            testData = await response.json();
        } else {
            showStatus('Failed to load test data', 'error');
        }
    } catch (error) {
        console.error('Error loading test data:', error);
        showStatus('Error loading test data', 'error');
    }
}

// Load current category/subcategory data
function loadCurrentData() {
    if (!currentCategory) return;
    
    if (!testData[currentCategory]) {
        testData[currentCategory] = {};
    }
    
    if (currentSubcategory && !testData[currentCategory][currentSubcategory]) {
        testData[currentCategory][currentSubcategory] = [];
    }
}

// Get current requests array
function getCurrentRequests() {
    if (!currentCategory) return [];
    
    if (currentSubcategory) {
        return testData[currentCategory]?.[currentSubcategory] || [];
    } else {
        return testData[currentCategory] || [];
    }
}

// Set current requests array
function setCurrentRequests(requests) {
    if (!currentCategory) return;
    
    if (currentSubcategory) {
        if (!testData[currentCategory]) testData[currentCategory] = {};
        testData[currentCategory][currentSubcategory] = requests;
    } else {
        testData[currentCategory] = requests;
    }
    
    hasUnsavedChanges = true;
}

// Update statistics
function updateStats() {
    const requests = getCurrentRequests();
    
    document.getElementById('totalRequests').textContent = requests.length;
    
    const avgLength = requests.length > 0 
        ? Math.round(requests.reduce((sum, req) => sum + req.length, 0) / requests.length)
        : 0;
    document.getElementById('avgLength').textContent = avgLength;
    
    document.getElementById('lastModified').textContent = hasUnsavedChanges ? 'Modified' : 'Saved';
}

// Render requests list
function renderRequestsList() {
    const requestsList = document.getElementById('requestsList');
    const requests = getCurrentRequests();
    
    requestsList.innerHTML = '';
    
    requests.forEach((request, index) => {
        const item = document.createElement('div');
        item.className = 'request-item';
        
        item.innerHTML = `
            <input type="text" class="request-text" value="${escapeHtml(request)}" 
                   onchange="updateRequest(${index}, this.value)"
                   onkeydown="handleRequestKeydown(event, ${index})">
            <div class="request-actions">
                <button class="action-btn duplicate" onclick="duplicateRequest(${index})" title="Duplicate">📋</button>
                <button class="action-btn delete" onclick="deleteRequest(${index})" title="Delete">🗑️</button>
            </div>
        `;
        
        requestsList.appendChild(item);
    });
}

// Handle keyboard shortcuts in request inputs
function handleRequestKeydown(event, index) {
    if (event.key === 'Enter' && event.ctrlKey) {
        event.preventDefault();
        duplicateRequest(index);
    } else if (event.key === 'Delete' && event.ctrlKey) {
        event.preventDefault();
        deleteRequest(index);
    }
}

// Add new request
function addRequest() {
    const input = document.getElementById('newRequestInput');
    const text = input.value.trim();
    
    if (!text) {
        alert('Please enter a request text');
        return;
    }
    
    const requests = getCurrentRequests();
    requests.push(text);
    setCurrentRequests(requests);
    
    input.value = '';
    renderRequestsList();
    updateStats();
    updateBulkExport();
    
    // Focus on the new request
    setTimeout(() => {
        const lastInput = document.querySelector('.request-item:last-child .request-text');
        if (lastInput) lastInput.focus();
    }, 100);
}

// Update existing request
function updateRequest(index, newValue) {
    const requests = getCurrentRequests();
    requests[index] = newValue;
    setCurrentRequests(requests);
    updateStats();
    updateBulkExport();
}

// Duplicate request
function duplicateRequest(index) {
    const requests = getCurrentRequests();
    const duplicated = requests[index];
    requests.splice(index + 1, 0, duplicated);
    setCurrentRequests(requests);
    renderRequestsList();
    updateStats();
    updateBulkExport();
}

// Delete request
function deleteRequest(index) {
    if (!confirm('Delete this request?')) return;
    
    const requests = getCurrentRequests();
    requests.splice(index, 1);
    setCurrentRequests(requests);
    renderRequestsList();
    updateStats();
    updateBulkExport();
}

// Bulk import
function bulkImport() {
    const text = document.getElementById('bulkImportText').value.trim();
    if (!text) {
        alert('Please enter some text to import');
        return;
    }
    
    const newRequests = text.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);
    
    if (newRequests.length === 0) {
        alert('No valid requests found');
        return;
    }
    
    const action = confirm(`Import ${newRequests.length} requests? Choose:\nOK = Add to existing\nCancel = Replace all`);
    
    const requests = action ? getCurrentRequests() : [];
    requests.push(...newRequests);
    setCurrentRequests(requests);
    
    document.getElementById('bulkImportText').value = '';
    renderRequestsList();
    updateStats();
    updateBulkExport();
    
    showStatus(`Imported ${newRequests.length} requests`, 'success');
}

// Update bulk export
function updateBulkExport() {
    const requests = getCurrentRequests();
    document.getElementById('bulkExportText').value = requests.join('\n');
}

// Copy to clipboard
async function copyToClipboard() {
    const text = document.getElementById('bulkExportText').value;
    try {
        await navigator.clipboard.writeText(text);
        showStatus('Copied to clipboard', 'success');
    } catch (error) {
        // Fallback for older browsers
        const textarea = document.getElementById('bulkExportText');
        textarea.select();
        document.execCommand('copy');
        showStatus('Copied to clipboard', 'success');
    }
}

// Clear category
function clearCategory() {
    const categoryName = currentSubcategory 
        ? `${categoryConfig[currentCategory].name} > ${formatSubcategoryName(currentSubcategory)}`
        : categoryConfig[currentCategory].name;
    
    if (!confirm(`Clear all requests in ${categoryName}? This cannot be undone.`)) {
        return;
    }
    
    setCurrentRequests([]);
    renderRequestsList();
    updateStats();
    updateBulkExport();
    showStatus('Category cleared', 'success');
}

// Save changes
async function saveChanges() {
    try {
        const response = await fetch('/api/test-data', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(testData)
        });
        
        if (response.ok) {
            hasUnsavedChanges = false;
            updateStats();
            showStatus('Changes saved successfully', 'success');
        } else {
            const error = await response.text();
            showStatus(`Failed to save: ${error}`, 'error');
        }
    } catch (error) {
        console.error('Error saving:', error);
        showStatus('Error saving changes', 'error');
    }
}

// Reload data
async function reloadData() {
    if (hasUnsavedChanges && !confirm('You have unsaved changes. Reload anyway?')) {
        return;
    }
    
    await loadAllTestData();
    if (currentCategory) {
        loadCurrentData();
        renderRequestsList();
        updateStats();
        updateBulkExport();
    }
    hasUnsavedChanges = false;
    showStatus('Data reloaded', 'success');
}

// Show status message
function showStatus(message, type) {
    const status = document.getElementById('saveStatus');
    status.textContent = message;
    status.className = `save-status ${type}`;
    status.style.display = 'block';
    
    setTimeout(() => {
        status.style.display = 'none';
    }, 3000);
}

// Setup keyboard shortcuts
function setupKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey || e.metaKey) {
            switch (e.key) {
                case 's':
                    e.preventDefault();
                    saveChanges();
                    break;
                case 'r':
                    e.preventDefault();
                    reloadData();
                    break;
            }
        }
    });
    
    // Add new request on Enter in input
    document.getElementById('newRequestInput').addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            addRequest();
        }
    });
}

// Setup before unload warning
function setupBeforeUnload() {
    window.addEventListener('beforeunload', (e) => {
        if (hasUnsavedChanges) {
            e.preventDefault();
            e.returnValue = 'You have unsaved changes. Are you sure you want to leave?';
            return e.returnValue;
        }
    });
}

// Utility function to escape HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
} 