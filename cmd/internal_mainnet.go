package main

import (
	"log"
	"time"

	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/data/fetcher"
	"github.com/KyberNetwork/reserve-data/data/storage"
	"github.com/KyberNetwork/reserve-data/http"
	"github.com/KyberNetwork/reserve-data/signer"
	ethereum "github.com/ethereum/go-ethereum/common"
)

func GetConfigForInternalMainnet() *Config {
	settingPath := "/go/src/github.com/KyberNetwork/reserve-data/cmd/internal_mainnet_setting.json"
	addressConfig, err := common.GetAddressConfigFromFile(settingPath)
	if err != nil {
		log.Fatalf("Config file %s is not found. Error: %s", settingPath, err)
	}
	wrapperAddr := ethereum.HexToAddress(addressConfig.Wrapper)
	reserveAddr := ethereum.HexToAddress(addressConfig.Reserve)
	pricingAddr := ethereum.HexToAddress(addressConfig.Pricing)
	burnerAddr := ethereum.HexToAddress(addressConfig.FeeBurner)
	networkAddr := ethereum.HexToAddress(addressConfig.Network)

	common.SupportedTokens = map[string]common.Token{}
	tokens := []common.Token{}
	for id, t := range addressConfig.Tokens {
		tok := common.Token{
			id, t.Address, t.Decimals,
		}
		common.SupportedTokens[id] = tok
		tokens = append(tokens, tok)
	}

	storage, err := storage.NewBoltStorage("/go/src/github.com/KyberNetwork/reserve-data/cmd/internal_mainnet.db")
	if err != nil {
		panic(err)
	}

	fetcherRunner := fetcher.NewTickerRunner(3*time.Second, 2*time.Second, 3*time.Second, 5*time.Second, 5*time.Second)

	fileSigner := signer.NewFileSigner("/go/src/github.com/KyberNetwork/reserve-data/cmd/internal_mainnet_config.json")

	exchangePool := NewMainnetExchangePool(
		addressConfig, fileSigner, storage,
	)

	hmac512auth := http.KNAuthentication{
		fileSigner.KNSecret,
		fileSigner.KNReadOnly,
		fileSigner.KNConfiguration,
	}

	endpoint := "https://mainnet.infura.io"

	return &Config{
		ActivityStorage:      storage,
		DataStorage:          storage,
		FetcherStorage:       storage,
		MetricStorage:        storage,
		FetcherRunner:        fetcherRunner,
		FetcherExchanges:     exchangePool.FetcherExchanges(),
		Exchanges:            exchangePool.CoreExchanges(),
		BlockchainSigner:     fileSigner,
		EnableAuthentication: true,
		AuthEngine:           hmac512auth,
		EthereumEndpoint:     endpoint,
		SupportedTokens:      tokens,
		WrapperAddress:       wrapperAddr,
		PricingAddress:       pricingAddr,
		ReserveAddress:       reserveAddr,
		FeeBurnerAddress:     burnerAddr,
		NetworkAddress:       networkAddr,
	}
}
