package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/KyberNetwork/reserve-data/boltutil"
	"github.com/KyberNetwork/reserve-data/common"
	"github.com/KyberNetwork/reserve-data/settings"
	"github.com/boltdb/bolt"
	ethereum "github.com/ethereum/go-ethereum/common"
)

const token_version = "token_version"

func getTokenCreationTime(b *bolt.Bucket, key []byte) (uint64, error) {
	v := b.Get(key)
	if v == nil {
		return common.GetTimepoint(), nil
	}
	var temp common.Token
	err := json.Unmarshal(v, &temp)
	if err != nil {
		return 0, err
	}
	return temp.CreationTime, nil
}

func addTokenByID(tx *bolt.Tx, t common.Token) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ID))
	if uErr != nil {
		return uErr
	}
	key := []byte(strings.ToLower(t.ID))
	creationTime, uErr := getTokenCreationTime(b, key)
	if uErr != nil {
		return uErr
	}
	t.CreationTime = creationTime
	dataJSON, uErr := json.Marshal(t)
	if uErr != nil {
		return uErr
	}
	return b.Put(key, dataJSON)
}

func addTokenByAddress(tx *bolt.Tx, t common.Token) error {
	b, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ADDRESS))
	if uErr != nil {
		return uErr
	}
	key := []byte(strings.ToLower(t.Address))
	creationTime, uErr := getTokenCreationTime(b, key)
	if uErr != nil {
		return uErr
	}
	t.CreationTime = creationTime
	dataJson, uErr := json.Marshal(t)
	if uErr != nil {
		return uErr
	}
	return b.Put(key, dataJson)
}

func updateTokenVersion(tx *bolt.Tx, timestamp uint64) error {
	b := tx.Bucket([]byte(token_version))
	if uErr := b.Put([]byte(token_version), boltutil.Uint64ToBytes(timestamp)); uErr != nil {
		return uErr
	}
	return nil
}

func (boltSettingStorage *BoltSettingStorage) UpdateToken(t common.Token, timestamp uint64) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		if uErr := addTokenByID(tx, t); uErr != nil {
			return uErr
		}
		if uErr := addTokenByAddress(tx, t); uErr != nil {
			return uErr
		}
		if uErr := updateTokenVersion(tx, timestamp); uErr != nil {
			return uErr
		}
		return nil
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) AddTokenByID(t common.Token, timestamp uint64) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		b, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ID))
		if uErr != nil {
			return uErr
		}
		dataJSON, uErr := json.Marshal(t)
		if uErr != nil {
			return uErr
		}
		if uErr := updateTokenVersion(tx, timestamp); uErr != nil {
			return uErr
		}
		return b.Put([]byte(strings.ToLower(t.ID)), dataJSON)
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) AddTokenByAddress(t common.Token, timestamp uint64) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		b, uErr := tx.CreateBucketIfNotExists([]byte(TOKEN_BUCKET_BY_ADDRESS))
		if uErr != nil {
			return uErr
		}
		dataJson, uErr := json.Marshal(t)
		if uErr != nil {
			return uErr
		}
		if uErr := updateTokenVersion(tx, timestamp); uErr != nil {
			return uErr
		}
		return b.Put([]byte(strings.ToLower(t.Address)), dataJson)
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) getTokensWithFilter(filter FilterFunction) (result []common.Token, err error) {
	err = boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TOKEN_BUCKET_BY_ID))
		if b == nil {
			return fmt.Errorf("Bucket doesn't exist yet")
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var temp common.Token
			uErr := json.Unmarshal(v, &temp)
			if uErr != nil {
				return uErr
			}
			if filter(temp) {
				result = append(result, temp)
			}
		}
		return nil
	})
	return result, err
}

func (boltSettingStorage *BoltSettingStorage) GetAllTokens() (result []common.Token, err error) {
	return boltSettingStorage.getTokensWithFilter(isToken)
}

func (boltSettingStorage *BoltSettingStorage) GetActiveTokens() (result []common.Token, err error) {
	return boltSettingStorage.getTokensWithFilter(isActive)
}

func (boltSettingStorage *BoltSettingStorage) GetInternalTokens() (result []common.Token, err error) {
	return boltSettingStorage.getTokensWithFilter(isInternal)
}

func (boltSettingStorage *BoltSettingStorage) GetExternalTokens() (result []common.Token, err error) {
	return boltSettingStorage.getTokensWithFilter(isExternal)
}

func (boltSettingStorage *BoltSettingStorage) getTokenByIDWithFiltering(id string, filter FilterFunction) (t common.Token, err error) {
	id = strings.ToLower(id)
	err = boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TOKEN_BUCKET_BY_ID))
		if b == nil {
			return fmt.Errorf("Bucket doesn't exist yet")
		}
		v := b.Get([]byte(id))
		if v == nil {
			return settings.ErrTokenNotFound
		}
		uErr := json.Unmarshal(v, &t)
		if uErr != nil {
			return uErr
		}
		if !filter(t) {
			return settings.ErrTokenNotFound
		}
		return nil
	})
	return t, err
}

func (boltSettingStorage *BoltSettingStorage) GetTokenByID(id string) (common.Token, error) {
	return boltSettingStorage.getTokenByIDWithFiltering(id, isToken)
}

func (boltSettingStorage *BoltSettingStorage) GetActiveTokenByID(id string) (common.Token, error) {
	return boltSettingStorage.getTokenByIDWithFiltering(id, isActive)
}

