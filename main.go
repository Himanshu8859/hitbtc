package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-hitbtc-challenge/hitbtc"
	"github.com/go-hitbtc-challenge/hitbtcwrapper"
	"github.com/gorilla/mux"
)

// MaxBodyBytes is maxbodybytes
const MaxBodyBytes = int64(65536)

var (
	API_KEY    = ""
	API_SECRET = ""
)

type HandleRequests struct {
	HitWrapper *hitbtcwrapper.HitBtcWrapper
}

func (h *HandleRequests) handleRequests() {
	// creates a new instance of a mux router
	myRouter := mux.NewRouter().StrictSlash(true)
	// replace http.HandleFunc with myRouter.HandleFunc
	myRouter.HandleFunc("/currency/{symbol}", h.handleCurrencyBySymbol).Methods("GET")
	myRouter.HandleFunc("/currency/", h.handleAllCurrency).Methods("GET")
	log.Fatal(http.ListenAndServe(":8081", myRouter))
}

func main() {
	fmt.Println("Rest API v2.0 - HIT BTC")
	h := &HandleRequests{
		HitWrapper: hitbtcwrapper.NewHitBtcV2Wrapper(API_KEY, API_SECRET),
	}
	err := h.HitWrapper.CacheAllSymbols()
	if err != nil {
		fmt.Println(err)
	}
	err = h.HitWrapper.CacheFullName()
	if err != nil {
		fmt.Println(err)
	}
	err = h.subscribeMarketFeeds()
	if err != nil {
		fmt.Println(err)
	}

	h.handleRequests()
}

type Response struct {
	Currencies []*hitbtc.Ticker `json:"currencies"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *HandleRequests) handleAllCurrency(w http.ResponseWriter, req *http.Request) {
	currencies, err := h.GetAllCurrencies()
	if err != nil {
		errorBody, _ := json.Marshal(&ErrorResponse{Error: err.Error()})
		writeResponse(w, http.StatusInternalServerError, errorBody)
		return
	}
	if len(currencies) == 0 {
		errorBody, _ := json.Marshal(&ErrorResponse{Error: "No data Found"})
		writeResponse(w, http.StatusNotFound, errorBody)
		return
	}

	var response Response
	response.Currencies = currencies
	currenciesJSON, err := json.Marshal(response)
	if err != nil {
		errorBody, _ := json.Marshal(&ErrorResponse{Error: err.Error()})
		writeResponse(w, http.StatusInternalServerError, errorBody)
		return
	}
	writeResponse(w, http.StatusOK, currenciesJSON)
}

func (h *HandleRequests) handleCurrencyBySymbol(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["symbol"]
	var currenciesJSON []byte
	if h.HitWrapper.Contains(h.HitWrapper.AllSymbols, key) {
		currency, err := h.HitWrapper.GetMarketSummary(key)
		if err != nil {
			errorBody, _ := json.Marshal(&ErrorResponse{Error: err.Error()})
			writeResponse(w, http.StatusInternalServerError, errorBody)
			return
		}
		if currency == nil {
			errorBody, _ := json.Marshal(&ErrorResponse{Error: "No data Found"})
			writeResponse(w, http.StatusNotFound, errorBody)
			return
		}
		currenciesJSON, err = json.Marshal(currency)
		if err != nil {
			errorBody, _ := json.Marshal(&ErrorResponse{Error: err.Error()})
			writeResponse(w, http.StatusInternalServerError, errorBody)
			return
		}
	} else {
		errorBody, _ := json.Marshal(&ErrorResponse{Error: "Not a valid Symbol"})
		writeResponse(w, http.StatusNotFound, errorBody)
		return
	}

	writeResponse(w, http.StatusOK, currenciesJSON)
}

func (h *HandleRequests) subscribeMarketFeeds() error {
	err := h.HitWrapper.FeedConnect()
	if err != nil {
		return err
	}
	return nil
}

// isSymbolValid checks if a string is present in a slice
func isSymbolValid(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func (h *HandleRequests) GetAllCurrencies() ([]*hitbtc.Ticker, error) {
	data, err := h.HitWrapper.GetCurrenciesFromCache()
	if err != nil {
		return nil, err
	}
	return data, nil
}
func writeResponse(w http.ResponseWriter, code int, response []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
