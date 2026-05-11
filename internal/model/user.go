package model

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type InviteCode struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	MaxUses   int    `json:"maxUses"`
	UsedCount int    `json:"usedCount"`
	CreatedBy int    `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	IsActive  bool   `json:"isActive"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	InviteCode string `json:"inviteCode" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateInviteCodeRequest struct {
	MaxUses int `json:"maxUses"`
	Count   int `json:"count"`
}
