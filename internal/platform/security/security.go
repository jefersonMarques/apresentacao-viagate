package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
)

func RandomToken(bytes int) (string, []byte, error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate random token: %w", err)
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plain))
	return plain, hash[:], nil
}

func HashToken(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}

func RandomOTP() (string, []byte, error) {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate OTP: %w", err)
	}
	value := int(raw[0])<<24 | int(raw[1])<<16 | int(raw[2])<<8 | int(raw[3])
	if value < 0 {
		value = -value
	}
	otp := fmt.Sprintf("%06d", value%1000000)
	return otp, HashToken(otp), nil
}

func HashPassword(password string) (string, error) {
	if len(password) < 10 {
		return "", fmt.Errorf("password must have at least 10 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var memory uint64
	var timeCost uint64
	var threads uint64
	for _, value := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(value, "=", 2)
		if len(pair) != 2 {
			return false
		}
		number, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}
		switch pair[0] {
		case "m": memory = number
		case "t": timeCost = number
		case "p": threads = number
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(timeCost), uint32(memory), uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func SHA256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
