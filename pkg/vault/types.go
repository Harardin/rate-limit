package vault

import "time"

type tokenData struct {
	isRoot         bool
	isRenewable    bool
	expirationTime time.Time
}
