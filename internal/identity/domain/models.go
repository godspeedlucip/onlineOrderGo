package domain

import "time"

type AccountType string

const (
	AccountTypeEmployee AccountType = "employee"
	AccountTypeUser     AccountType = "user"
)

type AccountStatus string

const (
	AccountStatusEnabled  AccountStatus = "ENABLED"
	AccountStatusDisabled AccountStatus = "DISABLED"
)

type Account struct {
	ID           int64
	Type         AccountType
	Username     string
	DisplayName  string
	PasswordHash string
	Status       AccountStatus
}

type Claims struct {
	AccountID    int64
	AccountType  AccountType
	TokenID      string
	TokenVersion int64
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

type Principal struct {
	AccountID    int64       `json:"accountId"`
	AccountType  AccountType `json:"accountType"`
	DisplayName  string      `json:"displayName"`
	TokenID      string      `json:"tokenId,omitempty"`
	TokenVersion int64       `json:"tokenVersion,omitempty"`
}

type LoginInput struct {
	AccountType AccountType
	Identifier  string
	Password    string
}

type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Principal Principal `json:"principal"`
}