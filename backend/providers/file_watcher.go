package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileWatcher watches a file for changes and triggers callbacks
type FileWatcher struct {
	filePath     string
	callback     func()
	stopChan     chan struct{}
	modTime      time.Time
	checkInterval time.Duration
	running      bool
	mutex        sync.RWMutex
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(filePath string, callback func()) (*FileWatcher, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get initial modification time
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &FileWatcher{
		filePath:      absPath,
		callback:      callback,
		stopChan:      make(chan struct{}),
		modTime:       info.ModTime(),
		checkInterval: 2 * time.Second, // Check every 2 seconds
		running:       false,
	}, nil
}

// Start starts watching the file for changes
func (fw *FileWatcher) Start() error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	if fw.running {
		return fmt.Errorf("file watcher already running")
	}

	fw.running = true
	go fw.watchLoop()

	return nil
}

// Stop stops watching the file
func (fw *FileWatcher) Stop() error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	if !fw.running {
		return nil // Already stopped
	}

	close(fw.stopChan)
	fw.running = false

	// Create new stop channel for next start
	fw.stopChan = make(chan struct{})

	return nil
}

// IsRunning returns whether the file watcher is currently running
func (fw *FileWatcher) IsRunning() bool {
	fw.mutex.RLock()
	defer fw.mutex.RUnlock()
	return fw.running
}

// SetCheckInterval sets the interval for checking file changes
func (fw *FileWatcher) SetCheckInterval(interval time.Duration) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()
	fw.checkInterval = interval
}

// watchLoop is the main loop that checks for file changes
func (fw *FileWatcher) watchLoop() {
	ticker := time.NewTicker(fw.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.stopChan:
			return
		case <-ticker.C:
			fw.checkFileChange()
		}
	}
}

// checkFileChange checks if the file has been modified
func (fw *FileWatcher) checkFileChange() {
	info, err := os.Stat(fw.filePath)
	if err != nil {
		// File might have been deleted or moved
		fmt.Printf("File watcher error: %v\n", err)
		return
	}

	fw.mutex.Lock()
	currentModTime := fw.modTime
	fw.mutex.Unlock()

	if info.ModTime().After(currentModTime) {
		fw.mutex.Lock()
		fw.modTime = info.ModTime()
		fw.mutex.Unlock()

		// File has been modified, trigger callback
		if fw.callback != nil {
			fw.callback()
		}
	}
}

// GetFilePath returns the file path being watched
func (fw *FileWatcher) GetFilePath() string {
	return fw.filePath
}

// GetLastModTime returns the last known modification time
func (fw *FileWatcher) GetLastModTime() time.Time {
	fw.mutex.RLock()
	defer fw.mutex.RUnlock()
	return fw.modTime
}

// GetFileInfo returns current file information
func (fw *FileWatcher) GetFileInfo() (os.FileInfo, error) {
	return os.Stat(fw.filePath)
}

// MultiFileWatcher watches multiple files for changes
type MultiFileWatcher struct {
	watchers map[string]*FileWatcher
	callback func(filePath string)
	mutex    sync.RWMutex
}

// NewMultiFileWatcher creates a new multi-file watcher
func NewMultiFileWatcher(callback func(filePath string)) *MultiFileWatcher {
	return &MultiFileWatcher{
		watchers: make(map[string]*FileWatcher),
		callback: callback,
	}
}

// AddFile adds a file to watch
func (mfw *MultiFileWatcher) AddFile(filePath string) error {
	mfw.mutex.Lock()
	defer mfw.mutex.Unlock()

	if _, exists := mfw.watchers[filePath]; exists {
		return fmt.Errorf("file %s is already being watched", filePath)
	}

	watcher, err := NewFileWatcher(filePath, func() {
		if mfw.callback != nil {
			mfw.callback(filePath)
		}
	})
	if err != nil {
		return err
	}

	if err := watcher.Start(); err != nil {
		return err
	}

	mfw.watchers[filePath] = watcher
	return nil
}

// RemoveFile removes a file from watching
func (mfw *MultiFileWatcher) RemoveFile(filePath string) error {
	mfw.mutex.Lock()
	defer mfw.mutex.Unlock()

	watcher, exists := mfw.watchers[filePath]
	if !exists {
		return fmt.Errorf("file %s is not being watched", filePath)
	}

	if err := watcher.Stop(); err != nil {
		return err
	}

	delete(mfw.watchers, filePath)
	return nil
}

// Start starts watching all files
func (mfw *MultiFileWatcher) Start() error {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()

	for _, watcher := range mfw.watchers {
		if !watcher.IsRunning() {
			if err := watcher.Start(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Stop stops watching all files
func (mfw *MultiFileWatcher) Stop() error {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()

	var lastErr error
	for _, watcher := range mfw.watchers {
		if err := watcher.Stop(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetWatchedFiles returns a list of watched file paths
func (mfw *MultiFileWatcher) GetWatchedFiles() []string {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()

	files := make([]string, 0, len(mfw.watchers))
	for filePath := range mfw.watchers {
		files = append(files, filePath)
	}

	return files
}

// GetFileWatcher returns the file watcher for a specific file
func (mfw *MultiFileWatcher) GetFileWatcher(filePath string) (*FileWatcher, bool) {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()

	watcher, exists := mfw.watchers[filePath]
	return watcher, exists
}

// IsWatching returns whether a file is being watched
func (mfw *MultiFileWatcher) IsWatching(filePath string) bool {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()

	_, exists := mfw.watchers[filePath]
	return exists
}

// GetWatchedFileCount returns the number of files being watched
func (mfw *MultiFileWatcher) GetWatchedFileCount() int {
	mfw.mutex.RLock()
	defer mfw.mutex.RUnlock()
	return len(mfw.watchers)
}