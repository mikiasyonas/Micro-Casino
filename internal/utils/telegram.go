package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func ValidateTelegramInitData(botToken string, initData string) (bool, error) {
	parsed, err := url.ParseQuery(initData)
	if err != nil {
		return false, err
	}

	hash := parsed.Get("hash")
	if hash == "" {
		return false, fmt.Errorf("hash not found in initData")
	}

	parsed.Del("hash")

	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckArr []string
	for _, k := range keys {
		v := parsed.Get(k)
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, v))
	}
	dataCheckString := strings.Join(dataCheckArr, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	return calculatedHash == hash, nil
}

func CheckInitDataAge(initData string) (bool, error) {
	parsed, err := url.ParseQuery(initData)
	if err != nil {
		return false, err
	}

	authDateStr := parsed.Get("auth_date")
	if authDateStr == "" {
		return false, fmt.Errorf("auth_date not found")
	}

	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return false, err
	}

	now := time.Now().Unix()
	maxAge := int64(24 * 60 * 60)

	return (now - authDate) < maxAge, nil
}
