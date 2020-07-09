package repositories

import (
	"github.com/jinzhu/gorm"

	"bbs-go/model"
)

var UserROBORepository = newUserROBORepository()

func newUserROBORepository() *userROBORepository {
	return &userROBORepository{}
}

type userROBORepository struct {
}

func (r *userROBORepository) Get(db *gorm.DB, id int64) *model.ROBOUser {
	ret := &model.ROBOUser{}
	if err := db.First(ret, "id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}
