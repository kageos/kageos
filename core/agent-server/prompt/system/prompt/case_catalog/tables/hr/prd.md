# 案例：招聘投递系统（多 Table）

## 一、项目概要

- **类型**：主从两表，两个 GET Table，无独立 POST Form（或 Form 仅辅助投递）。
- **路由**：`hr_job_list.table`（职位列表）、`hr_resume_list.table`（简历/投递列表）；路由组 `/tables/hr`。
- **关系**：职位 1:N 投递；职位列表可带「投递简历」link；投递选职位用 **OnSelectFuzzy**。
- **适合参考**：主从表、两 .go 两 GET、link、select 关联另一表、files（简历附件）。

---

## 二、结构化 PRD

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 只保留实现参考、SDK 写法和注意事项，不再承载旧 PRD 表格。

## 三、文件与路由

| 文件               | 说明           | 注册路由            |
|--------------------|----------------|---------------------|
| hr_job_list.go     | 职位管理       | GET hr_job_list.table   |
| hr_resume_list.go  | 简历/投递管理  | GET hr_resume_list.table |

---

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/tables/hr`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### hr_job_list.go

```go
package hr

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 职位信息管理 ================

// HrJob 职位信息表
type HrJob struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:职位ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time          `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Title        string `json:"title" gorm:"column:title;comment:职位名称" widget:"name:职位名称;type:input" validate:"required,min=2,max=100"`
	Department   string `json:"department" gorm:"column:department;comment:部门" widget:"name:部门;type:select;options:技术,产品,设计,运营,市场,销售,人事,财务,其他;options_colors:409EFF,67C23A,E6A23C,909399,F56C6C,9C27B0,FF9800,607D8B" validate:"required"`
	JobType      string `json:"job_type" gorm:"column:job_type;comment:工作类型;default:全职" widget:"name:工作类型;type:select;options:全职,兼职,实习,外包;options_colors:409EFF,67C23A,E6A23C,909399;render_default:全职" validate:"required,oneof=全职 兼职 实习 外包"`
	Experience   string `json:"experience" gorm:"column:experience;comment:工作经验;default:不限" widget:"name:工作经验;type:select;options:1-3年,3-5年,5-10年,10年以上,不限;options_colors:909399,409EFF,67C23A,E6A23C,9E9E9E;render_default:不限" validate:"required"`
	Education    string `json:"education" gorm:"column:education;comment:学历要求;default:不限" widget:"name:学历要求;type:select;options:本科,硕士,博士,不限;options_colors:909399,409EFF,67C23A,9E9E9E;render_default:不限" validate:"required"`
	Location     string `json:"location" gorm:"column:location;comment:工作地点" widget:"name:工作地点;type:input" validate:"required,min=2,max=100"`
	MinSalary    int    `json:"min_salary" gorm:"column:min_salary;comment:最低薪资(元)" widget:"name:最低薪资;type:integer" validate:"gte=0"`
	MaxSalary    int    `json:"max_salary" gorm:"column:max_salary;comment:最高薪资(元)" widget:"name:最高薪资;type:integer" validate:"gte=0"`
	Description  string `json:"description" gorm:"column:description;type:text;comment:职位描述" widget:"name:职位描述;type:text_area" validate:"required,min=10"`
	Requirements string `json:"requirements" gorm:"column:requirements;type:text;comment:任职要求" widget:"name:任职要求;type:text_area"`
	RecruitCount int    `json:"recruit_count" gorm:"column:recruit_count;comment:招聘人数;default:1" widget:"name:招聘人数;type:integer" validate:"required,min=1"`
	Status       string `json:"status" gorm:"column:status;comment:招聘状态;default:招聘中" widget:"name:招聘状态;type:select;options:招聘中,已暂停,已关闭,已招满;options_colors:67C23A,E6A23C,F56C6C,909399;render_default:招聘中" validate:"required,oneof=招聘中 已暂停 已关闭 已招满"`
	PublishTime  types.Time  `json:"publish_time" gorm:"column:publish_time;type:datetime;comment:发布时间" widget:"name:发布时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	Deadline     types.Time  `json:"deadline" gorm:"column:deadline;type:datetime;comment:截止时间" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedBy     string `json:"created_by" gorm:"column:created_by" widget:"name:发布人;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	ApplyLink    string `json:"apply_link" gorm:"-" widget:"name:投递简历;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (HrJob) TableName() string {
	return "hr_job"
}

// HrJobListReq 职位列表请求
type HrJobListReq struct {
	Title      string `json:"title" form:"title" widget:"name:职位名称;type:input"`
	Department string `json:"department" form:"department" widget:"name:部门;type:select;options:技术,产品,设计,运营,市场,销售,人事,财务,其他;options_colors:409EFF,67C23A,E6A23C,909399,F56C6C,9C27B0,FF9800,607D8B"`
	JobType    string `json:"job_type" form:"job_type" widget:"name:工作类型;type:select;options:全职,兼职,实习,外包;options_colors:409EFF,67C23A,E6A23C,909399"`
	Status     string `json:"status" form:"status" widget:"name:招聘状态;type:select;options:招聘中,已暂停,已关闭,已招满;options_colors:67C23A,E6A23C,F56C6C,909399"`

	query.PageSortReq `widget:"-"`
}

// HrJobList 职位管理
func HrJobList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req HrJobListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&HrJob{})
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Department != "" {
		queryDB = queryDB.Where("department = ?", req.Department)
	}
	if req.JobType != "" {
		queryDB = queryDB.Where("job_type = ?", req.JobType)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var jobs []HrJob
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&jobs).Error; err != nil {
		return err
	}

	for i := range jobs {
		if jobs[i].Status == "招聘中" {
			now := time.Now()
			if jobs[i].Deadline.IsZero() || jobs[i].Deadline.Time().After(now) {
				params := HrResume{
					JobID: jobs[i].ID,
				}
				jobs[i].ApplyLink, _ = ctx.BuildFunctionUrlWithText("hr_resume_list.table?_tab=OnTableAddRow", params, "投递简历")
			}
		}
	}

	return resp.Table(response.TableResult{
		Items:      jobs,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// HrJobListTemplate 职位管理配置
var HrJobListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "职位管理",
		Desc:         `职位信息的增删改查管理，包括职位名称、部门、工作类型、薪资范围、职位描述、任职要求、招聘状态等`,
		Tags:         []string{"招聘系统", "职位管理"},
		Request:      &HrJobListReq{},
		CreateTables: []interface{}{&HrJob{}},
	},
	AutoCrudTable: &HrJob{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row HrJob
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}

		if row.MaxSalary > 0 && row.MinSalary > 0 && row.MaxSalary < row.MinSalary {
			return nil, fmt.Errorf("最高薪资不能低于最低薪资")
		}

		if !row.Deadline.IsZero() && !row.PublishTime.IsZero() && !row.Deadline.Time().After(row.PublishTime.Time()) {
			return nil, fmt.Errorf("截止时间必须晚于发布时间")
		}

		row.CreatedBy = ctx.GetRequestUser()

		if row.PublishTime.IsZero() {
			row.PublishTime = types.Time(time.Now())
		}

		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create hr_job err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields HrJob
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()

		if req.IsFieldUpdated("min_salary") || req.IsFieldUpdated("max_salary") {
			var currentJob HrJob
			if err := db.Where("id = ?", req.GetId()).First(&currentJob).Error; err != nil {
				return nil, fmt.Errorf("职位记录不存在")
			}

			tempJob := HrJob{
				MinSalary: currentJob.MinSalary,
				MaxSalary: currentJob.MaxSalary,
			}

			if req.IsFieldUpdated("min_salary") {
				tempJob.MinSalary = updateFields.MinSalary
			}
			if req.IsFieldUpdated("max_salary") {
				tempJob.MaxSalary = updateFields.MaxSalary
			}

			if tempJob.MaxSalary > 0 && tempJob.MinSalary > 0 && tempJob.MaxSalary < tempJob.MinSalary {
				return nil, fmt.Errorf("最高薪资不能低于最低薪资")
			}
		}

		if req.IsFieldUpdated("publish_time") || req.IsFieldUpdated("deadline") {
			var currentJob HrJob
			if err := db.Where("id = ?", req.GetId()).First(&currentJob).Error; err != nil {
				return nil, fmt.Errorf("职位记录不存在")
			}

			tempJob := HrJob{
				PublishTime: currentJob.PublishTime,
				Deadline:    currentJob.Deadline,
			}

			if req.IsFieldUpdated("publish_time") {
				tempJob.PublishTime = updateFields.PublishTime
			}
			if req.IsFieldUpdated("deadline") {
				tempJob.Deadline = updateFields.Deadline
			}

			if !tempJob.Deadline.IsZero() && !tempJob.PublishTime.IsZero() && !tempJob.Deadline.Time().After(tempJob.PublishTime.Time()) {
				return nil, fmt.Errorf("截止时间必须晚于发布时间")
			}
		}

		err := db.Model(&HrJob{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update hr_job err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		var count int64
		if err := db.Model(&HrResume{}).Where("job_id in ?", req.GetIds()).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("检查简历投递记录失败: %v", err)
		}

		if count > 0 {
			return nil, fmt.Errorf("该职位下已有 %d 份简历投递，无法删除。请先处理简历投递记录", count)
		}

		err := db.Model(&HrJob{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": ctx.GetRequestUser(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "Delete hr_job err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("hr_job_list.table", HrJobList, HrJobListTemplate)
}
```

