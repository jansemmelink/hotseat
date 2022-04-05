package db

//all rows that exist on their own and does not belong to an account, must embed this
type ItemRow struct {
	ID string `json:"id"`
}

//all rows that exist on their own and belongs to an account, must embed this
type AccountItemRow struct {
	ItemRow
	AccountID string `json:"account_id"`
}