func (boltSettingStorage *BoltSettingStorage) GetInternalTokenByID(id string) (common.Token, error) {
	return boltSettingStorage.getTokenByIDWithFiltering(id, isInternal)
}

func (boltSettingStorage *BoltSettingStorage) GetExternalTokenByID(id string) (common.Token, error) {
	return boltSettingStorage.getTokenByIDWithFiltering(id, isExternal)
}

func (boltSettingStorage *BoltSettingStorage) getTokenByAddressWithFiltering(addr string, filter FilterFunction) (t common.Token, err error) {
	addr = strings.ToLower(addr)
	err = boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TOKEN_BUCKET_BY_ADDRESS))
		if b == nil {
			return fmt.Errorf("Bucket doesn't exist yet")
		}
		c := b.Cursor()
		k, v := c.Seek([]byte(addr))
		if bytes.Compare(k, []byte(addr)) != 0 {
			return settings.ErrTokenNotFound
		}
		uErr := json.Unmarshal(v, &t)
		if uErr != nil {
			return uErr
		}
		if !filter(t) {
			return settings.ErrTokenNotFound
		}
		return nil
	})
	return t, err
}

func (boltSettingStorage *BoltSettingStorage) GetActiveTokenByAddress(Addr ethereum.Address) (common.Token, error) {
	return boltSettingStorage.getTokenByAddressWithFiltering(Addr.Hex(), isActive)
}

func (boltSettingStorage *BoltSettingStorage) GetTokenByAddress(Addr ethereum.Address) (common.Token, error) {
	return boltSettingStorage.getTokenByAddressWithFiltering(Addr.Hex(), isToken)
}

func (boltSettingStorage *BoltSettingStorage) GetInternalTokenByAddress(Addr ethereum.Address) (common.Token, error) {
	return boltSettingStorage.getTokenByAddressWithFiltering(Addr.Hex(), isInternal)
}

func (boltSettingStorage *BoltSettingStorage) GetExternalTokenByAddress(Addr ethereum.Address) (common.Token, error) {
	return boltSettingStorage.getTokenByAddressWithFiltering(Addr.Hex(), isExternal)
}

// UpdateTokenWithExchangeSetting will attempt to apply all the token and exchange settings
// as well as remove pending Token listing in one TX. reroll and return err if occur
func (boltSettingStorage *BoltSettingStorage) UpdateTokenWithExchangeSetting(tokens []common.Token, exSetting map[settings.ExchangeName]*common.ExchangeSetting, timestamp uint64) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		//Apply tokens setting
		for _, t := range tokens {
			if uErr := addTokenByID(tx, t); uErr != nil {
				return uErr
			}
			if uErr := addTokenByAddress(tx, t); uErr != nil {
				return uErr
			}
		}
		//Apply exchanges setting
		for exName, exSett := range exSetting {
			if uErr := putDepositAddress(tx, exName, exSett.DepositAddress); uErr != nil {
				return uErr
			}
			if uErr := putExchangeInfo(tx, exName, exSett.Info); uErr != nil {
				return uErr
			}
			if uErr := putFee(tx, exName, exSett.Fee); uErr != nil {
				return uErr
			}
			if uErr := putMinDeposit(tx, exName, exSett.MinDeposit); uErr != nil {
				return uErr
			}
		}
		if uErr := updateExchangeVersion(tx, timestamp); uErr != nil {
			return uErr
		}

		if uErr := updateTokenVersion(tx, timestamp); uErr != nil {
			return uErr
		}
		//delete pending token listings
		if uErr := deletePendingTokenUpdates(tx); uErr != nil {
			return uErr
		}
		return nil
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) StorePendingTokenUpdates(trs map[string]common.TokenUpdate) error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		b, uErr := tx.CreateBucketIfNotExists([]byte(PENDING_TOKEN_REQUEST))
		if uErr != nil {
			return uErr
		}
		for tokenID, tr := range trs {
			dataJSON, vErr := json.Marshal(tr)
			if vErr != nil {
				return vErr
			}
			if vErr = b.Put([]byte(tokenID), dataJSON); vErr != nil {
				return vErr
			}
		}
		return nil
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) GetPendingTokenUpdates() (map[string]common.TokenUpdate, error) {
	result := make(map[string]common.TokenUpdate)
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(PENDING_TOKEN_REQUEST))
		if b == nil {
			return fmt.Errorf("Bucket %s does not exist yet", PENDING_TOKEN_REQUEST)
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var tokenUpdate common.TokenUpdate
			if vErr := json.Unmarshal(v, &tokenUpdate); vErr != nil {
				return vErr
			}
			result[string(k)] = tokenUpdate
		}
		return nil
	})
	return result, err
}

func deletePendingTokenUpdates(tx *bolt.Tx) error {
	b := tx.Bucket([]byte(PENDING_TOKEN_REQUEST))
	if b == nil {
		return fmt.Errorf("Bucket %s does not exist yet", PENDING_TOKEN_REQUEST)
	}
	c := b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		if err := b.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (boltSettingStorage *BoltSettingStorage) RemovePendingTokenUpdates() error {
	err := boltSettingStorage.db.Update(func(tx *bolt.Tx) error {
		return deletePendingTokenUpdates(tx)
	})
	return err
}

func (boltSettingStorage *BoltSettingStorage) GetTokenVersion() (uint64, error) {
	var result uint64
	err := boltSettingStorage.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(token_version))
		data := b.Get([]byte(token_version))
		if data == nil {
			return errors.New("No version is currently available")
		}
		result = boltutil.BytesToUint64(data)
		return nil
	})
	return result, err
}
