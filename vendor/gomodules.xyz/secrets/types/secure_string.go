package types

import (
	"context"
	"encoding/json"
	"fmt"

	"gocloud.dev/secrets"
)

type SecureString struct {
	URL    string `json:"u"`
	Data   string `json:"-"` // Value
	Cipher []byte `json:"c"`
}

func (s *SecureString) FromDB(data []byte) error {
	if len(data) <= 2 || (data[0] != '{' || data[len(data)-1] != '}') {
		s.Data = string(data)
		return nil
	}

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	ctx := context.Background()
	k, err := secrets.OpenKeeper(ctx, s.URL)
	if err != nil {
		return err
	}

	val, err := k.Decrypt(ctx, s.Cipher)
	if err != nil {
		return err
	}
	s.Data = string(val)

	return nil
}

func (s *SecureString) ToDB() ([]byte, error) {
	m.RLock()
	fn := urlFn
	m.RUnlock()

	if s.URL == "" {
		if fn == nil {
			return []byte(s.Data), nil
		} else {
			s.URL = fn()
		}
	}

	ctx := context.Background()
	k, err := secrets.OpenKeeper(ctx, s.URL)
	if err != nil {
		return nil, err
	}
	if s.Cipher, err = k.Encrypt(ctx, []byte(s.Data)); err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

func (s *SecureString) String() string {
	return fmt.Sprintf("%v:%v", s.URL, s.Data)
}
