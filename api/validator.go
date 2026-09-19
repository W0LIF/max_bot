package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func ValidateInitData(initData, botToken string) (bool, error) {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return false, err
	}

	originalHash := params.Get("hash")
	if originalHash == "" {
		return false, fmt.Errorf("hash не найден")
	}
	params.Del("hash")

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var launchParams strings.Builder
	for i, k := range keys {
		if i > 0 {
			launchParams.WriteString("\n")
		}
		launchParams.WriteString(k + "=" + params.Get(k))
	}

	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	h2 := hmac.New(sha256.New, secretKey)
	h2.Write([]byte(launchParams.String()))
	signature := hex.EncodeToString(h2.Sum(nil))

	return signature == originalHash, nil
}