### hr_resume_list.go

```go
package hr

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 简历投递管理 ================

// HrResume 简历投递表
type HrResume struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:投递ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:投递时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time          `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	JobID         int    `json:"job_id" gorm:"column:job_id;comment:职位ID;index" widget:"name:投递职位;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Job           *HrJob `json:"-" widget:"-" gorm:"foreignKey:JobID;references:ID"`
	JobTitle      string `json:"job_title" gorm:"-" widget:"name:职位名称;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	JobDepartment string `json:"job_department" gorm:"-" widget:"name:部门;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	JobLink       string `json:"job_link" gorm:"-" widget:"name:职位详情;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	Name            string       `json:"name" gorm:"column:name;comment:姓名" widget:"name:姓名;type:input" validate:"required,min=2,max=20"`
	Phone           string       `json:"phone" gorm:"column:phone;comment:联系电话" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`
	Email           string       `json:"email" gorm:"column:email;comment:邮箱" widget:"name:邮箱;type:input" validate:"required,email"`
	Gender          string       `json:"gender" gorm:"column:gender;comment:性别" widget:"name:性别;type:select;options:男,女,其他;options_colors:409EFF,67C23A,909399"`
	Age             int          `json:"age" gorm:"column:age;comment:年龄" widget:"name:年龄;type:integer" validate:"min=18,max=100"`
	Education       string       `json:"education" gorm:"column:education;comment:学历" widget:"name:学历;type:select;options:本科,硕士,博士,其他;options_colors:909399,409EFF,67C23A,9E9E9E"`
	Experience      string       `json:"experience" gorm:"column:experience;comment:工作经验" widget:"name:工作经验;type:select;options:1-3年,3-5年,5-10年,10年以上,应届生;options_colors:909399,409EFF,67C23A,E6A23C,9E9E9E"`
	CurrentCompany  string       `json:"current_company" gorm:"column:current_company;comment:当前公司" widget:"name:当前公司;type:input"`
	CurrentPosition string       `json:"current_position" gorm:"column:current_position;comment:当前职位" widget:"name:当前职位;type:input"`
	ResumeContent   string       `json:"resume_content" gorm:"column:resume_content;type:text;comment:简历内容" widget:"name:简历内容;type:text_area" validate:"required,min=10"`
	ResumeFile      string `json:"resume_file" gorm:"column:resume_file;type:text;comment:简历附件" widget:"name:简历附件;type:files"`
	Status          string       `json:"status" gorm:"column:status;comment:投递状态;default:待筛选" widget:"name:投递状态;type:select;options:待筛选,已通过,已拒绝,待面试,已录用;options_colors:909399,67C23A,F56C6C,E6A23C,67C23A;render_default:待筛选" validate:"required,oneof=待筛选 已通过 已拒绝 待面试 已录用"`
	Remark          string       `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
	Applicant       string       `json:"applicant" gorm:"column:applicant;comment:投递人" widget:"name:投递人;type:user;render_default:Me()" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (HrResume) TableName() string {
	return "hr_resume"
}

// HrResumeListReq 简历投递列表请求
type HrResumeListReq struct {
	JobTitle      string `json:"job_title" form:"job_title" gorm:"-" widget:"name:职位名称;type:input"`
	JobDepartment string `json:"job_department" form:"job_department" widget:"name:部门;type:select;options:技术,产品,设计,运营,市场,销售,人事,财务,其他;options_colors:409EFF,67C23A,E6A23C,909399,F56C6C,9C27B0,FF9800,607D8B"`
	Name          string `json:"name" form:"name" widget:"name:姓名;type:input"`
	Phone         string `json:"phone" form:"phone" widget:"name:联系电话;type:input"`
	Status        string `json:"status" form:"status" widget:"name:投递状态;type:select;options:待筛选,已通过,已拒绝,待面试,已录用;options_colors:909399,67C23A,F56C6C,E6A23C,67C23A"`

	query.PageSortReq `widget:"-"`
}

// HrResumeList 投递管理
func HrResumeList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req HrResumeListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&HrResume{})

	if req.JobTitle != "" {
		var jobIDs []int
		if err := db.Model(&HrJob{}).
			Where("title LIKE ?", "%"+req.JobTitle+"%").
			Pluck("id", &jobIDs).Error; err == nil && len(jobIDs) > 0 {
			queryDB = queryDB.Where("job_id IN ?", jobIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}

	if req.JobDepartment != "" {
		var jobIDs []int
		if err := db.Model(&HrJob{}).
			Where("department = ?", req.JobDepartment).
			Pluck("id", &jobIDs).Error; err == nil && len(jobIDs) > 0 {
			queryDB = queryDB.Where("job_id IN ?", jobIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Phone != "" {
		queryDB = queryDB.Where("phone LIKE ?", "%"+req.Phone+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	queryDB = queryDB.Preload("Job")

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var resumes []HrResume
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&resumes).Error; err != nil {
		return err
	}

	for i := range resumes {
		if resumes[i].Job != nil {
			resumes[i].JobTitle = resumes[i].Job.Title
			resumes[i].JobDepartment = resumes[i].Job.Department
		}

		params := HrJob{
			ID: resumes[i].JobID,
		}
		resumes[i].JobLink, _ = ctx.BuildFunctionUrlWithText("hr_job_list.table", params, "查看职位详情")
	}

	return resp.Table(response.TableResult{
		Items:      resumes,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// HrResumeListTemplate 投递管理配置
var HrResumeListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投递管理",
		Desc:         `简历投递的增删改查管理，包括职位选择、投递人信息、简历内容、投递状态等`,
		Tags:         []string{"招聘系统", "投递管理"},
		Request:      &HrResumeListReq{},
		CreateTables: []interface{}{&HrResume{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"job_id": onSelectFuzzyJob,
		},
	},
	AutoCrudTable: &HrResume{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row HrResume
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}

		var job HrJob
		if err := db.Where("id = ?", row.JobID).First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("职位不存在")
			}
			return nil, fmt.Errorf("查询职位失败: %v", err)
		}

		if job.Status != "招聘中" {
			return nil, fmt.Errorf("职位 %s 当前状态为 %s，无法投递简历", job.Title, job.Status)
		}

		if !job.Deadline.IsZero() && !job.Deadline.Time().After(time.Now()) {
			return nil, fmt.Errorf("职位 %s 已过期，无法投递简历", job.Title)
		}

		var existingResume HrResume
		err := db.Where("job_id = ? AND applicant = ? AND deleted_at IS NULL", row.JobID, ctx.GetRequestUser()).
			First(&existingResume).Error
		if err == nil {
			return nil, fmt.Errorf("您已经投递过该职位，请勿重复投递")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查投递记录失败: %v", err)
		}

		row.Applicant = ctx.GetRequestUser()

		err = db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create hr_resume err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields HrResume
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()

		if req.IsFieldUpdated("job_id") {
			var job HrJob
			if err := db.Where("id = ?", updateFields.JobID).First(&job).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("职位不存在")
				}
				return nil, fmt.Errorf("查询职位失败: %v", err)
			}

			if job.Status != "招聘中" {
				return nil, fmt.Errorf("职位 %s 当前状态为 %s，无法投递简历", job.Title, job.Status)
			}
		}

		err := db.Model(&HrResume{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update hr_resume err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&HrResume{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": ctx.GetRequestUser(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "Delete hr_resume err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// onSelectFuzzyJob 职位选择的模糊搜索回调
func onSelectFuzzyJob(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var jobs []HrJob

	db = db.Model(&HrJob{}).
		Where("status = ?", "招聘中")

	now := time.Now()
	db = db.Where("(deadline IS NULL OR deadline > ?)", now)

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("title LIKE ? OR department LIKE ? OR location LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
			Limit(20)
	}

	db.Find(&jobs)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, j := range jobs {
		salaryDisplay := "面议"
		if j.MinSalary > 0 && j.MaxSalary > 0 {
			salaryDisplay = fmt.Sprintf("%d-%d元", j.MinSalary, j.MaxSalary)
		} else if j.MinSalary > 0 {
			salaryDisplay = fmt.Sprintf("%d元以上", j.MinSalary)
		}

		items = append(items, &callback.SelectFuzzyItem{
			Value: j.ID,
			Label: fmt.Sprintf("%s - %s (%s, %s)", j.Title, j.Department, j.Location, salaryDisplay),
			DisplayInfo: map[string]interface{}{
				"职位名称":   j.Title,
				"部门":     j.Department,
				"工作类型":   j.JobType,
				"工作地点":   j.Location,
				"薪资范围":   salaryDisplay,
				"工作经验要求": j.Experience,
				"学历要求":   j.Education,
				"招聘人数":   j.RecruitCount,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"选中职位":   statistics.Value("职位名称"),
			"部门":     statistics.Value("部门"),
			"工作类型":   statistics.Value("工作类型"),
			"工作地点":   statistics.Value("工作地点"),
			"薪资范围":   statistics.Value("薪资范围"),
			"工作经验要求": statistics.Value("工作经验要求"),
			"学历要求":   statistics.Value("学历要求"),
			"招聘人数":   statistics.Value("招聘人数"),
		},
	}, nil
}

func init() {
	packageContext.GET("hr_resume_list.table", HrResumeList, HrResumeListTemplate)
}
```
