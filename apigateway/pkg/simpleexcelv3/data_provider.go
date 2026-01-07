package simpleexcelv3

import (
	"fmt"
	"reflect"
	"sync"
)

// DataProvider defines the contract for accessing data row-by-row
type DataProvider interface {
	// GetRow returns the data for a specific row index
	// Returns nil if row doesn't exist
	GetRow(rowIndex int) (interface{}, error)

	// GetRowCount returns the total number of rows and whether it's known
	// Returns (0, false) if unknown (streaming data)
	GetRowCount() (int, bool)

	// HasMoreRows returns true if there are more rows available
	HasMoreRows() bool

	// Close releases any resources held by the provider
	Close() error
}

// SliceDataProvider implements DataProvider for in-memory slices
type SliceDataProvider struct {
	data      interface{}
	rowCount  int
	valueType reflect.Type
	currentRow int
	mu        sync.RWMutex
}

// NewSliceDataProvider creates a DataProvider for slice data
func NewSliceDataProvider(data interface{}) (*SliceDataProvider, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("data must be a slice, got %s", v.Kind())
	}

	return &SliceDataProvider{
		data:      data,
		rowCount:  v.Len(),
		valueType: v.Type().Elem(),
		currentRow: 0,
	}, nil
}

func (p *SliceDataProvider) GetRow(rowIndex int) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	v := reflect.ValueOf(p.data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if rowIndex < 0 || rowIndex >= v.Len() {
		return nil, nil
	}

	return v.Index(rowIndex).Interface(), nil
}

func (p *SliceDataProvider) GetRowCount() (int, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rowCount, true
}

func (p *SliceDataProvider) HasMoreRows() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentRow < p.rowCount
}

func (p *SliceDataProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentRow = 0
	return nil
}

// ChannelDataProvider implements DataProvider for streaming data
type ChannelDataProvider struct {
	dataChan <-chan interface{}
	buffer   []interface{}
	closed   bool
	mu       sync.RWMutex
}

// NewChannelDataProvider creates a DataProvider for channel data
func NewChannelDataProvider(dataChan <-chan interface{}) *ChannelDataProvider {
	return &ChannelDataProvider{
		dataChan: dataChan,
		buffer:   make([]interface{}, 0),
		closed:   false,
	}
}

func (p *ChannelDataProvider) GetRow(rowIndex int) (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Fill buffer if needed
	if rowIndex >= len(p.buffer) && !p.closed {
		p.fillBuffer(rowIndex + 1)
	}

	if rowIndex < len(p.buffer) {
		return p.buffer[rowIndex], nil
	}

	return nil, nil
}

func (p *ChannelDataProvider) fillBuffer(targetSize int) {
	for len(p.buffer) < targetSize && !p.closed {
		select {
		case item, ok := <-p.dataChan:
			if !ok {
				p.closed = true
				return
			}
			p.buffer = append(p.buffer, item)
		}
	}
}

func (p *ChannelDataProvider) GetRowCount() (int, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return len(p.buffer), true
	}
	return 0, false // Unknown until channel is closed
}

func (p *ChannelDataProvider) HasMoreRows() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed || len(p.buffer) > 0
}

func (p *ChannelDataProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.buffer = nil
	return nil
}

// IteratorDataProvider implements DataProvider for custom iteration logic
type IteratorDataProvider struct {
	iterator func() (interface{}, bool, error)
	currentRow int
	hasNext   bool
	nextItem  interface{}
	err       error
	mu        sync.RWMutex
}

// NewIteratorDataProvider creates a DataProvider for custom iterator functions
func NewIteratorDataProvider(iterator func() (interface{}, bool, error)) *IteratorDataProvider {
	return &IteratorDataProvider{
		iterator: iterator,
		currentRow: 0,
		hasNext:  true,
	}
}

func (p *IteratorDataProvider) GetRow(rowIndex int) (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If we're requesting a row beyond what we've iterated, continue iterating
	for p.currentRow <= rowIndex && p.hasNext && p.err == nil {
		p.nextItem, p.hasNext, p.err = p.iterator()
		if p.err != nil {
			return nil, p.err
		}
		if p.hasNext {
			p.currentRow++
		}
	}

	// Check if we have the requested row
	if rowIndex < p.currentRow {
		// We've already passed this row, can't go back
		return nil, fmt.Errorf("cannot access row %d, already passed", rowIndex)
	}

	if p.err != nil {
		return nil, p.err
	}

	if !p.hasNext && rowIndex >= p.currentRow {
		return nil, nil
	}

	return p.nextItem, nil
}

func (p *IteratorDataProvider) GetRowCount() (int, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// For iterator, we don't know the count until we've iterated through everything
	return 0, false
}

func (p *IteratorDataProvider) HasMoreRows() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hasNext && p.err == nil
}

func (p *IteratorDataProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hasNext = false
	p.nextItem = nil
	p.err = nil
	return nil
}