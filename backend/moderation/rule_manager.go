package moderation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// RuleManager handles rule loading, hot-reloading, and lifecycle management
type RuleManager struct {
	engine         *RuleEngine
	config         *RuleEngineConfig
	watcher        *fsnotify.Watcher
	watchedPaths   map[string]bool
	reloadChan     chan string
	stopChan       chan struct{}
	isWatching     bool
	mutex          sync.RWMutex
	lastReloadTime time.Time
	reloadCallback func([]*ModerationRule, error)
}

// RuleChangeEvent represents a rule file change event
type RuleChangeEvent struct {
	Path      string
	Operation string // created, updated, deleted
	Timestamp time.Time
	Error     error
}

// NewRuleManager creates a new rule manager instance
func NewRuleManager(engine *RuleEngine, config *RuleEngineConfig) *RuleManager {
	return &RuleManager{
		engine:       engine,
		config:       config,
		watchedPaths: make(map[string]bool),
		reloadChan:   make(chan string, 100),
		stopChan:     make(chan struct{}),
		isWatching:   false,
	}
}

// LoadRules loads rules from the configured rules path
func (rm *RuleManager) LoadRules() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rulesPath := rm.config.RulesPath
	if rulesPath == "" {
		return fmt.Errorf("rules path not configured")
	}

	// Check if path exists
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		return fmt.Errorf("rules path does not exist: %s", rulesPath)
	}

	// Load rules based on path type
	var rules []*ModerationRule
	var err error

	fileInfo, err := os.Stat(rulesPath)
	if err != nil {
		return fmt.Errorf("failed to stat rules path: %w", err)
	}

	if fileInfo.IsDir() {
		rules, err = rm.loadRulesFromDirectory(rulesPath)
	} else {
		rules, err = rm.loadRulesFromFile(rulesPath)
	}

	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	// Validate total rule count
	if rm.config.MaxRules > 0 && len(rules) > rm.config.MaxRules {
		return fmt.Errorf("rule count (%d) exceeds maximum allowed (%d)", len(rules), rm.config.MaxRules)
	}

	// Update engine with new rules
	err = rm.engine.SetRules(rules)
	if err != nil {
		return fmt.Errorf("failed to set rules in engine: %w", err)
	}

	rm.lastReloadTime = time.Now()
	rm.updateReloadStats()

	log.Printf("Loaded %d rules from %s", len(rules), rulesPath)

	// Trigger callback if configured
	if rm.reloadCallback != nil {
		rm.reloadCallback(rules, nil)
	}

	return nil
}

// loadRulesFromFile loads rules from a single YAML file
func (rm *RuleManager) loadRulesFromFile(filePath string) ([]*ModerationRule, error) {
	return rm.engine.parser.ParseRulesFromFile(filePath)
}

// loadRulesFromDirectory loads rules from all YAML files in a directory
func (rm *RuleManager) loadRulesFromDirectory(dirPath string) ([]*ModerationRule, error) {
	var allRules []*ModerationRule

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() || (!isYAMLFile(path) && !isYMLFile(path)) {
			return nil
		}

		rules, err := rm.loadRulesFromFile(path)
		if err != nil {
			log.Printf("Warning: failed to load rules from %s: %v", path, err)
			return nil // Continue processing other files
		}

		allRules = append(allRules, rules...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk rules directory: %w", err)
	}

	return allRules, nil
}

