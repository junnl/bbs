package services

import (
	"bbs-go/model/constants"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple"

	databases "bbs-go/bases"
	"bbs-go/cache"
	"bbs-go/model"
	"bbs-go/repositories"
)

var UserTokenService = newUserTokenService()

func newUserTokenService() *userTokenService {
	return &userTokenService{}
}

type userTokenService struct {
}

func (s *userTokenService) Get(id int64) *model.UserToken {
	return repositories.UserTokenRepository.Get(databases.DB(), id)
}

func (s *userTokenService) Take(where ...interface{}) *model.UserToken {
	return repositories.UserTokenRepository.Take(databases.DB(), where...)
}

func (s *userTokenService) Find(cnd *simple.SqlCnd) []model.UserToken {
	return repositories.UserTokenRepository.Find(databases.DB(), cnd)
}

func (s *userTokenService) FindOne(cnd *simple.SqlCnd) *model.UserToken {
	return repositories.UserTokenRepository.FindOne(databases.DB(), cnd)
}

func (s *userTokenService) FindPageByParams(params *simple.QueryParams) (list []model.UserToken, paging *simple.Paging) {
	return repositories.UserTokenRepository.FindPageByParams(databases.DB(), params)
}

func (s *userTokenService) FindPageByCnd(cnd *simple.SqlCnd) (list []model.UserToken, paging *simple.Paging) {
	return repositories.UserTokenRepository.FindPageByCnd(databases.DB(), cnd)
}

// 获取当前登录用户的id
func (s *userTokenService) GetCurrentUserId(ctx iris.Context) int64 {
	user := s.GetCurrent(ctx)
	if user != nil {
		return user.Id
	}
	return 0
}

// 后台创建用户 (无验证)
func (s *userTokenService) SimpleSignUp(userID int64, username, email, nickname, password, avatarUrl string) (*model.User, error) {

	user := &model.User{
		Id:         userID,
		Username:   simple.SqlNullString(username),
		Email:      simple.SqlNullString(email),
		Avatar:     avatarUrl,
		Nickname:   nickname,
		Password:   password,
		Status:     constants.StatusOk,
		CreateTime: simple.NowTimestamp(),
		UpdateTime: simple.NowTimestamp(),
	}

	err := simple.Tx(simple.DB(), func(tx *gorm.DB) error {
		if err := repositories.UserRepository.Create(tx, user); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return user, nil
}

// 获取当前登录用户
func (s *userTokenService) GetCurrent(ctx iris.Context) *model.User {
	token := s.GetUserToken(ctx)
	userToken := repositories.UserTokenRepository.GetByToken(databases.DB(), token)
	// 没找到授权
	if userToken == nil {
		return nil
	}
	// 授权过期
	if userToken.ExpiresAt.Unix() <= time.Now().Unix() {
		return nil
	}
	user := repositories.UserRepository.Get(simple.DB(), userToken.UserId)
	if user == nil {
		roboUser := repositories.UserROBORepository.Get(databases.DB(), userToken.UserId)
		userName := strconv.FormatInt(roboUser.Id, 10)
		newUser, err := s.SimpleSignUp(roboUser.Id, userName, roboUser.Email, roboUser.NickName, roboUser.PasswrodHash, roboUser.HeadImg)
		if err != nil {
			return nil
		}
		return newUser
	}
	if user == nil || user.Status != constants.StatusOk {
		return nil
	}
	return user
}

// CheckLogin 检查登录状态
func (s *userTokenService) CheckLogin(ctx iris.Context) (*model.User, *simple.CodeError) {
	user := s.GetCurrent(ctx)
	if user == nil {
		return nil, simple.ErrorNotLogin
	}
	return user, nil
}

// 退出登录
func (s *userTokenService) Signout(ctx iris.Context) error {
	token := s.GetUserToken(ctx)
	userToken := repositories.UserTokenRepository.GetByToken(databases.DB(), token)
	if userToken == nil {
		return nil
	}
	return repositories.UserTokenRepository.UpdateColumn(databases.DB(), userToken.Id, "status", constants.StatusDeleted)
}

// 从请求体中获取UserToken
func (s *userTokenService) GetUserToken(ctx iris.Context) string {
	userToken := ctx.FormValue("userToken")
	if len(userToken) > 0 {
		return userToken
	}
	return ctx.GetHeader("X-User-Token")
}

// 生成
func (s *userTokenService) Generate(userId int64) (string, error) {
	token := simple.UUID()
	expiredAt := time.Now().Add(time.Hour * 24 * 365) // 365天后过期
	userToken := &model.UserToken{
		Key:        token,
		UserId:     userId,
		ExpiresAt:  expiredAt,
		CreateTime: time.Now(),
	}
	err := repositories.UserTokenRepository.Create(databases.DB(), userToken)
	if err != nil {
		return "", err
	}
	return token, nil
}

// 禁用
func (s *userTokenService) Disable(token string) error {
	t := repositories.UserTokenRepository.GetByToken(databases.DB(), token)
	if t == nil {
		return nil
	}
	err := repositories.UserTokenRepository.UpdateColumn(databases.DB(), t.Id, "status", constants.StatusDeleted)
	if err != nil {
		cache.UserTokenCache.Invalidate(token)
	}
	return err
}
