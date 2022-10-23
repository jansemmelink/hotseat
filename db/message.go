package db

import (
	"time"

	"github.com/go-msvc/errors"
	"github.com/google/uuid"
)

type Message struct {
	ID           string   `json:"id"`
	FromUserID   string   `json:"from_user_id"`
	FromUsername *string  `json:"from_username,omitempty"`
	ToUserID     string   `json:"to_user_id"`
	ToUsername   *string  `json:"to_username,omitempty"`
	Message      string   `json:"message"`
	TimeSent     SqlTime  `json:"time_sent"`
	TimeRead     *SqlTime `json:"time_read"`
	//todo: add optional link to another forwarded message
}

func (fromUser User) SendMessage(toUser *User, message string) (messageID string, err error) {
	if toUser == nil {
		return "", errors.Errorf("cannot send toUser==nil")
	}
	if message == "" {
		return "", errors.Errorf("cannot send empty message")
	}
	id := uuid.New().String()
	if _, err := db.NamedExec(
		"INSERT INTO `messages` SET id=:id,from_user_id=:from_uid,to_user_id=:to_uid,message=:message,time_sent=:time_sent",
		map[string]interface{}{
			"id":        id,
			"from_uid":  fromUser.ID,
			"to_uid":    toUser.ID,
			"message":   message,
			"time_sent": SqlTime(time.Now()),
		},
	); err != nil {
		return "", errors.Wrapf(err, "failed to create message")
	}
	return id, nil
} //User.SendMessage()

type MessageRow struct {
	ID           string   `db:"id"`
	FromUserID   string   `db:"from_user_id"`
	FromUsername string   `db:"from_username"`
	ToUserID     string   `db:"to_user_id"`
	ToUsername   string   `db:"to_username"`
	MessageText  string   `db:"message"`
	TimeSent     SqlTime  `db:"time_sent"`
	TimeRead     *SqlTime `db:"time_read"`
}

func (mr MessageRow) Message() Message {
	return Message{
		ID:           mr.ID,
		FromUserID:   mr.FromUserID,
		FromUsername: &mr.FromUsername,
		ToUserID:     mr.ToUserID,
		ToUsername:   &mr.ToUsername,
		Message:      mr.MessageText,
		TimeSent:     mr.TimeSent,
		TimeRead:     mr.TimeRead,
	}
}

func (toUser User) Inbox(status string, limit int) ([]Message, error) {
	query := "SELECT m.id,m.from_user_id,u1.username as from_username,m.to_user_id,u2.username as to_username,m.message,m.time_sent" +
		" FROM `messages` as m JOIN users as u1 ON m.from_user_id=u1.id JOIN users as u2 ON m.to_user_id=u2.id" +
		" WHERE m.to_user_id=:to_user_id"
	switch status {
	case "read":
		query += " AND time_read<>null"
	case "unread":
		query += " AND time_read=null"
	default:
		//not filtering on status read|unread
	}

	var rows []MessageRow
	if err := NamedSelect(
		&rows,
		query+" ORDER BY m.time_sent LIMIT :limit",
		map[string]interface{}{
			"to_user_id": toUser.ID,
			"limit":      limit,
		},
	); err != nil {
		return nil, errors.Wrapf(err, "failed to read inbox")
	}

	inbox := make([]Message, len(rows))
	for i, mr := range rows {
		inbox[i] = mr.Message()
	}
	return inbox, nil
} //User.Inbox()

func (m Message) MarkRead(read bool) (Message, error) {
	if read {
		t := SqlTime(time.Now())
		m.TimeRead = &t
	} else {
		m.TimeRead = nil
	}
	if _, err := db.NamedExec(
		"UPDATE `messages` SET time_read=:time_read WHERE id=:id",
		map[string]interface{}{
			"id":        m.ID,
			"time_read": m.TimeRead,
		},
	); err != nil {
		return Message{}, errors.Wrapf(err, "failed to mark read")
	}
	return m, nil
} //Message.MarkRead()

func (m Message) Delete() error {
	if _, err := db.Exec(
		"DELETE FROM `messages` WHERE id=?",
		m.ID,
	); err != nil {
		return errors.Wrapf(err, "failed to delete")
	}
	return nil
}
