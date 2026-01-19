package model

type UserModel struct {
	ID       string `gorm:"column:id;type:uuid;primaryKey"`
	Username string `gorm:"column:username;type:varchar(255);not null"`
}

func (UserModel) TableName() string {
	return "users"
}
