package sales_leads

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ==================== Model ====================

type SalesLead struct {
	ID           int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	LeadNo       string         `json:"lead_no" gorm:"column:lead_no" widget:"name:线索ID;type:input" hide:"create,update"`
	CustomerName string         `json:"customer_name" gorm:"column:customer_name" widget:"name:客户名称;type:input" validate:"required"`
	Contact      string         `json:"contact" gorm:"column:contact" widget:"name:联系人;type:input"`
	Phone        string         `json:"phone" gorm:"column:phone" widget:"name:电话;type:input"`
	Email        string         `json:"email" gorm:"column:email" widget:"name:邮箱;type:input"`
	Source       string         `json:"source" gorm:"column:source" widget:"name:线索来源;type:select;options:线上推广,展会获取,电话营销,老客户推荐,合作伙伴;options_colors:409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	CompanyScale string         `json:"company_scale" gorm:"column:company_scale" widget:"name:公司规模;type:select;options:大型企业,中型企业,小型企业;options_colors:F56C6C,E6A23C,67C23A" validate:"required"`
	Industry     string         `json:"industry" gorm:"column:industry" widget:"name:所在行业;type:select;options:互联网,通信设备,金融,零售,制造;options_colors:409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	City         string         `json:"city" gorm:"column:city" widget:"name:城市;type:input"`
	Status       string         `json:"status" gorm:"column:status" widget:"name:线索状态;type:select;options:初步接触,需求确认,方案报价,商务谈判,已成交,已流失;options_colors:909399,409EFF,67C23A,E6A23C,52C41A,F56C6C" validate:"required"`
	EstAmount    float64        `json:"est_amount" gorm:"column:est_amount" widget:"name:预计成交金额;type:float;min:0;precision:2;step:0.01;unit:万元"`
	EstCloseDate types.Time     `json:"est_close_date" gorm:"column:est_close_date;type:datetime" widget:"name:预计成交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	LeadScore    float64        `json:"lead_score" gorm:"column:lead_score" widget:"name:线索评分;type:float;min:1;max:10;step:0.1"`
	AssignedTo   string         `json:"assigned_to" gorm:"column:assigned_to" widget:"name:负责销售;type:user"`
	Remark       string         `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`
	CreatedAt    types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CreatedBy    string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	UpdatedAt    types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"column:deleted_at" widget:"-"`
}

// TableName 设置表名
func (SalesLead) TableName() string {
	return "sales_lead"
}

// ==================== Request Structs ====================

