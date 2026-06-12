package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
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
