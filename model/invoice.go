package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	InvoiceStatusPending   = 1 // 待开票
	InvoiceStatusCompleted = 2 // 已开票
	InvoiceStatusRejected  = 3 // 已驳回

	InvoiceTitleTypePersonal = 1 // 个人
	InvoiceTitleTypeCompany  = 2 // 企业

	InvoiceOrderTypeTopup        = "topup"
	InvoiceOrderTypeSubscription = "subscription"

	InvoiceFileMaxSize = 10 << 20 // 10MB
)

var (
	ErrInvoiceNotFound      = errors.New("发票申请不存在")
	ErrInvoiceStatusInvalid = errors.New("发票申请状态不允许此操作")
	ErrInvoiceOrderOccupied = errors.New("所选订单已在其他开票申请中")
	ErrInvoiceNoOrders      = errors.New("未选择任何订单")
	ErrInvoiceFileNotFound  = errors.New("发票文件不存在")
)

type Invoice struct {
	Id           int     `json:"id"`
	UserId       int     `json:"user_id" gorm:"index"`
	InvoiceNo    string  `json:"invoice_no" gorm:"type:varchar(64);uniqueIndex"`
	TitleType    int     `json:"title_type"`
	TitleName    string  `json:"title_name" gorm:"type:varchar(255)"`
	TaxNo        string  `json:"tax_no" gorm:"type:varchar(64)"`
	Email        string  `json:"email" gorm:"type:varchar(255)"`
	Money        float64 `json:"money"`
	Status       int     `json:"status" gorm:"index;default:1"`
	RejectReason string  `json:"reject_reason" gorm:"type:text"`
	Remark       string  `json:"remark" gorm:"type:text"`
	CreateTime   int64   `json:"create_time" gorm:"bigint"`
	CompleteTime int64   `json:"complete_time" gorm:"bigint"`

	Username string          `json:"username,omitempty" gorm:"-"`
	Orders   []*InvoiceOrder `json:"orders,omitempty" gorm:"-"`
	HasFile  bool            `json:"has_file" gorm:"-"`
}

type InvoiceOrder struct {
	Id        int     `json:"id"`
	InvoiceId int     `json:"invoice_id" gorm:"index"`
	OrderType string  `json:"order_type" gorm:"type:varchar(32);uniqueIndex:uk_invoice_order,priority:1"`
	OrderId   int     `json:"order_id" gorm:"uniqueIndex:uk_invoice_order,priority:2"`
	TradeNo   string  `json:"trade_no" gorm:"type:varchar(255)"`
	Money     float64 `json:"money"`
}

type InvoiceFile struct {
	Id         int    `json:"id"`
	InvoiceId  int    `json:"invoice_id" gorm:"uniqueIndex"`
	Filename   string `json:"filename" gorm:"type:varchar(255)"`
	MimeType   string `json:"mime_type" gorm:"type:varchar(64)"`
	Size       int64  `json:"size"`
	Data       []byte `json:"-"`
	UploadTime int64  `json:"upload_time" gorm:"bigint"`
}

func GenerateInvoiceNo() string {
	return fmt.Sprintf("INV%d%s", common.GetTimestamp(), strings.ToUpper(common.GetRandomString(8)))
}

// InvoiceableOrder 可开票订单（充值/订阅统一视图）
type InvoiceableOrder struct {
	OrderType  string  `json:"order_type"`
	OrderId    int     `json:"order_id"`
	TradeNo    string  `json:"trade_no"`
	Money      float64 `json:"money"`
	CreateTime int64   `json:"create_time"`
}

// CreateInvoiceWithOrders 事务创建申请并占用订单；金额服务端累加。
// 撞 (order_type, order_id) 唯一索引时返回 ErrInvoiceOrderOccupied。
func CreateInvoiceWithOrders(invoice *Invoice, orders []*InvoiceOrder) error {
	if len(orders) == 0 {
		return ErrInvoiceNoOrders
	}
	invoice.InvoiceNo = GenerateInvoiceNo()
	invoice.Status = InvoiceStatusPending
	invoice.CreateTime = common.GetTimestamp()
	invoice.Money = 0
	for _, o := range orders {
		invoice.Money += o.Money
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}
		for _, o := range orders {
			o.InvoiceId = invoice.Id
			if err := tx.Create(o).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// 三库的唯一约束错误文案不同，统一按占用处理。
		// 注：invoice_no 撞唯一索引（时间戳+8位随机，概率极低）也会落入此分支，可接受。
		if strings.Contains(strings.ToLower(err.Error()), "unique") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return ErrInvoiceOrderOccupied
		}
		return err
	}
	return nil
}

func GetInvoiceById(id int) (*Invoice, error) {
	var inv Invoice
	if err := DB.First(&inv, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	var orders []*InvoiceOrder
	if err := DB.Where("invoice_id = ?", id).Find(&orders).Error; err != nil {
		return nil, err
	}
	inv.Orders = orders
	var fileCnt int64
	if err := DB.Model(&InvoiceFile{}).Where("invoice_id = ?", id).Count(&fileCnt).Error; err != nil {
		return nil, err
	}
	inv.HasFile = fileCnt > 0
	return &inv, nil
}

// CancelInvoice 用户撤销：仅本人、仅待开票；物理删除申请与关联（释放订单）
func CancelInvoice(id int, userId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var inv Invoice
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&inv, "id = ? AND user_id = ?", id, userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceNotFound
			}
			return err
		}
		if inv.Status != InvoiceStatusPending {
			return ErrInvoiceStatusInvalid
		}
		if err := tx.Where("invoice_id = ?", id).Delete(&InvoiceOrder{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Invoice{}, id).Error
	})
}

