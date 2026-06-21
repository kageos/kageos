// start_game.go
// 开始游戏表单

package werewolf

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// StartGameReq 开始游戏请求
type StartGameReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
}

// StartGameResp 开始游戏响应
type StartGameResp struct {
	Result           string `json:"result" widget:"name:结果;type:input"`
	RoleDistribution string `json:"role_distribution" widget:"name:角色分配;type:text_area"`
	Message          string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 角色配置 ================

// roleConfig 角色配置
type roleConfig struct {
	Role  string
	Count int
}

// getRoleConfigs 根据人数获取角色配置
func getRoleConfigs(playerCount int) []roleConfig {
	switch playerCount {
	case 4:
		return []roleConfig{
			{"狼人", 1},
			{"村民", 3},
		}
	case 5:
		return []roleConfig{
			{"狼人", 1},
			{"预言家", 1},
			{"村民", 3},
		}
	case 6:
		return []roleConfig{
			{"狼人", 2},
			{"预言家", 1},
			{"女巫", 1},
			{"村民", 2},
		}
	case 7:
		return []roleConfig{
			{"狼人", 2},
			{"预言家", 1},
			{"女巫", 1},
			{"村民", 3},
		}
	case 8:
		return []roleConfig{
			{"狼人", 2},
			{"预言家", 1},
			{"女巫", 1},
			{"猎人", 1},
			{"村民", 3},
		}
	case 9:
		return []roleConfig{
			{"狼人", 3},
			{"预言家", 1},
			{"女巫", 1},
			{"猎人", 1},
			{"村民", 3},
		}
	case 10:
		return []roleConfig{
			{"狼人", 3},
			{"预言家", 1},
			{"女巫", 1},
			{"猎人", 1},
			{"村民", 4},
		}
	case 11:
		return []roleConfig{
			{"狼人", 3},
			{"预言家", 1},
			{"女巫", 1},
			{"猎人", 1},
			{"村民", 5},
		}
	case 12:
		return []roleConfig{
			{"狼人", 4},
			{"预言家", 1},
			{"女巫", 1},
			{"猎人", 1},
			{"村民", 5},
		}
	default:
		return []roleConfig{
			{"狼人", 2},
			{"预言家", 1},
			{"女巫", 1},
			{"村民", 2},
		}
	}
}

// ================ 开始游戏 ================

// StartGame 开始游戏入口
func StartGame(ctx *app.Context, resp response.Response) error {
	var req StartGameReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoStartGame(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoStartGame 开始游戏业务逻辑
func DoStartGame(ctx *app.Context, req *StartGameReq) (*StartGameResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoStartGame] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoStartGame]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
		}
		logger.Errorf(ctx, "[系统错误]-[DoStartGame] 查询房间失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoStartGame]： 查询房间失败, req: %+v, err: %w", req, err)
	}

	if room.Status != "等待开始" {
		return nil, fmt.Errorf("游戏已经开始，无法重新开始")
	}

	if room.PlayerCount < 4 {
		return nil, fmt.Errorf("房间人数不足，至少需要4人才能开始游戏")
	}

	var players []Player
	if err := db.Where("room_no = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, "存活").Find(&players).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoStartGame] 查询玩家失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoStartGame]： 查询玩家失败, req: %+v, err: %w", req, err)
	}

	if len(players) < 4 {
		return nil, fmt.Errorf("存活玩家不足，至少需要4人才能开始游戏")
	}

	roleConfigs := getRoleConfigs(len(players))
	var allRoles []string
	for _, config := range roleConfigs {
		for i := 0; i < config.Count; i++ {
			allRoles = append(allRoles, config.Role)
		}
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(allRoles), func(i, j int) {
		allRoles[i], allRoles[j] = allRoles[j], allRoles[i]
	})

	var distribution []string
	err := db.Transaction(func(tx *gorm.DB) error {
		for i, player := range players {
			role := allRoles[i]
			if err := tx.Model(&Player{}).Where("id = ?", player.ID).Update("role", role).Error; err != nil {
				return err
			}
			distribution = append(distribution, fmt.Sprintf("%s-%s", player.PlayerName, role))
		}

		if err := tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).Updates(map[string]interface{}{
			"status":        "夜晚",
			"current_round": 1,
		}).Error; err != nil {
			return err
		}

		record := &GameRecord{
			RecordID:   fmt.Sprintf("REC%s%d", req.RoomNo, time.Now().UnixNano()/1e6),
			RoomNo:     req.RoomNo,
			Round:      1,
			Phase:      "夜晚",
			PlayerName: "系统",
			Content:    fmt.Sprintf("游戏开始！第1夜，请狼人选择击杀目标。"),
		}
		return tx.Create(record).Error
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoStartGame] 分配角色失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoStartGame]： 分配角色失败, req: %+v, err: %w", req, err)
	}

	return &StartGameResp{
		Result:           "游戏开始",
		RoleDistribution: strings.Join(distribution, ", "),
		Message:          fmt.Sprintf("游戏开始！共 %d 名玩家，第1夜，请狼人选择击杀目标。", len(players)),
	}, nil
}

// StartGameTemplate 开始游戏配置
var StartGameTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "开始游戏",
		Desc:     `开始游戏，由房主触发，随机分配角色并进入第一夜`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &StartGameReq{},
		Response: &StartGameResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("start_game.form", StartGame, StartGameTemplate)
}
