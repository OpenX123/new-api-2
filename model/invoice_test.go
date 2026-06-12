package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

// migrateInvoiceTables 幂等迁移本功能所需表（TestMain 的列表不含发票表）
func migrateInvoiceTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Invoice{}, &InvoiceOrder{}, &InvoiceFile{}, &TopUp{}, &SubscriptionOrder{}))
}

func clearInvoiceTables(t *testing.T) {
	t.Helper()
	migrateInvoiceTables(t)
	require.NoError(t, DB.Exec("DELETE FROM invoice_files").Error)
	require.NoError(t, DB.Exec("DELETE FROM invoice_orders").Error)
	require.NoError(t, DB.Exec("DELETE FROM invoices").Error)
	require.NoError(t, DB.Exec("DELETE FROM top_ups").Error)
	require.NoError(t, DB.Exec("DELETE FROM subscription_orders").Error)
}

func TestInvoiceOrderUniqueIndex(t *testing.T) {
	clearInvoiceTables(t)
	inv := &Invoice{UserId: 1, InvoiceNo: "INV-T1", TitleType: InvoiceTitleTypePersonal,
		TitleName: "张三", Email: "a@b.c", Money: 10, Status: InvoiceStatusPending, CreateTime: 1}
	require.NoError(t, DB.Create(inv).Error)
	o1 := &InvoiceOrder{InvoiceId: inv.Id, OrderType: InvoiceOrderTypeTopup, OrderId: 100, TradeNo: "T100", Money: 10}
	require.NoError(t, DB.Create(o1).Error)
	// 同一订单再次挂接必须违反唯一索引
	o2 := &InvoiceOrder{InvoiceId: inv.Id + 999, OrderType: InvoiceOrderTypeTopup, OrderId: 100, TradeNo: "T100", Money: 10}
	require.Error(t, DB.Create(o2).Error)
}

func seedSuccessTopup(t *testing.T, id, userId int, money float64) {
	t.Helper()
	require.NoError(t, DB.Create(&TopUp{Id: id, UserId: userId, Money: money,
		TradeNo: fmt.Sprintf("TP%d", id), Status: common.TopUpStatusSuccess, CreateTime: 1000}).Error)
}

func TestCreateInvoiceWithOrders(t *testing.T) {
	clearInvoiceTables(t)
	seedSuccessTopup(t, 1, 7, 10.5)
	seedSuccessTopup(t, 2, 7, 4.5)

	inv := &Invoice{UserId: 7, TitleType: InvoiceTitleTypeCompany, TitleName: "X公司",
		TaxNo: "91xxx", Email: "x@y.z"}
	orders := []*InvoiceOrder{
		{OrderType: InvoiceOrderTypeTopup, OrderId: 1, TradeNo: "TP1", Money: 10.5},
		{OrderType: InvoiceOrderTypeTopup, OrderId: 2, TradeNo: "TP2", Money: 4.5},
	}
	require.NoError(t, CreateInvoiceWithOrders(inv, orders))
	require.Equal(t, 15.0, inv.Money)         // 服务端累加
	require.Equal(t, InvoiceStatusPending, inv.Status)
	require.NotEmpty(t, inv.InvoiceNo)

	// 同一订单重复申请必须失败
	inv2 := &Invoice{UserId: 7, TitleType: 1, TitleName: "张三", Email: "x@y.z"}
	err := CreateInvoiceWithOrders(inv2, []*InvoiceOrder{
		{OrderType: InvoiceOrderTypeTopup, OrderId: 1, TradeNo: "TP1", Money: 10.5},
	})
	require.ErrorIs(t, err, ErrInvoiceOrderOccupied)
	// 失败时不残留主记录
	var cnt int64
	require.NoError(t, DB.Model(&Invoice{}).Count(&cnt).Error)
	require.EqualValues(t, 1, cnt)
}

func TestCreateInvoiceWithOrdersEmptyGuard(t *testing.T) {
	// 空订单列表
	require.ErrorIs(t, CreateInvoiceWithOrders(&Invoice{UserId: 7, TitleName: "x", Email: "a@b.c"}, nil), ErrInvoiceNoOrders)
}

func TestGetInvoiceableOrders(t *testing.T) {
	clearInvoiceTables(t)
	seedSuccessTopup(t, 1, 7, 10)
	seedSuccessTopup(t, 2, 7, 20)
	seedSuccessTopup(t, 3, 8, 30) // 其他用户
	require.NoError(t, DB.Create(&TopUp{Id: 4, UserId: 7, Money: 40, TradeNo: "TP4",
		Status: common.TopUpStatusPending, CreateTime: 1000}).Error) // 未支付
	require.NoError(t, DB.Create(&SubscriptionOrder{Id: 9, UserId: 7, Money: 9.9,
		TradeNo: "SUB9", Status: common.TopUpStatusSuccess, CreateTime: 1000}).Error)

	// 占用订单 1
	inv := &Invoice{UserId: 7, TitleType: 1, TitleName: "张三", Email: "a@b.c"}
	require.NoError(t, CreateInvoiceWithOrders(inv, []*InvoiceOrder{
		{OrderType: InvoiceOrderTypeTopup, OrderId: 1, TradeNo: "TP1", Money: 10},
	}))

	list, err := GetInvoiceableOrders(7)
	require.NoError(t, err)
	require.Len(t, list, 2) // topup#2 + sub#9；不含他人/未支付/已占用
	keys := map[string]bool{}
	for _, o := range list {
		keys[o.OrderType+fmt.Sprint(o.OrderId)] = true
	}
	require.True(t, keys["topup2"])
	require.True(t, keys["subscription9"])
}

