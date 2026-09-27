package user

type User struct {
	UID  uint   `json:"uid"`
	Name string `json:"name"`
}

type UserSession struct {
	UID   uint   `json:"uid"`
	Token string `json:"token"`
}
