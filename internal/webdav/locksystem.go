package webdav

import (
	"strconv"
	"time"

	xwebdav "golang.org/x/net/webdav"
)

const lockTokenPrefix = "opaquelocktoken:noop-"

type noLockSystem struct{}

func NewNoLockSystem() xwebdav.LockSystem {
	return noLockSystem{}
}

func (noLockSystem) Confirm(now time.Time, name0, name1 string, conditions ...xwebdav.Condition) (func(), error) {
	return func() {}, nil
}

func (noLockSystem) Create(now time.Time, details xwebdav.LockDetails) (string, error) {
	token := lockTokenPrefix + strconv.FormatInt(time.Now().UnixNano(), 10)
	return token, nil
}

func (noLockSystem) Refresh(now time.Time, token string, duration time.Duration) (xwebdav.LockDetails, error) {
	return xwebdav.LockDetails{Duration: duration}, nil
}

func (noLockSystem) Unlock(now time.Time, token string) error {
	return nil
}