func mustCreateInvoice(t *testing.T, userId int, orderId int) *Invoice {
	t.Helper()
	seedSuccessTopup(t, orderId, userId, 10)
	inv := &Invoice{UserId: userId, TitleType: 1, TitleName: "张三", Email: "a@b.c"}
	require.NoError(t, CreateInvoiceWithOrders(inv, []*InvoiceOrder{
		{OrderType: InvoiceOrderTypeTopup, OrderId: orderId, TradeNo: fmt.Sprintf("TP%d", orderId), Money: 10},
	}))
	return inv
}

func TestCancelInvoice(t *testing.T) {
	clearInvoiceTables(t)
	inv := mustCreateInvoice(t, 7, 1)
	// 他人不能撤销
	require.ErrorIs(t, CancelInvoice(inv.Id, 999), ErrInvoiceNotFound)
	// 本人撤销成功，订单释放
	require.NoError(t, CancelInvoice(inv.Id, 7))
	list, _ := GetInvoiceableOrders(7)
	require.Len(t, list, 1)
	var cnt int64
	require.NoError(t, DB.Model(&InvoiceOrder{}).Count(&cnt).Error)
	require.EqualValues(t, 0, cnt)
}

func TestRejectAndCompleteInvoice(t *testing.T) {
	clearInvoiceTables(t)
	inv := mustCreateInvoice(t, 7, 1)

	// 驳回：必须释放订单、记录原因
	require.NoError(t, RejectInvoice(inv.Id, "信息不全"))
	got, err := GetInvoiceById(inv.Id)
	require.NoError(t, err)
	require.Equal(t, InvoiceStatusRejected, got.Status)
	require.Equal(t, "信息不全", got.RejectReason)
	list, _ := GetInvoiceableOrders(7)
	require.Len(t, list, 1) // 已释放
	// 已驳回不能再驳回/完成/撤销
	require.ErrorIs(t, RejectInvoice(inv.Id, "again"), ErrInvoiceStatusInvalid)
	require.ErrorIs(t, CompleteInvoiceWithFile(inv.Id, "a.pdf", "application/pdf", []byte("%PDF")), ErrInvoiceStatusInvalid)
	require.ErrorIs(t, CancelInvoice(inv.Id, 7), ErrInvoiceStatusInvalid)

	// 重新申请 → 上传完成
	inv2 := mustCreateInvoice(t, 7, 2)
	require.NoError(t, CompleteInvoiceWithFile(inv2.Id, "fp.pdf", "application/pdf", []byte("%PDF-1.7")))
	got2, _ := GetInvoiceById(inv2.Id)
	require.Equal(t, InvoiceStatusCompleted, got2.Status)
	require.NotZero(t, got2.CompleteTime)
	f, err := GetInvoiceFile(inv2.Id)
	require.NoError(t, err)
	require.Equal(t, "fp.pdf", f.Filename)
	require.EqualValues(t, 8, f.Size)
	// 已开票可重新上传（换票），文件覆盖
	require.NoError(t, CompleteInvoiceWithFile(inv2.Id, "fp2.pdf", "application/pdf", []byte("%PDF-1.7-v2")))
	f2, _ := GetInvoiceFile(inv2.Id)
	require.Equal(t, "fp2.pdf", f2.Filename)
	var fileCnt int64
	require.NoError(t, DB.Model(&InvoiceFile{}).Count(&fileCnt).Error)
	require.EqualValues(t, 1, fileCnt)
	// 已开票不能撤销
	require.ErrorIs(t, CancelInvoice(inv2.Id, 7), ErrInvoiceStatusInvalid)
}

func TestInvoiceQueries(t *testing.T) {
	clearInvoiceTables(t)
	// seed users for username assertion; clean up at end to avoid polluting other tests
	require.NoError(t, DB.Exec("INSERT INTO users (id, username, password, display_name, role, status) VALUES (?, ?, ?, ?, ?, ?)", 7, "u7", "placeholder", "u7", 1, 1).Error)
	require.NoError(t, DB.Exec("INSERT INTO users (id, username, password, display_name, role, status) VALUES (?, ?, ?, ?, ?, ?)", 8, "u8", "placeholder", "u8", 1, 1).Error)
	defer func() {
		DB.Exec("DELETE FROM users WHERE id IN (7, 8)")
	}()

	mustCreateInvoice(t, 7, 1)
	mustCreateInvoice(t, 8, 2)
	page := &common.PageInfo{Page: 1, PageSize: 10}
	mine, total, err := GetUserInvoices(7, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, mine, 1)
	all, totalAll, err := GetAllInvoices("", 0, page)
	require.NoError(t, err)
	require.EqualValues(t, 2, totalAll)
	require.Len(t, all, 2)
	pending, totalPending, err := GetAllInvoices("", InvoiceStatusPending, page)
	require.NoError(t, err)
	require.EqualValues(t, 2, totalPending)
	require.Len(t, pending, 2)

	// assert usernames are populated
	usernameMap := map[int]string{}
	for _, inv := range all {
		usernameMap[inv.UserId] = inv.Username
	}
	require.Equal(t, "u7", usernameMap[7])
	require.Equal(t, "u8", usernameMap[8])
}