type SalesLeadListReq struct {
	CustomerName      string `json:"customer_name" form:"customer_name" widget:"name:客户名称;type:input"`
	Status            string `json:"status" form:"status" widget:"name:线索状态;type:select;options:初步接触,需求确认,方案报价,商务谈判,已成交,已流失"`
	Source            string `json:"source" form:"source" widget:"name:线索来源;type:select;options:线上推广,展会获取,电话营销,老客户推荐,合作伙伴"`
	AssignedTo        string `json:"assigned_to" form:"assigned_to" widget:"name:负责销售;type:user"`
	StartTime         string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime           string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

type AddLeadReq struct {
	CustomerName string     `json:"customer_name" widget:"name:客户名称;type:input" validate:"required"`
	Contact      string     `json:"contact" widget:"name:联系人;type:input" validate:"required"`
	Phone        string     `json:"phone" widget:"name:电话;type:input" validate:"required"`
	Email        string     `json:"email" widget:"name:邮箱;type:input"`
	Source       string     `json:"source" widget:"name:线索来源;type:select;options:线上推广,展会获取,电话营销,老客户推荐,合作伙伴;options_colors:409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	CompanyScale string     `json:"company_scale" widget:"name:公司规模;type:select;options:大型企业,中型企业,小型企业;options_colors:F56C6C,E6A23C,67C23A" validate:"required"`
	Industry     string     `json:"industry" widget:"name:所在行业;type:select;options:互联网,通信设备,金融,零售,制造;options_colors:409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	City         string     `json:"city" widget:"name:城市;type:input"`
	Status       string     `json:"status" widget:"name:线索状态;type:select;options:初步接触,需求确认,方案报价,商务谈判,已成交,已流失;options_colors:909399,409EFF,67C23A,E6A23C,52C41A,F56C6C" validate:"required"`
	EstAmount    float64    `json:"est_amount" widget:"name:预计成交金额;type:float;min:0;precision:2;step:0.01;unit:万元"`
	EstCloseDate types.Time `json:"est_close_date" widget:"name:预计成交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	LeadScore    float64    `json:"lead_score" widget:"name:线索评分;type:float;min:1;max:10;step:0.1"`
	AssignedTo   string     `json:"assigned_to" widget:"name:负责销售;type:user"`
	Remark       string     `json:"remark" widget:"name:备注;type:text_area"`
}

type AddLeadResp struct {
	Result string `json:"result" widget:"name:操作结果;type:input"`
}

type BatchImportReq struct {
	File string `json:"file" widget:"name:上传文件;type:files;accept:.xlsx,.csv" validate:"required"`
}

type BatchImportResp struct {
	TotalRows   int    `json:"total_rows" widget:"name:总行数;type:integer"`
	SuccessRows int    `json:"success_rows" widget:"name:成功行数;type:integer"`
	FailedRows  int    `json:"failed_rows" widget:"name:失败行数;type:integer"`
	Result      string `json:"result" widget:"name:导入结果;type:text_area"`
}

// ==================== Table Template & Handlers ====================

var SalesLeadTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "销售线索列表",
		Request:      &SalesLeadListReq{},
		CreateTables: []interface{}{&SalesLead{}},
	},
	AutoCrudTable:     &SalesLead{},
	OnTableAddRow:     onTableAddRow,
	OnTableUpdateRow:  onTableUpdateRow,
	OnTableDeleteRows: onTableDeleteRows,
}

func onTableAddRow(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
	var row SalesLead
	if err := ctx.ShouldBindValidate(&row); err != nil {
		return nil, err
	}
	row.CreatedBy = ctx.GetRequestUser()
	row.LeadNo = generateLeadNo()
	db := ctx.GetGormDB()
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &callback.OnTableAddRowResp{Data: &row}, nil
}

func onTableUpdateRow(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
	var updateFields SalesLead
	if err := req.BindChangedFields(&updateFields); err != nil {
		return nil, err
	}
	updates := req.ChangedFields()
	db := ctx.GetGormDB()
	err := db.Model(&SalesLead{}).Where("id = ?", req.GetId()).Updates(updates).Error
	if err != nil {
		return nil, err
	}
	return &callback.OnTableUpdateRowResp{}, nil
}

func onTableDeleteRows(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
	db := ctx.GetGormDB()
	err := db.Model(&SalesLead{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
		"deleted_at": time.Now(),
		"deleted_by": ctx.GetRequestUser(),
	}).Error
	if err != nil {
		return nil, err
	}
	return &callback.OnTableDeleteRowsResp{}, nil
}

// ==================== List Handler ====================

func SalesLeadList(ctx *app.Context, resp response.Response) error {
	var req SalesLeadListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesLead{})

	if req.CustomerName != "" {
		queryDB = queryDB.Where("customer_name LIKE ?", "%"+req.CustomerName+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Source != "" {
		queryDB = queryDB.Where("source = ?", req.Source)
	}
	if req.AssignedTo != "" {
		queryDB = queryDB.Where("assigned_to = ?", req.AssignedTo)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("created_at DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var lists []*SalesLead
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// ==================== Add Lead Form ====================

var AddLeadTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "新增线索",
		Request:  &AddLeadReq{},
		Response: &AddLeadResp{},
	},
}

func AddLead(ctx *app.Context, resp response.Response) error {
	var req AddLeadReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	row := SalesLead{
		CustomerName: req.CustomerName,
		Contact:      req.Contact,
		Phone:        req.Phone,
		Email:        req.Email,
		Source:       req.Source,
		CompanyScale: req.CompanyScale,
		Industry:     req.Industry,
		City:         req.City,
		Status:       req.Status,
		EstAmount:    req.EstAmount,
		EstCloseDate: req.EstCloseDate,
		LeadScore:    req.LeadScore,
		AssignedTo:   req.AssignedTo,
		Remark:       req.Remark,
		CreatedBy:    ctx.GetRequestUser(),
		LeadNo:       generateLeadNo(),
	}
	if err := db.Create(&row).Error; err != nil {
		return fmt.Errorf("[系统错误]-[AddLead] 创建线索失败, req: %+v, err: %w", req, err)
	}
	return resp.Form(&AddLeadResp{Result: "新增成功"}).Build()
}

// ==================== Batch Import Form ====================

var BatchImportTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "批量导入线索",
		Request:  &BatchImportReq{},
		Response: &BatchImportResp{},
	},
}

func BatchImport(ctx *app.Context, resp response.Response) error {
	var req BatchImportReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.File)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 || inputFiles[0] == "" {
		return fmt.Errorf("[系统错误]-[BatchImport] 未找到上传文件")
	}

	filePath := inputFiles[0]
	db := ctx.GetGormDB()
	createdBy := ctx.GetRequestUser()

	var totalRows, successRows, failedRows int
	var failedDetails []string

	// 根据文件扩展名选择解析方式
	if strings.HasSuffix(strings.ToLower(filePath), ".csv") {
		totalRows, successRows, failedRows, failedDetails = parseCSVAndImport(ctx, filePath, db, createdBy)
	} else {
		totalRows, successRows, failedRows, failedDetails = parseExcelAndImport(ctx, filePath, db, createdBy)
	}

	// 构建结果文本
	resultText := fmt.Sprintf("导入完成！\n总行数: %d\n成功: %d\n失败: %d", totalRows, successRows, failedRows)
	if len(failedDetails) > 0 {
		resultText += "\n\n失败详情:\n" + strings.Join(failedDetails, "\n")
	}

	return resp.Form(&BatchImportResp{
		TotalRows:   totalRows,
		SuccessRows: successRows,
		FailedRows:  failedRows,
		Result:      resultText,
	}).Build()
}

// parseCSVAndImport 解析CSV文件并导入
func parseCSVAndImport(ctx *app.Context, filePath string, db *gorm.DB, createdBy string) (total, success, failed int, failedDetails []string) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, 0, []string{"打开文件失败: " + err.Error()}
	}
	defer file.Close()

	// 检测并处理UTF-8 BOM
	bom := make([]byte, 3)
	n, err := file.Read(bom)
	if err != nil && err != io.EOF {
		return 0, 0, 0, []string{"读取文件失败: " + err.Error()}
	}
	reader := csv.NewReader(file)
	if n == 3 && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		// UTF-8 BOM，reader已正确处理
	} else {
		// 非BOM，重置文件指针
		file.Seek(0, 0)
	}

	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return 0, 0, 0, []string{"读取CSV失败: " + err.Error()}
	}

	if len(records) < 2 {
		return 0, 0, 0, []string{"CSV文件数据少于2行（需要表头和数据）"}
	}

	// 第一行是表头
	header := records[0]
	fieldMap := buildFieldMapping(header)

	// 逐行导入（跳过表头）
	for i := 1; i < len(records); i++ {
		row := records[i]
		if isEmptyRow(row) {
			continue
		}

		lead, err := mapRowToLead(row, header, fieldMap)
		if err != nil {
			failed++
			failedDetails = append(failedDetails, fmt.Sprintf("第%d行: %v", i+1, err))
			continue
		}

		lead.CreatedBy = createdBy
		lead.LeadNo = generateLeadNo()

		if err := db.Create(&lead).Error; err != nil {
			failed++
			failedDetails = append(failedDetails, fmt.Sprintf("第%d行[%s]: %v", i+1, lead.CustomerName, err))
			continue
		}
		success++
	}

	return len(records) - 1, success, failed, failedDetails
}

// parseExcelAndImport 解析Excel文件并导入
func parseExcelAndImport(ctx *app.Context, filePath string, db *gorm.DB, createdBy string) (total, success, failed int, failedDetails []string) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return 0, 0, 0, []string{"打开Excel文件失败: " + err.Error()}
	}
	defer f.Close()

	sheetName := f.GetSheetList()[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, 0, 0, []string{"读取工作表失败: " + err.Error()}
	}

	if len(rows) < 2 {
		return 0, 0, 0, []string{"Excel文件数据少于2行（需要表头和数据）"}
	}

	header := rows[0]
	fieldMap := buildFieldMapping(header)

	// 逐行导入（跳过表头）
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if isEmptyRow(row) {
			continue
		}

		lead, err := mapRowToLead(row, header, fieldMap)
		if err != nil {
			failed++
			failedDetails = append(failedDetails, fmt.Sprintf("第%d行: %v", i+1, err))
			continue
		}

		lead.CreatedBy = createdBy
		lead.LeadNo = generateLeadNo()

		if err := db.Create(&lead).Error; err != nil {
			failed++
			failedDetails = append(failedDetails, fmt.Sprintf("第%d行[%s]: %v", i+1, lead.CustomerName, err))
			continue
		}
		success++
	}

	return len(rows) - 1, success, failed, failedDetails
}

// buildFieldMapping 根据表头构建字段映射
func buildFieldMapping(header []string) map[string]int {
	fieldMap := make(map[string]int)
	// 字段名到列索引的映射（支持多种表头名称）
	fieldNames := map[string]string{
		"客户名称":   "customer_name",
		"客户":     "customer_name",
		"公司名称":   "customer_name",
		"联系人":    "contact",
		"电话":     "phone",
		"手机":     "phone",
		"联系方式":   "phone",
		"邮箱":     "email",
		"邮件":     "email",
		"来源":     "source",
		"线索来源":   "source",
		"公司规模":   "company_scale",
		"规模":     "company_scale",
		"行业":     "industry",
		"所在行业":   "industry",
		"城市":     "city",
		"状态":     "status",
		"线索状态":   "status",
		"预计金额":   "est_amount",
		"预计成交金额": "est_amount",
		"成交金额":   "est_amount",
		"预计时间":   "est_close_date",
		"预计成交时间": "est_close_date",
		"评分":     "lead_score",
		"线索评分":   "lead_score",
		"负责":     "assigned_to",
		"负责销售":   "assigned_to",
		"销售人员":   "assigned_to",
		"备注":     "remark",
	}

	for i, col := range header {
		col = strings.TrimSpace(col)
		if fieldName, ok := fieldNames[col]; ok {
			fieldMap[fieldName] = i
		}
	}
	return fieldMap
}

// mapRowToLead 将行数据映射为SalesLead
func mapRowToLead(row []string, header []string, fieldMap map[string]int) (SalesLead, error) {
	lead := SalesLead{}

	// 获取值（带索引边界检查）
	getValue := func(fieldName string) string {
		if idx, ok := fieldMap[fieldName]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// 客户名称（必填）
	customerName := getValue("customer_name")
	if customerName == "" {
		return lead, fmt.Errorf("客户名称不能为空")
	}
	lead.CustomerName = customerName

	// 联系人
	lead.Contact = getValue("contact")

	// 电话
	lead.Phone = getValue("phone")

	// 邮箱
	lead.Email = getValue("email")

	// 线索来源（必填，校验枚举值）
	source := getValue("source")
	if source != "" {
		if !contains([]string{"线上推广", "展会获取", "电话营销", "老客户推荐", "合作伙伴"}, source) {
			return lead, fmt.Errorf("无效的线索来源: %s（可选: 线上推广,展会获取,电话营销,老客户推荐,合作伙伴）", source)
		}
		lead.Source = source
	} else {
		lead.Source = "线上推广" // 默认值
	}

	// 公司规模（必填）
	companyScale := getValue("company_scale")
	if companyScale != "" {
		if !contains([]string{"大型企业", "中型企业", "小型企业"}, companyScale) {
			return lead, fmt.Errorf("无效的公司规模: %s（可选: 大型企业,中型企业,小型企业）", companyScale)
		}
		lead.CompanyScale = companyScale
	} else {
		lead.CompanyScale = "中型企业" // 默认值
	}

	// 所在行业（必填）
	industry := getValue("industry")
	if industry != "" {
		if !contains([]string{"互联网", "通信设备", "金融", "零售", "制造"}, industry) {
			return lead, fmt.Errorf("无效的所在行业: %s（可选: 互联网,通信设备,金融,零售,制造）", industry)
		}
		lead.Industry = industry
	} else {
		lead.Industry = "互联网" // 默认值
	}

	// 城市
	lead.City = getValue("city")

	// 线索状态（必填）
	status := getValue("status")
	if status != "" {
		if !contains([]string{"初步接触", "需求确认", "方案报价", "商务谈判", "已成交", "已流失"}, status) {
			return lead, fmt.Errorf("无效的线索状态: %s（可选: 初步接触,需求确认,方案报价,商务谈判,已成交,已流失）", status)
		}
		lead.Status = status
	} else {
		lead.Status = "初步接触" // 默认值
	}

	// 预计成交金额
	if estAmountStr := getValue("est_amount"); estAmountStr != "" {
		if estAmount, err := strconv.ParseFloat(estAmountStr, 64); err == nil {
			lead.EstAmount = estAmount
		}
	}

	// 预计成交时间
	if estDateStr := getValue("est_close_date"); estDateStr != "" {
		if parsedTime, err := types.ParseTime(estDateStr); err == nil {
			lead.EstCloseDate = parsedTime
		}
	}

	// 线索评分
	if scoreStr := getValue("lead_score"); scoreStr != "" {
		if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
			if score < 1 {
				score = 1
			} else if score > 10 {
				score = 10
			}
			lead.LeadScore = score
		}
	}

	// 负责销售
	lead.AssignedTo = getValue("assigned_to")

	// 备注
	lead.Remark = getValue("remark")

	return lead, nil
}

// isEmptyRow 检查是否为空行
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ==================== Charts ====================

type LeadStatusChartReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func LeadStatusDistributionChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesLead{})

	type StatusStat struct {
		Status string  `gorm:"column:status"`
		Count  int     `gorm:"column:count"`
		Amount float64 `gorm:"column:amount"`
	}
	var stats []StatusStat
	err := queryDB.Select("status, COUNT(*) as count, COALESCE(SUM(est_amount), 0) as amount").
		Group("status").Find(&stats).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[LeadStatusDistributionChart] 查询失败, err: %w", err)
	}

	statuses := []string{"初步接触", "需求确认", "方案报价", "商务谈判", "已成交", "已流失"}
	colors := []string{"#909399", "#409EFF", "#67C23A", "#E6A23C", "#52C41A", "#F56C6C"}
	countMap := make(map[string]int)
	amountMap := make(map[string]float64)
	for _, s := range stats {
		countMap[s.Status] = s.Count
		amountMap[s.Status] = s.Amount
	}

	xAxis := []string{}
	counts := []interface{}{}
	amounts := []interface{}{}
	seriesConfig := []map[string]interface{}{}
	for i, status := range statuses {
		xAxis = append(xAxis, status)
		counts = append(counts, countMap[status])
		amounts = append(amounts, amountMap[status])
		seriesConfig = append(seriesConfig, map[string]interface{}{
			"itemStyle": map[string]interface{}{"color": colors[i]},
		})
	}

	c := &chart.BarChart{
		Title: "线索状态分布",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "线索数量", Data: counts},
			{Name: "预计金额(万)", Data: amounts},
		},
		Metadata: map[string]interface{}{
			"总线索数":     len(stats),
			"预计总金额(万)": amountMap["已成交"],
		},
	}
	return resp.Chart(c).Build()
}

var LeadStatusChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "线索状态分布",
		Request:  &LeadStatusChartReq{},
		Response: &chart.BarChart{},
	},
}

type LeadSourceChartReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func LeadSourceAnalysisChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesLead{})

	type SourceStat struct {
		Source string `gorm:"column:source"`
		Count  int    `gorm:"column:count"`
	}
	var stats []SourceStat
	err := queryDB.Select("source, COUNT(*) as count").
		Group("source").Find(&stats).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[LeadSourceAnalysisChart] 查询失败, err: %w", err)
	}

	sources := []string{"线上推广", "展会获取", "电话营销", "老客户推荐", "合作伙伴"}
	colors := []string{"#409EFF", "#67C23A", "#E6A23C", "#F56C6C", "#9C27B0"}
	sourceMap := make(map[string]int)
	for _, s := range stats {
		sourceMap[s.Source] = s.Count
	}

	xAxis := []string{}
	counts := []interface{}{}
	seriesConfig := []map[string]interface{}{}
	for i, src := range sources {
		xAxis = append(xAxis, src)
		counts = append(counts, sourceMap[src])
		seriesConfig = append(seriesConfig, map[string]interface{}{
			"itemStyle": map[string]interface{}{"color": colors[i]},
		})
	}

	c := &chart.BarChart{
		Title: "线索来源分析",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "线索数量", Data: counts},
		},
		Metadata: map[string]interface{}{
			"总线索数": len(stats),
		},
	}
	return resp.Chart(c).Build()
}

var LeadSourceChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "线索来源分析",
		Request:  &LeadSourceChartReq{},
		Response: &chart.BarChart{},
	},
}

// ==================== Helper ====================

func generateLeadNo() string {
	return fmt.Sprintf("LEAD%s%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%10000)
}

// ==================== Init ====================

func init() {
	packageContext.GET("sales_lead_list.table", SalesLeadList, SalesLeadTemplate)
	packageContext.POST("add_lead.form", AddLead, AddLeadTemplate)
	packageContext.POST("batch_import_lead.form", BatchImport, BatchImportTemplate)
	packageContext.GET("lead_status_distribution.chart", LeadStatusDistributionChart, LeadStatusChartTemplate)
	packageContext.GET("lead_source_analysis.chart", LeadSourceAnalysisChart, LeadSourceChartTemplate)
}
