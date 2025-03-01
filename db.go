package main

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateGameVoteMaster(gameName string, targetTime *time.Time, messageID string) (uint, error) {
	var t sql.NullTime
	if targetTime != nil {
		t.Valid = true
		t.Time = *targetTime
	}

	master := &GameVoteMaster{
		GameName: gameName, TargetTime: t, MessageID: messageID,
	}
	result := db.Create(&master)
	if result.Error != nil {
		return 0, result.Error
	}

	return master.ID, nil
}

func GetVoteMasterByMessageID(messageID string) (*GameVoteMaster, error) {
	var master GameVoteMaster
	err := db.Where("message_id = ?", messageID).First(&master).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &master, nil
}

func VoteToGame(voteMasterID uint, userID string, voteType VoteType) error {
	return db.Transaction(func(tx *gorm.DB) error {
		vote := &GameVote{
			GameVoteMasterID: voteMasterID,
			UserID:           userID,
			VoteType:         voteType,
		}
		result := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "game_vote_master_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(
				map[string]interface{}{
					"vote_type": voteType,
				},
			),
		}).Create(&vote)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
}

type VoteResult struct {
	Yes     []string
	No      []string
	Pending []string
}

func GetVoteResult(voteMasterID uint) (VoteResult, error) {
	var votes []GameVote
	db.Where("game_vote_master_id = ?", voteMasterID).Order(
		clause.OrderBy{Columns: []clause.OrderByColumn{
			{
				Column: clause.Column{Name: "updated_at"},
			},
		}}).Find(&votes)

	yes := make([]string, 0)
	no := make([]string, 0)
	pending := make([]string, 0)
	for _, vote := range votes {
		userID := vote.UserID
		switch vote.VoteType {
		case VoteTypeYes:
			yes = append(yes, userID)
		case VoteTypeNo:
			no = append(no, userID)
		case VoteTypePending:
			pending = append(pending, userID)
		}
	}

	return VoteResult{
		Yes:     yes,
		No:      no,
		Pending: pending,
	}, nil
}

func getYesOrPendingUsers(voteMasterID uint) ([]string, error) {
	voteResult, err := GetVoteResult(voteMasterID)
	if err != nil {
		return nil, err
	}

	users := make([]string, 0)
	users = append(users, voteResult.Yes...)
	users = append(users, voteResult.Pending...)
	return users, nil
}