// RejectInvoice 管理员驳回：仅待开票；释放订单、记录原因
func RejectInvoice(id int, reason string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var inv Invoice
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&inv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceNotFound
			}
			return err
		}
		if inv.Status != InvoiceStatusPending {
			return ErrInvoiceStatusInvalid
		}
		if err := tx.Where("invoice_id = ?", id).Delete(&InvoiceOrder{}).Error; err != nil {
			return err
		}
		return tx.Model(&Invoice{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":        InvoiceStatusRejected,
			"reject_reason": reason,
			"complete_time": common.GetTimestamp(),
		}).Error
	})
}

// CompleteInvoiceWithFile 管理员上传发票：待开票→已开票；已开票可重新上传（覆盖文件）
func CompleteInvoiceWithFile(id int, filename string, mimeType string, data []byte) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var inv Invoice
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&inv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvoiceNotFound
			}
			return err
		}
		if inv.Status != InvoiceStatusPending && inv.Status != InvoiceStatusCompleted {
			return ErrInvoiceStatusInvalid
		}
		if err := tx.Where("invoice_id = ?", id).Delete(&InvoiceFile{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&InvoiceFile{InvoiceId: id, Filename: filename, MimeType: mimeType,
			Size: int64(len(data)), Data: data, UploadTime: common.GetTimestamp()}).Error; err != nil {
			return err
		}
		return tx.Model(&Invoice{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":        InvoiceStatusCompleted,
			"complete_time": common.GetTimestamp(),
		}).Error
	})
}

func GetInvoiceFile(invoiceId int) (*InvoiceFile, error) {
	var f InvoiceFile
	if err := DB.First(&f, "invoice_id = ?", invoiceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceFileNotFound
		}
		return nil, err
	}
	return &f, nil
}

func GetUserInvoices(userId int, pageInfo *common.PageInfo) ([]*Invoice, int64, error) {
	var list []*Invoice
	var total int64
	q := DB.Model(&Invoice{}).Where("user_id = ?", userId)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	fillInvoiceHasFile(list)
	return list, total, nil
}

// GetAllInvoices 管理端列表；status=0 表示全部；keyword 匹配申请单号/抬头
func GetAllInvoices(keyword string, status int, pageInfo *common.PageInfo) ([]*Invoice, int64, error) {
	var list []*Invoice
	var total int64
	q := DB.Model(&Invoice{})
	if status != 0 {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("invoice_no LIKE ? OR title_name LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// 待开票置顶，其余按 id 倒序
	err := q.Order("status asc").Order("id desc").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	fillInvoiceUsernames(list)
	fillInvoiceHasFile(list)
	return list, total, nil
}

func fillInvoiceUsernames(list []*Invoice) {
	if len(list) == 0 {
		return
	}
	idSet := make(map[int]bool)
	for _, inv := range list {
		idSet[inv.UserId] = true
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var users []*User
	if err := DB.Select("id, username").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return
	}
	nameMap := make(map[int]string, len(users))
	for _, u := range users {
		nameMap[u.Id] = u.Username
	}
	for _, inv := range list {
		inv.Username = nameMap[inv.UserId]
	}
}

func fillInvoiceHasFile(list []*Invoice) {
	if len(list) == 0 {
		return
	}
	ids := make([]int, 0, len(list))
	for _, inv := range list {
		ids = append(ids, inv.Id)
	}
	var files []*InvoiceFile
	if err := DB.Select("invoice_id").Where("invoice_id IN ?", ids).Find(&files).Error; err != nil {
		return
	}
	fset := make(map[int]bool, len(files))
	for _, f := range files {
		fset[f.InvoiceId] = true
	}
	for _, inv := range list {
		inv.HasFile = fset[inv.Id]
	}
}

// GetInvoiceableOrders 用户可开票订单：支付成功、金额>0、未被任何申请占用
func GetInvoiceableOrders(userId int) ([]*InvoiceableOrder, error) {
	result := make([]*InvoiceableOrder, 0)

	occupiedTopup := DB.Model(&InvoiceOrder{}).Select("order_id").
		Where("order_type = ?", InvoiceOrderTypeTopup)
	var topups []*TopUp
	err := DB.Where("user_id = ? AND status = ? AND money > 0", userId, common.TopUpStatusSuccess).
		Where("id NOT IN (?)", occupiedTopup).
		Order("id desc").Find(&topups).Error
	if err != nil {
		return nil, err
	}
	for _, tp := range topups {
		result = append(result, &InvoiceableOrder{OrderType: InvoiceOrderTypeTopup,
			OrderId: tp.Id, TradeNo: tp.TradeNo, Money: tp.Money, CreateTime: tp.CreateTime})
	}

	occupiedSub := DB.Model(&InvoiceOrder{}).Select("order_id").
		Where("order_type = ?", InvoiceOrderTypeSubscription)
	var subs []*SubscriptionOrder
	err = DB.Where("user_id = ? AND status = ? AND money > 0", userId, common.TopUpStatusSuccess).
		Where("id NOT IN (?)", occupiedSub).
		Order("id desc").Find(&subs).Error
	if err != nil {
		return nil, err
	}
	for _, so := range subs {
		result = append(result, &InvoiceableOrder{OrderType: InvoiceOrderTypeSubscription,
			OrderId: so.Id, TradeNo: so.TradeNo, Money: so.Money, CreateTime: so.CreateTime})
	}
	return result, nil
}
