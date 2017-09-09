package contrib

type User struct {
	Id       float64
	Url      string
	UserType string
	Score    float64
}

func ParseUser(user map[string]interface{}) *User {
	id := user["id"].(float64)
	url := user["url"].(string)
	userType := user["type"].(string)
	score := user["score"].(float64)
	return &User{id, url, userType, score}
}
