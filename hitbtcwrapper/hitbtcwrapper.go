package hitbtcwrapper

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-hitbtc-challenge/currencycache"
	"github.com/go-hitbtc-challenge/hitbtc"
)

var SymbolsFeeCurrency = make(map[string]string, 0)
var CurrencyFullName = make(map[string]string, 0)

type HitBtcWrapper struct {
	api         *hitbtc.HitBtc
	ws          *hitbtc.WSClient
	websocketOn bool
	summaries   *currencycache.CurrencyCache
	AllSymbols  []string
}

// NewHitBtcV2Wrapper creates a generic wrapper of the HitBtc API v2.0.
func NewHitBtcV2Wrapper(publicKey string, secretKey string) *HitBtcWrapper {
	ws, _ := hitbtc.NewWSClient()
	return &HitBtcWrapper{
		api:         hitbtc.New(publicKey, secretKey),
		ws:          ws,
		websocketOn: false,
		summaries:   currencycache.NewCurrencyCache(),
	}
}

// GetTicker gets the updated ticker for a market.
func (wrapper *HitBtcWrapper) GetTicker(symbol string) (*hitbtc.Ticker, error) {
	hitbtcTicker, err := wrapper.api.GetTicker(symbol)
	if err != nil {
		return nil, err
	}

	return &hitbtc.Ticker{
		Last:        hitbtcTicker.Last,
		Ask:         hitbtcTicker.Ask,
		Bid:         hitbtcTicker.Bid,
		Open:        hitbtcTicker.Open,
		Low:         hitbtcTicker.Low,
		High:        hitbtcTicker.High,
		Volume:      hitbtcTicker.Volume,
		VolumeQuote: hitbtcTicker.VolumeQuote,
		Symbol:      hitbtcTicker.Symbol,
		Timestamp:   hitbtcTicker.Timestamp,
	}, nil
}

// GetMarketSummary gets the current market summary.
func (wrapper *HitBtcWrapper) GetMarketSummary(symbol string) (*hitbtc.Ticker, error) {
	ret, exists := wrapper.summaries.Get(symbol)
	if !exists {
		hitbtcTicker, err := wrapper.GetTicker(symbol)
		if err != nil {
			return nil, err
		}
		feeCurrency := ""
		_, ok := SymbolsFeeCurrency[hitbtcTicker.Symbol]
		if ok {
			feeCurrency = SymbolsFeeCurrency[hitbtcTicker.Symbol]
		}
		fullName := ""
		_, ok = CurrencyFullName[feeCurrency]
		if ok {
			fullName = CurrencyFullName[feeCurrency]
		}
		ret = &hitbtc.Ticker{
			Last:        hitbtcTicker.Last,
			Ask:         hitbtcTicker.Ask,
			Bid:         hitbtcTicker.Bid,
			Open:        hitbtcTicker.Open,
			Low:         hitbtcTicker.Low,
			High:        hitbtcTicker.High,
			Volume:      hitbtcTicker.Volume,
			VolumeQuote: hitbtcTicker.VolumeQuote,
			Symbol:      hitbtcTicker.Symbol,
			Timestamp:   hitbtcTicker.Timestamp,
			FeeCurrency: feeCurrency,
			FullName:    fullName,
			ID:          hitbtcTicker.Symbol,
		}
		if wrapper.Contains(supportedSymbols, hitbtcTicker.Symbol) {
			wrapper.summaries.Set(symbol, ret)
		}
		return ret, nil
	}

	return ret, nil
}

// subscribeFeeds subscribes to the Market Summary Feed service.
func (wrapper *HitBtcWrapper) subscribeFeeds(symbol string, closeChan chan bool, c chan os.Signal) error {
	handleTicker := func(wrapper *HitBtcWrapper, currencyChannel <-chan hitbtc.WSNotificationTickerResponse, m string) {
		for {
			select {
			case <-closeChan:

				wrapper.Close(symbol)
				return
			default:
				hitbtcSummary, stillOpen := <-currencyChannel
				if !stillOpen {
					return
				}
				last, _ := strconv.ParseFloat(hitbtcSummary.Last, 64)
				ask, _ := strconv.ParseFloat(hitbtcSummary.Ask, 64)
				bid, _ := strconv.ParseFloat(hitbtcSummary.Bid, 64)
				open, _ := strconv.ParseFloat(hitbtcSummary.Open, 64)
				low, _ := strconv.ParseFloat(hitbtcSummary.Low, 64)
				high, _ := strconv.ParseFloat(hitbtcSummary.High, 64)
				volume, _ := strconv.ParseFloat(hitbtcSummary.Volume, 64)
				volumeQuota, _ := strconv.ParseFloat(hitbtcSummary.VolumeQuote, 64)
				feeCurrency := ""
				_, ok := SymbolsFeeCurrency[hitbtcSummary.Symbol]
				if ok {
					feeCurrency = SymbolsFeeCurrency[hitbtcSummary.Symbol]
				}
				fullName := ""
				_, ok = CurrencyFullName[feeCurrency]
				if ok {
					fullName = CurrencyFullName[feeCurrency]
				}
				sum := &hitbtc.Ticker{
					Last:        last,
					Ask:         ask,
					Bid:         bid,
					Open:        open,
					Low:         low,
					High:        high,
					Volume:      volume,
					VolumeQuote: volumeQuota,
					Symbol:      hitbtcSummary.Symbol,
					FeeCurrency: feeCurrency,
					FullName:    fullName,
					ID:          hitbtcSummary.Symbol,
				}
				if wrapper.Contains(supportedSymbols, hitbtcSummary.Symbol) {
					wrapper.summaries.Set(symbol, sum)
				}

			}
		}
	}
	summaryChannel, err := wrapper.ws.SubscribeTicker(symbol)
	if err != nil {
		return err
	}

	go handleTicker(wrapper, summaryChannel, symbol)
	return nil
}

// FeedConnect connects to the feed of the exchange.
func (wrapper *HitBtcWrapper) FeedConnect() error {
	wrapper.websocketOn = true
	closeChan := make(chan bool)
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		os.Exit(0)
	}()
	for _, m := range supportedSymbols {
		err := wrapper.subscribeFeeds(m, closeChan, ch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (wrapper *HitBtcWrapper) Close(m string) {
	wrapper.ws.UnsubscribeTicker(m)
}

func (wrapper *HitBtcWrapper) CacheAllSymbols() error {
	symbolsrecords, err := wrapper.api.GetSymbols()
	if err != nil {
		return err
	}
	var symbols []string
	for _, sym := range symbolsrecords {
		SymbolsFeeCurrency[sym.Id] = sym.FeeCurrency
		symbols = append(symbols, sym.Id)
	}
	wrapper.AllSymbols = symbols
	return nil
}

func (wrapper *HitBtcWrapper) GetCurrenciesFromCache() ([]*hitbtc.Ticker, error) {
	allRecords, err := wrapper.summaries.GetAll()
	if err != nil {
		return nil, err
	}
	return allRecords, nil
}

func (wrapper *HitBtcWrapper) CacheFullName() error {
	currencyRecords, err := wrapper.api.GetCurrencies()
	if err != nil {
		return err
	}
	for _, currency := range currencyRecords {
		CurrencyFullName[currency.Id] = currency.FullName
	}
	return nil
}

// Contains checks if a string is present in a slice
func (wrapper *HitBtcWrapper) Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
