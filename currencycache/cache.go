package currencycache

import (
	"errors"
	"sync"

	"github.com/go-hitbtc-challenge/hitbtc"
)

// CurrencyCache represents a local summary cache for every exchange. To allow dinamic polling from multiple sources (REST + Websocket)
type CurrencyCache struct {
	mutex    *sync.RWMutex
	internal map[string]*hitbtc.Ticker
}

// NewCurrencyCache creates a new SummaryCache Object
func NewCurrencyCache() *CurrencyCache {
	return &CurrencyCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[string]*hitbtc.Ticker),
	}
}

// Set sets a value for the specified key.
func (sc *CurrencyCache) Set(currencySymbol string, data *hitbtc.Ticker) *hitbtc.Ticker {
	sc.mutex.Lock()
	old := sc.internal[currencySymbol]
	sc.internal[currencySymbol] = data
	sc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (sc *CurrencyCache) Get(currencySymbol string) (*hitbtc.Ticker, bool) {
	sc.mutex.RLock()
	ret, isSet := sc.internal[currencySymbol]
	sc.mutex.RUnlock()
	return ret, isSet
}

// GetAll gets the value for the whole data.
func (sc *CurrencyCache) GetAll() ([]*hitbtc.Ticker, error) {
	allData := make([]*hitbtc.Ticker, 0)
	for i, _ := range sc.internal {
		sc.mutex.RLock()
		ret := sc.internal[i]
		allData = append(allData, ret)
		sc.mutex.RUnlock()
	}
	if len(allData) == 0 {
		return nil, errors.New("No data present")
	}
	return allData, nil
}
