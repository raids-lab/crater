package main

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/raids-lab/crater/dao/model"
	"gorm.io/gorm"
)

func kthenaTurnLeaseMigration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202610031200",
		Migrate: func(tx *gorm.DB) error {
			for _, field := range []string{"TurnLease", "TurnExpiresAt"} {
				if err := addColumnIfMissing(tx, "kthena_chat_sessions", &model.KthenaChatSession{}, field); err != nil {
					return err
				}
			}
			return addColumnIfMissing(tx, "kthena_chat_messages", &model.KthenaChatMessage{}, "RequestHash")
		},
		Rollback: func(tx *gorm.DB) error {
			if err := dropColumnIfPresent(tx, "kthena_chat_messages", &model.KthenaChatMessage{}, "RequestHash"); err != nil {
				return err
			}
			for _, field := range []string{"TurnLease", "TurnExpiresAt"} {
				if err := dropColumnIfPresent(tx, "kthena_chat_sessions", &model.KthenaChatSession{}, field); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
