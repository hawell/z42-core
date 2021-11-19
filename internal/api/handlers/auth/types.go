package auth

type NewUser struct {
	Email    string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type loginCredentials struct {
	Email    string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type verification struct {
	Code string `form:"code" json:"code" binding:"required"`
}

type authenticationToken struct {
	Code   int    `form:"code" json:"code" binding:"required"`
	Token  string `form:"token" json:"token" binding:"required"`
	Expire string `form:"expire" json:"expire" binding:"required"`
}

type recovery struct {
	Email string `form:"email" json:"email" binding:"required"`
}

type passwordReset struct {
	Password string `form:"password" json:"password" binding:"required"`
	Code     string `form:"code" json:"code" binding:"required"`
}

type NewAPIKeyRequest struct {
	Name     string `json:"name" binding:"required"`
	ZoneName string `json:"zone_name" binding:"required"`
	Scope    string `json:"scope" binding:"required"`
	Enabled  bool   `json:"enabled"`
}

type NewAPIKeyResponse struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	ZoneName string `json:"zone_name"`
	Scope    string `json:"scope"`
	Enabled  bool   `json:"enabled"`
}

type UpdateAPIKeyRequest struct {
	Scope   string `json:"scope" binding:"required"`
	Enabled bool   `json:"enabled"`
}
