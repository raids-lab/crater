package common

import (
	"errors"

	"github.com/raids-lab/crater/internal/bizerr"

	"gorm.io/gorm"
)

func RaiseGormError(
	err error,
	gormMsg string,
	wrapMsg string,
) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.NotFound.DataBaseNotFound.New(gormMsg)
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return bizerr.Conflict.ResourceAlreadyExists.New(gormMsg)
	}

	return bizerr.Internal.DatabaseError.Wrap(err, wrapMsg)
}
