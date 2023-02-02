package lisk

type GetAccountsResponse struct {
	Data []GetAccountsData `json:"data,omitempty"`
}

type GetAccountsData struct {
	Summary  GetAccountsSummary  `json:"summary,omitempty"`
	Sequence GetAccountsSequence `json:"sequence,omitempty"`
}

type GetAccountsSummary struct {
	Address string `json:"address,omitempty"`
	Balance string `json:"balance,omitempty"`
}

type GetAccountsSequence struct {
	Nonce string `json:"nonce,omitempty"`
}