// StartHotReload starts monitoring rule files for changes
func (rm *RuleManager) StartHotReload() error {
	if !rm.config.HotReload {
		return fmt.Errorf("hot reload is disabled in configuration")
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.isWatching {
		return fmt.Errorf("hot reload is already running")
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	rm.watcher = watcher
	rm.isWatching = true

	// Add paths to watch
	err = rm.addWatchPaths()
	if err != nil {
		rm.watcher.Close()
		rm.isWatching = false
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	// Start monitoring goroutines
	go rm.watchForChanges()
	go rm.processReloadEvents()

	log.Printf("Hot reload started, watching: %s", rm.config.RulesPath)
	return nil
}

// StopHotReload stops monitoring rule files
func (rm *RuleManager) StopHotReload() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if !rm.isWatching {
		return fmt.Errorf("hot reload is not running")
	}

	// Signal stop
	close(rm.stopChan)

	// Close watcher
	if rm.watcher != nil {
		rm.watcher.Close()
	}

	rm.isWatching = false
	log.Println("Hot reload stopped")
	return nil
}

// addWatchPaths adds file/directory paths to the watcher
func (rm *RuleManager) addWatchPaths() error {
	rulesPath := rm.config.RulesPath

	fileInfo, err := os.Stat(rulesPath)
	if err != nil {
		return fmt.Errorf("failed to stat rules path: %w", err)
	}

	if fileInfo.IsDir() {
		// Watch directory and all YAML files in it
		err = rm.watcher.Add(rulesPath)
		if err != nil {
			return fmt.Errorf("failed to watch directory %s: %w", rulesPath, err)
		}
		rm.watchedPaths[rulesPath] = true

		// Also watch existing YAML files individually for more precise change detection
		err = filepath.Walk(rulesPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && (isYAMLFile(path) || isYMLFile(path)) {
				err = rm.watcher.Add(path)
				if err != nil {
					log.Printf("Warning: failed to watch file %s: %v", path, err)
				} else {
					rm.watchedPaths[path] = true
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk rules directory for watching: %w", err)
		}
	} else {
		// Watch single file
		err = rm.watcher.Add(rulesPath)
		if err != nil {
			return fmt.Errorf("failed to watch file %s: %w", rulesPath, err)
		}
		rm.watchedPaths[rulesPath] = true
	}

	return nil
}

// watchForChanges monitors file system events
func (rm *RuleManager) watchForChanges() {
	debounceMap := make(map[string]time.Time)
	debounceInterval := 1 * time.Second // Debounce rapid changes

	for {
		select {
		case event, ok := <-rm.watcher.Events:
			if !ok {
				return
			}

			// Check if this is a relevant file
			if !rm.isRelevantFile(event.Name) {
				continue
			}

			// Debounce rapid changes to the same file
			lastEvent, exists := debounceMap[event.Name]
			if exists && time.Since(lastEvent) < debounceInterval {
				continue
			}
			debounceMap[event.Name] = time.Now()

			// Handle different event types
			switch {
			case event.Op&fsnotify.Write == fsnotify.Write:
				log.Printf("Rule file modified: %s", event.Name)
				rm.scheduleReload(event.Name)

			case event.Op&fsnotify.Create == fsnotify.Create:
				log.Printf("Rule file created: %s", event.Name)
				// Add new files to watch list if they're YAML files
				if isYAMLFile(event.Name) || isYMLFile(event.Name) {
					err := rm.watcher.Add(event.Name)
					if err != nil {
						log.Printf("Failed to watch new file %s: %v", event.Name, err)
					} else {
						rm.watchedPaths[event.Name] = true
					}
				}
				rm.scheduleReload(event.Name)

			case event.Op&fsnotify.Remove == fsnotify.Remove:
				log.Printf("Rule file removed: %s", event.Name)
				delete(rm.watchedPaths, event.Name)
				rm.scheduleReload(event.Name)

			case event.Op&fsnotify.Rename == fsnotify.Rename:
				log.Printf("Rule file renamed: %s", event.Name)
				delete(rm.watchedPaths, event.Name)
				rm.scheduleReload(event.Name)
			}

		case err, ok := <-rm.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)

		case <-rm.stopChan:
			return
		}
	}
}

// processReloadEvents processes scheduled reload events
func (rm *RuleManager) processReloadEvents() {
	reloadTimer := time.NewTimer(rm.config.ReloadInterval)
	pendingReloads := make(map[string]bool)

	for {
		select {
		case path := <-rm.reloadChan:
			pendingReloads[path] = true
			// Reset timer to batch multiple changes
			if !reloadTimer.Stop() {
				<-reloadTimer.C
			}
			reloadTimer.Reset(rm.config.ReloadInterval)

		case <-reloadTimer.C:
			if len(pendingReloads) > 0 {
				log.Printf("Processing reload for %d changed files", len(pendingReloads))
				err := rm.LoadRules()
				if err != nil {
					log.Printf("Failed to reload rules: %v", err)
					if rm.reloadCallback != nil {
						rm.reloadCallback(nil, err)
					}
				}
				// Clear pending reloads
				pendingReloads = make(map[string]bool)
			}

		case <-rm.stopChan:
			if !reloadTimer.Stop() {
				<-reloadTimer.C
			}
			return
		}
	}
}

// scheduleReload schedules a rule reload
func (rm *RuleManager) scheduleReload(path string) {
	select {
	case rm.reloadChan <- path:
	default:
		// Channel is full, skip this event
		log.Printf("Reload channel full, skipping reload for %s", path)
	}
}

// isRelevantFile checks if a file change is relevant for rule reloading
func (rm *RuleManager) isRelevantFile(path string) bool {
	// Check if it's a YAML file
	if !isYAMLFile(path) && !isYMLFile(path) {
		return false
	}

	// Check if it's in our watched paths or a subdirectory
	rulesPath := rm.config.RulesPath
	
	// Make paths absolute for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	absRulesPath, err := filepath.Abs(rulesPath)
	if err != nil {
		return false
	}

	// Check if file is within rules directory
	rel, err := filepath.Rel(absRulesPath, absPath)
	if err != nil {
		return false
	}

	// File is relevant if it's within the rules directory tree
	return !filepath.IsAbs(rel) && !startsWith(rel, "..")
}

// SetReloadCallback sets a callback function to be called when rules are reloaded
func (rm *RuleManager) SetReloadCallback(callback func([]*ModerationRule, error)) {
	rm.reloadCallback = callback
}

// GetReloadStats returns statistics about rule reloading
func (rm *RuleManager) GetReloadStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	return map[string]interface{}{
		"hot_reload_enabled": rm.config.HotReload,
		"is_watching":        rm.isWatching,
		"watched_paths":      len(rm.watchedPaths),
		"last_reload_time":   rm.lastReloadTime,
		"reload_count":       rm.engine.stats.ReloadCount,
		"reload_interval":    rm.config.ReloadInterval.String(),
	}
}

// updateReloadStats updates reload statistics in the engine
func (rm *RuleManager) updateReloadStats() {
	rm.engine.stats.mutex.Lock()
	defer rm.engine.stats.mutex.Unlock()

	rm.engine.stats.ReloadCount++
	rm.engine.stats.LastReload = rm.lastReloadTime
}

// ForceReload forces an immediate rule reload
func (rm *RuleManager) ForceReload() error {
	log.Println("Forcing rule reload...")
	return rm.LoadRules()
}

// ValidateRulesPath validates that the configured rules path is accessible
func (rm *RuleManager) ValidateRulesPath() error {
	rulesPath := rm.config.RulesPath
	if rulesPath == "" {
		return fmt.Errorf("rules path is not configured")
	}

	fileInfo, err := os.Stat(rulesPath)
	if err != nil {
		return fmt.Errorf("rules path is not accessible: %w", err)
	}

	if fileInfo.IsDir() {
		// Check if directory contains any YAML files
		hasYAMLFiles := false
		err = filepath.Walk(rulesPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (isYAMLFile(path) || isYMLFile(path)) {
				hasYAMLFiles = true
				return filepath.SkipAll // Stop walking once we find a YAML file
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error checking directory contents: %w", err)
		}

		if !hasYAMLFiles {
			return fmt.Errorf("rules directory contains no YAML files")
		}
	} else {
		// Single file must be YAML
		if !isYAMLFile(rulesPath) && !isYMLFile(rulesPath) {
			return fmt.Errorf("rules file must be a YAML file (.yaml or .yml)")
		}
	}

	// Try to parse rules to validate syntax
	if fileInfo.IsDir() {
		_, err = rm.loadRulesFromDirectory(rulesPath)
	} else {
		_, err = rm.loadRulesFromFile(rulesPath)
	}

	if err != nil {
		return fmt.Errorf("rules validation failed: %w", err)
	}

	return nil
}

// Helper functions

// isYAMLFile checks if a file has .yaml extension
func isYAMLFile(path string) bool {
	return filepath.Ext(path) == ".yaml"
}

// isYMLFile checks if a file has .yml extension
func isYMLFile(path string) bool {
	return filepath.Ext(path) == ".yml"
}

// startsWith checks if a string starts with a prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// SetRules updates the engine with new rules (implementing the missing method)
func (re *RuleEngine) SetRules(rules []*ModerationRule) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Validate all rules
	for _, rule := range rules {
		if err := re.parser.ValidateRule(rule); err != nil {
			return fmt.Errorf("rule validation failed: %w", err)
		}
	}

	// Clear existing rules
	re.rules = make([]*ModerationRule, 0, len(rules))
	re.rulesByID = make(map[string]*ModerationRule)
	re.rulesByType = make(map[string][]*ModerationRule)

	// Add new rules
	for _, rule := range rules {
		re.rules = append(re.rules, rule)
		re.rulesByID[rule.ID] = rule
		
		// Group by type
		if _, exists := re.rulesByType[rule.Type]; !exists {
			re.rulesByType[rule.Type] = make([]*ModerationRule, 0)
		}
		re.rulesByType[rule.Type] = append(re.rulesByType[rule.Type], rule)
	}

	// Update statistics
	re.stats.mutex.Lock()
	re.stats.TotalRules = int64(len(rules))
	re.stats.ActiveRules = int64(re.countActiveRules())
	
	// Update rule type statistics
	for ruleType := range re.stats.RuleTypeStats {
		re.stats.RuleTypeStats[ruleType].Count = 0
	}
	
	for _, rule := range rules {
		if typeStats, exists := re.stats.RuleTypeStats[rule.Type]; exists {
			typeStats.Count++
		} else {
			// Create stats for new rule type
			re.stats.RuleTypeStats[rule.Type] = &RuleTypeStats{
				Count: 1,
			}
		}
	}
	re.stats.mutex.Unlock()

	re.lastReload = time.Now()
	return nil
}

// countActiveRules counts the number of enabled rules
func (re *RuleEngine) countActiveRules() int {
	count := 0
	for _, rule := range re.rules {
		if rule.Enabled {
			count++
		}
	}
	return count
}