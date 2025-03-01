package main

import (
	"database/sql"

	"gorm.io/gorm"
)

type VoteType string

const (
	VoteTypeYes     VoteType = "yes"
	VoteTypeNo      VoteType = "no"
	VoteTypePending VoteType = "pending"
)

type GameVoteMaster struct {
	gorm.Model
	GameName   string
	TargetTime sql.NullTime
	MessageID  string `gorm:"uniqueIndex"`
}

type GameVote struct {
	gorm.Model
	UserID           string `gorm:"uniqueIndex:idx_game_vote_unique"`
	VoteType         VoteType
	GameVoteMasterID uint `gorm:"uniqueIndex:idx_game_vote_unique"`
	GameVoteMaster   GameVoteMaster
}
